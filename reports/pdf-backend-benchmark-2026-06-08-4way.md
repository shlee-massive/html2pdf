# PDF 백엔드 4-way 벤치마크 — 2026-06-08

> 이전 [3-way 벤치마크 (2026-05-26)](./pdf-backend-benchmark-2026-05-26.md) 에 **`@react-pdf/renderer` 사이드카** 를 4번째 백엔드로 추가한 후속 측정.
>
> 측정 데이터: `reports/matrix-4way-2026-06-08.csv` (39 셀, 헤더 포함 40 행)
> 측정 환경: macOS Darwin 25.1, Docker Desktop, baseURL `http://localhost:8090`
> 측정 시간: 8분 56초

---

## 0. 한 줄 결론

이전 결론 (Gotenberg 1순위, WeasyPrint 정확도 우선, DocRaptor zero-ops) 은 **유지**. `@react-pdf/renderer` 는 **invoice 시나리오 한정** 측정에서 Gotenberg 와 동등 또는 우세한 baseline (177~282ms) 을 보였으나, **HTML 을 받지 않는 패러다임 차이** 때문에 다른 3 백엔드와 직접 비교축이 다르다. 운영 도입 검토 시 _보고서 종류가 1~2개로 고정_ 이고 _디자인을 코드로 관리하기 원할 때_ 만 후보로 고려 가능.

---

## 1. 측정 범위 변경 사항

| 항목 | 2026-05-26 (이전) | 2026-06-08 (본 측정) |
|------|-------------------|----------------------|
| 백엔드 수 | 3 (Gotenberg / WeasyPrint / DocRaptor) | **4** (+ ReactPdf) |
| 시나리오 | 18 (invoice + nested-* × 3 로케일) | 동일 |
| 동시성 | conc=4, total=8 | 동일 |
| 측정 셀 | 54 | **39 측정 + 15 skip** (reactpdf 는 nested-* 입력 불가) |
| DocRaptor | 측정 | **본 측정에선 제외** (test API 키 미설정) |
| 전송 입력 | 모두 HTML | reactpdf 만 Invoice JSON, 나머지 HTML |

ReactPdf 만 `application/json` body 를 `:5002/pdf` 에 직접 POST (HTTP 서버 우회). 다른 백엔드는 기존대로 `:8090/api/convert?backend=X` 에 HTML body POST.

⚠️ **비교축 차이**: ReactPdf 칸의 측정값은 "**같은 콘텐츠를 다른 패러다임으로 만든 비용**" 이지 "동일 HTML 변환 비용" 이 아님. CSV 의 `size_bytes` 컬럼이 입력 형식 차이를 그대로 노출 (reactpdf 3~4KB JSON vs 나머지 12~26KB HTML).

---

## 2. 결과 매트릭스 — baseline (단일 요청, ms)

| template | lng | gotenberg | weasyprint | **reactpdf** | 입력 KB (HTML/JSON) |
|----------|-----|----------:|-----------:|--------------:|---------------------|
| invoice | ko | 252 | **4,383** ⚠️ | **208** | 12 / 4 |
| invoice | ja | 203 | **4,652** ⚠️ | 282 | 12 / 4 |
| invoice | en | 190 | **4,072** ⚠️ | **177** | 12 / 4 |
| nested-basic | ko | 190 | 3,828 | _skip_ | 5 |
| nested-basic | ja | 159 | 6,868 | _skip_ | 5 |
| nested-basic | en | 79 | **96** | _skip_ | 5 |
| nested-deep | ko | 190 | 3,850 | _skip_ | 13 |
| nested-deep | ja | 198 | 3,459 | _skip_ | 13 |
| nested-deep | en | 76 | 334 | _skip_ | 13 |
| nested-split | ko | 143 | 3,252 | _skip_ | 14 |
| nested-split | ja | 161 | 3,846 | _skip_ | 14 |
| nested-split | en | 87 | 768 | _skip_ | 13 |
| nested-projects | ko | 143 | 3,645 | _skip_ | 21 |
| nested-projects | ja | 168 | 3,822 | _skip_ | 22 |
| nested-projects | en | 86 | 350 | _skip_ | 21 |
| nested-stress | ko | 153 | 4,915 | _skip_ | 26 |
| nested-stress | ja | 243 | **6,510** | _skip_ | 26 |
| nested-stress | en | 110 | 640 | _skip_ | 25 |

볼드: 각 행의 가장 빠른 백엔드. ⚠️ 는 _콘텐츠 자체 비용이 아님_ (§4.1 참조).

---

## 3. 결과 매트릭스 — concurrent (conc=4, total=8)

### 3.1 Throughput (req/s)

