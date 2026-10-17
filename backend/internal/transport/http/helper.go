package http

import (
	"encoding/json"
	"errors"
	"expense-tracker/backend/internal/domain"
	"log"
	"net/http"
	"strconv"
	"time"
)

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.RequestURI(), time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ve.Msg})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "expense not found"})
	default:
		log.Printf("internal error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func (h *Handler) today() string { return time.Now().In(h.loc).Format(domain.DateLayout) }

// optUserID читает необязательный ?user_id=. Если он задан, доступ ограничен тратами этого пользователя.
func optUserID(r *http.Request) (*int64, error) {
	s := r.URL.Query().Get("user_id")
	if s == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, errors.New("user_id must be an integer")
	}
	return &id, nil
}

// idAndOwner разбирает {id} из пути и ?user_id=.
func idAndOwner(r *http.Request) (int64, *int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, nil, errors.New("invalid id")
	}
	owner, err := optUserID(r)
	return id, owner, err
}
