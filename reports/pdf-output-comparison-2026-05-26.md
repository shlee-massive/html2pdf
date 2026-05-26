# HTML → PDF 변환 백엔드 비교 — Nested Table 5종 × 3 로케일 × 3 백엔드 = 45개 PDF

> 이 보고서는 **출력물의 페이지 수·파일 크기·레이아웃 충실도**를 본다 (변환 시간은 부차적).
> 처리량/지연 중심 성능 보고서는 [`load-test-2026-05-26.md`](load-test-2026-05-26.md) (동시성 곡선),
> [`matrix-2026-05-26.md`](matrix-2026-05-26.md) (36 셀 매트릭스) 참조.
> 인덱스: [`README.md`](README.md)

생성: 2026-05-26 · 환경: macOS / Docker · 샘플 서버: `http://localhost:8080` · 산출물: [`../pdfs/`](../pdfs/)

---

## 1. 테스트 대상

### 샘플 5종 (각각 ko/ja/en 3 로케일)

| # | 파일 (ko 기준) | 의도 | ja/en 변형 |
|---|---|---|---|
| ① | `web/nested-basic.html` | **카테고리별 1단 중첩**. 외곽 표(카테고리) × 내부 표(품목 명세). 기본 케이스 | `-ja.html` (¥, Noto Sans JP), `-en.html` ($, 시스템 폰트) |
| ② | `web/nested-projects.html` | **프로젝트별 다중 nested**. 7개 프로젝트 행, 각 행에 grid 로 인력/자재/기타 표 3개. `page-break-inside: avoid` 적용 | 동일 |
| ③ | `web/nested-deep.html` | **3단 중첩**. 부서 → 프로젝트 → 항목 (L1/L2/L3 표 중첩) | 동일 |
| ④ | `web/nested-split.html` | **행 잘림 테스트**. ②와 같은 구조이나 `page-break-inside: auto` + 데이터 분량 조절로 대형 프로젝트 행이 페이지 경계 횡단 | 동일 |
| ⑤ | `web/nested-stress.html` | **종합 스트레스 (A+B+C+D+E+F)**. 한 문서에 6가지 패턴을 모두 녹임 | 동일 |

### ⑤ 종합 스트레스에 녹인 패턴 (A~F)

| 패턴 | 확인 위치 |
|---|---|
| **A.** 50+ 행 단일 표, thead 자동 반복 | `세부 청구 명세 / 詳細明細 / Itemized Details` 표 (2~3 페이지 걸침) |
| **B.** rowspan 매트릭스 (`rowspan="3"`, `rowspan="2"` 그룹 셀, `writing-mode: vertical-rl` 그룹 라벨) | `월별 청구 매트릭스 / 月別請求マトリックス / Monthly Billing Matrix` 표 (1 페이지) |
| **C.** 14 컬럼 가로폭 (월×12 + 항목 + 합계) — A4 가로폭 한계 | 같은 매트릭스 |
| **D.** 끊을 데 없는 긴 URL · 부서명 — `word-break: break-all` | 명세 표의 일부 셀 (1번 / 5번 / 10번 / 23번 / 30번 행) |
| **E.** `@page` 마진 박스 — 페이지 번호 `N / N`, 상단 식별자, 하단 발행일 | 모든 페이지 상하단 |
| **F.** 인라인 SVG — 로고 / 바코드 / QR / 사인 | 헤더 / 푸터 사인 박스 |

### 백엔드 3종

| 백엔드 | 엔진 | 비고 |
|---|---|---|
| `gotenberg` | Chromium Headless 147 (Skia/PDF m147) | 로컬 Docker |
| `weasyprint` | WeasyPrint 62.3 + pydyf 0.10.0 | 로컬 Docker. pydyf 0.11+ 와 호환 안 됨 → 0.10.0 핀 |
| `docraptor` | Prince 15.1 (PrinceXML) | 외부 클라우드 API, 워터마크 포함 |

---

## 2. 페이지 수 × 파일 크기 × 변환 시간 (전체 45개)