| template | lng | gotenberg | weasyprint | **reactpdf** |
|----------|-----|----------:|-----------:|--------------:|
| invoice | ko | 7.90 | 0.23 | **10.60** |
| invoice | ja | 7.51 | 0.19 | 6.25 |
| invoice | en | 6.50 | 0.25 | **8.64** |
| nested-basic | en | **20.39** | 7.71 | _skip_ |
| nested-deep | en | **24.64** | 7.11 | _skip_ |
| nested-split | en | **22.84** | 2.30 | _skip_ |
| nested-projects | en | **22.55** | 2.74 | _skip_ |
| nested-stress | en | **17.73** | 1.82 | _skip_ |
| nested-stress | ko | **12.36** | 0.15 | _skip_ |
| nested-stress | ja | **6.97** | 0.14 | _skip_ |

### 3.2 p95 latency (ms)

| template | lng | gotenberg | weasyprint | **reactpdf** |
|----------|-----|----------:|-----------:|--------------:|
| invoice | ko | 579 | 17,909 | 524 |
| invoice | ja | 546 | 21,296 | 859 |
| invoice | en | 648 | 16,929 | 536 |
| nested-stress | ko | 342 | **30,711** | _skip_ |
| nested-stress | ja | 591 | **32,305** | _skip_ |

### 3.3 실패율

**모든 39 셀이 8/0 (100% 성공)**. WeasyPrint 가 `threaded=False` 픽스 덕분에 SIGSEGV 회귀 없음 — 이전 보고서의 안정성 fix 가 본 측정에서도 유효함을 확인.

---

## 4. 핵심 발견

### 4.1 WeasyPrint invoice 의 4~7초는 _CDN fetch 비용_ 이지 처리 비용이 아니다

invoice 시나리오만 WeasyPrint 가 4초+ baseline 인 이유:

```css
/* templates/invoice.html.tmpl:7 */
@import url('https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@400;500;700&family=Noto+Sans+KR:wght@400;500;700&family=Inter:wght@400;500;700&display=swap');
```

WeasyPrint 는 매 요청마다 외부 Google Fonts CDN 호출. nested-*.html 들은 인라인 CSS 만 사용 (외부 fetch 없음) 이라 영문에서 80~770ms 로 빠르다. 진짜 처리 비용은:

| WeasyPrint baseline | invoice (CDN fetch 포함) | nested-deep/en (인라인 CSS) | 비율 |
|---------------------|--------------------------|------------------------------|------|
| 영문 | 4,072ms | 334ms | **~12×** |

→ **본 측정의 invoice/weasyprint 행은 CDN latency 가 변수**. 운영에서 WeasyPrint 도입 검토 시 폰트는 컨테이너 내장 (Dockerfile `fonts-noto-cjk`) 사용하고 CSS @import 제거 필요.

Gotenberg 는 Chrome 의 HTTP 캐시가 있어 CDN 영향 작음. reactpdf 는 폰트 패키지 임베드이므로 외부 fetch 0.

### 4.2 reactpdf 의 invoice baseline 은 Gotenberg 와 동등 또는 우세

| 로케일 | gotenberg | reactpdf | reactpdf 우열 |
|--------|----------:|---------:|--------------|
| ko | 252ms | **208ms** | ▲ 1.21× 빠름 |
| ja | 203ms | 282ms | ▽ 1.39× 느림 |
| en | 190ms | **177ms** | ▲ 1.07× 빠름 |

다만 다음 단서 필수:
- 입력 형식 다름 (4KB JSON vs 12KB HTML) — _HTML 파싱 비용이 빠진 수치_
- 콜드스타트 1회 (폰트 등록 + JIT) 는 본 측정에서 워밍업으로 흡수됨
- invoice 만 측정 — 표가 많거나 복잡 레이아웃 시나리오로 일반화 불가

### 4.3 reactpdf throughput 이 일부 케이스에서 Gotenberg 보다 높음

| 로케일 | gotenberg req/s | reactpdf req/s |
|--------|----------------:|----------------:|
| ko | 7.90 | **10.60** ▲ |
| ja | 7.51 | 6.25 ▽ |
| en | 6.50 | **8.64** ▲ |

Node 이벤트 루프 + 단일 프로세스 처리로도 conc=4 에서 Gotenberg 의 멀티 Chromium worker 와 비등. 단 본 측정의 conc=8 까지 — sweep 이 아니라 정점이 어디인지는 미측정.

### 4.4 Gotenberg 의 영문 처리량이 압도적

