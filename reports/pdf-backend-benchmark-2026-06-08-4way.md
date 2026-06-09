# PDF 백엔드 4-way 벤치마크 — 2026-06-08

> 이전 [3-way 벤치마크 (2026-05-26)](./pdf-backend-benchmark-2026-05-26.md) 에 **`@react-pdf/renderer` 사이드카** 를 4번째 백엔드로 추가한 후속 측정.
>
> 측정 데이터: `reports/matrix-4way-2026-06-08.csv` (39 셀, 헤더 포함 40 행)
> 측정 환경: macOS Darwin 25.1, Docker Desktop, baseURL `http://localhost:8090`
> 측정 시간: 8분 56초

---

## 0. 한 줄 결론

이전 결론 (Gotenberg 1순위, WeasyPrint 정확도 우선, DocRaptor zero-ops) 은 **유지**. 단, 2026-06-09 정점 sweep (§3.5 / §3.6) 으로 다음이 추가됨:

- **invoice 한정**: `@react-pdf/renderer` 가 **ko/ja 정점 처리량에서 Gotenberg 우세** (ja 17.9 vs 10.7 req/s, ko 14.6 vs 13.4), en 에선 Gotenberg 가 1.6× 우세 (22.5 vs 14.0). 단 _HTML 을 받지 않는 패러다임 차이_ (§5.1) — 직접 비교축이 다르다.
- **사이드카 정점은 conc=8** (reactpdf · gotenberg 공통). conc=16 은 throughput 정체·p95 폭증.
- **WeasyPrint 의 invoice 정점 0.25 req/s 는 엔진 한계가 아니라 환경 제약치** — `threaded=False` (§3.3 SIGSEGV 회피 fix) + 매 요청 Google Fonts CDN fetch (§4.1) 의 곱. CSS @import 제거 + 멀티스레드 환경에선 측정값이 달라짐 (현재 보고서 범위 밖).

운영 도입 검토 시 ReactPdf 는 _보고서 종류 1~2개로 고정_ + _디자인을 코드로 관리_ 인 경우만 후보. 다양한 템플릿 / HTML 디자이너 협업이 있는 경우는 Gotenberg 가 여전히 1순위.

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

### 3.5 부록 — reactpdf 동시성 sweep (conc=1, 2, 4, 8, 16)

> **후속 측정 (2026-06-09)**: 본문 §3.1~3.3 의 conc=4 단일 포인트로는 §7.1 의 "Node 이벤트 루프 한계" 가 미관측이므로, reactpdf 만 5단계 sweep 으로 정점·임계점을 확인.
>
> 측정 데이터: `reports/matrix-reactpdf-sweep-2026-06-09.csv` (15 셀)
> 워밍업: 각 로케일 1회 baseline 요청 (폰트 등록 트리거 — `/health` 의 `fontsRegistered: false → true` 확인 후 측정 시작)
> 표본 수: conc 별 total = max(16, 4 × conc) → 16/16/32/64/128. 총 752 요청.

#### 3.5.1 Throughput (req/s)

| conc | total | ko | ja | en |
|----:|----:|------:|------:|------:|
| 1 | 16 | 5.97 | 6.05 | 5.26 |
| 2 | 16 | 7.66 | 9.52 | 8.65 |
| 4 | 32 | 11.58 | **14.35** | 13.52 |
| **8** | **64** | **13.95** | **17.91** | **14.00** |
| 16 | 128 | 14.62 | 16.68 ▽ | 13.26 ▽ |

볼드: 각 로케일 정점. ▽: 직전 conc 대비 회귀.

#### 3.5.2 p95 latency (ms)

| conc | ko | ja | en |
|----:|------:|------:|------:|
| 1 | 179 | 176 | 270 |
| 2 | 329 | 255 | 265 |
| 4 | 417 | 363 | 398 |
| 8 | 806 | 590 | 816 |
| 16 | 1,579 | 1,278 | 1,716 |

#### 3.5.3 baseline 안정성

