# HTML → PDF 백엔드 선정 — 임원 보고용 요약

> 작성일: 2026-06-09 · 1 페이지 요약 · 상세는 [pdf-backend-benchmark-2026-06-08-4way.md](./pdf-backend-benchmark-2026-06-08-4way.md)

## TL;DR

PDF 변환 백엔드 4종 (Gotenberg / WeasyPrint / DocRaptor / @react-pdf/renderer) 의 처리량·안정성·운영 위험을 측정. **Gotenberg 가 종합 1순위**. 단일 양식·인프라 최소화가 목표라면 `@react-pdf/renderer` 도 후보 — 단, 측정 외 운영 위험 4가지 (lock-in, paged media 성숙도, SMP 한자 누락, Node 이벤트 루프 블로킹) 가 도입 조건을 좁힘.

## 측정 범위

- **백엔드 4종**: Gotenberg (Chrome 기반), WeasyPrint (Python), DocRaptor (SaaS, 이번 측정 제외), @react-pdf/renderer (Node 사이드카)
- **시나리오**: invoice + nested-* 5종, **로케일** ko/ja/en
- **부하**: 정점 sweep conc=1/2/4/8/16, **총 2,192 요청 · 0 실패**

## 측정 결과 — 핵심 수치

| 백엔드 | invoice 정점 처리량 (req/s) | invoice 정점 conc | nested-* 시나리오 |
|---|---|---|---|
| **Gotenberg** | ko 13.4 / ja 10.7 / en **22.5** | 8 | 5종 전부 처리 가능 |
| **@react-pdf/renderer** | ko **14.6** / ja **17.9** / en 14.0 | 8 | **처리 불가** (JSON 입력만) |
| **WeasyPrint** | ko 0.26 / ja 0.22 / en 0.26 | conc 무관 | 5종 전부 처리 가능 |
| DocRaptor | 본 측정 제외 (test API 키 미설정) | — | — |

**의외 발견**: invoice 한정 ko/ja 정점에선 reactpdf 가 Gotenberg 보다 빠름 (ja 1.67×, ko 1.09×). 단 영문에선 Gotenberg 가 1.61× 우세.

**WeasyPrint 의 0.25 req/s 는 엔진 한계가 아니라 환경 제약치** — `threaded=False` (안정성 fix) + 매 요청 Google Fonts CDN fetch 의 곱. 환경 개선 시 재측정 필요.

## ⚠️ 측정의 한계 (결론을 보기 전 필수 참조)

본 측정의 수치는 다음 한계 아래에서 읽어야 함. **현 측정 환경의 실측치이지 엔진 자체의 능력 비교가 아닐 수 있는 항목** 위주:

1. **WeasyPrint invoice 의 4초+ baseline 은 Google Fonts CDN fetch 비용** — invoice 템플릿만 `@import` 로 외부 폰트 호출. 매 요청 외부 fetch. nested-* 시나리오 (인라인 CSS) 에선 동일 엔진이 80~770ms 로 동작. **즉 §3 의 WeasyPrint invoice 정점 0.25 req/s 는 엔진 한계가 아니라 환경 제약치**.
2. **WeasyPrint `threaded=False` 는 SIGSEGV 회피 fix** — Flask 단일 스레드로 고정. conc 와 무관하게 직렬 처리 → 0.25 req/s 천장의 또 다른 원인. 운영 환경에서 멀티스레드 활성화 시 천장 사라질 가능성.
3. **ReactPdf 는 다른 작업을 측정** — 입력이 JSON body (3~4KB) 직접 POST 이므로 HTML 파싱 비용 0. 다른 백엔드 (HTML 12~26KB) 와 비교축이 다름. **실제 운영 파이프라인에선 JSON→HTML 변환 단계가 어딘가에 있으므로 reactpdf 의 정점 우위 (ko 1.09×, ja 1.67×) 는 공정 비교가 아닐 수 있음**.
4. **gotenberg/weasyprint sweep cold-start 미보정** — reactpdf 만 폰트 등록 워밍업 받음. gotenberg conc=1/ko baseline 736ms (콜드) vs conc=2/ko 153ms 의 격차가 그 증거. sweep 시작점 약간 왜곡.
5. **백엔드별 invoice 템플릿이 다른 코드** — Gotenberg/WeasyPrint 는 HTML 템플릿, ReactPdf 는 JSX. _시각적으로 동일한 결과를 만든다는 가정만 있고 자동 비교 안 됨_. 다른 PDF 를 만들고 있다면 throughput 비교 자체가 다른 작업 간 비교.

추가 단서: **단일 호스트 측정** (macOS Docker Desktop, 서비스 간 CPU 경쟁) · **DocRaptor 누락** (사실상 3-way) · **invoice 1 종만** (양식 다양성 미반영) · **macOS ARM** (운영 일반적 Linux x86_64 와 다름).

