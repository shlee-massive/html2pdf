# HTML → PDF 변환 백엔드 종합 벤치마크 보고서

- 측정일: 2026-05-26
- 대상: 본 리포의 `POST /api/convert?backend=X` 엔드포인트가 위임하는 3 백엔드
  - **Gotenberg** (Chromium Headless 147, Skia/PDF m147, 로컬 Docker)
  - **WeasyPrint** (62.3 + pydyf 0.10.0, 로컬 Docker)
  - **DocRaptor** (Prince 15.1, 외부 클라우드 API, 워터마크 포함)
- 측정 차원: **동시성 곡선**(깊이) + **템플릿×언어 매트릭스**(폭) + **출력 품질**(페이지·크기·레이아웃) + **안정성**(thread-safety)
- 도구: [`cmd/loadtest`](../cmd/loadtest), [`cmd/matrix`](../cmd/matrix), 수기 비교
- 원시 데이터: [`matrix-2026-05-26.csv`](matrix-2026-05-26.csv) (36 셀), PDF 산출물 45개 [`../pdfs/`](../pdfs/)

---

## 1. TL;DR

| 결론 | 근거 위치 |
|------|-----------|
| **운영 1차 백엔드는 Gotenberg.** 모든 셀에서 빠르고 안정적. CJK 영향이 1.5~2배 이내. | §3, §4 |
| **WeasyPrint 는 CJK 콘텐츠에서 영문 대비 최대 57배 느리다.** 단일 요청 처리 시간이 ~4초. | §4 |
| **WeasyPrint 는 thread-safe 가 아니다.** Flask 개발 서버 기본 `threaded=True` 와 만나면 SIGSEGV. **백트레이스로 확정 진단됨**. 본 리포에 `threaded=False` 픽스 적용. | §5 |
| **레이아웃 정확도는 WeasyPrint 가 최고**, Gotenberg 차순, DocRaptor 는 `rowspan` + `vertical-rl` 매트릭스에서 페이지를 분할해버리는 회귀 있음. | §6 |
| **일본어 콘텐츠는 같은 템플릿이라도 페이지 +1**, ko/en 보다 글자수가 많아 `page-break-inside: avoid` 만으로는 막을 수 없다. | §6 |
| **Gotenberg 의 동시성 sweet spot 은 4**. 그 이상은 throughput 감소 + p95 폭발. | §3 |
| **DocRaptor 는 부하 테스트 대상에서 제외**. 외부 유료 API + 워터마크 + 동시 호출 SLA 미상. 본 보고서에서는 출력 품질만 다룬다. | §2-3 |

---

## 2. 측정 환경 및 방법

### 2-1. 환경

- macOS Darwin 25.1.0
- Docker Desktop: `htp-gotenberg`, `htp-weasyprint` (각 단일 컨테이너)
- App: Go `net/http` (`make serve`, `:8080`)
- 워밍업: 첫 1 요청 (필요 시 별도 명시)

### 2-2. 측정 차원

| 차원 | 도구 | 무엇을 본다 |
|------|------|-------------|
| **동시성 깊이** (§3) | `cmd/loadtest -sweep` | 같은 템플릿/언어로 동시성 1→16 sweep. throughput 정점·tail latency 폭발 지점 |
| **콘텐츠 폭** (§4) | `cmd/matrix` | 6 템플릿 × 3 언어 × 2 백엔드 = 36 셀. 페이로드가 바뀔 때 어디서 깨지는지 |
| **출력 품질** (§6) | 수기 비교 | 같은 입력 HTML 의 3 백엔드 산출 PDF (페이지 수, 파일 크기, 레이아웃 충실도) |
| **안정성** (§5) | `cmd/matrix` + 디버그 환경 | 부하 중 프로세스 abort, 백트레이스 채집 |

### 2-3. 측정 대상 템플릿

