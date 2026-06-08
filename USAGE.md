# 사용법 (USAGE)

`html-to-pdf-go-server-report.md` 의 결론을 실제로 검증하기 위한 PoC 샘플의 사용 매뉴얼입니다. 동일한 데이터를 **Gotenberg · WeasyPrint · DocRaptor · ReactPdf** 네 엔진에 보내 PDF를 비교합니다.

> ⚠️ ReactPdf 만 입력 형식이 다릅니다 — HTML 이 아닌 Invoice JSON 을 받습니다. 측정 결과 해석 시 README.md 의 "비교축 차이" 단서 참조.

- 작성 기준일: 2026-05-24
- 대상: Go 서버에서 HTML → PDF 변환 도구 선정 PoC 담당자
- 사전 지식: Go, Docker, 약간의 HTTP/JSON

---

## 1. 빠른 시작 (5분)

```bash
# 1) 사이드카 컨테이너 빌드·기동
make up

# 2) 헬스 체크
make health
# → Gotenberg / WeasyPrint / ReactPdf 모두 200 OK 가 떠야 다음 단계로 진행

# 3-A) 배치 모드: 3 로케일 × 4 백엔드 = PDF 12개 생성
make run                    # output/{ko,ja,en}-{gotenberg,weasyprint,docraptor,reactpdf}.pdf

# 3-B) 또는 브라우저 데모
make serve                  # → http://localhost:8080
```

처음 `make up` 은 WeasyPrint 이미지 빌드(Noto CJK · IPAex 폰트 포함) 때문에 약 2분 걸립니다. 두 번째부터는 캐시되어 수 초.

---

## 2. 두 가지 실행 모드

### 2-1. CLI 모드 (배치 변환)

`output/` 디렉터리에 PDF 파일로 떨어뜨리는 용도. CI나 회귀 테스트, 결과 보관에 적합합니다.

```bash
# 전체 (기본값: locale=all, backend=all)
go run . -dump-html

# 특정 로케일만
go run . -locale ja

# 특정 백엔드만
go run . -backend weasyprint

# 조합
go run . -locale ja,ko -backend gotenberg,weasyprint

# 출력 디렉터리 변경
go run . -out /tmp/htp-output
```

콘솔에는 로케일·백엔드별 변환 시간과 결과 PDF 크기가 함께 찍힙니다:

```
[ja/gotenberg]  OK 412ms (87 KB)
[ja/weasyprint] OK 678ms (92 KB)
[ja/docraptor]  OK 1.4s  (88 KB)
```

### 2-2. 서버 모드 (브라우저 3-패널 데모)

```bash
make serve              # 기본 :8080
# 또는 포트 변경
go run . -serve -addr :9000
```

브라우저에서 `http://localhost:8080` 열면:

```
┌──────────────────┬──────────────────────┬──────────────────────┐
│ ① 데이터 (JSON)   │ ② HTML 미리보기       │ ③ PDF 결과            │
│ [편집 가능]       │ [iframe srcdoc]       │ [iframe PDF]          │
│                  │                       │                       │
│ → HTML 렌더링    │ Gotenberg|Weasy|Doc   │ "weasyprint 487ms"    │
└──────────────────┴──────────────────────┴──────────────────────┘
```

- 상단 로케일 셀렉터로 `ko/ja/en` 샘플 데이터를 즉시 교체
- 가운데 패널 하단 3개 버튼이 동일 HTML을 세 백엔드에 보내 ③ 패널에 PDF 표시
- 변환 시간(ms)·PDF 크기(KB) 가 ③ 패널 하단 상태바에 누적 표시되어 즉석 비교

---

## 3. 시나리오별 검증 방법

보고서 본문과 매핑된 핵심 비교 시나리오입니다.

### 시나리오 A. 표 페이지 나눔 / thead 반복 (보고서 §1)

샘플 JSON에는 25줄짜리 라인아이템이 들어 있어 자동으로 표가 2~3 페이지로 분할됩니다.

```bash
make run-ja
open output/ja-gotenberg.pdf output/ja-weasyprint.pdf output/ja-docraptor.pdf
```