### ① 카테고리별1단

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 1p · 150 KB · 0.15s | 1p · 289 KB · 2.63s | 1p · 56 KB · 2.59s |
| **ja** | 1p · 135 KB · 0.17s | 1p · 292 KB · 2.79s | 1p · 45 KB · 1.80s |
| **en** | 1p · **35 KB** · 0.10s | 1p · **13 KB** · **0.06s** | 1p · 50 KB · 2.26s |

### ② 프로젝트분할

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 266 KB · 0.19s | 2p · 318 KB · 3.35s | **5p** · 100 KB · 1.95s |
| **ja** | **3p** · 287 KB · 0.25s | **3p** · 326 KB · 3.30s | **6p** · 97 KB · 1.82s |
| **en** | 2p · **54 KB** · 0.12s | 2p · **27 KB** · 0.29s | **5p** · 82 KB · 1.76s |

### ③ 3단중첩

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 247 KB · 0.17s | 2p · 306 KB · 2.92s | 2p · 99 KB · 3.15s |
| **ja** | 2p · 267 KB · 0.22s | 2p · 316 KB · 2.94s | 2p · 97 KB · 1.79s |
| **en** | 2p · **56 KB** · 0.11s | 2p · **24 KB** · 0.27s | 2p · 85 KB · 1.88s |

### ④ 행잘림테스트

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 2p · 270 KB · 0.20s | 2p · 310 KB · 2.93s | **3p** · 94 KB · 1.80s |
| **ja** | 2p · 331 KB · 0.23s | 2p · 325 KB · 3.23s | **3p** · 96 KB · 1.80s |
| **en** | 2p · **69 KB** · 0.12s | 2p · **29 KB** · 0.55s | **3p** · 80 KB · 1.81s |

### ⑤ 종합스트레스

| 로케일 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **ko** | 3p · 333 KB · 0.22s | 3p · 466 KB · 4.78s | **6p** · 150 KB · 1.87s |
| **ja** | 3p · **480 KB** · 0.24s | 3p · 489 KB · 5.16s | **6p** · 144 KB · 1.70s |
| **en** | 3p · **66 KB** · 0.14s | 3p · **36 KB** · 0.43s | **6p** · 100 KB · 1.79s |

---

## 3. 로케일별 패턴 — 핵심 관찰

### 3-1. 영어가 모든 면에서 가장 가벼움
- **파일 크기**: 영어 WeasyPrint 결과가 한국어의 **1/13 ~ 1/22**
  - basic: 13 KB(en) vs 289 KB(ko)
  - stress: 36 KB(en) vs 466 KB(ko)
- **변환 시간**: 영어 WeasyPrint 가 0.06 ~ 0.55s 로 한국어/일본어(2.6 ~ 5.2s)보다 압도적으로 빠름
- **원인**: CJK 폰트(Noto Sans JP/Noto Sans KR) 글리프 임베드가 영어에서는 불필요. WeasyPrint 가 폰트 풀 임베드를 하기 때문에 가장 큰 영향.
- **의미**: 같은 백엔드라도 콘텐츠 언어에 따라 산출물 크기·속도가 10배 이상 차이남. **다국어 청구서 시스템에서 평균 PDF 크기 추산할 때 언어 비율을 반드시 고려해야 함.**

### 3-2. 일본어 ② 프로젝트분할이 페이지 1장 더 늘어남
- ko: 2p → ja: **3p** → en: 2p
- gotenberg 와 weasyprint 모두 같은 패턴 (docraptor 도 +1)
- **원인 추정**: 일본어 텍스트가 같은 의미를 표현할 때 한국어/영어보다 글자수가 더 많아짐 (특히 役職名 같은 직위·역할 라벨, 한국어 "PM"/"백엔드 엔지니어" vs 일본어 "PM"/"バックエンドエンジニア"). 7번째 프로젝트의 내부 nested grid 가 한 줄 더 차지하면서 페이지를 넘김.
- **의미**: `page-break-inside: avoid` 가 잘 동작하더라도, 콘텐츠 길이 차이로 페이지 수 자체가 늘어남.

