async function req(path, opts = {}) {
  const res = await fetch('/api' + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      msg = (await res.json()).error || msg
    } catch {
      /* тело не JSON */
    }
    throw new Error(msg)
  }
  return res.status === 204 ? null : res.json()
}

export const api = {
  users: () => req('/users'),
  list: (filters) => {
    const q = new URLSearchParams()
    Object.entries(filters).forEach(([k, v]) => v && q.set(k, v))
    return req('/expenses?' + q.toString())
  },
  create: (body) => req('/expenses', { method: 'POST', body: JSON.stringify(body) }),
  update: (id, body) => req(`/expenses/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  remove: (id) => req(`/expenses/${id}`, { method: 'DELETE' }),
}
