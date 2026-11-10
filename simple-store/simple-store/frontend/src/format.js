const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })

export const money = (n) => usd.format(Number(n) || 0)

export const dateTime = (iso) => {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '-' : d.toLocaleString('th-TH', { dateStyle: 'medium', timeStyle: 'short' })
}

export const STATUS_LABEL = {
  pending: 'รอชำระเงิน',
  paid: 'ชำระแล้ว',
  shipped: 'จัดส่งแล้ว',
  cancelled: 'ยกเลิก',
}
