package geocoder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"order-service/internal/domain/order"
)

type OpenCageGeocoder struct {
	apiKey string
	client *http.Client
}

func NewOpenCageGeocoder(apiKey string) *OpenCageGeocoder {
	return &OpenCageGeocoder{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (g *OpenCageGeocoder) Geocode(ctx context.Context, addr order.Address) (float64, float64, error) {
	baseURL := "https://api.opencagedata.com/geocode/v1/json"

	query := fmt.Sprintf(
		"%s %s, %s %s",
		addr.House,
		addr.Street,
		addr.PostalCode,
		addr.City,
	)

	u, err := url.Parse(baseURL)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("key", g.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, 0, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("opencage geocode failed: status=%s", resp.Status)
	}

	var data struct {
		Results []struct {
			Geometry struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"geometry"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, fmt.Errorf("decode response failed: %w", err)
	}

	if len(data.Results) == 0 {
		return 0, 0, fmt.Errorf("no geocoding results found")
	}

	return data.Results[0].Geometry.Lat, data.Results[0].Geometry.Lng, nil
}
