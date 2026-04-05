package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

const defaultBaseURL = "https://api.anthropic.com"

type ClaudeScorer struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
	logger  *zerolog.Logger
}

func NewClaudeScorer(apiKey, model string, logger *zerolog.Logger) *ClaudeScorer {
	return &ClaudeScorer{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		logger:  logger,
	}
}

func (s *ClaudeScorer) ComputeScore(ctx context.Context, input *entity.ScoringInput) (*entity.CreditScore, error) {
	if s.apiKey == "" {
		s.logger.Warn().Msg("no Anthropic API key, using fallback scorer")
		return s.fallbackScore(input), nil
	}

	score, err := s.callAPI(ctx, input)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Claude API failed, using fallback scorer")
		return s.fallbackScore(input), nil
	}
	return score, nil
}

func (s *ClaudeScorer) callAPI(ctx context.Context, input *entity.ScoringInput) (*entity.CreditScore, error) {
	prompt := s.buildPrompt(input)

	reqBody, err := json.Marshal(map[string]any{
		"model":      s.model,
		"max_tokens": 1024,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	if err != nil {
		return nil, fmt.Errorf("claude scorer marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("claude scorer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude scorer call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude scorer status %d: %s", resp.StatusCode, body)
	}

	return s.parseResponse(resp.Body, input)
}

func (s *ClaudeScorer) buildPrompt(input *entity.ScoringInput) string {
	ndviScores := make([]float64, len(input.NDVIHistory))
	for i, c := range input.NDVIHistory {
		ndviScores[i] = c.NDVIScore
	}

	data, _ := json.Marshal(map[string]any{
		"cadastral":       input.CadastralNumber,
		"area_ha":         input.AreaHa,
		"land_class":      input.LandClass,
		"oblast":          input.Oblast,
		"ndvi_scores":     ndviScores,
		"active_liens":    input.ActiveLiens,
		"total_liens":     input.TotalLiens,
		"disputes":        input.Disputes,
		"dormant_seasons": input.DormantSeasons,
	})

	return fmt.Sprintf(`Analyze this agricultural parcel and return a JSON credit score.
Data: %s

Return ONLY valid JSON with these fields:
{"ai_score": 0-100, "recommended_ltv": 0.0-1.0, "collateral_grade": "A"|"B"|"C"|"D",
 "estimated_value_tenge": integer, "explanation": "string", "risk_factors": ["string"]}`, data)
}

func (s *ClaudeScorer) parseResponse(body io.Reader, input *entity.ScoringInput) (*entity.CreditScore, error) {
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("claude scorer decode response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("claude scorer: empty response")
	}

	var parsed struct {
		AIScore        int      `json:"ai_score"`
		RecommendedLTV float64  `json:"recommended_ltv"`
		Grade          string   `json:"collateral_grade"`
		EstValue       int64    `json:"estimated_value_tenge"`
		Explanation    string   `json:"explanation"`
		RiskFactors    []string `json:"risk_factors"`
	}
	if err := json.Unmarshal([]byte(apiResp.Content[0].Text), &parsed); err != nil {
		return nil, fmt.Errorf("claude scorer parse score JSON: %w", err)
	}

	return &entity.CreditScore{
		ID:                  uuid.New(),
		CadastralNumber:     input.CadastralNumber,
		AIScore:             parsed.AIScore,
		RecommendedLTV:      parsed.RecommendedLTV,
		CollateralGrade:     parsed.Grade,
		EstimatedValueTenge: parsed.EstValue,
		ModelVersion:        s.model,
		Explanation:         parsed.Explanation,
		RiskFactors:         parsed.RiskFactors,
		ComputedAt:          time.Now(),
	}, nil
}

func (s *ClaudeScorer) fallbackScore(input *entity.ScoringInput) *entity.CreditScore {
	score := 50

	if avgNDVI(input) > 0.6 {
		score += 20
	}
	if input.ActiveLiens == 0 {
		score += 15
	}
	if input.LandClass <= 3 {
		score += 10
	}
	score -= input.DormantSeasons * 10

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	grade, ltv := gradeFromScore(score)

	return &entity.CreditScore{
		ID:                  uuid.New(),
		CadastralNumber:     input.CadastralNumber,
		AIScore:             score,
		RecommendedLTV:      ltv,
		CollateralGrade:     grade,
		EstimatedValueTenge: int64(input.AreaHa * 500000),
		ModelVersion:        "fallback-v1",
		Explanation:         "Score computed using deterministic fallback formula",
		RiskFactors:         buildRiskFactors(input),
		ComputedAt:          time.Now(),
	}
}

func avgNDVI(input *entity.ScoringInput) float64 {
	if len(input.NDVIHistory) == 0 {
		return 0
	}
	var sum float64
	for _, c := range input.NDVIHistory {
		sum += c.NDVIScore
	}
	return sum / float64(len(input.NDVIHistory))
}

func buildRiskFactors(input *entity.ScoringInput) []string {
	var factors []string
	if avgNDVI(input) < 0.4 {
		factors = append(factors, "Low NDVI indicates poor vegetation health")
	}
	if input.ActiveLiens > 0 {
		factors = append(factors, "Active liens reduce collateral availability")
	}
	if input.DormantSeasons > 0 {
		factors = append(factors, fmt.Sprintf("%d dormant season(s) detected", input.DormantSeasons))
	}
	if input.Disputes > 0 {
		factors = append(factors, "Land disputes present")
	}
	return factors
}

func gradeFromScore(score int) (string, float64) {
	switch {
	case score >= 80:
		return "A", 0.7
	case score >= 60:
		return "B", 0.5
	case score >= 40:
		return "C", 0.3
	default:
		return "D", 0.1
	}
}
