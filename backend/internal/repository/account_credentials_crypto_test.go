package repository

import (
	"context"
	"encoding/base64"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type accountCredentialTestEncryptor struct{}

func (accountCredentialTestEncryptor) Encrypt(plaintext string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (accountCredentialTestEncryptor) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	return string(raw), err
}

func TestAccountCredentialsEncryptRoundTrip(t *testing.T) {
	credentials := map[string]any{
		"api_key":       "sk-upstream-secret",
		"refresh_token": "refresh-secret",
		"base_url":      "https://api.example.com",
	}

	stored, err := encryptAccountCredentials(accountCredentialTestEncryptor{}, credentials)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.NotContains(t, stored, "api_key")

	decoded, err := decryptAccountCredentials(accountCredentialTestEncryptor{}, stored)
	require.NoError(t, err)
	require.Equal(t, credentials, decoded)
}

func TestMigrateAccountCredentialsAtRestEncryptsLegacyRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, credentials FROM accounts ORDER BY id")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "credentials"}).
			AddRow(int64(7), []byte(`{"api_key":"sk-upstream-secret"}`)).
			AddRow(int64(8), []byte(`{"__sub2api_encrypted_credentials_v1":"YWxyZWFkeQ=="}`)))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE accounts SET credentials = $1::jsonb WHERE id = $2")).
		WithArgs(sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = migrateAccountCredentialsAtRest(context.Background(), db, accountCredentialTestEncryptor{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
