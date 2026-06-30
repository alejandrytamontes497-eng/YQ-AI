package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type callAuditRepository struct {
	db *sql.DB
}

func NewCallAuditRepository(db *sql.DB) service.CallAuditRepository {
	return &callAuditRepository{db: db}
}

func (r *callAuditRepository) Create(ctx context.Context, input service.CreateCallAuditLogInput) error {
	if r == nil || r.db == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO call_audit_logs (
			user_id, api_key_id, account_id, group_id, request_id, method, path, model,
			status_code, request_excerpt, response_excerpt, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, input.UserID, input.APIKeyID, nullableInt64(input.AccountID), nullableInt64(input.GroupID),
		input.RequestID, input.Method, input.Path, input.Model, input.StatusCode,
		input.RequestExcerpt, input.ResponseExcerpt, input.CreatedAt)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		DELETE FROM call_audit_logs
		 WHERE id IN (
			SELECT id
			  FROM call_audit_logs
			 WHERE user_id = $1
			 ORDER BY created_at DESC, id DESC
			 OFFSET 10
		 )
	`, input.UserID)
	return err
}

func (r *callAuditRepository) ListLatestByUser(ctx context.Context, userID int64, limit int) ([]service.CallAuditLog, error) {
	if r == nil || r.db == nil {
		return []service.CallAuditLog{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, api_key_id, account_id, group_id, request_id, method, path, model,
		       status_code, request_excerpt, response_excerpt, created_at
		  FROM call_audit_logs
		 WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.CallAuditLog, 0, limit)
	for rows.Next() {
		var item service.CallAuditLog
		var accountID, groupID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.UserID, &item.APIKeyID, &accountID, &groupID,
			&item.RequestID, &item.Method, &item.Path, &item.Model, &item.StatusCode,
			&item.RequestExcerpt, &item.ResponseExcerpt, &item.CreatedAt); err != nil {
			return nil, err
		}
		if accountID.Valid {
			v := accountID.Int64
			item.AccountID = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			item.GroupID = &v
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
