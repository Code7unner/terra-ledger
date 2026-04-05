package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

func newTestScorer(baseURL string) *ClaudeScorer {
	logger := zerolog.Nop()
	s := NewClaudeScorer("test-api-key", "claude-test", &logger)
	s.baseURL = baseURL
	return s
}

func fakeScoringInput() *entity.ScoringInput {
	return &entity.ScoringInput{
		CadastralNumber: fmt.Sprintf("KZ-%s-%03d", gofakeit.LetterN(4), gofakeit.IntRange(1, 999)),
		AreaHa:          gofakeit.Float64Range(100, 5000),
		LandClass:       gofakeit.IntRange(1, 5),
		Oblast:          gofakeit.State(),
		NDVIHistory: []entity.NDVICertificate{
			{NDVIScore: gofakeit.Float64Range(0.3, 0.9)},
			{NDVIScore: gofakeit.Float64Range(0.3, 0.9)},
		},
		ActiveLiens:    gofakeit.IntRange(0, 3),
		TotalLiens:     gofakeit.IntRange(0, 5),
		Disputes:       gofakeit.IntRange(0, 2),
		DormantSeasons: gofakeit.IntRange(0, 3),
	}
}

// claudeAPIResponse builds a mock Anthropic Messages API response body.
func claudeAPIResponse(scoreJSON string) string {
	resp := map[string]any{
		"content": []map[string]string{
			{"type": "text", "text": scoreJSON},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func TestClaudeScorer_ComputeScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		handler         http.HandlerFunc
		noAPIKey        bool
		cancelCtx       bool
		wantFallback    bool
		wantErr         bool
		wantAIScore     int
		wantGrade       string
		wantLTV         float64
		wantExplanation string
	}{
		{
			name: "successful_api_response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
				assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				scoreJSON := `{"ai_score": 75, "recommended_ltv": 0.55, "collateral_grade": "B", "risk_factors": ["seasonal variance"], "explanation": "Moderate risk"}`
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, claudeAPIResponse(scoreJSON))
			},
			wantAIScore:     75,
			wantGrade:       "B",
			wantLTV:         0.55,
			wantExplanation: "Moderate risk",
		},
		{
			name: "api_returns_error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error": "internal server error"}`)
			},
			wantFallback: true,
		},
		{
			name: "api_returns_malformed_json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"content": [{"type": "text", "text": "not valid json at all!!!"}]}`)
			},
			wantFallback: true,
		},
		{
			name:       "context_cancellation",
			handler:    func(http.ResponseWriter, *http.Request) {},
			cancelCtx:  true,
			wantErr:    false, // fallback is used on error
			wantFallback: true,
		},
		{
			name:         "no_api_key_uses_fallback",
			handler:      func(http.ResponseWriter, *http.Request) {},
			noAPIKey:     true,
			wantFallback: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			scorer := newTestScorer(srv.URL)
			if tc.noAPIKey {
				scorer.apiKey = ""
			}

			ctx := context.Background()
			if tc.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // cancel immediately
			}

			input := fakeScoringInput()
			result, err := scorer.ComputeScore(ctx, input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, input.CadastralNumber, result.CadastralNumber)
			assert.False(t, result.ComputedAt.IsZero())

			if tc.wantFallback {
				assert.Equal(t, "fallback-v1", result.ModelVersion)
				assert.Equal(t, "Score computed using deterministic fallback formula", result.Explanation)
				return
			}

			assert.Equal(t, tc.wantAIScore, result.AIScore)
			assert.Equal(t, tc.wantGrade, result.CollateralGrade)
			assert.InDelta(t, tc.wantLTV, result.RecommendedLTV, 0.001)
			assert.Equal(t, tc.wantExplanation, result.Explanation)
			assert.Equal(t, "claude-test", result.ModelVersion)
		})
	}
}

