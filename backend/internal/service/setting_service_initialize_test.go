package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingInitializeRepoStub struct {
	values  map[string]string
	updates map[string]string
}

func (s *settingInitializeRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingInitializeRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingInitializeRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *settingInitializeRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingInitializeRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.updates[key] = value
		s.values[key] = value
	}
	return nil
}

func (s *settingInitializeRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *settingInitializeRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingServiceInitializeDefaultSettingsFillsMissingSettings(t *testing.T) {
	repo := &settingInitializeRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.InitializeDefaultSettings(context.Background()))

	require.Equal(t, "true", repo.values[SettingKeyRegistrationEnabled])
	require.Equal(t, "false", repo.values[SettingKeyEmailVerifyEnabled])
	require.Equal(t, "Sub2API", repo.values[SettingKeySiteName])
}

func TestSettingServiceInitializeDefaultSettingsDoesNotOverwriteExistingSettings(t *testing.T) {
	repo := &settingInitializeRepoStub{values: map[string]string{
		SettingKeyRegistrationEnabled: "false",
		SettingKeySiteName:            "Custom Site",
	}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.InitializeDefaultSettings(context.Background()))

	require.Equal(t, "false", repo.values[SettingKeyRegistrationEnabled])
	require.Equal(t, "Custom Site", repo.values[SettingKeySiteName])
	require.NotContains(t, repo.updates, SettingKeyRegistrationEnabled)
	require.NotContains(t, repo.updates, SettingKeySiteName)
	require.Equal(t, "false", repo.updates[SettingKeyEmailVerifyEnabled])
}
