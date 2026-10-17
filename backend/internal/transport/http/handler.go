package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"expense-tracker/backend/internal/domain"
	"expense-tracker/backend/internal/repository"
)

type Handler struct {
	expenses repository.Expenses
	users    repository.Users
	loc      *time.Location
}

func NewRouter(expenses repository.Expenses, users repository.Users, loc *time.Location) http.Handler {
	h := &Handler{expenses: expenses, users: users, loc: loc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, domain.Categories) })
	mux.HandleFunc("GET /api/users", h.listUsers)
	mux.HandleFunc("POST /api/users", h.upsertUser)
	mux.HandleFunc("GET /api/expenses", h.list)
	mux.HandleFunc("POST /api/expenses", h.create)
	mux.HandleFunc("GET /api/expenses/{id}", h.get)
	mux.HandleFunc("PATCH /api/expenses/{id}", h.update)
	mux.HandleFunc("DELETE /api/expenses/{id}", h.delete)
	return logging(mux)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, err := optUserID(r)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := domain.Filter{
		UserID:         uid,
		Category:       q.Get("category"),
		From:           q.Get("from"),
		To:             q.Get("to"),
		Query:          q.Get("q"),
		OrderByCreated: q.Get("order") == "created",
		Limit:          limit,
	}
	if err := f.Normalize(); err != nil {
		writeErr(w, err)
		return
	}
	items, err := h.expenses.List(r.Context(), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, items)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in createReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	e := domain.Expense{UserID: in.UserID, Amount: in.Amount, Category: in.Category, SpentAt: in.SpentAt, Comment: in.Comment}
	if err := e.Normalize(h.today()); err != nil {
		writeErr(w, err)
		return
	}
	out, err := h.expenses.Create(r.Context(), e)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, owner, err := idAndOwner(r)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	e, err := h.expenses.Get(r.Context(), id, owner)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, e)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, owner, err := idAndOwner(r)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	var in patchReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		badRequest(w, "invalid JSON")
		return
	}
	p := domain.Patch{Amount: in.Amount, Category: in.Category, SpentAt: in.SpentAt, Comment: in.Comment}
	if err := p.Normalize(); err != nil {
		writeErr(w, err)
		return
	}
	e, err := h.expenses.Update(r.Context(), id, owner, p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, e)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, owner, err := idAndOwner(r)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := h.expenses.Delete(r.Context(), id, owner); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, users)
}

func (h *Handler) upsertUser(w http.ResponseWriter, r *http.Request) {
	var u domain.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil || u.ID == 0 {
		badRequest(w, "id is required")
		return
	}
	if err := h.users.Upsert(r.Context(), u); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, u)
}