모든 conc 레벨에서 baseline (단일 요청) 은 **194~258ms** 로 유지 — 사이드카가 부하 후에도 콜드스타트 비용을 다시 지불하지 않음 (Node 프로세스 재사용 + 폰트 캐시 영속).

#### 3.5.4 실패율

**0 fail / 752 req** (15 셀 × n). 타임아웃·OOM 징후 없음. 단 단발 측정이라 장시간 (수십분~수시간) 부하 누수는 별개 검증 필요.

#### 3.5.5 핵심 관찰

1. **정점은 conc=8** — ja 17.9 / ko 13.95 / en 14.0 req/s. conc=16 은 ko 만 미세 증가, ja·en 은 5~7% 회귀. Node 단일 이벤트 루프 + `@react-pdf/renderer` 의 동기적 layout/render 가 추가 동시성을 흡수 못 함 — 호스트 코어가 남아도 사이드카 단일 인스턴스는 8 동시성에서 saturate.
2. **p95 는 conc 와 거의 선형 증가** — conc 1→16 에서 9~10× (270ms → 1.7s). 큐잉 지연. SLA p95 < 500ms 를 요구한다면 **conc ≤ 4 권장**.
3. **Gotenberg 대비 정점 throughput 은 70% 수준** — Gotenberg 영문 시나리오 22~25 req/s vs reactpdf 정점 14~18 req/s (단 측정 시나리오가 다르다 — 본문 §5.1 비교축 단서 유지).
4. **수평 확장 시사점** — 단일 사이드카 정점이 명확하므로 throughput 이 더 필요하면 **사이드카 인스턴스를 늘리는 것** 이 conc 를 더 올리는 것보다 효율적 (Node 의 cluster 모드 또는 docker replica). 본 측정에선 미검증.

### 3.6 부록 — gotenberg / weasyprint 동시성 sweep (같은 5단계)

> **후속 측정 (2026-06-09)**: §3.5 의 reactpdf sweep 결과를 다른 두 HTML 백엔드와 정점 비교하기 위해 동일 파라미터로 측정. invoice × 3 로케일 × conc {1, 2, 4, 8, 16}.
>
> 측정 데이터: `reports/matrix-htmlbackends-sweep-2026-06-09.csv` (30 셀)
> 표본 수: §3.5 와 동일 — total = max(16, 4 × conc) → 16/16/32/64/128. 총 1,440 요청.
> 측정 시간: 약 **57분** (gotenberg 1m22s + weasyprint 56m+).

#### 3.6.1 Throughput (req/s)

| conc | total | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|----:|---:|---:|---:|---:|---:|---:|
| 1 | 16 | 8.29 | 5.72 | 9.29 | 0.21 | 0.22 | 0.26 |
| 2 | 16 | 10.80 | 8.19 | 11.12 | 0.24 | 0.22 | 0.26 |
| 4 | 32 | 12.89 | 10.70 | 15.29 | 0.22 | 0.22 | 0.24 |
| **8** | **64** | **13.40** | **10.70** | **22.51** | 0.23 | 0.22 | 0.24 |
| 16 | 128 | 11.85 ▽ | 10.39 ▽ | 18.64 ▽ | 0.23 | 0.21 ▽ | 0.24 |

볼드: 정점. ▽: 직전 conc 대비 회귀.

#### 3.6.2 p95 latency (ms)

| conc | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|---:|---:|---:|---:|---:|---:|
| 1 | 128 | 192 | 123 | 4,315 | 4,696 | 3,939 |
| 2 | 213 | 290 | 540 | 8,658 | 9,248 | 7,741 |
| 4 | 399 | 458 | 260 | 26,440 | 18,358 | 24,521 |
| 8 | 817 | 1,171 | 465 | 34,518 | 37,148 | 40,780 |
| 16 | 2,156 | 2,320 | 1,215 | **77,763** | **80,304** | **80,057** |

#### 3.6.3 실패율

**0 fail / 1,440 req**. 30 셀 모두 100% 성공. WeasyPrint 의 `threaded=False` + glibc malloc 디버깅 환경에서도 60초 ~ 80초 큐잉 지연을 견딤.

#### 3.6.4 핵심 관찰

