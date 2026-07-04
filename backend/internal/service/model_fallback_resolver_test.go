package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMostSimilarAvailableModel(t *testing.T) {
	models := []string{
		"claude-opus-4-6-thinking",
		"claude-sonnet-4-6",
		"gemini-3-pro-preview",
		"gpt-5.4",
	}

	got, ok := mostSimilarAvailableModel("claude-sonet-4-6", models)
	require.True(t, ok)
	require.Equal(t, "claude-sonnet-4-6", got)

	got, ok = mostSimilarAvailableModel("claude-opus-4-6-think", models)
	require.True(t, ok)
	require.Equal(t, "claude-opus-4-6-thinking", got)

	_, ok = mostSimilarAvailableModel("llama-unknown", models)
	require.False(t, ok)
}

func TestModelListSupportsRequestedModel(t *testing.T) {
	require.True(t, modelListSupportsRequestedModel(nil, "anything"), "empty model list means unrestricted")
	require.True(t, modelListSupportsRequestedModel([]string{"claude-*"}, "claude-sonnet-4-6"))
	require.True(t, modelListSupportsRequestedModel([]string{"GPT-5.4"}, "gpt-5.4"))
	require.False(t, modelListSupportsRequestedModel([]string{"claude-sonnet-4-6"}, "claude-opus-4-6"))
}

func TestModelSimilarityScore(t *testing.T) {
	require.Greater(t, modelSimilarityScore("gpt-5.4-xhihg", "gpt-5.4-xhigh"), modelSimilarityThreshold)
	require.Less(t, modelSimilarityScore("claude-sonnet", "gemini-pro"), modelSimilarityThreshold)
}
