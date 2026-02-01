package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ForecastDay represents a daily wind forecast snapshot for a location.
type ForecastDay struct {
	Date         time.Time
	WindSpeedMax float64
	WindGustMax  float64
	WindDirMean  float64 // in degrees, 0 = North
}

// RainForecast represents rain data for a day with hourly detail.
type RainForecast struct {
	Date            time.Time
	PrecipProb      int       // daily max precipitation probability %
	PrecipMM        float64   // daily total precipitation mm
	MorningRainProb []int     // hourly rain probability 6am-10am (indices 0-4)
	MorningRainMM   []float64 // hourly precipitation 6am-10am
	AfternoonProb   []int     // hourly rain probability 15-18 (indices 0-3)
}

// Forecaster fetches a set of daily wind forecasts.
type Forecaster interface {
	Fetch(ctx context.Context, days int) ([]ForecastDay, error)
}

// RainForecaster fetches rain forecasts.
type RainForecaster interface {
	FetchRain(ctx context.Context, days int) ([]RainForecast, error)
}

// OpenMeteoClient hits the public Open-Meteo API (no API key needed).
type OpenMeteoClient struct {
	Latitude   float64
	Longitude  float64
	HTTPClient *http.Client
}

const openMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"

// Fetch retrieves up to `days` worth of daily max wind speeds and gusts.
func (c *OpenMeteoClient) Fetch(ctx context.Context, days int) ([]ForecastDay, error) {
	if days < 1 {
		return nil, errors.New("days must be >= 1")
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	query := url.Values{}
	query.Set("latitude", fmt.Sprintf("%f", c.Latitude))
	query.Set("longitude", fmt.Sprintf("%f", c.Longitude))
	query.Set("daily", "windspeed_10m_max,windgusts_10m_max,winddirection_10m_dominant")
	query.Set("forecast_days", fmt.Sprintf("%d", days))
	query.Set("timezone", "auto")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoBaseURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call open-meteo: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo returned %s", resp.Status)
	}

	var payload openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode open-meteo response: %w", err)
	}

	if payload.Daily == nil {
		return nil, errors.New("open-meteo response missing daily block")
	}

	return payload.Daily.toForecastDays()
}

type openMeteoResponse struct {
	Daily  *openMeteoDaily  `json:"daily"`
	Hourly *openMeteoHourly `json:"hourly"`
}

type openMeteoHourly struct {
	Time        []string  `json:"time"`
	PrecipProb  []int     `json:"precipitation_probability"`
	Precip      []float64 `json:"precipitation"`
}

type openMeteoDaily struct {
	Time         []string  `json:"time"`
	WindSpeedMax []float64 `json:"windspeed_10m_max"`
	WindGustMax  []float64 `json:"windgusts_10m_max"`
	WindDirMean  []float64 `json:"winddirection_10m_dominant"`
}

// FetchRain retrieves rain forecast with hourly morning data.
func (c *OpenMeteoClient) FetchRain(ctx context.Context, days int) ([]RainForecast, error) {
	if days < 1 {
		return nil, errors.New("days must be >= 1")
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	query := url.Values{}
	query.Set("latitude", fmt.Sprintf("%f", c.Latitude))
	query.Set("longitude", fmt.Sprintf("%f", c.Longitude))
	query.Set("daily", "precipitation_sum,precipitation_probability_max")
	query.Set("hourly", "precipitation_probability,precipitation")
	query.Set("forecast_days", fmt.Sprintf("%d", days))
	query.Set("timezone", "Europe/London")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoBaseURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call open-meteo: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo returned %s", resp.Status)
	}

	var payload rainResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode open-meteo response: %w", err)
	}

	return payload.toRainForecasts()
}

type rainResponse struct {
	Daily  rainDaily  `json:"daily"`
	Hourly rainHourly `json:"hourly"`
}

type rainDaily struct {
	Time       []string  `json:"time"`
	PrecipSum  []float64 `json:"precipitation_sum"`
	PrecipProb []int     `json:"precipitation_probability_max"`
}

type rainHourly struct {
	Time       []string  `json:"time"`
	PrecipProb []int     `json:"precipitation_probability"`
	Precip     []float64 `json:"precipitation"`
}

func (r *rainResponse) toRainForecasts() ([]RainForecast, error) {
	if len(r.Daily.Time) == 0 {
		return nil, errors.New("no daily rain data")
	}

	out := make([]RainForecast, 0, len(r.Daily.Time))

	for i, dateStr := range r.Daily.Time {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}

		rf := RainForecast{
			Date:       date,
			PrecipProb: r.Daily.PrecipProb[i],
			PrecipMM:   r.Daily.PrecipSum[i],
		}

		// Extract hourly data for school times
		for j, hourStr := range r.Hourly.Time {
			hourTime, err := time.Parse("2006-01-02T15:04", hourStr)
			if err != nil {
				continue
			}
			if hourTime.Year() == date.Year() && hourTime.Month() == date.Month() && hourTime.Day() == date.Day() {
				hour := hourTime.Hour()
				// Morning: 6am-10am for drop-off
				if hour >= 6 && hour <= 10 {
					rf.MorningRainProb = append(rf.MorningRainProb, r.Hourly.PrecipProb[j])
					rf.MorningRainMM = append(rf.MorningRainMM, r.Hourly.Precip[j])
				}
				// Afternoon: 15-18 for pickup (Wed 15-16, others 17-18)
				if hour >= 15 && hour <= 18 {
					rf.AfternoonProb = append(rf.AfternoonProb, r.Hourly.PrecipProb[j])
				}
			}
		}

		out = append(out, rf)
	}

	return out, nil
}

func (d *openMeteoDaily) toForecastDays() ([]ForecastDay, error) {
	if len(d.Time) == 0 {
		return nil, errors.New("no daily data returned")
	}
	if len(d.Time) != len(d.WindSpeedMax) || len(d.Time) != len(d.WindGustMax) || len(d.Time) != len(d.WindDirMean) {
		return nil, errors.New("open-meteo arrays differ in length")
	}

	out := make([]ForecastDay, 0, len(d.Time))
	for idx := range d.Time {
		date, err := time.Parse("2006-01-02", d.Time[idx])
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", d.Time[idx], err)
		}
		out = append(out, ForecastDay{
			Date:         date,
			WindSpeedMax: d.WindSpeedMax[idx],
			WindGustMax:  d.WindGustMax[idx],
			WindDirMean:  d.WindDirMean[idx],
		})
	}
	return out, nil
}