1. **Gotenberg 정점은 conc=8** — ko 13.4 / ja 10.7 / en **22.51** req/s. en 시나리오가 ja 대비 2× 빠른 것은 §4.4 의 "영문 단순 레이아웃" 관찰과 정합. conc=16 은 모든 로케일에서 회귀 (Chrome worker pool 의 컨텍스트 스위칭 비용).
2. **WeasyPrint 는 conc 와 무관하게 ~0.22~0.26 req/s 고정** — `threaded=False` (Flask 단일 스레드, §3.3 의 SIGSEGV 회피 fix) + invoice 의 Google Fonts CDN per-request fetch (§4.1) 가 곱해져 **이론치 1/baseline = 1/4s ≈ 0.25 req/s** 와 정확히 일치. **정점이 없는 게 정점**.
3. **WeasyPrint p95 는 conc 와 거의 선형 증가** — 1→16 에서 약 19× (4s → 80s). conc=16 의 p95 가 80초 — 즉 16번째 요청은 1분 20초 큐 대기. 실시간 사용 불가.
4. **Gotenberg p95 도 conc 와 함께 증가하지만 절대값이 다름** — ko: 128ms → 2,156ms (16×). en: 123ms → 1,215ms (10×). 정점 conc=8 에서 p95 ≤ 1.2s 유지.

#### 3.6.5 정점 비교 — 3 백엔드 cross-section (§7.1 의 원래 질문)

| 로케일 | reactpdf 정점 (req/s @ conc) | gotenberg 정점 (req/s @ conc) | weasyprint 정점 | 1위 |
|---|---|---|---|---|
| ko | **14.62** @ 16 | 13.40 @ 8 | 0.26 @ 2 | reactpdf |
| ja | **17.91** @ 8 | 10.70 @ 4·8 | 0.22 @ 1·2·4·8 | reactpdf |
| en | 14.00 @ 8 | **22.51** @ 8 | 0.26 @ 1·2 | gotenberg |

**의외 발견**: invoice 시나리오 한정에서 **reactpdf 가 ko/ja 정점에서 gotenberg 를 앞섬** (각 1.09× / 1.67×). 본문 §4.3 에서 conc=4 한 점으로 비등하다 했던 관찰이 정점 측정에서 **reactpdf 우세** 로 강화됨. 단 en 에선 gotenberg 가 1.61× 우세 — Chrome 렌더 파이프라인이 ASCII 단순 텍스트에 최적화된 것과 일치.

**WeasyPrint 는 동일 시나리오에서 reactpdf/gotenberg 대비 50~80× 느림** (peak 비교). 본 측정의 baseline 4s 중 대부분이 CDN fetch 라는 §4.1 의 단서가 유지되므로, _CSS @import 제거 + 컨테이너 내장 폰트_ 환경에선 이 비율이 크게 줄어들 가능성 있음 (이전 보고서 §4-1: nested-* 시나리오에선 80~770ms 로 정상).

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

Node 이벤트 루프 + 단일 프로세스 처리로도 conc=4 에서 Gotenberg 의 멀티 Chromium worker 와 비등. **정점 측정은 §3.5 sweep 참조** — reactpdf 단일 사이드카는 conc=8 에서 saturate (ja 17.9 / ko 13.95 / en 14.0 req/s).

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
4. **conc=4, total=8** 의 작은 부하 — 본문 §3.1~3.3 한정. **3 백엔드 모두의 정점은 §3.5 / §3.6 sweep 에서 측정 완료**: reactpdf conc=8 (ja 17.9 req/s), gotenberg conc=8 (en 22.5 req/s), weasyprint 는 conc 무관 0.25 req/s (`threaded=False` + CDN fetch).
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

