# HTML → PDF 변환 비교 샘플

`html-to-pdf-go-server-report.md` 의 결론(Gotenberg → WeasyPrint 이전 검토)을 실제 PDF로 비교하기 위한 PoC 샘플입니다.

3개 백엔드를 **같은 HTML / 같은 데이터**로 돌려서, 보고서의 핵심 pain point인 **표 페이지 나눔 / CJK 폰트 / 통화 정렬** 차이를 눈으로 확인하는 것이 목적입니다.

| 백엔드 | 엔진 | 실행 형태 | 비용 |
|---|---|---|---|
| Gotenberg | Chromium `--print-to-pdf` | Docker 사이드카 (`:3000`) | 무료 |
| WeasyPrint | Pango + CSS Paged Media | Docker 사이드카 (`:5001`) — 자체 Flask 래퍼 | 무료 |
| DocRaptor | Prince XML | SaaS REST API (test mode) | **test 모드는 무료** (워터마크) |

## 빠른 시작

```bash
make up        # 사이드카 컨테이너 빌드·기동 (Gotenberg + WeasyPrint)
make run       # 3 로케일 × 3 백엔드 = output/*.pdf 9개
make serve     # 브라우저 3-패널 데모: http://localhost:8080
```

자세한 사용법, 시나리오별 검증, CLI/환경변수/API 레퍼런스, 트러블슈팅은 **[USAGE.md](./USAGE.md)** 를 참고하세요.

## 디렉터리 구조

```
html-to-pdf/
├── README.md                     ← 개요 (이 파일)
├── USAGE.md                      ← 사용법 매뉴얼
├── html-to-pdf-go-server-report.md   PoC 근거 보고서
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
├── main.go                       CLI 진입점 + Backend 어댑터
├── serve.go                      HTTP 서버 + 데모 API
├── render_test.go                템플릿/통화 테스트
├── output/                       PDF/HTML 출력 (gitignore)
├── go.mod
└── Makefile
```

## 비교 포인트 (보고서와 매핑)

| 체크 항목 | 보고서 근거 | 확인 방법 |
|---|---|---|
| 표 페이지 나눔 / thead 반복 | §1 | `*-gotenberg.pdf` vs `*-weasyprint.pdf` 2페이지 헤더 |
| `border-collapse` 페이지 경계 | §1 | 페이지 경계의 셀 테두리 어긋남 |
| CJK 폰트 셰이핑 (¥ ₩ $ 정렬) | §2 | `.amount` 열 우측 정렬 일관성 |
| JIS X 0213 한자 (髙 﨑 𠮷) | §2 | `ja-*.pdf` 의 "髙﨑𠮷田商事" 글리프 |
| @page margin boxes | §1 | 풋터 페이지 번호 정상 여부 |
| 1매 변환 시간 / CPU·메모리 | §1, §6 | 콘솔 로그 + `docker stats` |

## 범위 밖

- 시나리오 B (후처리: PDF/A-2b, PAdES 서명, RFC 3161 타임스탬프). `WeasyPrint + UniDoc UniPDF` 조합 검증은 별도 PoC 가지에서.
- 부하/동시성 측정. `make serve` 의 `/api/convert` 엔드포인트에 `vegeta` / `hey` 를 붙여 별도 측정 권장 — [USAGE.md §3 시나리오 D](./USAGE.md#시나리오-d-변환-성능-측정-보고서-6-poc-체크리스트).
