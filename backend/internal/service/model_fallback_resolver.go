package service

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strings"
)

const modelSimilarityThreshold = 0.62

var modelFallbackTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

type ModelFallbackResolution struct {
	Model   string
	Changed bool
	Reason  string
}

func (s *GatewayService) ResolveSimilarOrFallbackModel(ctx context.Context, groupID *int64, platform string, requestedModel string) ModelFallbackResolution {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" || s == nil || s.settingService == nil || !s.settingService.IsModelFallbackEnabled(ctx) {
		return ModelFallbackResolution{Model: requestedModel}
	}

	availableModels := s.GetAvailableModels(ctx, groupID, platform)
	if modelListSupportsRequestedModel(availableModels, requestedModel) {
		return ModelFallbackResolution{Model: requestedModel}
	}

	if similarModel, ok := mostSimilarAvailableModel(requestedModel, availableModels); ok {
		return ModelFallbackResolution{Model: similarModel, Changed: true, Reason: "similar"}
	}

	fallbackModel := strings.TrimSpace(s.settingService.GetFallbackModel(ctx, platform))
	if fallbackModel == "" || strings.EqualFold(fallbackModel, requestedModel) {
		return ModelFallbackResolution{Model: requestedModel}
	}
	return ModelFallbackResolution{Model: fallbackModel, Changed: true, Reason: "fallback"}
}

func modelListSupportsRequestedModel(models []string, requestedModel string) bool {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" || len(models) == 0 {
		return true
	}
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if strings.EqualFold(model, requestedModel) || matchWildcard(model, requestedModel) {
			return true
		}
	}
	return false
}

func mostSimilarAvailableModel(requestedModel string, availableModels []string) (string, bool) {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" || len(availableModels) == 0 {
		return "", false
	}

	requestedTokens := significantModelTokens(requestedModel)
	bestModel := ""
	bestScore := 0.0
	for _, candidate := range availableModels {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.ContainsAny(candidate, "*?") {
			continue
		}
		if !hasSharedModelToken(requestedTokens, significantModelTokens(candidate)) {
			continue
		}
		score := modelSimilarityScore(requestedModel, candidate)
		if score > bestScore || (score == bestScore && candidate < bestModel) {
			bestScore = score
			bestModel = candidate
		}
	}
	if bestModel == "" || bestScore < modelSimilarityThreshold {
		return "", false
	}
	return bestModel, true
}

func modelSimilarityScore(a, b string) float64 {
	a = normalizeModelForSimilarity(a)
	b = normalizeModelForSimilarity(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	distance := levenshteinDistance(a, b)
	base := 1 - float64(distance)/float64(maxModelFallbackInt(len(a), len(b)))
	return math.Max(0, base)
}

func normalizeModelForSimilarity(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	parts := modelFallbackTokenPattern.FindAllString(model, -1)
	return strings.Join(parts, "-")
}

func significantModelTokens(model string) map[string]struct{} {
	parts := modelFallbackTokenPattern.FindAllString(strings.ToLower(strings.TrimSpace(model)), -1)
	out := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if len(part) < 3 || isAllDigits(part) {
			continue
		}
		out[part] = struct{}{}
	}
	return out
}

func hasSharedModelToken(a, b map[string]struct{}) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for token := range a {
		if _, ok := b[token]; ok {
			return true
		}
	}
	return false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minModelFallbackInt(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minModelFallbackInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxModelFallbackInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *OpenAIGatewayService) ResolveSimilarOrFallbackModel(ctx context.Context, groupID *int64, platform string, requestedModel string) ModelFallbackResolution {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" || s == nil || s.settingService == nil || !s.settingService.IsModelFallbackEnabled(ctx) {
		return ModelFallbackResolution{Model: requestedModel}
	}

	availableModels := s.availableModelsForFallback(ctx, groupID, platform)
	if modelListSupportsRequestedModel(availableModels, requestedModel) {
		return ModelFallbackResolution{Model: requestedModel}
	}
	if similarModel, ok := mostSimilarAvailableModel(requestedModel, availableModels); ok {
		return ModelFallbackResolution{Model: similarModel, Changed: true, Reason: "similar"}
	}
	fallbackModel := strings.TrimSpace(s.settingService.GetFallbackModel(ctx, platform))
	if fallbackModel == "" || strings.EqualFold(fallbackModel, requestedModel) {
		return ModelFallbackResolution{Model: requestedModel}
	}
	return ModelFallbackResolution{Model: fallbackModel, Changed: true, Reason: "fallback"}
}

func (s *OpenAIGatewayService) availableModelsForFallback(ctx context.Context, groupID *int64, platform string) []string {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	var accounts []Account
	var err error
	if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupID(ctx, *groupID)
	} else {
		accounts, err = s.accountRepo.ListSchedulable(ctx)
	}
	if err != nil || len(accounts) == 0 {
		return nil
	}
	return collectMappedModelCandidates(accounts, platform)
}

func collectMappedModelCandidates(accounts []Account, platform string) []string {
	modelSet := make(map[string]struct{})
	hasAnyMapping := false
	for _, acc := range accounts {
		if platform != "" && acc.Platform != platform {
			continue
		}
		mapping := acc.GetModelMapping()
		if len(mapping) == 0 {
			continue
		}
		hasAnyMapping = true
		for model := range mapping {
			model = strings.TrimSpace(model)
			if model != "" {
				modelSet[model] = struct{}{}
			}
		}
	}
	if !hasAnyMapping {
		return nil
	}
	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}
