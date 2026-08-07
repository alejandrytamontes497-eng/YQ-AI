package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	openAIInlineImageNormalizeThreshold = 256 << 10
	openAIInlineImageTargetBytes        = 192 << 10
	openAIInlineImageMaxEdge            = 1024
	openAIInlineImageMaxPixels          = 40_000_000
)

func normalizeOpenAIInlineImage(data []byte) ([]byte, string, error) {
	config, _, configErr := image.DecodeConfig(bytes.NewReader(data))
	needsNormalization := len(data) > openAIInlineImageNormalizeThreshold
	if configErr == nil && max(config.Width, config.Height) > openAIInlineImageMaxEdge {
		needsNormalization = true
	}
	if !needsNormalization {
		return data, "", nil
	}
	if configErr != nil {
		return nil, "", fmt.Errorf("decode image metadata: %w", configErr)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > openAIInlineImageMaxPixels {
		return nil, "", fmt.Errorf("image dimensions are unsupported")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	encoded := encodeOpenAIInlineJPEG(source, openAIInlineImageMaxEdge, []int{82, 72, 62, 52})
	if len(encoded) > openAIInlineImageTargetBytes {
		encoded = encodeOpenAIInlineJPEG(source, 768, []int{62, 52, 42})
	}
	if len(encoded) == 0 {
		return nil, "", fmt.Errorf("encode normalized image")
	}
	return encoded, "image/jpeg", nil
}

func encodeOpenAIInlineJPEG(source image.Image, maxEdge int, qualities []int) []byte {
	normalized := resizeOpenAIInlineImage(source, maxEdge)
	var smallest []byte
	for _, quality := range qualities {
		var buffer bytes.Buffer
		if err := jpeg.Encode(&buffer, normalized, &jpeg.Options{Quality: quality}); err != nil {
			return nil
		}
		encoded := append([]byte(nil), buffer.Bytes()...)
		if len(smallest) == 0 || len(encoded) < len(smallest) {
			smallest = encoded
		}
		if len(encoded) <= openAIInlineImageTargetBytes {
			return encoded
		}
	}
	return smallest
}

func resizeOpenAIInlineImage(source image.Image, maxEdge int) *image.RGBA {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > maxEdge || height > maxEdge {
		scale := float64(maxEdge) / float64(max(width, height))
		width = max(1, int(float64(width)*scale+0.5))
		height = max(1, int(float64(height)*scale+0.5))
	}
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(destination, destination.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(destination, destination.Bounds(), source, bounds, draw.Over, nil)
	return destination
}