확인 포인트:
- 2페이지 이후에도 `thead` (#, 품목, 수량, 단가, 금액) 가 정확히 반복되는가
- 페이지 경계에서 `border-bottom` 이 깨지지 않는가 (border-collapse 버그)
- 행이 셀 중간에서 끊겨 다음 페이지로 이어지지 않는가 (`break-inside: avoid`)

### 시나리오 B. JIS X 0213 한자 글리프 (보고서 §2)

`data/ja.json` 의 "髙﨑𠮷田商事 株式会社" — 자주 누락되는 일본 인명·상호용 한자.

```bash
go run . -locale ja -backend all -dump-html
# output/ja.html 을 텍스트 에디터로 열어 원본에 글리프가 있음을 확인 후
# output/ja-*.pdf 세 개를 비교
```

확인 포인트:
- "髙﨑𠮷田" 4글자가 모두 정상 렌더링되는가
- 글리프 누락 시 사각형(.notdef) / 다른 폰트 fallback 으로 표시되는가
- 영문 폰트 fallback 으로 인한 행 높이 흔들림이 발생하는가

### 시나리오 C. 통화 정렬 (보고서 §2)

`tabular-nums` + `white-space: nowrap` 처리 검증. ¥ / ₩ / $ 가 우측 정렬 시 정수부 자릿수가 변해도 소수점/심볼이 흔들리지 않아야 합니다.

```bash
make run                # 9개 PDF
# ko-*.pdf, ja-*.pdf, en-*.pdf 의 "금액" 열 우측 정렬 라인 일치 확인
```

### 시나리오 D. 변환 성능 측정 (보고서 §6 PoC 체크리스트)

```bash
make serve &
# 같은 HTML을 100회 변환 (예: GNU parallel)
seq 1 100 | parallel -j 10 'curl -s -X POST \
  -H "Content-Type: text/html" \
  --data-binary @output/ja.html \
  "http://localhost:8080/api/convert?backend=weasyprint" \
  -o /dev/null -w "%{time_total}\n"' | \
  awk '{s+=$1; if($1>m)m=$1} END{print "avg=" s/NR " max=" m}'
```

목표(보고서 §6): **1매 1초 이내, 동시 50요청 부하 대응**.

CPU/메모리는 다른 창에서:
```bash
docker stats htp-gotenberg htp-weasyprint
```

### 시나리오 E. 자체 데이터로 변환

`data/` 에 새 JSON 파일을 추가하거나 기존 파일을 수정합니다. 스키마는 `data/ko.json` 참조 (`Invoice` 구조체와 1:1):

```json
{
  "locale": "ko",
  "currency": "KRW",
  "invoice_number": "...",
  "issue_date": "...",
  "due_date": "...",
  "tax_rate": 10,
  "from": { "name": "...", "address": "...", "tax_id": "...", "contact": "..." },
  "to":   { "name": "...", "address": "...", "tax_id": "...", "contact": "..." },
  "items": [ { "description": "...", "sku": "...", "quantity": 1, "unit_price": 100, "amount": 100 } ],
  "notes": "..."
}
```

- `locale`: `ko` / `ja` / `en` 중 하나 (라벨 다국어 매핑이 이 키 기준)
- `currency`: `KRW` / `JPY` / `USD` (심볼·소수점 자리수 자동)
- `subtotal` / `tax` / `total` 은 코드가 재계산하므로 JSON에 없어도 됨

서버 모드라면 그냥 ① 패널에서 JSON을 직접 편집 → "HTML 렌더링" 버튼 → 그대로 PDF 변환 흐름이 더 빠릅니다.

---

## 4. 옵션 레퍼런스

### CLI 플래그

| 플래그 | 기본값 | 설명 |
|---|---|---|
| `-serve` | `false` | HTTP 서버 모드 |
| `-addr` | `:8080` | 서버 모드 리스닝 주소 |
| `-locale` | `all` | `ko` / `ja` / `en` / `all` / 콤마 구분 |
| `-backend` | `all` | `gotenberg` / `weasyprint` / `docraptor` / `reactpdf` / `all` / 콤마 구분 |
| `-template` | `templates/invoice.html.tmpl` | 템플릿 경로 |
| `-data` | `data` | `{locale}.json` 디렉터리 |
| `-out` | `output` | PDF 출력 디렉터리 |
| `-dump-html` | `false` | 렌더된 HTML 도 `output/{locale}.html` 로 저장 |
| `-gotenberg-url` | `http://localhost:3000` | Gotenberg 베이스 URL |
| `-weasyprint-url` | `http://localhost:5001` | WeasyPrint 베이스 URL |
| `-reactpdf-url` | `http://localhost:5002` | ReactPdf 사이드카 베이스 URL |
| `-docraptor-key` | `YOUR_API_KEY_HERE` | DocRaptor API 키 |
| `-docraptor-test` | `true` | DocRaptor test 모드 (무료/워터마크) |

### 환경 변수 (CLI 플래그와 동등, 플래그가 우선)

| 환경변수 | 대응 플래그 |
|---|---|
| `GOTENBERG_URL` | `-gotenberg-url` |
| `WEASYPRINT_URL` | `-weasyprint-url` |
| `DOCRAPTOR_API_KEY` | `-docraptor-key` |
| `DOCRAPTOR_TEST` | `-docraptor-test` (`true` / `false`) |

### DocRaptor 키 운용

기본값은 **test 모드 + 더미 키** 라 계정 없이 변환이 됩니다. 출력 PDF 전체에 "TEST" 워터마크가 박힙니다 (Prince 엔진의 실제 렌더링 품질은 그대로 확인 가능).

워터마크 없는 결과가 필요할 때:

```bash
export DOCRAPTOR_API_KEY="실제-키"
export DOCRAPTOR_TEST=false
make run-docraptor
```

DocRaptor 무료 플랜은 월 5건 (워터마크 없음). 그 이상은 유료. PoC 단계에서는 test 모드로 충분합니다.

---

## 5. HTTP API 레퍼런스 (서버 모드)

| 메서드 | 경로 | 요청 | 응답 |
|---|---|---|---|
| `GET` | `/` | — | 데모 페이지 (`web/index.html`) |
| `GET` | `/api/sample?locale=ko` | — | `application/json` — 해당 로케일 샘플 |
| `POST` | `/api/render` | `application/json` (Invoice JSON) | `text/html` — 렌더된 HTML |
| `POST` | `/api/convert?backend=gotenberg` | `text/html` | `application/pdf` |
| `GET` | `/api/health` | — | `{"ok":true,"backends":[...]}` |

응답 헤더:
- `X-Convert-Backend`: 사용된 백엔드 이름
- `X-Convert-Elapsed-Ms`: 변환 소요 시간(ms)

예시 (curl):

```bash
# 샘플 가져와서 그대로 PDF 변환
curl -s "http://localhost:8080/api/sample?locale=ja" \
  | curl -s -X POST -H "Content-Type: application/json" --data-binary @- \
         "http://localhost:8080/api/render" \
  | curl -s -X POST -H "Content-Type: text/html" --data-binary @- \
         "http://localhost:8080/api/convert?backend=weasyprint" \
  -o out.pdf
```

---

## 6. 디렉터리 구조

```
html-to-pdf/
├── README.md                     개요 / 비교 매트릭스
├── USAGE.md                      ← 이 파일
├── html-to-pdf-go-server-report.md   PoC 의 근거 보고서
├── docker-compose.yml            Gotenberg + WeasyPrint
├── weasyprint/                   WeasyPrint Flask 래퍼 (커스텀 이미지)
│   ├── Dockerfile
│   └── server.py
├── templates/
│   └── invoice.html.tmpl         다국어 청구서 (Go text/template, CSS Paged Media)
├── data/
│   ├── ko.json                   한국어 (₩, 사업자번호)
│   ├── ja.json                   일본어 (¥, JIS X 0213 髙﨑𠮷, T번호)
│   └── en.json                   영어  ($, EIN)
├── web/
│   └── index.html                3-패널 브라우저 데모 (embed.FS)
├── output/                       PDF/HTML 출력 (gitignore)
├── main.go                       CLI 진입점 + Backend 어댑터
├── serve.go                      HTTP 서버 + 데모 API
├── render_test.go                템플릿/통화 테스트
├── go.mod
└── Makefile
```

---

## 7. 트러블슈팅

### Gotenberg 변환 시 한자/한글이 두부(豆腐) 또는 사각형으로 나옴

Gotenberg 기본 이미지에 CJK 폰트가 없어서 발생. 현재 템플릿은 Google Fonts CDN(`@import` Noto Sans KR/JP)을 끌어와 우회하므로 **컨테이너에서 outbound HTTPS** 가 가능해야 합니다.

오프라인 환경이라면 커스텀 Dockerfile 로 `fonts-noto-cjk` 를 추가하세요:

```dockerfile
FROM gotenberg/gotenberg:8
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
        fonts-noto-cjk fonts-ipaexfont \
    && rm -rf /var/lib/apt/lists/*
USER gotenberg
```

### WeasyPrint 빌드 실패 (`fonts-ipaexfont` 패키지 없음)

배포판 버전 차이. `weasyprint/Dockerfile` 에서 해당 라인을 제거하고 다시 빌드. IPAex 가 없으면 일부 JIS X 0213 글리프 fallback 만 영향받습니다.

### DocRaptor 호출이 403/timeout

- 사내 프록시·방화벽이 `docraptor.com` 으로의 outbound HTTPS 를 막고 있는지 확인
- `-docraptor-key` 값이 빈 문자열이면 test 모드라도 403 가 떨어짐 (기본 placeholder 사용)

### 변환 직후 PDF 가 비어 보임 / 깨짐

- `Content-Type` 헤더 확인: convert API 는 `text/html` 만 받음
- HTML 안에서 외부 리소스(`<img src=>`) 로딩이 실패하면 WeasyPrint 는 해당 영역만 비우고 계속 진행하지만 Gotenberg 는 페이지 통째로 비어 보일 수 있음 → 네트워크 접근 가능성 확인

### `make up` 후에도 컨테이너가 떠 있지 않음

```bash
docker compose logs gotenberg
docker compose logs weasyprint
```

대부분 포트 충돌(`3000`, `5001`)이 원인. `docker-compose.yml` 에서 외부 포트만 바꾸고, 그에 맞춰 `-gotenberg-url` / `-weasyprint-url` 도 같이 조정.

### 동시 요청 시 WeasyPrint 가 느려짐

보고서 §3 에 명시된 **CJK 폰트 사용 시 6배 저하 (Issue #2120)** 그대로 재현. 대응:
- WeasyPrint 컨테이너를 여러 개 띄우고 앞단에 로드밸런서
- 폰트 캐시 워밍업 후 측정
- 단일 인스턴스에서 무리하지 말고 동시성 워커 수를 보수적으로 산정

---

## 8. FAQ

**Q. 후처리(PDF/A, 디지털 서명, 타임스탬프)도 이 샘플에서 검증되나요?**
A. 아닙니다. 본 샘플은 보고서 **시나리오 A (변환 품질)** 검증용입니다. 시나리오 B (UniDoc UniPDF 후처리, RFC 3161 TSA, PAdES) 는 별도 PoC 가지에서 진행하세요.

**Q. 폰트를 로컬에 둘 수 있나요?**
A. 가능. `weasyprint/Dockerfile` 에 폰트 파일을 `COPY` 하고 `fontconfig` 캐시를 갱신한 뒤, 템플릿의 `@font-face` 를 `url(file:///fonts/...)` 로 바꾸면 됩니다. Gotenberg 도 동일한 방식으로 커스텀 이미지에 폰트를 묶을 수 있습니다.

**Q. 변환된 PDF 가 보고서의 표 어긋남 문제를 보여주지 않습니다.**
A. 25줄 데이터로는 약한 케이스도 있습니다. `data/ja.json` 의 `items` 를 50~80줄로 늘리고, 일부 행에 `<br>` 가 포함된 긴 description 을 섞으면 페이지 경계 버그가 더 잘 드러납니다.

**Q. 서버 모드를 프로덕션에 그대로 쓸 수 있나요?**
A. 권장하지 않습니다. 이 샘플은 PoC 비교 용도라 인증/레이트리밋/입력 사이즈 제한이 없습니다. 프로덕션 통합 시에는 변환 호출을 내부 워커 큐 뒤에 두는 형태로 분리하세요.

**Q. 다른 템플릿(견적서 등)을 추가하려면?**
A. `templates/` 에 새 `.tmpl` 을 만들고 `-template` 플래그로 지정. 현재 코드 흐름은 단일 템플릿을 가정하므로, 다중 템플릿이 필요하면 `main.go` 의 `renderHTML` 분기점만 늘리면 됩니다.
