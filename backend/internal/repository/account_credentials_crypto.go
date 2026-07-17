package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const accountCredentialsCiphertextKey = "__sub2api_encrypted_credentials_v1"

func encryptAccountCredentials(encryptor service.SecretEncryptor, credentials map[string]any) (map[string]any, error) {
	if encryptor == nil {
		return copyJSONMap(credentials), nil
	}
	plaintext, err := json.Marshal(normalizeJSONMap(credentials))
	if err != nil {
		return nil, fmt.Errorf("marshal account credentials: %w", err)
	}
	ciphertext, err := encryptor.Encrypt(string(plaintext))
	if err != nil {
		return nil, fmt.Errorf("encrypt account credentials: %w", err)
	}
	return map[string]any{accountCredentialsCiphertextKey: ciphertext}, nil
}

func decryptAccountCredentials(encryptor service.SecretEncryptor, stored map[string]any) (map[string]any, error) {
	ciphertext, encrypted := accountCredentialsCiphertext(stored)
	if !encrypted {
		return copyJSONMap(stored), nil
	}
	if encryptor == nil {
		return nil, fmt.Errorf("account credentials are encrypted but no encryptor is configured")
	}
	plaintext, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt account credentials: %w", err)
	}
	var credentials map[string]any
	if err := json.Unmarshal([]byte(plaintext), &credentials); err != nil {
		return nil, fmt.Errorf("decode account credentials: %w", err)
	}
	return normalizeJSONMap(credentials), nil
}

func accountCredentialsCiphertext(stored map[string]any) (string, bool) {
	if len(stored) != 1 {
		return "", false
	}
	value, ok := stored[accountCredentialsCiphertextKey].(string)
	return value, ok && value != ""
}

func migrateAccountCredentialsAtRest(ctx context.Context, db *sql.DB, encryptor service.SecretEncryptor) error {
	if db == nil || encryptor == nil {
		return fmt.Errorf("account credential encryption dependencies are not configured")
	}

	rows, err := db.QueryContext(ctx, `SELECT id, credentials FROM accounts ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list account credentials for encryption migration: %w", err)
	}
	type legacyAccountCredentials struct {
		id          int64
		credentials map[string]any
	}
	legacy := make([]legacyAccountCredentials, 0)
	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan account credentials for encryption migration: %w", err)
		}
		var credentials map[string]any
		if err := json.Unmarshal(raw, &credentials); err != nil {
			_ = rows.Close()
			return fmt.Errorf("decode account %d credentials for encryption migration: %w", id, err)
		}
		if _, encrypted := accountCredentialsCiphertext(credentials); encrypted {
			continue
		}
		legacy = append(legacy, legacyAccountCredentials{id: id, credentials: credentials})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate account credentials for encryption migration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close account credentials migration rows: %w", err)
	}
	if len(legacy) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account credentials encryption migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range legacy {
		encrypted, err := encryptAccountCredentials(encryptor, item.credentials)
		if err != nil {
			return fmt.Errorf("encrypt account %d credentials: %w", item.id, err)
		}
		raw, err := json.Marshal(encrypted)
		if err != nil {
			return fmt.Errorf("marshal encrypted account %d credentials: %w", item.id, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET credentials = $1::jsonb WHERE id = $2`, raw, item.id); err != nil {
			return fmt.Errorf("persist encrypted account %d credentials: %w", item.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account credentials encryption migration: %w", err)
	}
	return nil
}
