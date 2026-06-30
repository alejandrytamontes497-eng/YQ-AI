package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const CallAuditExcerptChars = 30

type CallAuditLog struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	APIKeyID        int64     `json:"api_key_id"`
	AccountID       *int64    `json:"account_id,omitempty"`
	GroupID         *int64    `json:"group_id,omitempty"`
	RequestID       string    `json:"request_id"`
	Method          string    `json:"method"`
	Path            string    `json:"path"`
	Model           string    `json:"model"`
	StatusCode      int       `json:"status_code"`
	RequestExcerpt  string    `json:"request_excerpt"`
	ResponseExcerpt string    `json:"response_excerpt"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateCallAuditLogInput struct {
	UserID          int64
	APIKeyID        int64
	AccountID       *int64
	GroupID         *int64
	RequestID       string
	Method          string
	Path            string
	Model           string
	StatusCode      int
	RequestExcerpt  string
	ResponseExcerpt string
	CreatedAt       time.Time
}

type CallAuditRepository interface {
	Create(ctx context.Context, input CreateCallAuditLogInput) error
	ListLatestByUser(ctx context.Context, userID int64, limit int) ([]CallAuditLog, error)
}

type CallAuditService struct {
	repo CallAuditRepository
}

func NewCallAuditService(repo CallAuditRepository) *CallAuditService {
	return &CallAuditService{repo: repo}
}

func (s *CallAuditService) Create(ctx context.Context, input CreateCallAuditLogInput) error {
	if s == nil || s.repo == nil || input.UserID <= 0 || input.APIKeyID <= 0 {
		return nil
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}
	input.Method = truncateRunes(normalizeAuditText(input.Method), 16)
	input.Path = truncateRunes(normalizeAuditText(input.Path), 256)
	input.Model = truncateRunes(normalizeAuditText(input.Model), 128)
	input.RequestID = truncateRunes(normalizeAuditText(input.RequestID), 128)
	input.RequestExcerpt = truncateRunes(normalizeAuditText(input.RequestExcerpt), CallAuditExcerptChars)
	input.ResponseExcerpt = truncateRunes(normalizeAuditText(input.ResponseExcerpt), CallAuditExcerptChars)
	return s.repo.Create(ctx, input)
}

func (s *CallAuditService) ListLatestByUser(ctx context.Context, userID int64, limit int) ([]CallAuditLog, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return []CallAuditLog{}, nil
	}
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	return s.repo.ListLatestByUser(ctx, userID, limit)
}

func CallAuditExcerpt(raw []byte) string {
	return truncateRunes(normalizeAuditText(string(raw)), CallAuditExcerptChars)
}

func normalizeAuditText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
