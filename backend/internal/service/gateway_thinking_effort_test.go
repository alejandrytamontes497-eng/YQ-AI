package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSanitizeAnthropicThinkingEffort(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantChanged bool
		wantExists  bool
		wantEffort  string
	}{
		{
			name:        "xhigh maps to Claude max",
			body:        `{"model":"claude-opus-4-7-xhigh","output_config":{"effort":"xhigh"},"messages":[]}`,
			wantChanged: true,
			wantExists:  true,
			wantEffort:  "max",
		},
		{
			name:        "x-high alias maps to Claude max",
			body:        `{"output_config":{"effort":"x-high"},"messages":[]}`,
			wantChanged: true,
			wantExists:  true,
			wantEffort:  "max",
		},
		{
			name:        "uppercase legal effort is normalized",
			body:        `{"output_config":{"effort":"HIGH"},"messages":[]}`,
			wantChanged: true,
			wantExists:  true,
			wantEffort:  "high",
		},
		{
			name:        "unknown effort is removed",
			body:        `{"output_config":{"effort":"ultra"},"messages":[]}`,
			wantChanged: true,
			wantExists:  false,
		},
		{
			name:        "valid max unchanged",
			body:        `{"output_config":{"effort":"max"},"messages":[]}`,
			wantChanged: false,
			wantExists:  true,
			wantEffort:  "max",
		},
		{
			name:        "model suffix xhigh is preserved without output effort",
			body:        `{"model":"claude-opus-4-7-xhigh","messages":[]}`,
			wantChanged: false,
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, changed := sanitizeAnthropicThinkingEffort([]byte(tt.body))
			require.Equal(t, tt.wantChanged, changed)
			effort := gjson.GetBytes(out, "output_config.effort")
			require.Equal(t, tt.wantExists, effort.Exists())
			if tt.wantExists {
				require.Equal(t, tt.wantEffort, effort.String())
			}
			require.Equal(t, gjson.Get(tt.body, "model").String(), gjson.GetBytes(out, "model").String())
		})
	}
}