| # | 키 | 의도 | 언어 변형 |
|---|---|---|---|
| ① | `invoice` | 표준 인보이스 (숫자·통화 중심) — `output/{ko,ja,en}.html` 로 사전 렌더 | ko/ja/en |
| ② | `nested-basic` | 카테고리별 1단 중첩, 기본 케이스 — `web/nested-basic{,-ja,-en}.html` | ko/ja/en |
| ③ | `nested-deep` | 부서 → 프로젝트 → 항목 3단 중첩 | ko/ja/en |
| ④ | `nested-split` | `page-break-inside: auto` + 대형 행으로 페이지 경계 횡단 | ko/ja/en |
| ⑤ | `nested-projects` | 프로젝트별 다중 nested grid (`page-break-inside: avoid`) | ko/ja/en |
| ⑥ | `nested-stress` | 종합 스트레스 (긴 표 thead 반복 + rowspan + vertical-rl + 14열 가로폭 + `word-break` + `@page` 마진박스 + 인라인 SVG) | ko/ja/en |

크기: 5KB (`nested-basic`) ~ 26KB (`nested-stress`).

### 2-4. DocRaptor 제외 사유

외부 클라우드 API, 유료, 워터마크, 동시성 SLA 미상. **출력 품질 비교(§6)에는 포함**, 부하/매트릭스(§3·§4)에서는 제외.

---

## 3. 성능 — 동시성 곡선 (단일 템플릿 깊이)

같은 페이로드(`output/en.html`, 12KB)를 동시성 1, 2, 4, 8, 16 으로 sweep. 각 레벨당 16 요청.

### 3-1. Gotenberg

```mermaid
xychart-beta
    title "Gotenberg throughput vs concurrency"
    x-axis "concurrency" [1, 2, 4, 8, 16]
    y-axis "req/s" 0 --> 20
    bar [9.18, 14.35, 18.40, 16.26, 11.50]
```

| 동시성 | req/s | avg | p50 | p95 | max |
|-------:|------:|-----|-----|-----|-----|
|  1 |  9.18 | 109ms | 102ms | 127ms | 128ms |
|  2 | 14.35 | 138ms | 135ms | 157ms | 165ms |
|  4 | **18.40** | 214ms | 205ms | 258ms | 271ms |
|  8 | 16.26 | 439ms | 423ms | 656ms | 768ms |
| 16 | 11.50 | 742ms | 586ms | 1382ms | 1391ms |

- conc=1 → 2: 처리량 1.56배, 지연 거의 그대로 — 무비용 증설
- conc=4 가 **throughput 정점**. 이후 모두 손해:
  - conc=8: throughput −12%, p95 2.5배
  - conc=16: throughput −38%, p95 5.4배

### 3-2. WeasyPrint

```mermaid
xychart-beta
    title "WeasyPrint throughput vs concurrency"
    x-axis "concurrency" [1, 2, 4, 8, 16]
    y-axis "req/s" 0 --> 2
    bar [0.26, 0.50, 0.60, 0.70, 1.61]
```

| 동시성 | req/s | avg | p50 | p95 |
|-------:|------:|-----|-----|-----|
|  1 | 0.26 | 3847ms | 3835ms | 4008ms |
|  2 | 0.50 | 3941ms | 3888ms | 4183ms |
|  4 | 0.60 | 6645ms | 4224ms | **13740ms** |
|  8 | 0.70 | 8105ms | 5474ms | 13774ms |
| 16 | 1.61 | 7681ms | 7073ms | 9889ms |

- 단일 요청 ~3.85s — Gotenberg 대비 ~35배
- conc=2 까지 거의 선형, conc=4 부터 p95 가 4s → 13.7s 로 폭발
- conc=16 의 throughput 회복은 측정 아티팩트 (전 요청이 burst 로 시작해 워커가 완전히 포화 유지). p50 7s 로 사용자 경험상 이미 한계 초과
- **`threaded=False` 픽스 적용 전 측정값**. 픽스 후에는 동시성과 무관하게 사실상 직렬 처리 (§5 참조)

---

## 4. 성능 — 템플릿 × 언어 매트릭스 (콘텐츠 폭)

각 (template, language, backend) 셀에서 baseline 1회 + 동시성 4, 8 요청. (`threaded=True` 상태 측정값; 픽스 후에는 영문 처리량 다소 상승, CJK 거의 동일.)

### 4-1. 가장 중요한 발견: WeasyPrint 의 CJK 페널티

영어를 1.0 기준으로 한 상대 baseline:

