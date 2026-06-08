// invoice.html.tmpl 의 시각 등가물을 @react-pdf/renderer 컴포넌트로 표현.
// JSX 미사용 — React.createElement 를 h 로 별칭하여 transpile toolchain 회피.
import React from 'react'
import { Document, Page, Text, View, StyleSheet } from '@react-pdf/renderer'
import { formatMoney } from './money.mjs'
import { localeStrings } from './strings.mjs'

const h = React.createElement

const COLORS = {
  ink: '#1a1a1a',
  sub: '#444',
  mute: '#666',
  faint: '#888',
  line: '#e5e5e5',
  panel: '#f6f6f8',
  noteBg: '#fffbe6',
  noteBar: '#f0c000',
  stampBorder: '#bbb',
  stampText: '#aaa',
}

// 로케일별 fontFamily — 단일 family fallback 으론 CJK 글리프 매칭이 깨진다.
// (server.mjs 의 Font.register 와 1:1 대응. 미등록된 family 는 fallback 으로 첫 등록 family 사용.)
const FONT_BY_LOCALE = {
  ko: 'NotoSansKR',
  ja: 'NotoSansJP',
  en: 'Inter',
}
const DEFAULT_FONT = 'NotoSansJP'

const styles = StyleSheet.create({
  page: {
    paddingTop: 20 * 2.834,        // 20mm → pt (1mm ≈ 2.834pt)
    paddingBottom: 25 * 2.834,
    paddingHorizontal: 15 * 2.834,
    // fontFamily 는 InvoiceDoc 에서 로케일별로 인라인 주입 — styles 에 고정값 두지 않음
    fontSize: 10,
    color: COLORS.ink,
    lineHeight: 1.55,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    borderBottomWidth: 2,
    borderBottomColor: COLORS.ink,
    paddingBottom: 10,
    marginBottom: 18,
  },
  h1: {
    fontSize: 26,
    fontWeight: 700,
    letterSpacing: 1.04,  // 26pt × 0.04em
    marginBottom: 4,
    // h1 자체 행간을 줄여서 invoiceNo 와의 baseline 충돌 회피
    lineHeight: 1.15,
  },
  invoiceNo: {
    color: COLORS.mute,
    fontSize: 9,
    marginTop: 4,  // h1 baseline 과의 여백 확보
  },
  meta: { textAlign: 'right', fontSize: 9 },
  metaRow: { flexDirection: 'row', justifyContent: 'flex-end', marginBottom: 2 },
  metaLabel: { color: COLORS.faint, marginRight: 6 },
  metaValue: { fontWeight: 500 },

  parties: { flexDirection: 'row', gap: 16, marginBottom: 18 },
  party: {
    flex: 1,
    padding: 12,
    backgroundColor: COLORS.panel,
    borderRadius: 4,
  },
  partyH3: {
    fontSize: 9,
    color: COLORS.mute,
    textTransform: 'uppercase',
    letterSpacing: 1.08,  // 9pt × 0.12em
    fontWeight: 500,
    marginBottom: 8,
  },
  partyName: { fontSize: 12, fontWeight: 700, marginBottom: 2 },
  partyLine: { fontSize: 9.5, color: COLORS.sub },

  tableHead: {
    flexDirection: 'row',
    backgroundColor: COLORS.ink,
    color: '#fff',
    paddingVertical: 7,
    paddingHorizontal: 8,
    fontSize: 9,
    fontWeight: 500,
    letterSpacing: 0.36,  // 9pt × 0.04em (이전 1.04 는 'D e s c r i pt i o n' 식 과잉 자간)
  },
  tableHeadCell: { color: '#fff' },
  tableRow: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: COLORS.line,
    paddingVertical: 7,
    paddingHorizontal: 8,
    fontSize: 10,
    // 페이지 경계에서 행이 잘리지 않게 — @react-pdf 의 wrap=false
  },
  cellIdx: { width: '5%', textAlign: 'center' },
  cellDesc: { width: '50%' },
  cellQty: { width: '10%', textAlign: 'right' },
  cellUnit: { width: '17%', textAlign: 'right' },
  cellAmt: { width: '18%', textAlign: 'right' },
  sku: { color: COLORS.faint, fontSize: 8.5, marginTop: 2 },

  summaryWrap: { marginTop: 14, flexDirection: 'row', justifyContent: 'flex-end' },
  summary: { width: '45%' },
  summaryRow: { flexDirection: 'row', paddingVertical: 5, paddingHorizontal: 8 },
  summaryLabel: { width: '50%', color: COLORS.mute },
  summaryNum: { width: '50%', textAlign: 'right' },
  summaryTotalRow: {
    flexDirection: 'row',
    paddingTop: 9,
    paddingHorizontal: 8,
    paddingBottom: 5,
    borderTopWidth: 2,
    borderTopColor: COLORS.ink,
  },
  summaryTotalLabel: { width: '50%', fontSize: 13, fontWeight: 700, color: COLORS.ink },
  summaryTotalNum: { width: '50%', fontSize: 13, fontWeight: 700, textAlign: 'right' },

  notes: {
    marginTop: 26,
    paddingVertical: 12,
    paddingHorizontal: 14,
    backgroundColor: COLORS.noteBg,
    borderLeftWidth: 4,
    borderLeftColor: COLORS.noteBar,
    fontSize: 9,
    lineHeight: 1.6,
  },

  footerSign: { marginTop: 30, flexDirection: 'row', justifyContent: 'flex-end' },
  stampBox: {
    width: 70, height: 70,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: COLORS.stampBorder,
    borderRadius: 4,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stampText: { color: COLORS.stampText, fontSize: 9 },

  pageNumber: {
    position: 'absolute',
    bottom: 12 * 2.834,
    left: 0, right: 0,
    textAlign: 'center',
    fontSize: 9,
    color: COLORS.faint,
  },
})

