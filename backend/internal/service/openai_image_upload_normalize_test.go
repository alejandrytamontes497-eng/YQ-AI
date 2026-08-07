package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIImageUploadToDataURLNormalizesLargeReference(t *testing.T) {
	input := image.NewNRGBA(image.Rect(0, 0, 1257, 939))
	state := uint32(1)
	for index := range input.Pix {
		state = state*1664525 + 1013904223
		input.Pix[index] = byte(state >> 24)
	}
	var source bytes.Buffer
	require.NoError(t, png.Encode(&source, input))
	require.Greater(t, source.Len(), openAIInlineImageNormalizeThreshold)

	dataURL, err := openAIImageUploadToDataURL(OpenAIImagesUpload{
		FileName: "reference.png", ContentType: "image/png", Data: source.Bytes(),
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(dataURL, "data:image/jpeg;base64,"))
	encoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, "data:image/jpeg;base64,"))
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded), openAIInlineImageTargetBytes)
	config, format, err := image.DecodeConfig(bytes.NewReader(encoded))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.LessOrEqual(t, max(config.Width, config.Height), openAIInlineImageMaxEdge)
}