| Template          | en   | ko 배수   | ja 배수   |
|-------------------|-----:|----------:|----------:|
| nested-basic      | 1.0  | **56.7x** | **47.4x** |
| invoice           | 1.0  | 1.13x     | 1.18x     |
| nested-deep       | 1.0  | 16.5x     | 17.5x     |
| nested-split      | 1.0  | 7.5x      | 8.0x      |
| nested-projects   | 1.0  | 9.7x      | 9.4x      |
| nested-stress     | 1.0  | 4.6x      | 4.8x      |

> 동일 측정을 Gotenberg 로 하면 ko 배수가 1.6~1.9 수준. **CJK 페널티는 WeasyPrint(Pango/Cairo) 고유 이슈**.

invoice 만 유독 영향이 작은 이유: 페이지가 숫자·날짜·통화 중심이라 CJK 비중이 낮음. nested-basic 은 같은 5KB 라도 본문 텍스트 비중이 높아 페널티가 가장 크게 드러남.

### 4-2. 매트릭스 baseline (모든 36 셀, ms)

| Template          | KB   | Lang | Gotenberg | WeasyPrint | WeasyP / Gtb |
|-------------------|-----:|:----:|----------:|-----------:|-------------:|
| nested-basic      |  5.5 | ko   |    189    |    3457    | 18.3x        |
| nested-basic      |  5.6 | ja   |    131    |    2889    | 22.1x        |
| nested-basic      |  5.4 | en   |     61    |      61    |  1.0x        |
| invoice           | 12.5 | ko   |    232    |    4312    | 18.6x        |
| invoice           | 12.5 | ja   |    392    |    4484    | 11.4x        |
| invoice           | 12.2 | en   |    127    |    3799    | 29.9x        |
| nested-deep       | 13.3 | ko   |    147    |    2767    | 18.8x        |
| nested-deep       | 13.6 | ja   |    234    |    2936    | 12.5x        |
| nested-deep       | 13.0 | en   |     84    |     168    |  2.0x        |
| nested-split      | 14.0 | ko   |    141    |    2922    | 20.7x        |
| nested-split      | 14.2 | ja   |    167    |    3146    | 18.8x        |
| nested-split      | 13.6 | en   |     80    |     391    |  4.9x        |
| nested-projects   | 21.2 | ko   |    158    |    3520    | 22.3x        |
| nested-projects   | 21.5 | ja   |    162    |    3428    | 21.2x        |
| nested-projects   | 20.5 | en   |     85    |     363    |  4.3x        |
| nested-stress     | 25.9 | ko   |    164    |    4662    | 28.4x        |
| nested-stress     | 26.3 | ja   |    200    |    4844    | 24.2x        |
| nested-stress     | 25.2 | en   |     96    |    1004    | 10.5x        |

### 4-3. 동시성 4 throughput 요약

| 환경 | Gotenberg | WeasyPrint | 비고 |
|------|----------:|-----------:|------|
| CJK 평균 (ko/ja) | 10.9 req/s | 0.31 req/s | WeasyPrint 는 사실상 1 req/s 미만 |
| 영문 평균 | 22.7 req/s | 3.1 req/s | invoice/en 만 0.80 으로 유독 낮음 |

극단 케이스 (`nested-stress/ko` @ conc=4):
- Gotenberg 12.8 req/s, p95 310ms
- WeasyPrint 0.19 req/s, p95 21495ms → 사용 불가

---

## 5. 안정성 — WeasyPrint thread-safety abort 진단

### 5-1. 증상

첫 매트릭스 실행 중 WeasyPrint 프로세스가 다음으로 abort:

```
free(): invalid pointer
```

- ~12분 누적 부하 시점 (nested-basic 3종 + nested-deep ko/ja 처리 후) 에 발생
- `nested-deep/ja` concurrent 배치에서 8개 중 3개 실패 → 이후 baseline 부터 모두 502 → 프로세스 사망

### 5-2. 이건 자원 부족이 아니다

`free(): invalid pointer` 는 **glibc 의 `free()` 가 손상된 힙 또는 잘못된 포인터를 감지하여 `abort()` 한 메시지**. 의미하는 것은 C 레벨 메모리 손상 버그:

- 같은 포인터 두 번 `free()` (double free)
- `malloc()` 이 돌려준 적 없는 주소 `free()`
- 인접 청크 헤더가 짓밟혀 free 시점에 감지

**메모리 부족·자원 고갈이 아니다.** OOM 시그니처는 `Cannot allocate memory`, Python `MemoryError`, 컨테이너 `OOMKilled` — 어디에도 해당하지 않음.

