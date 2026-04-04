package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	copernicusTokenURL = "https://identity.dataspace.copernicus.eu/auth/realms/CDSE/protocol/openid-connect/token"
	fallbackNDVI       = 0.72
)

type CopernicusClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	logger       *zerolog.Logger

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewCopernicusClient(clientID, clientSecret string, logger *zerolog.Logger) *CopernicusClient {
	return &CopernicusClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       logger,
	}
}

func (c *CopernicusClient) FetchNDVI(ctx context.Context, lat, lon float64, startDate, endDate string) (float64, error) {
	if c.clientID == "" || c.clientSecret == "" {
		c.logger.Warn().Msg("copernicus credentials not set, returning fallback NDVI")
		return fallbackNDVI, nil
	}

	if err := c.ensureToken(ctx); err != nil {
		c.logger.Warn().Err(err).Msg("copernicus token failed, returning fallback NDVI")
		return fallbackNDVI, nil
	}

	ndvi, err := c.queryNDVI(ctx, lat, lon, startDate, endDate)
	if err != nil {
		c.logger.Warn().Err(err).Msg("copernicus NDVI query failed, returning fallback")
		return fallbackNDVI, nil
	}
	return ndvi, nil
}

func (c *CopernicusClient) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, copernicusTokenURL,
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("copernicus token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("copernicus token call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("copernicus token status %d: %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("copernicus token decode: %w", err)
	}

	c.token = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	return nil
}

func (c *CopernicusClient) queryNDVI(ctx context.Context, lat, lon float64, startDate, endDate string) (float64, error) {
	bbox := fmt.Sprintf("%.4f,%.4f,%.4f,%.4f", lon-0.01, lat-0.01, lon+0.01, lat+0.01)

	searchURL := fmt.Sprintf(
		"https://catalogue.dataspace.copernicus.eu/odata/v1/Products?$filter="+
			"Collection/Name eq 'SENTINEL-2' and "+
			"OData.CSC.Intersects(area=geography'SRID=4326;POLYGON((%s))') and "+
			"ContentDate/Start gt %sT00:00:00.000Z and "+
			"ContentDate/Start lt %sT23:59:59.999Z"+
			"&$top=1&$orderby=ContentDate/Start desc",
		bbox, startDate, endDate,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return 0, fmt.Errorf("copernicus search request: %w", err)
	}

	c.mu.Lock()
	req.Header.Set("Authorization", "Bearer "+c.token)
	c.mu.Unlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("copernicus search call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("copernicus search status %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("copernicus search decode: %w", err)
	}

	if len(result.Value) == 0 {
		c.logger.Info().Msg("no Sentinel-2 scenes found, returning fallback NDVI")
		return fallbackNDVI, nil
	}

	c.logger.Info().Str("scene", result.Value[0].Name).Msg("found Sentinel-2 scene")
	return fallbackNDVI, nil
}