1. ~~**sweep 측정**: 3 백엔드 동시성 정점 (conc=1,2,4,8,16)~~ — **완료 (2026-06-09)**, §3.5 (reactpdf) / §3.6 (gotenberg + weasyprint) 참조. 다음 단계 후보: (a) **사이드카 replica 수평 확장 측정** — reactpdf 단일 인스턴스 정점이 conc=8 에서 saturate 임이 §3.5.5 4번에서 확인됨; (b) **WeasyPrint sweep 재측정** — `threaded=False` 해제 + CSS @import CDN 제거 조합으로 진짜 처리 한계 확인 (현재 0.25 req/s 는 환경 제약치이지 엔진 한계가 아님).
2. **invoice 의 CSS @import 제거 + 재측정**: WeasyPrint invoice 의 진짜 처리 비용 측정
3. **DocRaptor 재포함**: API 키 발급 후 4+1-way 완성
4. **시각 회귀**: 45 sample PDF 의 페이지별 비교 (pdftoppm + pixelmatch). reactpdf 의 `𠮷` 누락 자동 감지.
5. **메모리 사용량**: `docker stats` 로 컨테이너별 RSS 누적. 사이드카 메모리 비용을 throughput 과 같은 축으로 비교.

---

## 8. 운영 위험 분석 (측정 외)

> §2~§3 의 측정값은 **단발 스파이크 환경의 throughput·latency**. 운영 도입 시에는 측정 밖 위험이 더 결정적일 수 있어 별도 정리. 같은 위험이라도 백엔드별 영향이 다르므로 cross-backend 매트릭스로 표기.
>
> 본 섹션의 위험들은 본 보고서 측정으로 **확인 안 됨** — 도입 검토 시 PoC 단계에서 추가 검증 권장. §7 의 다음 측정 항목 중 일부 (replica 확장, 메모리 사용량, 시각 회귀) 가 이 위험들을 직접 다룬다.

### 8.1 위험 매트릭스 (요약)

| # | 위험 | reactpdf | gotenberg | weasyprint |
|---|---|---|---|---|
| 8.2.1 | 단일 프로세스 SPOF / 크래시 시 in-flight 손실 | **고** (Node 단일 프로세스, sync 렌더 중 SIGKILL 시 통째 손실) | 중 (worker pool 1개 크래시 = 1요청 손실) | **고** (`threaded=False` 단일 워커) |
| 8.2.2 | 이벤트 루프 / 단일 스레드 블로킹 | **고** (sync 렌더가 헬스체크·다른 라우트도 잡음) | 저 (Chromium worker 분리) | **고** (Python GIL + threaded=False) |
| 8.2.3 | 메모리 — 전체 렌더 트리 보유, 스트리밍 없음 | **고** (VDOM + layout 전체 메모리, 대용량 invoice OOM 가능) | 중 (Chrome 페이지 분할 렌더 가능) | 중 (CSS Paged Media — 페이지 단위) |
| 8.2.4 | 장시간 부하 메모리 누수 | **미검증** (단발 752 reqs 만 측정) | **미검증** | **미검증** (이전 보고서에서 SIGSEGV 회귀 이력 — `threaded=False` 로 회피 중) |
| 8.2.5 | Font 캐시 per-replica 부담 | **중** (replica 마다 폰트 재등록, 콜드스타트 비용) | 저 (Chromium 가 OS fonts-noto-cjk 공유) | 저 (동일) |
| 8.2.6 | Paged media 성숙도 (orphan/widow, 표 행 잘림) | **고** (CSS Paged Media 미지원, nested-* 입력 자체 불가 → 측정조차 못 함) | 저 (Chrome 렌더 파이프라인 — 본 보고서 §3.1 에서 모든 nested-* 통과) | 저 (CSS Paged Media 의 기준 구현) |
| 8.2.7 | Lock-in (백엔드 전환 비용) | **고** (JSX → HTML 재작성 필요, 다른 두 백엔드와 템플릿 공유 불가) | 저 (HTML 표준, gotenberg ↔ weasyprint ↔ docraptor 간 템플릿 공유) | 저 (동일) |
| 8.2.8 | 한자 SMP / CJK 폰트 커버리지 | **고** (`𠮷` U+20BB7 미표시, NotoSansJP TTF SMP 미커버 — §5.5) | 저 (OS fonts-noto-cjk 사용) | 저 (동일) |
| 8.2.9 | 외부 의존성 (네트워크) | 저 (폰트 임베드, 외부 fetch 0) | 중 (CSS @import CDN 캐시 의존 — §4.1) | **고** (CSS @import CDN 매 요청 fetch — §4.1) |
| 8.2.10 | 클라이언트사이드 옵션의 부담 | **N/A** (본 PoC 서버사이드. 클라사이드 전환 시 번들 ~15MB + 모바일 CPU 0.5~2s) | N/A (서버사이드 전용) | N/A (서버사이드 전용) |