function Item({ idx, item, currency }) {
  return h(View, { style: styles.tableRow, wrap: false },
    h(Text, { style: styles.cellIdx }, String(idx + 1)),
    h(View, { style: styles.cellDesc },
      h(Text, null, item.description),
      item.sku ? h(Text, { style: styles.sku }, 'SKU: ' + item.sku) : null,
    ),
    h(Text, { style: styles.cellQty }, String(item.quantity)),
    h(Text, { style: styles.cellUnit }, formatMoney(item.unit_price, currency)),
    h(Text, { style: styles.cellAmt }, formatMoney(item.amount, currency)),
  )
}

function TableHead(strings) {
  return h(View, { style: styles.tableHead, fixed: true },
    h(Text, { style: [styles.tableHeadCell, styles.cellIdx] }, '#'),
    h(Text, { style: [styles.tableHeadCell, styles.cellDesc] }, strings.description),
    h(Text, { style: [styles.tableHeadCell, styles.cellQty] }, strings.quantity),
    h(Text, { style: [styles.tableHeadCell, styles.cellUnit] }, strings.unitPrice),
    h(Text, { style: [styles.tableHeadCell, styles.cellAmt] }, strings.amount),
  )
}

export function InvoiceDoc(inv) {
  const strings = localeStrings[inv.locale] || localeStrings.en
  const currency = inv.currency
  const subtotal = inv.items.reduce((s, it) => s + it.amount, 0)
  const tax = Math.round(subtotal * inv.tax_rate) / 100
  const total = subtotal + tax
  const fontFamily = FONT_BY_LOCALE[inv.locale] || DEFAULT_FONT

  return h(Document, {
    title: `${strings.title} ${inv.invoice_number}`,
    language: inv.locale,
  },
    h(Page, { size: 'A4', style: [styles.page, { fontFamily }] },
      // Header
      h(View, { style: styles.header },
        h(View, null,
          h(Text, { style: styles.h1 }, strings.title),
          // nested Text 인라인 처리 불안정 → 단일 Text 로 단순화 (bold 강조는 포기)
          h(Text, { style: styles.invoiceNo }, `${strings.invoiceNo}: ${inv.invoice_number}`),
        ),
        h(View, { style: styles.meta },
          h(View, { style: styles.metaRow },
            h(Text, { style: styles.metaLabel }, strings.issueDate),
            h(Text, { style: styles.metaValue }, inv.issue_date),
          ),
          h(View, { style: styles.metaRow },
            h(Text, { style: styles.metaLabel }, strings.dueDate),
            h(Text, { style: styles.metaValue }, inv.due_date),
          ),
        ),
      ),

      // Parties
      h(View, { style: styles.parties },
        h(View, { style: styles.party },
          h(Text, { style: styles.partyH3 }, strings.from),
          h(Text, { style: styles.partyName }, inv.from.name),
          h(Text, { style: styles.partyLine }, inv.from.address),
          inv.from.tax_id ? h(Text, { style: styles.partyLine }, `${strings.taxId}: ${inv.from.tax_id}`) : null,
          inv.from.contact ? h(Text, { style: styles.partyLine }, inv.from.contact) : null,
        ),
        h(View, { style: styles.party },
          h(Text, { style: styles.partyH3 }, strings.to),
          h(Text, { style: styles.partyName }, inv.to.name),
          h(Text, { style: styles.partyLine }, inv.to.address),
          inv.to.tax_id ? h(Text, { style: styles.partyLine }, `${strings.taxId}: ${inv.to.tax_id}`) : null,
          inv.to.contact ? h(Text, { style: styles.partyLine }, inv.to.contact) : null,
        ),
      ),

      // Items table
      TableHead(strings),
      ...inv.items.map((it, idx) => h(Item, { key: idx, idx, item: it, currency })),

      // Summary
      h(View, { style: styles.summaryWrap, wrap: false },
        h(View, { style: styles.summary },
          h(View, { style: styles.summaryRow },
            h(Text, { style: styles.summaryLabel }, strings.subtotal),
            h(Text, { style: styles.summaryNum }, formatMoney(subtotal, currency)),
          ),
          h(View, { style: styles.summaryRow },
            h(Text, { style: styles.summaryLabel }, `${strings.tax} (${inv.tax_rate}%)`),
            h(Text, { style: styles.summaryNum }, formatMoney(tax, currency)),
          ),
          h(View, { style: styles.summaryTotalRow },
            h(Text, { style: styles.summaryTotalLabel }, strings.total),
            h(Text, { style: styles.summaryTotalNum }, formatMoney(total, currency)),
          ),
        ),
      ),

      // Notes
      inv.notes ? h(View, { style: styles.notes, wrap: false }, h(Text, null, inv.notes)) : null,

      // Stamp
      h(View, { style: styles.footerSign, wrap: false },
        h(View, { style: styles.stampBox },
          h(Text, { style: styles.stampText }, strings.stamp),
        ),
      ),

      // Page number (fixed)
      h(Text, {
        style: styles.pageNumber,
        fixed: true,
        render: ({ pageNumber, totalPages }) => `${strings.page} ${pageNumber} / ${totalPages}`,
      }),
    ),
  )
}
