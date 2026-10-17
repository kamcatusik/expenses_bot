import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from './api'
import { formatDate, money } from './constants'
import ExpenseForm from './components/ExpenseForm'
import Filters from './components/Filters'

const EMPTY = { user_id: '', category: '', from: '', to: '' }

export default function App() {
  const [filters, setFilters] = useState(EMPTY)
  const [items, setItems] = useState([])
  const [users, setUsers] = useState([])
  const [editing, setEditing] = useState(null) // null | {} (новая) | трата
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const userName = useMemo(
    () => Object.fromEntries(users.map((u) => [u.id, u.name || `ID ${u.id}`])),
    [users],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setItems(await api.list(filters))
      setError('')
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [filters])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    api.users().then(setUsers).catch((e) => setError(e.message))
  }, [])

  const total = items.reduce((s, e) => s + e.amount, 0)

  async function save(payload) {
    if (editing.id) await api.update(editing.id, payload)
    else await api.create(payload)
    setEditing(null)
    await load()
    api.users().then(setUsers).catch(() => {})
  }

  async function remove(e) {
    const what = e.comment ? `«${e.comment}», ${money(e.amount)}` : money(e.amount)
    if (!window.confirm(`Удалить трату ${what}?`)) return
    try {
      await api.remove(e.id)
      await load()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <main>
      <header>
        <h1>Расходы семьи</h1>
        <button className="primary" onClick={() => setEditing({})}>
          Добавить трату
        </button>
      </header>

      <Filters filters={filters} setFilters={setFilters} users={users} />

      {error && <p className="error banner">{error}</p>}

      <div className="summary">
        <span>
          Записей: <b>{items.length}</b>
        </span>
        <span>
          Итого: <b className="num">{money(total)}</b>
        </span>
      </div>

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Дата</th>
              <th>Кто</th>
              <th>Категория</th>
              <th>Комментарий</th>
              <th className="num">Сумма</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {items.map((e) => (
              <tr key={e.id}>
                <td>{formatDate(e.spent_at)}</td>
                <td>{userName[e.user_id] || `ID ${e.user_id}`}</td>
                <td>
                  <span className={`tag c-${e.category}`}>{e.category}</span>
                </td>
                <td className="muted">{e.comment}</td>
                <td className="num">{money(e.amount)}</td>
                <td className="row-actions">
                  <button className="ghost" onClick={() => setEditing(e)}>Изменить</button>
                  <button className="ghost danger" onClick={() => remove(e)}>Удалить</button>
                </td>
              </tr>
            ))}
            {!loading && items.length === 0 && (
              <tr>
                <td colSpan="6" className="empty">
                  Трат по этим условиям нет. Измените фильтры или напишите боту «кофе 350».
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editing && (
        <ExpenseForm initial={editing} users={users} onSave={save} onCancel={() => setEditing(null)} />
      )}
    </main>
  )
}
