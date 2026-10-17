export const CATEGORIES = ['еда', 'транспорт', 'жильё', 'развлечения', 'здоровье', 'покупки', 'другое']

const pad = (n) => String(n).padStart(2, '0')
export const iso = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`

export const money = (n) =>
  new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 0, maximumFractionDigits: 2 }).format(n)

export const formatDate = (s) => {
  const [y, m, d] = s.split('-')
  return `${d}.${m}.${y}`
}