### 3-3. 일본어 ⑤ 종합스트레스의 gotenberg 결과가 한국어보다 큼
- ko gotenberg: 333 KB → ja gotenberg: **480 KB** (+44%)
- 같은 stress 의 WeasyPrint 는 466 → 489 KB 로 거의 비슷 (+5%)
- **원인 추정**: Chromium 의 폰트 임베드 전략이 사용 글리프 수에 더 민감. 일본어는 한국어보다 표시 글리프 수가 더 많아 (한국어는 한글 조합 자모 + 일부 한자, 일본어는 히라가나 + 가타카나 + 한자) 임베드 서브셋이 커짐.

### 3-4. DocRaptor 의 페이지 분할은 로케일 무관
- ② 프로젝트분할 docraptor: ko 5p → ja 6p → en 5p (rowspan/매트릭스 분할 동작 + 일본어 추가 1p)
- ⑤ 종합스트레스 docraptor: ko/ja/en **모두 6p** — rowspan 그룹마다 페이지 갈라버리는 동작은 콘텐츠 언어와 무관하게 일관됨
- **의미**: DocRaptor 의 매트릭스 분할 이슈는 콘텐츠가 아닌 **구조(rowspan + writing-mode)** 에서 비롯됨. 로케일을 바꿔도 해결 안 됨.

---

## 4. 패턴별 동작 비교 (A~F)

(⑤ 종합 스트레스 기준, 3 로케일 모두에서 일관되게 관찰됨)

| 패턴 | gotenberg | weasyprint | docraptor |
|---|---|---|---|
| **A.** thead 반복 (긴 표) | OK | OK | OK |
| **B.** rowspan 매트릭스 | OK (vertical-rl 약간 어색) | **가장 깔끔** (vertical-rl 라벨이 그룹 셀 안쪽에 세로 정렬) | **rowspan 그룹마다 페이지 분할** + vertical-rl 라벨이 셀 한가운데로 흘러나옴 |
| **C.** 12열 가로폭 | OK | OK | OK |
| **D.** 긴 URL / 끊을 수 없는 CJK 문자열 | OK (`word-break: break-all`) | OK | OK |
| **E.** `@page` 마진 박스 (페이지 번호) | OK (Chromium 147 지원) | OK | OK |
| **F.** 인라인 SVG | OK | OK | OK |

### CJK 처리 추가 관찰
- `nested-stress` 의 `.longcell` 안에 일부러 넣은 긴 CJK 부서명 (`株式会社マッシブリンクスデータプラットフォーム本部...` / `MassiveLinksIncDataPlatformDivision...`) 도 모든 백엔드에서 `word-break: break-all` 로 정상 줄바꿈
- 일본어의 한자/가나 혼용 시 줄바꿈 위치가 백엔드별로 미세하게 다름 — 본 비교에선 큰 가독성 차이 없음

---

## 5. rowspan 매트릭스 — 백엔드 차이가 가장 크게 드러나는 케이스

### 무엇이 다른가
- `종합스트레스` 의 `월별 청구 매트릭스` : 7행 × 14열, 외곽 컬럼은 `rowspan="3"` (인프라) / `rowspan="2"` (SaaS, 서비스) 의 그룹 라벨
- 그룹 라벨 셀에 `writing-mode: vertical-rl` 로 세로쓰기

### 결과 (3 로케일 동일 패턴)

| 백엔드 | 매트릭스 페이지 수 | 그룹 라벨 표현 |
|---|---|---|
| weasyprint | 1 페이지 (전체) | 그룹 셀 안에 세로쓰기로 정상 배치 — **가장 의도대로** |
| gotenberg | 1 페이지 | 세로쓰기 OK, 위치는 약간 어색 — **실용적** |
| docraptor | **3 페이지로 분할** (인프라 / SaaS / 서비스 각각) | vertical-rl 텍스트가 셀 가운데로 흘러나옴, 그룹 결합 깨짐 |

### 확인 방법

