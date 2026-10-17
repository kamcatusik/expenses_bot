package repository

import (
	"expense-tracker/backend/internal/domain"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse(domain.DateLayout, s) // формат уже проверен в domain
	return t
}

func datePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := date(*s)
	return &t
}
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dateOrNil(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := date(s)
	return &t
}
