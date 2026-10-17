import { CATEGORIES, iso } from '../constants'

export default function Filters({ filters, setFilters, users }) {
  const set = (k) => (e) => setFilters({ ...filters, [k]: e.target.value })

  const preset = (days) => {
    const to = new Date()
    const from = new Date()
    from.setDate(to.getDate() - days + 1)
    setFilters({ ...filters, from: iso(from), to: iso(to) })
  }
  const thisMonth = () => {
    const now = new Date()
    setFilters({ ...filters, from: iso(new Date(now.getFullYear(), now.getMonth(), 1)), to: iso(now) })
  }

  return (
    <section className="filters">
      <label>
        Член семьи
        <select value={filters.user_id} onChange={set('user_id')}>
          <option value="">Все</option>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.name || `ID ${u.id}`}
            </option>
          ))}
        </select>
      </label>
      <label>
        Категория
        <select value={filters.category} onChange={set('category')}>
          <option value="">Все</option>
          {CATEGORIES.map((c) => (
            <option key={c}>{c}</option>
          ))}
        </select>
      </label>
      <label>
        С
        <input type="date" value={filters.from} onChange={set('from')} />
      </label>
      <label>
        По
        <input type="date" value={filters.to} onChange={set('to')} />
      </label>
      <div className="presets">
        <button type="button" className="ghost" onClick={() => preset(7)}>Неделя</button>
        <button type="button" className="ghost" onClick={thisMonth}>Этот месяц</button>
        <button type="button" className="ghost" onClick={() => setFilters({ user_id: '', category: '', from: '', to: '' })}>
          Сбросить
        </button>
      </div>
    </section>
  )
}
