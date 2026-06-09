# PDF 백엔드 4-way 벤치마크 — 2026-06-08

> 이전 [3-way 벤치마크 (2026-05-26)](./pdf-backend-benchmark-2026-05-26.md) 에 **`@react-pdf/renderer` 사이드카** 를 4번째 백엔드로 추가한 후속 측정 + 정점 sweep (2026-06-09).
>
> 본 문서는 **측정 데이터** 만 다룸. 결과 해석·권고·운영 위험은 [executive-summary-2026-06-09.md](./executive-summary-2026-06-09.md) 참조.

---

## 0. 한 줄 요약

4 백엔드 × 6 시나리오 × 3 로케일 baseline + conc=4 + 정점 sweep (conc=1/2/4/8/16). 측정 셀 합계 84, **0 실패**. 정점: gotenberg en **22.5** req/s @ conc=8, reactpdf ja **17.9** req/s @ conc=8, weasyprint conc 무관 **0.25** req/s.

---

## 1. 측정 범위 / 환경

### 1.1 백엔드

| 백엔드 | 입력 형식 | 엔드포인트 | 비고 |
|---|---|---|---|
| Gotenberg | HTML | `:8090/api/convert?backend=gotenberg` → `:3000` | Chrome `--print-to-pdf` |
| WeasyPrint | HTML | `:8090/api/convert?backend=weasyprint` → `:5001` | `threaded=False` (SIGSEGV 회피 fix) |
| ReactPdf | Invoice JSON | `:5002/pdf` 직접 (HTTP 서버 우회) | Node 사이드카 + `@react-pdf/renderer` v4.5.1 |
| DocRaptor | — | — | 본 측정 제외 (test API 키 미설정) |

### 1.2 시나리오 매트릭스

| template | 입력 크기 | reactpdf 처리 |
|---|---|---|
| invoice | 12 KB (HTML) · 3~4 KB (JSON) | ✓ |
| nested-basic | 5 KB | × (HTML 입력 미지원) |
| nested-deep | 13 KB | × |
| nested-split | 14 KB | × |
| nested-projects | 21 KB | × |
| nested-stress | 26 KB | × |

로케일: ko / ja / en. 본문 §2/§3 = 18 시나리오 × 3 백엔드 = 54 후보 중 **39 측정 + 15 skip** (reactpdf × nested-*).

### 1.3 동시성 차원

- **본문 §2** — baseline: 1 req 직렬
- **본문 §3.1~3.3** — concurrent: conc=4, total=8
- **§3.5 (reactpdf)** + **§3.6 (gotenberg/weasyprint)** — sweep: conc=1/2/4/8/16, total=max(16, 4×conc) → 16/16/32/64/128

### 1.4 환경

| 항목 | 값 |
|---|---|
| OS | macOS Darwin 25.1 |
| 컨테이너 | Docker Desktop |
| baseURL | `http://localhost:8090` (gotenberg/weasyprint) · `http://localhost:5002` (reactpdf) |
| 본문 측정 일시 | 2026-06-08, 8분 56초 |
| Sweep 측정 일시 | 2026-06-09. reactpdf ~1분, gotenberg ~1분 22초, weasyprint ~57분 |

### 1.5 메소드 노트

- **WeasyPrint invoice baseline (~4s)** 에는 `templates/invoice.html.tmpl` 의 Google Fonts CDN `@import` per-request fetch 비용이 포함됨. nested-* 는 인라인 CSS — 외부 fetch 0.
- **ReactPdf 의 `size_bytes`** 는 JSON body 기준 (3~4KB). 다른 백엔드의 HTML body (12~26KB) 와 비교축이 다름.
- **ReactPdf sweep 워밍업** — 각 로케일 1회 baseline 요청으로 폰트 등록 트리거 (`/health` 의 `fontsRegistered: false → true` 확인 후 측정 시작).
- **Sweep 셀당 요청 수** — baseline 1 + concurrent total. conc=1 셀 = 17 req, conc=16 셀 = 129 req.

---

## 2. 결과 — baseline (단일 요청, ms)