func TestFallbackScoring(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      *entity.ScoringInput
		wantScore  int
		wantGrade  string
		wantLTV    float64
		wantRisks  []string
	}{
		{
			name: "high_ndvi_no_liens_low_class",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-001",
				AreaHa:          500,
				LandClass:       2,
				NDVIHistory: []entity.NDVICertificate{
					{NDVIScore: 0.8},
					{NDVIScore: 0.7},
				},
				ActiveLiens:    0,
				DormantSeasons: 0,
			},
			wantScore: 95, // 50 + 20 (ndvi>0.6) + 15 (no liens) + 10 (class<=3)
			wantGrade: "A",
			wantLTV:   0.7,
			wantRisks: nil,
		},
		{
			name: "low_ndvi_active_liens_dormant",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-002",
				AreaHa:          200,
				LandClass:       4,
				NDVIHistory: []entity.NDVICertificate{
					{NDVIScore: 0.2},
					{NDVIScore: 0.3},
				},
				ActiveLiens:    2,
				DormantSeasons: 2,
				Disputes:       1,
			},
			wantScore: 30, // 50 + 0 (ndvi<0.6) + 0 (liens) + 0 (class>3) - 20 (2 dormant)
			wantGrade: "D",
			wantLTV:   0.1,
			wantRisks: []string{
				"Low NDVI indicates poor vegetation health",
				"Active liens reduce collateral availability",
				"2 dormant season(s) detected",
				"Land disputes present",
			},
		},
		{
			name: "empty_ndvi_history",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-003",
				AreaHa:          1000,
				LandClass:       3,
				NDVIHistory:     nil,
				ActiveLiens:     0,
				DormantSeasons:  0,
			},
			wantScore: 75, // 50 + 0 (avg=0, not>0.6) + 15 (no liens) + 10 (class<=3)
			wantGrade: "B",
			wantLTV:   0.5,
			wantRisks: []string{
				"Low NDVI indicates poor vegetation health",
			},
		},
		{
			name: "score_clamped_to_zero",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-004",
				AreaHa:          100,
				LandClass:       5,
				NDVIHistory: []entity.NDVICertificate{
					{NDVIScore: 0.1},
				},
				ActiveLiens:    3,
				DormantSeasons: 8,
			},
			wantScore: 0, // 50 + 0 + 0 + 0 - 80 = -30, clamped to 0
			wantGrade: "D",
			wantLTV:   0.1,
			wantRisks: []string{
				"Low NDVI indicates poor vegetation health",
				"Active liens reduce collateral availability",
				"8 dormant season(s) detected",
			},
		},
		{
			name: "score_clamped_to_100",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-005",
				AreaHa:          3000,
				LandClass:       1,
				NDVIHistory: []entity.NDVICertificate{
					{NDVIScore: 0.9},
					{NDVIScore: 0.85},
				},
				ActiveLiens:    0,
				DormantSeasons: 0,
			},
			wantScore: 95, // 50 + 20 + 15 + 10 = 95 (not > 100, no clamp needed)
			wantGrade: "A",
			wantLTV:   0.7,
			wantRisks: nil,
		},
		{
			name: "estimated_value_uses_area",
			input: &entity.ScoringInput{
				CadastralNumber: "KZ-TEST-006",
				AreaHa:          750,
				LandClass:       3,
				NDVIHistory: []entity.NDVICertificate{
					{NDVIScore: 0.5},
				},
				ActiveLiens:    0,
				DormantSeasons: 0,
			},
			wantScore: 75, // 50 + 0 (ndvi=0.5) + 15 (no liens) + 10 (class<=3)
			wantGrade: "B",
			wantLTV:   0.5,
			wantRisks: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := zerolog.Nop()
			scorer := NewClaudeScorer("", "fallback", &logger)

			result, err := scorer.ComputeScore(context.Background(), tc.input)

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tc.wantScore, result.AIScore)
			assert.Equal(t, tc.wantGrade, result.CollateralGrade)
			assert.InDelta(t, tc.wantLTV, result.RecommendedLTV, 0.001)
			assert.Equal(t, "fallback-v1", result.ModelVersion)
			assert.Equal(t, tc.input.CadastralNumber, result.CadastralNumber)
			assert.Equal(t, "Score computed using deterministic fallback formula", result.Explanation)

			if tc.wantRisks == nil {
				assert.Empty(t, result.RiskFactors)
			} else {
				assert.Equal(t, tc.wantRisks, result.RiskFactors)
			}

			// Verify estimated value = area_ha * 500000
			expectedValue := int64(tc.input.AreaHa * 500000)
			assert.Equal(t, expectedValue, result.EstimatedValueTenge)
		})
	}
}
