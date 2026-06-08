// main.go 의 formatMoney 와 1:1 동작 일치 (KRW/JPY 0 decimals, USD 2 decimals, 천단위 콤마)
export function formatMoney(amount, currency) {
  let sym, decimals
  switch (currency) {
    case 'KRW': sym = '₩'; decimals = 0; break
    case 'JPY': sym = '¥'; decimals = 0; break
    case 'USD': sym = '$'; decimals = 2; break
    default:    sym = currency + ' '; decimals = 2; break
  }
  const neg = amount < 0
  if (neg) amount = -amount
  const fixed = amount.toFixed(decimals)
  const [intPart, fracPart] = fixed.split('.')
  const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  let out = sym + grouped + (fracPart ? '.' + fracPart : '')
  if (neg) out = '-' + out
  return out
}