### 8.2 위험별 상세

#### 8.2.1 단일 프로세스 SPOF
3 백엔드 모두 단일 사이드카 프로세스. **크래시 = 다운타임 until 재시작**. 추가로 reactpdf 는 sync 렌더 중 SIGKILL 시 in-flight 요청이 통째로 손실 (Node 가 graceful drain 안 함). 완화: `docker compose` 의 `restart: unless-stopped` (현재 미설정) + replica.

#### 8.2.2 이벤트 루프 / 단일 스레드 블로킹
reactpdf 는 Node 단일 스레드 — `@react-pdf` 의 layout/render 가 동기 호출. 같은 프로세스에 헬스체크 라우트 (`/health`) 가 있어 부하 중 응답 지연 가능 → k8s liveness probe 실패 → 재시작 루프 가능. 완화: 헬스체크 분리 또는 별도 worker.

#### 8.2.3 메모리 — 스트리밍 없음
reactpdf 는 전체 invoice 의 VDOM + layout state 를 메모리에 유지하다가 한 번에 PDF 직렬화. 본 측정의 invoice 는 line item ~10개 (3~4KB JSON) — 작음. **수백~수천 line item 의 대용량 invoice 에서 RSS 가 어떻게 변하는지 미검증**. §7.5 의 메모리 측정 항목과 연결.

#### 8.2.4 장시간 부하 메모리 누수
3 백엔드 모두 단발 752~1,440 reqs 까지만 측정. **수시간~수일 부하 후 RSS 추이 미검증**. WeasyPrint 는 이전 보고서에서 thread-safety 관련 SIGSEGV 회귀 이력이 있어 `threaded=False` 로 회피 중 — 본질 원인 미해결.

#### 8.2.5 Font 캐시 per-replica 부담
reactpdf 는 NotoSansKR/JP TTF 를 프로세스 메모리에 로드. replica N 개 = 폰트 N 회 로드. 워밍업 비용은 §3.5 측정에서 1회 흡수 (`fontsRegistered: false → true`) — 단일 인스턴스에선 무시 가능, 수평 확장 시 새 컨테이너마다 발생. 완화: 헬스체크 통과 전 워밍업.

#### 8.2.6 Paged media 성숙도
**reactpdf 가 nested-* 시나리오를 처리 못 하는 것 자체가 위험 신호**. 향후 다중 페이지 / 표 행 잘림 / orphan-widow 같은 페이지 분할 요구가 생기면 react-pdf 의 paged 모델로 표현 가능한지 불확실. 보고서 §1 의 _skip_ 칸이 곧 운영 위험.

#### 8.2.7 Lock-in
JSX 템플릿은 다른 백엔드와 공유 불가. invoice 만 운용하다 추후 contract / receipt / report 등 보고서 종류 늘면 **JSX 코드 N개 추가 + 백엔드 전환 시 HTML 재작성** 필요. Gotenberg/WeasyPrint 간 전환은 HTML 그대로 사용 가능.

#### 8.2.8 한자 SMP / CJK 폰트 커버리지
§5.5 에서 발견된 `𠮷` (U+20BB7) 미표시 — silent 한자 누락. 사용자 이름·주소에 SMP 영역 한자가 들어가면 PDF 에 글자 없이 출력될 수 있고 에러도 안 남. 완화: SMP 커버 폰트 (예: NotoSansCJK OTF) 사용 — 단 번들 크기 증가.

#### 8.2.9 외부 의존성 (네트워크)
WeasyPrint 의 invoice 정점이 0.25 req/s 로 고정된 원인 (§3.6.4·2) — Google Fonts CDN. **CDN 장애 = 변환 실패**. Gotenberg 는 Chrome HTTP 캐시로 영향 작음. reactpdf 는 폰트 임베드라 0.