| 시나리오 | gotenberg en throughput | weasyprint en throughput |
|----------|------------------------:|--------------------------:|
| nested-deep | **24.64 req/s** | 7.11 |
| nested-split | **22.84** | 2.30 |
| nested-projects | **22.55** | 2.74 |
| nested-basic | **20.39** | 7.71 |

Chromium 의 렌더 파이프라인 + Gotenberg 의 worker pool 이 영문 단순 레이아웃에서 정점 효율.

### 4.5 WeasyPrint CJK + 복잡 레이아웃은 여전히 최악 — 이전 결론 재확인

| 시나리오 | weasyprint p95 |
|----------|---------------:|
| nested-stress/ko | **30.7 s** |
| nested-stress/ja | **32.3 s** |
| nested-projects/ko | 13.8 s |
| nested-deep/ko | 16.9 s |

이전 보고서 §4-1 (CJK 페널티 41~69×) 의 진단이 본 측정에서도 유효.

---

## 5. 비교축 단서 — 결과 해석 가이드

본 보고서를 인용할 때 반드시 명시할 단서:

1. **reactpdf 셀의 latency 는 JSON body 직송 비용**. HTML→PDF 변환을 거치는 다른 백엔드와 _다른 작업을 측정_.
2. **WeasyPrint invoice 행은 Google Fonts CDN fetch 포함**. WeasyPrint 자체 성능 평가는 nested-* (인라인 CSS) 행 참조.
3. **DocRaptor 미포함**. 이전 보고서의 SaaS 비교는 별도 참고.
4. **conc=4, total=8** 의 작은 부하 — sweep 측정 아님. 정점 throughput 은 측정 못 함.
5. **시각 품질**: reactpdf 의 일본어 `𠮷` (U+20BB7) 미표시 (NotoSansJP TTF SMP 미커버). 다른 백엔드는 fonts-noto-cjk OS 폰트로 커버.

---

## 6. 권고 갱신 — 이전 보고서 §7 대비 추가 한 줄

| 우선 순위 | 권장 백엔드 | 추가/유지 |
|----------|------------|----------|
| 속도 (대량 배치, 실시간) | Gotenberg | 유지 |
| 레이아웃 정확도 | WeasyPrint | 유지 (단 CSS @import CDN 제거) |
| 운영 편의성 (zero-ops) | DocRaptor | 유지 |
| **JSX/React 친화 + 인프라 최소** | **ReactPdf** | **신규**. 단 (a) 보고서 종류 1~2개로 고정, (b) HTML 디자이너 협업 불필요, (c) 디자인을 코드로 관리하기 원할 때 |

---

## 7. 다음 측정 권고

1. **sweep 측정**: reactpdf 의 동시성 정점 (conc=1,2,4,8,16) 측정 — Node 이벤트 루프 한계 확인
2. **invoice 의 CSS @import 제거 + 재측정**: WeasyPrint invoice 의 진짜 처리 비용 측정
3. **DocRaptor 재포함**: API 키 발급 후 4+1-way 완성
4. **시각 회귀**: 45 sample PDF 의 페이지별 비교 (pdftoppm + pixelmatch). reactpdf 의 `𠮷` 누락 자동 감지.
5. **메모리 사용량**: `docker stats` 로 컨테이너별 RSS 누적. 사이드카 메모리 비용을 throughput 과 같은 축으로 비교.

---

## 부록 A. 원본 데이터

- CSV: [`reports/matrix-4way-2026-06-08.csv`](./matrix-4way-2026-06-08.csv) — 39 행
- 실행 명령: `go run ./cmd/matrix -url http://localhost:8090 -backends gotenberg,weasyprint,reactpdf -out reports/matrix-4way-2026-06-08.csv`

## 부록 B. 코드 변경점 (2026-05-26 → 2026-06-08)

| 영역 | 변경 |
|------|------|
| `main.go` | `InvoiceBackend` 옵셔널 인터페이스 + `ReactPdf` 어댑터 + `-reactpdf-url` 플래그 |
| `cmd/matrix/main.go` | reactpdf 분기 (사이드카 직접 호출 + nested-* skip) + `-reactpdf-url` / `-data-dir` 플래그 + `doOne(contentType)` 시그니처 |
| `react-pdf/` 신규 디렉토리 | Node sidecar (Express + @react-pdf/renderer v4.5.1) — `Dockerfile`, `server.mjs`, `invoice-doc.mjs`, `money.mjs`, `strings.mjs`, `package.json` |
| `docker-compose.yml` | `reactpdf` 서비스 (port 5002:5002) |
| `Makefile` | `run-reactpdf` 타깃 + `health` 갱신 |
| `README.md` / `USAGE.md` | 4-way 표 + 비교축 차이 단서 |
