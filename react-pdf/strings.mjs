// main.go 의 localeStrings 와 동일 (한/일/영 라벨)
export const localeStrings = {
  ko: {
    title: '청구서', invoiceNo: '청구번호',
    issueDate: '발행일', dueDate: '결제기한',
    from: '공급자', to: '공급받는자', taxId: '사업자등록번호',
    description: '품목', quantity: '수량',
    unitPrice: '단가', amount: '금액',
    subtotal: '공급가액', tax: '부가세', total: '합계',
    page: '페이지', stamp: '인감',
  },
  ja: {
    title: '請求書', invoiceNo: '請求番号',
    issueDate: '発行日', dueDate: '支払期限',
    from: '請求元', to: '請求先', taxId: '登録番号',
    description: '品目', quantity: '数量',
    unitPrice: '単価', amount: '金額',
    subtotal: '小計', tax: '消費税', total: '合計',
    page: 'ページ', stamp: '印',
  },
  en: {
    title: 'INVOICE', invoiceNo: 'Invoice No.',
    issueDate: 'Issue Date', dueDate: 'Due Date',
    from: 'From', to: 'Bill To', taxId: 'Tax ID',
    description: 'Description', quantity: 'Qty',
    unitPrice: 'Unit Price', amount: 'Amount',
    subtotal: 'Subtotal', tax: 'Tax', total: 'Total',
    page: 'Page', stamp: 'Seal',
  },
}
