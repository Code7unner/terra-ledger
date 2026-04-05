package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackNDVI_PerParcel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cadastral string
		wantDiff  bool
	}{
		{"known_parcel_001", "KZ11-0032-001", true},
		{"known_parcel_004", "KZ11-0032-004", true},
		{"unknown_parcel", "KZ99-9999-999", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ndvi := fallbackNDVI(tc.cadastral)
			assert.Greater(t, ndvi, 0.0)
			assert.LessOrEqual(t, ndvi, 1.0)
			if tc.wantDiff {
				other := fallbackNDVI("KZ11-0032-003")
				assert.NotEqual(t, ndvi, other, "different parcels should have different NDVI")
			}
		})
	}
}

func TestParseSentinelHubStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		want    float64
		wantErr bool
	}{
		{
			name: "valid_response",
			body: `{"data":[{"outputs":{"ndvi":{"bands":{"B0":{"stats":{"mean":0.73}}}}}}]}`,
			want: 0.73,
		},
		{
			name:    "empty_data",
			body:    `{"data":[]}`,
			wantErr: true,
		},
		{
			name:    "invalid_ndvi",
			body:    `{"data":[{"outputs":{"ndvi":{"bands":{"B0":{"stats":{"mean":5.0}}}}}}]}`,
			wantErr: true,
		},
		{
			name:    "malformed_json",
			body:    `{broken`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ndvi, err := parseSentinelHubStats(strings.NewReader(tc.body))
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tc.want, ndvi, 0.001)
		})
	}
}

func TestCopernicusClient_FetchNDVI_NoCredentials(t *testing.T) {
	logger := zerolog.Nop()
	c := NewCopernicusClient("", "", &logger)

	ndvi, err := c.FetchNDVI(context.Background(), "KZ11-0032-001", 51.13, 69.41, "2026-03-01", "2026-04-01")
	require.NoError(t, err)
	assert.Greater(t, ndvi, 0.0)
}

func TestCopernicusClient_FetchNDVI_TokenFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	logger := zerolog.Nop()
	c := NewCopernicusClient("id", "secret", &logger)
	c.httpClient = ts.Client()

	ndvi, err := c.FetchNDVI(context.Background(), "KZ11-0032-003", 51.14, 69.40, "2026-03-01", "2026-04-01")
	require.NoError(t, err)
	assert.Greater(t, ndvi, 0.0, "should use fallback on token failure")
}
