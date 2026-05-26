# 부하 테스트 보고서 인덱스

이 디렉터리는 `POST /api/convert` 엔드포인트의 PDF 변환 성능 측정 결과를 담는다.
대상 백엔드는 **Gotenberg**(Chromium 기반) 와 **WeasyPrint**(Cairo/Pango 기반)이며,
DocRaptor 는 외부 유료 API 라 부하 테스트 대상에서 제외했다.

## 어떤 보고서를 읽어야 하나

| 알고 싶은 것 | 보고서 | 측정 도구 |
|--------------|--------|-----------|
| 단일 템플릿에서 동시 요청 수에 따라 처리량·지연이 어떻게 변하는가 | [`load-test-2026-05-26.md`](load-test-2026-05-26.md) | [`cmd/loadtest`](../cmd/loadtest) |
| 템플릿/언어 조합 36개에서 baseline·동시성-4 성능이 어떻게 다른가 | [`matrix-2026-05-26.md`](matrix-2026-05-26.md) | [`cmd/matrix`](../cmd/matrix) |
| 출력 PDF의 페이지 수·파일 크기·레이아웃 충실도 (rowspan, vertical-rl 등) | [`pdf-output-comparison-2026-05-26.md`](pdf-output-comparison-2026-05-26.md) | 수기 비교 + 산출물 [`../pdfs/`](../pdfs/) |

원시 측정 데이터: [`matrix-2026-05-26.csv`](matrix-2026-05-26.csv)

## 세 보고서의 관계

| 보고서 | 축 | 묻는 질문 |
|--------|----|-----------|
| `load-test` | **동시성** (1→16) | "이 부하에서 무너지는 지점은 어디인가?" |
| `matrix` | **콘텐츠** (6 템플릿 × 3 언어) | "어떤 콘텐츠가 가장 느리고 어떤 백엔드가 그걸 못 견디는가?" |
| `pdf-output-comparison` | **산출물 품질** (페이지·크기·레이아웃) | "결과 PDF 자체는 백엔드별로 어떻게 다른가?" (성능 ×, 품질 ○) |

- `load-test` + `matrix` 는 **성능** 두 축, `pdf-output-comparison` 은 **품질** 축.
- 어느 백엔드를 운영에 쓸지 결정하려면 성능 보고서(throughput/latency) 와 품질 보고서(레이아웃 정확도) 를 함께 봐야 한다.

## 한 줄 결론

- **속도**: Gotenberg 가 모든 셀에서 압도적으로 빠르고 안정적. WeasyPrint 는 CJK 콘텐츠에서 영문 대비 최대 57배 느림.
- **안정성**: 부하 테스트 중 WeasyPrint 프로세스가 SIGSEGV 로 사망한 케이스를 백트레이스로 확정 진단함 — **werkzeug 의 기본 `threaded=True` + WeasyPrint thread-safety 미보장**의 충돌. 본 리포의 `weasyprint/server.py` 에 `threaded=False` 픽스 적용 완료. 자세한 분석은 [`matrix-2026-05-26.md`](matrix-2026-05-26.md) §4 참조.
- **품질**: 복잡한 nested · rowspan · vertical-rl 정확도는 WeasyPrint 가 가장 충실. Gotenberg 도 무난. DocRaptor 는 rowspan 매트릭스를 페이지마다 분할하는 회귀가 있다.
- **권장**: 일반 운영은 Gotenberg, 정밀 인쇄·복잡 레이아웃이 필요하면 WeasyPrint (단 CJK + 동시성은 회피).

## 재현

```bash
make up        # gotenberg + weasyprint 컨테이너
make serve &   # 데모 서버 (:8080)
make run-ko && make run-ja && make run-en   # output/*.html 생성

# load-test 재현
go run ./cmd/loadtest -backend gotenberg  -sweep -sweep-total 16
go run ./cmd/loadtest -backend weasyprint -sweep -sweep-total 16

# matrix 재현
go run ./cmd/matrix -out reports/matrix.csv
```

## 향후 측정 백로그

- WeasyPrint Pango 폰트 캐시 워밍 효과 (반복 동일 요청 시 비용 변화)
- Gotenberg `GOTENBERG_API_TIMEOUT`, Chromium worker 수 튜닝 영향
- 컨테이너 CPU·메모리 limit 별 곡선 이동
- DocRaptor 는 부하 테스트가 아니라 SLA 기반 소량 샘플링으로 별도 평가
