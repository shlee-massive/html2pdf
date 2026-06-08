// react-pdf 사이드카 — html-to-pdf PoC 4번째 백엔드.
//
// POST /pdf       application/json  Invoice 구조 → PDF Buffer
// GET  /health    { ok: true }
//
// 다른 백엔드와의 비교축 차이:
//  - gotenberg / weasyprint / docraptor 는 HTML+CSS 를 받는다.
//  - react-pdf 는 HTML 을 받지 않고 구조화된 데이터(Invoice JSON)를 받는다.
//  → 비교의 의미는 "같은 콘텐츠를 만드는 end-to-end 비용" 으로 한 단계 추상화된다.
//    (이 단서는 reports/pdf-backend-benchmark-*.md 에 명시).

import express from 'express'
import { Font, renderToBuffer } from '@react-pdf/renderer'
import { createRequire } from 'node:module'
import { InvoiceDoc } from './invoice-doc.mjs'

const require = createRequire(import.meta.url)

// ─────────────────────────────────────────────────────────────
// 폰트 등록 (idempotent, 첫 요청 콜드스타트 1회만 비용)
// ─────────────────────────────────────────────────────────────
let fontsRegistered = false
function registerFonts() {
  if (fontsRegistered) return
  fontsRegistered = true

  // @expo-google-fonts/* 패키지는 .ttf 를 패키지 디렉토리에 동봉.
  // require.resolve 로 정확 경로 해결.
  const notoJpRegular = require.resolve('@expo-google-fonts/noto-sans-jp/400Regular/NotoSansJP_400Regular.ttf')
  const notoJpBold    = require.resolve('@expo-google-fonts/noto-sans-jp/700Bold/NotoSansJP_700Bold.ttf')

  // NotoSansJP 는 라틴 글리프도 포함하지만, 한국어 글리프는 별도 패밀리.
  // 단일 fontFamily 'NotoSansJP' 로 본문을 통일하고, 한국어는 같은 family alias 로 fallback 등록.
  Font.register({
    family: 'NotoSansJP',
    fonts: [
      { src: notoJpRegular, fontWeight: 400 },
      { src: notoJpBold,    fontWeight: 700 },
    ],
  })

  // 한국어 — NotoSansKR (별도 family). 단일 family fallback 으론 한글 글리프가 누락되므로
  // 로케일별로 family 자체를 바꿈 (invoice-doc.mjs 참조).
  try {
    const notoKrRegular = require.resolve('@expo-google-fonts/noto-sans-kr/400Regular/NotoSansKR_400Regular.ttf')
    const notoKrBold    = require.resolve('@expo-google-fonts/noto-sans-kr/700Bold/NotoSansKR_700Bold.ttf')
    Font.register({
      family: 'NotoSansKR',
      fonts: [
        { src: notoKrRegular, fontWeight: 400 },
        { src: notoKrBold,    fontWeight: 700 },
      ],
    })
  } catch (e) {
    console.warn('[fonts] NotoSansKR 미등록 — 한국어 PDF 글리프 깨짐 가능:', e.message)
  }

  // 영문 — Inter (라틴 디자인). NotoSansJP 의 영문 글리프도 사용 가능하지만
  // 디자인 측면에서 invoice.html.tmpl 의 'Inter' 선언과 시각 일치 목적.
  try {
    const interRegular = require.resolve('@expo-google-fonts/inter/400Regular/Inter_400Regular.ttf')
    const interBold    = require.resolve('@expo-google-fonts/inter/700Bold/Inter_700Bold.ttf')
    Font.register({
      family: 'Inter',
      fonts: [
        { src: interRegular, fontWeight: 400 },
        { src: interBold,    fontWeight: 700 },
      ],
    })
  } catch (e) {
    console.warn('[fonts] Inter 미등록 — 영문은 NotoSansJP 의 라틴 글리프로 fallback:', e.message)
  }

  // CJK 하이픈네이션 비활성 — 글자 단위 줄바꿈
  Font.registerHyphenationCallback(word => Array.from(word).map(c => c))
}

// ─────────────────────────────────────────────────────────────
// HTTP
// ─────────────────────────────────────────────────────────────
const app = express()
app.use(express.json({ limit: '2mb' }))

app.post('/pdf', async (req, res) => {
  try {
    registerFonts()
    const inv = req.body
    if (!inv || !inv.locale || !Array.isArray(inv.items)) {
      return res.status(400).type('text/plain').send('invalid invoice payload: require {locale, items[], ...}')
    }
    const doc = InvoiceDoc(inv)
    const buf = await renderToBuffer(doc)
    res
      .status(200)
      .type('application/pdf')
      .set('Content-Disposition', `inline; filename="reactpdf.pdf"`)
      .send(buf)
  } catch (err) {
    console.error('[pdf] render failed:', err)
    res.status(500).type('text/plain').send('render failed: ' + (err?.message ?? String(err)))
  }
})

app.get('/health', (_req, res) => {
  res.json({ ok: true, fontsRegistered })
})

const port = Number(process.env.PORT ?? 5002)
app.listen(port, '0.0.0.0', () => {
  console.log(`[react-pdf sidecar] listening on :${port}`)
})