| template | lng | gotenberg | weasyprint | reactpdf | 입력 KB (HTML / JSON) |
|----------|-----|----------:|-----------:|---------:|---------------------|
| invoice | ko | 252 | 4,383 | **208** | 12 / 4 |
| invoice | ja | **203** | 4,652 | 282 | 12 / 4 |
| invoice | en | 190 | 4,072 | **177** | 12 / 4 |
| nested-basic | ko | **190** | 3,828 | _skip_ | 5 |
| nested-basic | ja | **159** | 6,868 | _skip_ | 5 |
| nested-basic | en | 79 | **96** | _skip_ | 5 |
| nested-deep | ko | **190** | 3,850 | _skip_ | 13 |
| nested-deep | ja | **198** | 3,459 | _skip_ | 13 |
| nested-deep | en | **76** | 334 | _skip_ | 13 |
| nested-split | ko | **143** | 3,252 | _skip_ | 14 |
| nested-split | ja | **161** | 3,846 | _skip_ | 14 |
| nested-split | en | **87** | 768 | _skip_ | 13 |
| nested-projects | ko | **143** | 3,645 | _skip_ | 21 |
| nested-projects | ja | **168** | 3,822 | _skip_ | 22 |
| nested-projects | en | **86** | 350 | _skip_ | 21 |
| nested-stress | ko | **153** | 4,915 | _skip_ | 26 |
| nested-stress | ja | **243** | 6,510 | _skip_ | 26 |
| nested-stress | en | **110** | 640 | _skip_ | 25 |

볼드: 각 행 최단 백엔드.

---

## 3. 결과 — concurrent

### 3.1 Throughput @ conc=4, total=8 (req/s)

| template | lng | gotenberg | weasyprint | reactpdf |
|----------|-----|----------:|-----------:|---------:|
| invoice | ko | 7.90 | 0.23 | **10.60** |
| invoice | ja | **7.51** | 0.19 | 6.25 |
| invoice | en | 6.50 | 0.25 | **8.64** |
| nested-basic | en | **20.39** | 7.71 | _skip_ |
| nested-deep | en | **24.64** | 7.11 | _skip_ |
| nested-split | en | **22.84** | 2.30 | _skip_ |
| nested-projects | en | **22.55** | 2.74 | _skip_ |
| nested-stress | en | **17.73** | 1.82 | _skip_ |
| nested-stress | ko | **12.36** | 0.15 | _skip_ |
| nested-stress | ja | **6.97** | 0.14 | _skip_ |

볼드: 각 행 최대 throughput.

### 3.2 p95 latency @ conc=4, total=8 (ms)

| template | lng | gotenberg | weasyprint | reactpdf |
|----------|-----|----------:|-----------:|---------:|
| invoice | ko | 579 | 17,909 | **524** |
| invoice | ja | **546** | 21,296 | 859 |
| invoice | en | 648 | 16,929 | **536** |
| nested-stress | ko | **342** | 30,711 | _skip_ |
| nested-stress | ja | **591** | 32,305 | _skip_ |

볼드: 각 행 최저 latency.

### 3.3 실패율

모든 39 셀이 **8/0 (100% 성공)**.

### 3.5 부록 — reactpdf 동시성 sweep (conc=1, 2, 4, 8, 16)

> 측정 데이터: `reports/matrix-reactpdf-sweep-2026-06-09.csv` (15 셀, conc 컬럼 prefix). invoice × 3 로케일 × conc 5단계 = 15 셀, concurrent 합계 768 req.

#### 3.5.1 Throughput (req/s)

| conc | total | ko | ja | en |
|----:|----:|------:|------:|------:|
| 1 | 16 | 5.97 | 6.05 | 5.26 |
| 2 | 16 | 7.66 | 9.52 | 8.65 |
| 4 | 32 | 11.58 | 14.35 | 13.52 |
| 8 | 64 | **13.95** | **17.91** | **14.00** |
| 16 | 128 | **14.62** | 16.68 ▽ | 13.26 ▽ |

볼드: 각 로케일 정점. ▽: 직전 conc 대비 회귀.

#### 3.5.2 p95 latency (ms)

| conc | ko | ja | en |
|----:|------:|------:|------:|
| 1 | 179 | 176 | 270 |
| 2 | 329 | 255 | 265 |
| 4 | 417 | 363 | 398 |
| 8 | 806 | 590 | 816 |
| 16 | 1,579 | 1,278 | 1,716 |

#### 3.5.3 실패율

**0 fail / 768 req**.

### 3.6 부록 — gotenberg / weasyprint 동시성 sweep (동일 5단계)

> 측정 데이터: `reports/matrix-htmlbackends-sweep-2026-06-09.csv` (30 셀). invoice × 3 로케일 × conc 5단계 × 2 백엔드 = 30 셀, concurrent 합계 1,536 req.

#### 3.6.1 Throughput (req/s)