→ 본 보고서의 권고는 위 한계 인지 하에 해석할 것. 결론을 흔들 수 있는 1~3번 항목은 PoC 단계에서 재측정 권장 (다음 액션 후보 참조). 상세 영향 분석은 [4way 본문 §1.6](./pdf-backend-benchmark-2026-06-08-4way.md#16-측정-한계-결과-해석-시-필수-참조) 참조.

## 부하 테스트 결과 (concurrency sweep)

> invoice × 3 로케일 × conc {1,2,4,8,16} 5단계 sweep. 총 reactpdf 768 req + gotenberg·weasyprint 1,536 req = **2,304 req · 0 실패**. 상세는 [4way 본문 §3.5~3.6](./pdf-backend-benchmark-2026-06-08-4way.md).

### Throughput (req/s, 로케일별 conc 별)

| conc | reactpdf ko | ja | en | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1   | 5.97 | 6.05 | 5.26 | 8.29 | 5.72 | 9.29 | 0.21 | 0.22 | 0.26 |
| 2   | 7.66 | 9.52 | 8.65 | 10.80 | 8.19 | 11.12 | 0.24 | 0.22 | **0.26** |
| 4   | 11.58 | 14.35 | 13.52 | 12.89 | 10.70 | 15.29 | 0.22 | 0.22 | 0.24 |
| 8   | 13.95 | **17.91** | **14.00** | **13.40** | **10.70** | **22.51** | **0.23** | **0.22** | 0.24 |
| 16  | **14.62** | 16.68 ▽ | 13.26 ▽ | 11.85 ▽ | 10.39 ▽ | 18.64 ▽ | **0.23** | 0.21 ▽ | 0.24 |

볼드: 각 (백엔드, 로케일) 정점 throughput. ▽: 직전 conc 대비 회귀.

### p95 latency (ms)

| conc | reactpdf ko | ja | en | gotenberg ko | ja | en | weasyprint ko | ja | en |
|----:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1   | 179 | 176 | 270 | 128 | 192 | 123 | 4,315 | 4,696 | 3,939 |
| 2   | 329 | 255 | 265 | 213 | 290 | 540 | 8,658 | 9,248 | 7,741 |
| 4   | 417 | 363 | 398 | 399 | 458 | 260 | 26,440 | 18,358 | 24,521 |
| 8   | 806 | 590 | 816 | 817 | 1,171 | 465 | 34,518 | 37,148 | 40,780 |
| 16  | 1,579 | 1,278 | 1,716 | 2,156 | 2,320 | 1,215 | 77,763 | 80,304 | 80,057 |

### 정점 cross-section (로케일별 최강 백엔드)

| 로케일 | reactpdf 정점 (req/s @ conc) | gotenberg 정점 (req/s @ conc) | weasyprint 정점 (req/s @ conc) | 1위 |
|---|---|---|---|---|
| ko | **14.62** @ 16 | 13.40 @ 8 | 0.26 @ 2 | reactpdf (1.09×) |
| ja | **17.91** @ 8 | 10.70 @ 4·8 | 0.22 @ 1·2·4·8 | reactpdf (1.67×) |
| en | 14.00 @ 8 | **22.51** @ 8 | 0.26 @ 1·2 | gotenberg (1.61×) |

**해석**: ko/ja 는 reactpdf 가 정점에서 앞서지만, conc=16 에서 ko 외 모두 회귀 ▽ → 8 이상에선 효용 체감. gotenberg 는 영문에서 압도적 (22.51 req/s), CJK 에선 폰트 사이즈로 인한 페이로드 증가가 처리량을 압박. weasyprint 는 conc 무관 0.21~0.26 req/s 천장 — `threaded=False` 와 CDN fetch 의 곱.

## 운영 위험 요약 (측정 외)

| 위험 항목 | Gotenberg | reactpdf | WeasyPrint |
|---|---|---|---|
| 다양한 보고서 양식 처리 | 저 | **고** (JSX 재작성) | 저 |
| Paged media (페이지 분할·표 행 잘림) 성숙도 | 저 | **고** (CSS Paged Media 미지원) | 저 |
| 한자 SMP 정확도 (`𠮷` 등) | 저 | **고** (silent 누락) | 저 |
| 단일 프로세스 SPOF | 중 | **고** (Node sync 렌더) | **고** (threaded=False) |
| 외부 CDN 의존 | 중 | 저 | **고** |
| HTML 디자이너 협업 | 가능 | **불가** (JSX 코드) | 가능 |

> **저/중/고 판정 기준** (도입 후 운영 단계에서 발생할 수 있는 사고·비용 부담의 상대 크기):
> - **저** — 표준·일반 사례를 그대로 처리, 별도 우회 코드/모니터링 불필요
> - **중** — 운영상 알아둬야 하나 일반적 완화책 (replica 추가, 폰트 사전 임베드 등) 으로 흡수 가능
> - **고** — 도입 전제 자체를 흔드는 위험. silent 실패 (한자 누락) 또는 양식 추가 시 코드 재작성/엔진 교체가 필요한 수준
>
> 각 행별 구체 기준:
> - **다양한 보고서 양식**: 저 = HTML 템플릿 1회로 처리 / 고 = 양식별 JSX 컴포넌트를 새로 작성
> - **Paged media**: 저 = CSS `@page`·`break-*`·`thead repeat` 표준 준수 / 고 = 표준 미지원, 페이지 분할 수동 구현 필요
> - **SMP 한자 정확도**: 저 = U+20000 이상 한자 정상 렌더 / 고 = silent 누락 (에러 없이 글자가 사라짐)
> - **단일 프로세스 SPOF**: 중 = HTTP 서버 1프로세스지만 내부 워커 풀 보유, replica 로 수평 확장 가능 / 고 = 동기 직렬 렌더, replica 외 다른 완화책 없음
> - **외부 CDN 의존**: 저 = 외부 fetch 없음 / 중 = 폰트만 OS·이미지 임베드로 회피 가능 / 고 = 매 요청 CDN fetch (현재 측정 환경)
> - **HTML 디자이너 협업**: 가능 = HTML/CSS 결과물 그대로 / 불가 = JSX 코드 수정 필요

## 권고

### 1순위 — Gotenberg
- **언제**: 다양한 보고서 양식이 있거나, HTML 디자이너 협업이 필요하거나, 일본어 한자 정확도가 요구될 때 (= 대부분의 일반적 케이스)
- **근거**: 처리량 정점 영문 22.5 req/s, 모든 시나리오 처리, OS 폰트 활용, Chrome 렌더 파이프라인 성숙도

### 2순위 (조건부) — @react-pdf/renderer
- **언제**: 다음 3 조건을 **모두** 만족할 때
  1. 보고서 종류가 1~2개로 고정 (invoice 같은 단일 양식)
  2. HTML 디자이너 협업 불필요
  3. 디자인을 코드로 관리하기 원함
- **근거**: ko/ja invoice 정점에서 reactpdf 우세 (ko 1.09×, ja 1.67×; 영문은 Gotenberg 가 1.61× 역우세), 인프라 최소 (외부 CDN 0, 폰트 임베드), JSX/React 친화
- **주의**: 위 조건이 깨지면 운영 위험 4종이 즉시 활성화

### 3순위 — WeasyPrint
- **언제**: 레이아웃 정확도 (CSS Paged Media 표준 준수) 가 최우선이고, 처리량은 부차적일 때
- **전제**: CSS `@import` (CDN fetch) 제거 + `threaded=False` 해제 가능한 환경. 현재 측정에선 둘 다 미해결로 정점 0.25 req/s 천장.

### 보류 — DocRaptor
이번 측정 미포함. zero-ops (SaaS) 매력이 있으나 비용·외부 의존성 별도 검토 필요.

## 결정에 필요한 추가 정보

도입 결정 전 확인 필요한 항목:

1. **운영 환경의 동시 처리 부하 추정** — 정점 ~15 req/s 가 충분한지, 부족하면 수평 확장 (replica) 전제
2. **장시간 부하 메모리 누수 검증** (PoC 단계 추가 테스트) — 본 측정은 단발 스파이크만
3. **CJK 한자 SMP 영역 사용 빈도** — reactpdf 후보면 사용자 이름·주소의 SMP 한자 빈도 조사 필요
4. **DocRaptor 비용 비교** — API 키 발급 후 4+1-way 측정 재실행 (다음 측정 권고 §7.3)

## 다음 액션 후보

- **즉시 진행 가능**: Gotenberg 단독 도입 (모든 시나리오 안전, 위험 최저)
- **PoC 필요**: reactpdf 도입 검토 시 — 장시간 부하 + SMP 한자 + 수평 확장 측정
- **환경 정비 필요**: WeasyPrint 도입 검토 시 — CSS @import 제거 + threaded=True 안정성 재검증

## 출처

- 본 요약은 [pdf-backend-benchmark-2026-06-08-4way.md](./pdf-backend-benchmark-2026-06-08-4way.md) (§0~§8) 의 발췌·요약
- 측정 CSV: `matrix-4way-2026-06-08.csv` · `matrix-reactpdf-sweep-2026-06-09.csv` · `matrix-htmlbackends-sweep-2026-06-09.csv`
- 샘플 PDF: `pdfs/청구서-{ko,ja,en}-reactpdf.pdf` 외 44개 (Gotenberg/WeasyPrint/DocRaptor × 5 시나리오 × 3 로케일)
