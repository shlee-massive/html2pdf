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

## 운영 위험 요약 (측정 외)

| 위험 항목 | Gotenberg | reactpdf | WeasyPrint |
|---|---|---|---|
| 다양한 보고서 양식 처리 | 저 | **고** (JSX 재작성) | 저 |
| Paged media (페이지 분할·표 행 잘림) 성숙도 | 저 | **고** (CSS Paged Media 미지원) | 저 |
| 한자 SMP 정확도 (`𠮷` 등) | 저 | **고** (silent 누락) | 저 |
| 단일 프로세스 SPOF | 중 | **고** (Node sync 렌더) | **고** (threaded=False) |
| 외부 CDN 의존 | 중 | 저 | **고** |
| HTML 디자이너 협업 | 가능 | **불가** (JSX 코드) | 가능 |

## 권고

### 1순위 — Gotenberg
- **언제**: 다양한 보고서 양식이 있거나, HTML 디자이너 협업이 필요하거나, 일본어 한자 정확도가 요구될 때 (= 대부분의 일반적 케이스)
- **근거**: 처리량 정점 영문 22.5 req/s, 모든 시나리오 처리, OS 폰트 활용, Chrome 렌더 파이프라인 성숙도

### 2순위 (조건부) — @react-pdf/renderer
- **언제**: 다음 3 조건을 **모두** 만족할 때
  1. 보고서 종류가 1~2개로 고정 (invoice 같은 단일 양식)
  2. HTML 디자이너 협업 불필요
  3. 디자인을 코드로 관리하기 원함
- **근거**: ko/ja invoice 정점에서 Gotenberg 우세 (1.09~1.67×), 인프라 최소 (외부 CDN 0, 폰트 임베드), JSX/React 친화
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