| conc | total | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|----:|---:|---:|---:|---:|---:|---:|
| 1 | 16 | 8.29 | 5.72 | 9.29 | 0.21 | 0.22 | 0.26 |
| 2 | 16 | 10.80 | 8.19 | 11.12 | 0.24 | 0.22 | **0.26** |
| 4 | 32 | 12.89 | 10.70 | 15.29 | 0.22 | 0.22 | 0.24 |
| 8 | 64 | **13.40** | **10.70** | **22.51** | **0.23** | **0.22** | 0.24 |
| 16 | 128 | 11.85 ▽ | 10.39 ▽ | 18.64 ▽ | **0.23** | 0.21 ▽ | 0.24 |

볼드: 각 (백엔드, 로케일) 정점. ▽: 직전 conc 대비 회귀.

#### 3.6.2 p95 latency (ms)

| conc | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|---:|---:|---:|---:|---:|---:|
| 1 | 128 | 192 | 123 | 4,315 | 4,696 | 3,939 |
| 2 | 213 | 290 | 540 | 8,658 | 9,248 | 7,741 |
| 4 | 399 | 458 | 260 | 26,440 | 18,358 | 24,521 |
| 8 | 817 | 1,171 | 465 | 34,518 | 37,148 | 40,780 |
| 16 | 2,156 | 2,320 | 1,215 | 77,763 | 80,304 | 80,057 |

#### 3.6.3 실패율

**0 fail / 1,536 req**.

#### 3.6.4 3 백엔드 정점 cross-section

| 로케일 | reactpdf 정점 (req/s @ conc) | gotenberg 정점 (req/s @ conc) | weasyprint 정점 (req/s @ conc) |
|---|---|---|---|
| ko | **14.62** @ 16 | 13.40 @ 8 | 0.26 @ 2 |
| ja | **17.91** @ 8 | 10.70 @ 4·8 | 0.22 @ 1·2·4·8 |
| en | 14.00 @ 8 | **22.51** @ 8 | 0.26 @ 1·2 |

볼드: 각 로케일 최대 throughput 백엔드.

---

## 부록 A. 원본 데이터

- CSV (본문 §2/§3.1~3.3): [`reports/matrix-4way-2026-06-08.csv`](./matrix-4way-2026-06-08.csv) — 39 행
- CSV (§3.5 sweep): [`reports/matrix-reactpdf-sweep-2026-06-09.csv`](./matrix-reactpdf-sweep-2026-06-09.csv) — 15 행, `conc` 컬럼 prefix
- CSV (§3.6 sweep): [`reports/matrix-htmlbackends-sweep-2026-06-09.csv`](./matrix-htmlbackends-sweep-2026-06-09.csv) — 30 행
- 샘플 PDF (reactpdf 증거물): `pdfs/청구서-{ko,ja,en}-reactpdf.pdf` (PDF-1.3, 25~60KB)
- 실행 명령 — 본문 §2/§3.1~3.3:
  ```
  go run ./cmd/matrix -url http://localhost:8090 \
    -backends gotenberg,weasyprint,reactpdf \
    -out reports/matrix-4way-2026-06-08.csv
  ```
- 실행 명령 — §3.5 / §3.6 (각 backend × conc 조합):
  ```
  go run ./cmd/matrix -url http://localhost:8090 \
    -backends <B> -templates invoice \
    -concurrent <C> -concurrent-total <N> -timeout 300s
  ```
  B=reactpdf/gotenberg/weasyprint, C=1/2/4/8/16, N=max(16, 4×C).

## 부록 B. 코드 변경점 (2026-05-26 → 2026-06-08)

| 영역 | 변경 |
|------|------|
| `main.go` | `InvoiceBackend` 옵셔널 인터페이스 + `ReactPdf` 어댑터 + `-reactpdf-url` 플래그 |
| `cmd/matrix/main.go` | reactpdf 분기 (사이드카 직접 호출 + nested-* skip) + `-reactpdf-url` / `-data-dir` 플래그 + `doOne(contentType)` 시그니처 |
| `react-pdf/` 신규 디렉토리 | Node sidecar (Express + @react-pdf/renderer v4.5.1) — `Dockerfile`, `server.mjs`, `invoice-doc.mjs`, `money.mjs`, `strings.mjs`, `package.json` |
| `docker-compose.yml` | `reactpdf` 서비스 (port 5002:5002) |
| `Makefile` | `run-reactpdf` 타깃 + `health` 갱신 |
| `README.md` / `USAGE.md` | 4-way 표 + 비교축 차이 단서 |
