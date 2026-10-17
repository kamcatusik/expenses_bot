import { useState } from 'react'
import { CATEGORIES, iso } from '../constants'

export default function ExpenseForm({ initial, users, onSave, onCancel }) {
  const isEdit = Boolean(initial?.id)
  const [form, setForm] = useState({
    user_id: initial?.user_id ?? users[0]?.id ?? '',
    amount: initial?.amount ?? '',
    category: initial?.category ?? 'еда',
    spent_at: initial?.spent_at ?? iso(new Date()),
    comment: initial?.comment ?? '',
  })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const set = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  async function submit(e) {
    e.preventDefault()
    setError('')
    const amount = Number(form.amount)
    if (!(amount > 0)) return setError('Сумма должна быть больше нуля')
    if (!isEdit && !form.user_id) return setError('Выберите члена семьи')

    const payload = { amount, category: form.category, spent_at: form.spent_at, comment: form.comment }
    if (!isEdit) payload.user_id = Number(form.user_id)

    setSaving(true)
    try {
      await onSave(payload)
    } catch (err) {
      setError(err.message)
      setSaving(false)
    }
  }

  return (
    <div className="overlay" onMouseDown={onCancel}>
      <form className="dialog" onSubmit={submit} onMouseDown={(e) => e.stopPropagation()}>
        <h2>{isEdit ? 'Изменить трату' : 'Новая трата'}</h2>

        <label>
          Член семьи
          {isEdit || users.length > 0 ? (
            <select value={form.user_id} onChange={set('user_id')} disabled={isEdit}>
              {users.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name || `ID ${u.id}`}
                </option>
              ))}
            </select>
          ) : (
            <input type="number" placeholder="Telegram ID" value={form.user_id} onChange={set('user_id')} />
          )}
        </label>

        <div className="row">
          <label>
            Сумма
            <input type="number" min="0" step="0.01" autoFocus value={form.amount} onChange={set('amount')} />
          </label>
          <label>
            Дата
            <input type="date" required value={form.spent_at} onChange={set('spent_at')} />
          </label>
        </div>

        <label>
          Категория
          <select value={form.category} onChange={set('category')}>
            {CATEGORIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </label>

        <label>
          Комментарий
          <input type="text" value={form.comment} onChange={set('comment')} placeholder="Необязательно" />
        </label>

        {error && <p className="error">{error}</p>}

        <div className="actions">
          <button type="button" className="ghost" onClick={onCancel}>
            Отмена
          </button>
          <button type="submit" className="primary" disabled={saving}>
            Сохранить
          </button>
        </div>
      </form>
    </div>
  )
}