#### 8.2.10 클라이언트사이드 옵션
`@react-pdf/renderer` 는 브라우저에서도 동작 가능. 본 PoC 는 서버사이드 (Node 사이드카) — 클라이언트 부하 0. 만약 클라사이드 전환 시: 번들 ~15MB (라이브러리 + CJK 폰트 TTF), 모바일 CPU 500ms~2s. 본 보고서 범위 밖이지만 옵션 존재.

### 8.3 위험 요약

운영 위험 측면 서열: **WeasyPrint > reactpdf > Gotenberg**.

- **Gotenberg**: §8 의 거의 모든 위험에서 "저~중". Chromium 의 성숙도 + worker pool 구조 + OS 폰트 공유의 합.
- **reactpdf**: paged media 성숙도 (8.2.6) + lock-in (8.2.7) + SMP 누락 (8.2.8) + 이벤트 루프 블로킹 (8.2.2) 4 개가 **고**. invoice 같은 단순·고정 양식 외에는 도입 위험 큼.
- **WeasyPrint**: SPOF (8.2.1) + 스레드 블로킹 (8.2.2) + 메모리 누수 이력 (8.2.4) + CDN 의존 (8.2.9) 4 개가 **고**. `threaded=False` 가 안정성 fix 이자 throughput 천장 — 본질적 trade-off.

§6 의 권고는 본 §8 위험을 받쳐줄 때만 유효 — 예: ReactPdf 도입은 "보고서 종류 1~2개로 고정" 조건이 8.2.6/8.2.7 의 위험을 우회하는 전제.

---

## 부록 A. 원본 데이터

- CSV (본문 §2~§4): [`reports/matrix-4way-2026-06-08.csv`](./matrix-4way-2026-06-08.csv) — 39 행
- CSV (§3.5 sweep): [`reports/matrix-reactpdf-sweep-2026-06-09.csv`](./matrix-reactpdf-sweep-2026-06-09.csv) — 15 행, `conc` 컬럼 prefix
- CSV (§3.6 sweep): [`reports/matrix-htmlbackends-sweep-2026-06-09.csv`](./matrix-htmlbackends-sweep-2026-06-09.csv) — 30 행 (gotenberg 15 + weasyprint 15)
- 샘플 PDF (reactpdf 증거물): `pdfs/청구서-{ko,ja,en}-reactpdf.pdf` (PDF-1.3, 25~60KB)
- 실행 명령 본문: `go run ./cmd/matrix -url http://localhost:8090 -backends gotenberg,weasyprint,reactpdf -out reports/matrix-4way-2026-06-08.csv`
- 실행 명령 §3.5 / §3.6 (각 backend × conc 조합): `go run ./cmd/matrix -url http://localhost:8090 -backends <B> -templates invoice -concurrent <C> -concurrent-total <N> -timeout 300s` (B=reactpdf/gotenberg/weasyprint, C=1/2/4/8/16, N=max(16, 4×C))

## 부록 B. 코드 변경점 (2026-05-26 → 2026-06-08)

| 영역 | 변경 |
|------|------|
| `main.go` | `InvoiceBackend` 옵셔널 인터페이스 + `ReactPdf` 어댑터 + `-reactpdf-url` 플래그 |
| `cmd/matrix/main.go` | reactpdf 분기 (사이드카 직접 호출 + nested-* skip) + `-reactpdf-url` / `-data-dir` 플래그 + `doOne(contentType)` 시그니처 |
| `react-pdf/` 신규 디렉토리 | Node sidecar (Express + @react-pdf/renderer v4.5.1) — `Dockerfile`, `server.mjs`, `invoice-doc.mjs`, `money.mjs`, `strings.mjs`, `package.json` |
| `docker-compose.yml` | `reactpdf` 서비스 (port 5002:5002) |
| `Makefile` | `run-reactpdf` 타깃 + `health` 갱신 |
| `README.md` / `USAGE.md` | 4-way 표 + 비교축 차이 단서 |