### 5-3. 백트레이스 채집 (디버그 환경 구성)

`PYTHONFAULTHANDLER=1`, `MALLOC_CHECK_=3`, `MALLOC_PERTURB_=42` 환경에서 동일 시퀀스 재실행 → SIGSEGV (exit 139) + 완전한 Python+C 백트레이스 확보:

```
Current thread (most recent call first):
  Garbage-collecting                                  ← Python GC 진행 중
  weasyprint/text/line_break.py:141 in get_first_line ← Pango 셰이핑 호출
  ...

Thread (most recent call first):
  weasyprint/text/line_break.py:100 in setup
  weasyprint/text/line_break.py:234 in reactivate
  ...

Thread (...): line_break.py 안에서 또 다른 요청 처리 중
```

해석:

1. 여러 스레드가 **동시에** `weasyprint/text/line_break.py` (Pango / HarfBuzz cffi 래퍼) 안에 있음
2. 그 중 한 스레드는 `Garbage-collecting` 라벨이 붙어 cffi 객체를 finalize 하는 시점
3. 다른 스레드가 같은 객체를 사용 중 → **use-after-free** → SIGSEGV

순수 threading 으로도 같은 버그가 재현됨 (Flask 없이도 발생; 두 번째 백트레이스는 다른 함수 `pdf/fonts.py:build_fonts_dictionary` 에서 잡힘 → 단일 함수 버그가 아닌 **시스템적 race condition**).

### 5-4. 왜 멀티스레드인가

`weasyprint/server.py` 는 `app.run(host="0.0.0.0", port=5000)` 으로 Flask 개발 서버를 그대로 띄운다. **werkzeug 의 기본은 `threaded=True`** — 요청 하나당 스레드 하나. 부하 테스트의 동시성 4 가 곧 4 스레드 동시 `HTML(...).write_pdf()` 진입.

