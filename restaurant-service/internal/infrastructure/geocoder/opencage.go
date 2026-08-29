package geocoder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"restaurant-service/internal/domain/restaurant"
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

func (g *OpenCageGeocoder) GeocodeAddress(
	ctx context.Context,
	addr restaurant.Address,
) (float64, float64, string, error) {
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
		return 0, 0, "", fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("key", g.apiKey)
	u.RawQuery = q.Encode()

	endpoint := u.String()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return 0, 0, "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", fmt.Errorf("opencage geocode failed: status=%s", resp.Status)
	}

	var data struct {
		Results []struct {
			Geometry struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"geometry"`
			Annotations struct {
				Timezone struct {
					Name string `json:"name"`
				} `json:"timezone"`
			} `json:"annotations"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, "", fmt.Errorf("decode response failed: %w", err)
	}

	if len(data.Results) == 0 {
		return 0, 0, "", fmt.Errorf("no geocoding results found")
	}

	result := data.Results[0]

	if result.Annotations.Timezone.Name == "" {
		return 0, 0, "", fmt.Errorf("no timezone found in geocoding result")
	}

	return result.Geometry.Lat, result.Geometry.Lng, result.Annotations.Timezone.Name, nil
}