```
pdfs/종합스트레스-{ko,ja,en}-weasyprint.pdf   ← 1 페이지에 매트릭스 전체
pdfs/종합스트레스-{ko,ja,en}-gotenberg.pdf    ← 1 페이지에 매트릭스 전체
pdfs/종합스트레스-{ko,ja,en}-docraptor.pdf    ← 1~3 페이지에 분할
```

같은 로케일끼리 백엔드별로 나란히 띄워서 비교하면 차이가 가장 명확하다.

---

## 6. 종합 추천

### 본 데모 시나리오(nested table 위주)에서

| 우선 순위 | 권장 백엔드 | 이유 |
|---|---|---|
| **속도가 중요** (대량 배치, 실시간) | gotenberg | 다른 두 백엔드의 10배 이상 빠름. 영어는 거의 즉시 |
| **레이아웃 정확도가 중요** (복잡한 nested · rowspan · vertical-rl · `@page` 풀 활용) | weasyprint | 표준 CSS 동작이 가장 충실. 단 CJK 콘텐츠에서 변환 시간 ↑ |
| **인쇄/출판 품질, 단순 명세** | docraptor (Prince) | 단순 청구서엔 좋지만 rowspan + vertical-rl 가장자리 케이스에선 페이지를 분할해버림. 미리 검증 필수 |

### 다국어 청구서 시스템을 짠다면

- **언어별 평균 PDF 크기 분포**가 크게 다름 → 스토리지 용량 추산 시 단순 평균 X, 언어 비율로 가중평균 필요
- **WeasyPrint + 영어** 조합은 거의 정적 사이트 만큼 가벼움 (13~36 KB / 0.06~0.55s) → 영어권 트래픽이 많다면 변환 비용 거의 무료
- **일본어 콘텐츠는 페이지 +1** 가능성을 청구서 디자인 단계부터 고려 (`page-break-inside: avoid` 만으로는 막을 수 없음 — 콘텐츠 자체 길이가 늘어남)

---

## 7. 부수 발견

- **WeasyPrint pydyf 호환성 이슈**: pip 최신 pydyf(0.11+) 는 `Stream.transform` 메서드를 제거했는데 weasyprint 62.3 은 이를 호출함 → `AttributeError`. 해결: `pydyf==0.10.0` 핀 (`weasyprint/Dockerfile`).
- **Chromium `@page` 마진 박스 진전**: 과거 Chrome 은 `@page @bottom-center` 등을 무시했으나, Chrome 147 / Skia PDF m147 에서는 `counter(page) " / " counter(pages)` 페이지 번호가 정상 출력됨. 비교 시 의외의 발견.
- **DocRaptor 의 워터마크는 페이지 수에 직접 영향 없음**: 페이지 분할은 콘텐츠 구조(rowspan/vertical-rl)에서 비롯되며 워터마크 때문이 아님.

---

## 8. 결과물 위치

```
pdfs/                                                            ← 리포 내 (7.9 MB)
├── 카테고리별1단-{ko,ja,en}-{gotenberg,weasyprint,docraptor}.pdf   (9개)
├── 프로젝트분할-{ko,ja,en}-{gotenberg,weasyprint,docraptor}.pdf      (9개)
├── 3단중첩-{ko,ja,en}-{gotenberg,weasyprint,docraptor}.pdf          (9개)
├── 행잘림테스트-{ko,ja,en}-{gotenberg,weasyprint,docraptor}.pdf      (9개)
└── 종합스트레스-{ko,ja,en}-{gotenberg,weasyprint,docraptor}.pdf      (9개)

총 45개 PDF, 7.9 MB
```

### 자료 활용

- **같은 표 구조의 다국어 비교**: `pdfs/카테고리별1단-ko-weasyprint.pdf` vs `pdfs/카테고리별1단-ja-weasyprint.pdf` vs `pdfs/카테고리별1단-en-weasyprint.pdf`
- **같은 콘텐츠의 백엔드 비교**: `pdfs/종합스트레스-ko-gotenberg.pdf` vs `pdfs/종합스트레스-ko-weasyprint.pdf` vs `pdfs/종합스트레스-ko-docraptor.pdf`
- **로케일 × 백엔드 매트릭스 전부 비교**: 위 두 축의 곱