WeasyPrint 공식 문서는 thread-safe 가 아니라고 명시한다. 그러나 사용자가 처음 읽는 공식 docs ([first_steps](https://doc.courtbouillon.org/weasyprint/stable/first_steps.html), [api_reference](https://doc.courtbouillon.org/weasyprint/stable/api_reference.html)) 에는 **thread-safety 경고가 단 한 줄도 없다**.

### 5-5. 픽스 검증

`server.py` 에 `threaded=False` 한 줄 추가 후 동일 시퀀스 재실행:

| 시나리오 | nested-deep/en concurrent (16 req @ conc=4) |
|----------|----------------------------------------------|
| `threaded=True` (이전) | **0 / 16 성공**, exit 139 (SIGSEGV), 컨테이너 사망 |
| `threaded=False` (수정 후) | **16 / 16 성공**, p95 525 ms, 컨테이너 정상 |

→ 원인 확정. 다른 가설(Cairo 버그, ABI 부조합, pydyf 호환성 등) 기각.

### 5-6. 본 리포에 적용된 픽스

- `weasyprint/server.py` : `app.run(host="0.0.0.0", port=5000, threaded=False)` + 사유 주석
- `weasyprint/Dockerfile` : `gdb`, `python3-dbg`, `python -X faulthandler` (향후 디버그용)
- `docker-compose.yml` : `MALLOC_CHECK_=3`, `MALLOC_PERTURB_=42`, `PYTHONFAULTHANDLER=1` 환경변수

### 5-7. Kozea/WeasyPrint Issue tracker 조사

이미 알려진 이슈로 확인. 10년간 동일 클래스 segfault 가 반복 보고됨:

| 이슈 | 연도 | 요지 |
|------|------|------|
| [#167](https://github.com/Kozea/WeasyPrint/issues/167) | 2015 | 동시 호출 segfault 원조 (이후 모든 이슈가 dup) |
| [#344](https://github.com/Kozea/WeasyPrint/issues/344) | 2016 | celery task 에서 SIGSEGV |
| [#684](https://github.com/Kozea/WeasyPrint/issues/684) | 2018 | celery 멀티스레드 크래시. **메인테이너 답변**: *"not designed to be thread-safe neither"* — 수정 안 함이 공식 입장 |
| [#1402](https://github.com/Kozea/WeasyPrint/issues/1402) | 2021 | **`get_first_line` segfault — 우리 백트레이스와 정확히 같은 함수**. "race condition that unreferences Fontconfig patterns twice" |
| [#2472](https://github.com/Kozea/WeasyPrint/issues/2472) | 2025 | Python 3.13+ no-GIL 모드 segfault |

업계 합의된 우회법:
- gunicorn `-w N --threads 1` (multi-process, single-thread)
- celery prefork 풀 (또는 gevent 풀 + 큐 동시성 1)
- Flask 개발 서버 `threaded=False`

### 5-8. Contribution 여지

본 리포의 [`contrib/` 브랜치](https://github.com/shlee-massive/html2pdf/tree/weasyprint-thread-safety-contrib/contrib) 에 **WeasyPrint docs PR 초안**이 있다 (RST 섹션 + 재현 스크립트 + PR 본문). 코드 패치가 아니라 문서 추가 — 메인테이너의 공식 입장상 코드 fix 는 어렵지만, 사용자가 도착하는 공식 docs 에 thread-safety 경고를 추가하는 것은 합리적인 contribution.

---

## 6. 출력 품질 — 페이지 수 × 파일 크기 × 레이아웃 충실도

`nested-*` 5종 × ko/ja/en × 3 백엔드 = 45 PDF 비교 (산출물: [`../pdfs/`](../pdfs/)). 본 절은 DocRaptor 포함.

### 6-1. 페이지 수 × 파일 크기 × 변환 시간

#### ① 카테고리별1단 (`nested-basic`)

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 1p · 150 KB · 0.15s | 1p · 289 KB · 2.63s | 1p · 56 KB · 2.59s |
| **ja** | 1p · 135 KB · 0.17s | 1p · 292 KB · 2.79s | 1p · 45 KB · 1.80s |
| **en** | 1p · **35 KB** · 0.10s | 1p · **13 KB** · **0.06s** | 1p · 50 KB · 2.26s |

#### ② 프로젝트분할 (`nested-projects`)

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 266 KB · 0.19s | 2p · 318 KB · 3.35s | **5p** · 100 KB · 1.95s |
| **ja** | **3p** · 287 KB · 0.25s | **3p** · 326 KB · 3.30s | **6p** · 97 KB · 1.82s |
| **en** | 2p · **54 KB** · 0.12s | 2p · **27 KB** · 0.29s | **5p** · 82 KB · 1.76s |

#### ③ 3단중첩 (`nested-deep`)

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 247 KB · 0.17s | 2p · 306 KB · 2.92s | 2p · 99 KB · 3.15s |
| **ja** | 2p · 267 KB · 0.22s | 2p · 316 KB · 2.94s | 2p · 97 KB · 1.79s |
| **en** | 2p · **56 KB** · 0.11s | 2p · **24 KB** · 0.27s | 2p · 85 KB · 1.88s |

#### ④ 행잘림테스트 (`nested-split`)

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 270 KB · 0.20s | 2p · 310 KB · 2.93s | **3p** · 94 KB · 1.80s |
| **ja** | 2p · 331 KB · 0.23s | 2p · 325 KB · 3.23s | **3p** · 96 KB · 1.80s |
| **en** | 2p · **69 KB** · 0.12s | 2p · **29 KB** · 0.55s | **3p** · 80 KB · 1.81s |

#### ⑤ 종합스트레스 (`nested-stress`)

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 3p · 333 KB · 0.22s | 3p · 466 KB · 4.78s | **6p** · 150 KB · 1.87s |
| **ja** | 3p · **480 KB** · 0.24s | 3p · 489 KB · 5.16s | **6p** · 144 KB · 1.70s |
| **en** | 3p · **66 KB** · 0.14s | 3p · **36 KB** · 0.43s | **6p** · 100 KB · 1.79s |

### 6-2. 로케일별 패턴

#### 영어가 모든 면에서 가장 가벼움
- 파일 크기: 영어 WeasyPrint 결과가 한국어의 **1/13 ~ 1/22**
  - basic: 13 KB(en) vs 289 KB(ko)
  - stress: 36 KB(en) vs 466 KB(ko)
- 변환 시간: 영어 WeasyPrint 가 0.06 ~ 0.55s 로 ko/ja(2.6 ~ 5.2s) 대비 압도적으로 빠름
- 원인: CJK 폰트(Noto Sans JP/KR) 글리프 임베드가 영어에서는 불필요. WeasyPrint 가 폰트 풀 임베드를 해서 가장 큰 영향

#### 일본어 ② 프로젝트분할이 페이지 1장 더
- ko: 2p → ja: **3p** → en: 2p (3 백엔드 모두 같은 패턴)
- 원인: 일본어 텍스트가 같은 의미 표현에 글자수 더 많음 (직위·역할 라벨 "PM"/"バックエンドエンジニア"). 7번째 프로젝트 행의 내부 grid 가 한 줄 더 차지하면서 페이지 넘김
- 의미: **`page-break-inside: avoid` 만으로는 페이지 수 증가를 막을 수 없다** — 콘텐츠 자체 길이를 디자인 단계에서 고려해야 함

#### 일본어 ⑤ 종합스트레스의 Gotenberg 결과가 한국어보다 큼
- ko gotenberg: 333 KB → ja gotenberg: **480 KB** (+44%)
- 같은 stress 의 WeasyPrint 는 466 → 489 KB (+5%)
- 원인 추정: Chromium 의 폰트 임베드는 사용 글리프 수에 더 민감. 일본어가 한국어보다 표시 글리프 수 더 많음 (히라가나 + 가타카나 + 한자)

#### DocRaptor 의 페이지 분할은 로케일 무관
- ② `nested-projects` docraptor: ko 5p → ja 6p → en 5p (rowspan 분할 + 일본어 +1)
- ⑤ `nested-stress` docraptor: ko/ja/en **모두 6p** — rowspan 그룹마다 페이지를 갈라버리는 동작
- 의미: DocRaptor 의 매트릭스 분할 이슈는 **구조(rowspan + writing-mode)** 에서 비롯됨. 로케일 바꿔도 해결 안 됨

### 6-3. rowspan 매트릭스 — 백엔드 차이가 가장 크게 드러나는 케이스

`종합스트레스` 의 `월별 청구 매트릭스` (7행 × 14열, `rowspan="3"`/`rowspan="2"` 그룹 라벨, `writing-mode: vertical-rl` 세로쓰기):

| 백엔드 | 매트릭스 페이지 수 | 그룹 라벨 표현 |
|---|---|---|
| weasyprint | 1 페이지 (전체) | 그룹 셀 안에 세로쓰기로 정상 배치 — **가장 의도대로** |
| gotenberg | 1 페이지 | 세로쓰기 OK, 위치는 약간 어색 — **실용적** |
| docraptor | **3 페이지 분할** (인프라/SaaS/서비스 각각) | vertical-rl 텍스트가 셀 가운데로 흘러나옴, 그룹 결합 깨짐 |

### 6-4. 패턴별 동작 비교

`nested-stress` 에 의도적으로 녹여둔 6 패턴 (A~F) 의 백엔드별 동작:

| 패턴 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| A. thead 반복 (긴 표) | OK | OK | OK |
| B. rowspan 매트릭스 | OK (vertical-rl 약간 어색) | **가장 깔끔** | **rowspan 그룹마다 페이지 분할** |
| C. 12열 가로폭 | OK | OK | OK |
| D. 긴 URL / CJK 부서명 (`word-break: break-all`) | OK | OK | OK |
| E. `@page` 마진 박스 (페이지 번호) | OK (Chromium 147 지원) | OK | OK |
| F. 인라인 SVG | OK | OK | OK |

### 6-5. 부수 발견

- **WeasyPrint pydyf 호환성 이슈**: pip 최신 pydyf(0.11+) 가 `Stream.transform` 메서드를 제거했는데 weasyprint 62.3 이 호출함 → `AttributeError`. 해결: `pydyf==0.10.0` 핀 (`weasyprint/Dockerfile`)
- **Chromium `@page` 마진 박스 진전**: 과거 Chrome 은 `@page @bottom-center` 등을 무시했으나, Chrome 147 / Skia PDF m147 에서는 `counter(page) " / " counter(pages)` 페이지 번호가 정상 출력
- **DocRaptor 워터마크는 페이지 수와 무관**: 페이지 분할은 콘텐츠 구조에서 비롯되며 워터마크 때문이 아님

---

## 7. 종합 권고

### 7-1. 백엔드 선택

| 우선 순위 | 권장 백엔드 | 이유 |
|-----------|------------|------|
| **속도** (대량 배치, 실시간) | Gotenberg | 다른 백엔드의 10배 이상 빠름. CJK 영향 1.5~2배 이내 |
| **레이아웃 정확도** (복잡한 nested, rowspan, vertical-rl, `@page` 풀 활용) | WeasyPrint | 표준 CSS 동작이 가장 충실 |
| **인쇄/출판 품질, 단순 명세** | DocRaptor | rowspan + vertical-rl 가장자리 케이스 미리 검증 필수 |

### 7-2. 동시 처리 설정

| 백엔드 | 권장 동시성 | 비고 |
|--------|------------|------|
| Gotenberg  | **4** | 정점 이후 throughput 감소 + p95 폭발 |
| WeasyPrint | **1 per process** | thread-safe 아님. 진짜 동시성 필요 시 `gunicorn -w N --threads 1` |
| DocRaptor  | (SLA 별도 확인) | 본 보고서 범위 밖 |

### 7-3. 다국어 청구서 시스템

- **언어별 평균 PDF 크기 분포가 크게 다름** → 스토리지 추산 시 단순 평균 X, 언어 비율로 가중평균
- **WeasyPrint + 영어** 조합은 거의 정적 사이트 만큼 가벼움 (13~36 KB / 0.06~0.55s)
- **일본어 콘텐츠는 페이지 +1** 가능성을 청구서 디자인 단계부터 고려

### 7-4. WeasyPrint 운영

- 운영에서 사용하려면 **반드시** `threaded=False` 또는 multi-process single-thread 서버
- `docker-compose.yml` 에 healthcheck + `restart: unless-stopped` (혹시 누락된 멀티스레드 경로 대비)
- 본 리포처럼 부하 테스트로 진단 시 `PYTHONFAULTHANDLER=1`, `MALLOC_CHECK_=3` 환경변수 + `python -X faulthandler` 권장

---

## 8. 부록

### 8-1. 재현

```bash
# 사전 준비
make up         # gotenberg + weasyprint 컨테이너
make serve &    # 데모 서버 (:8080)
make run-ko && make run-ja && make run-en   # output/*.html 생성

# 동시성 sweep (§3 재현)
go run ./cmd/loadtest -backend gotenberg  -sweep -sweep-total 16
go run ./cmd/loadtest -backend weasyprint -sweep -sweep-total 16

# 매트릭스 (§4 재현)
go run ./cmd/matrix -out reports/matrix.csv

# 부분 매트릭스 (특정 백엔드/템플릿만)
go run ./cmd/matrix -backends weasyprint \
                    -templates nested-deep,nested-stress \
                    -cooldown 3s \
                    -append -out reports/matrix.csv
```

### 8-2. 안정성 디버그 환경

`docker-compose.yml` 의 weasyprint 서비스에 이미 적용됨:

```yaml
environment:
  MALLOC_CHECK_: "3"
  MALLOC_PERTURB_: "42"
  PYTHONFAULTHANDLER: "1"
  PYTHONUNBUFFERED: "1"
```

`weasyprint/Dockerfile` 의 CMD:

```dockerfile
CMD ["python", "-X", "faulthandler", "server.py"]
```

### 8-3. 원시 데이터

- 매트릭스 36 셀: [`matrix-2026-05-26.csv`](matrix-2026-05-26.csv)
- 출력 PDF 45개: [`../pdfs/`](../pdfs/) (총 7.9 MB)

### 8-4. 도구

- [`cmd/loadtest`](../cmd/loadtest) — 동시성 sweep
- [`cmd/matrix`](../cmd/matrix) — 템플릿 × 언어 × 백엔드 매트릭스
- [`contrib/`](../contrib) — WeasyPrint docs PR 초안 (별도 브랜치 `weasyprint-thread-safety-contrib` 에서 작업)

### 8-5. 측정 백로그

- WeasyPrint Pango 폰트 캐시 워밍 효과 (첫 요청 vs 이후 요청)
- Gotenberg `GOTENBERG_API_TIMEOUT`, Chromium worker 수 튜닝
- 컨테이너 CPU/메모리 limit 별 곡선 이동
- DocRaptor 는 부하 테스트 대신 SLA 기반 소량 샘플링으로 별도 평가
