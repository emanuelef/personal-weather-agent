package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emanuelefumagalli/test-agent/internal/ollama"
	"github.com/emanuelefumagalli/test-agent/internal/weather"
)

// Config wires together the dependencies and runtime options for the agent.
type Config struct {
	LocationName   string
	ForecastDays   int
	Weather        weather.Forecaster
	Ollama         *ollama.Client
	TelegramToken  string
	TelegramChatID string
}

// Agent coordinates the weather fetch and Ollama summarization.
type Agent struct {
	cfg Config
}

// New returns a fully constructed Agent.
func New(cfg Config) *Agent {
	if cfg.ForecastDays <= 0 {
		cfg.ForecastDays = 15
	}
	return &Agent{cfg: cfg}
}

// Run executes one fetch-and-summarize pass.
// Run runs the agent 24/7, fetching and sending once a day.
func (a *Agent) Run(ctx context.Context) error {
	// Get run time from env (default 10:00 UTC)
	runHour := 10
	runMinute := 0
	if h := os.Getenv("WIND_CHECK_HOUR"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v >= 0 && v < 24 {
			runHour = v
		}
	}
	if m := os.Getenv("WIND_CHECK_MINUTE"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 0 && v < 60 {
			runMinute = v
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if a.cfg.Weather == nil {
			return fmt.Errorf("weather client is missing")
		}
		if a.cfg.Ollama == nil {
			return fmt.Errorf("ollama client is missing")
		}

		forecast, err := a.cfg.Weather.Fetch(ctx, a.cfg.ForecastDays)
		if err != nil {
			fmt.Printf("fetch forecast: %v\n", err)
		}

		location := fallbackLocation(a.cfg.LocationName)
		report := buildForecastTable(forecast)
		easterlyAnalysis := buildEasterlyAnalysis(forecast)

		fmt.Printf("\n%d-day %s wind forecast (km/h):\n", len(forecast), location)
		fmt.Println(report)
		fmt.Println(easterlyAnalysis)

		prompt := buildPrompt(location, forecast, report, easterlyAnalysis)
		fmt.Println("\nPrompt sent to Ollama:\n----------------------")
		fmt.Println(prompt)
		fmt.Println("----------------------")
		summary, err := a.cfg.Ollama.Generate(ctx, prompt)
		if err != nil {
			fmt.Printf("Ollama failed: %v\n", err)
			if a.cfg.TelegramToken != "" && a.cfg.TelegramChatID != "" {
				// Send table + easterly analysis as fallback
				fallbackMsg := formatTelegramTable(report) + "\n" + easterlyAnalysis
				err2 := sendTelegramMessage(&a.cfg, fallbackMsg)
				if err2 != nil {
					fmt.Printf("Failed to send Telegram message: %v\n", err2)
				} else {
					fmt.Println("Sent fallback wind table to Telegram.")
				}
			}
		} else {
			fmt.Println("\nOllama summary:")
			fmt.Println(summary)
			// Send to Telegram if configured
			if a.cfg.TelegramToken != "" && a.cfg.TelegramChatID != "" {
				// First send the formatted table with easterly analysis
				tableMsg := formatTelegramTable(report) + "\n" + easterlyAnalysis
				err := sendTelegramMessage(&a.cfg, tableMsg)
				if err != nil {
					fmt.Printf("Failed to send wind table to Telegram: %v\n", err)
				}
				// Then send the Ollama summary
				err = sendTelegramMessage(&a.cfg, summary)
				if err != nil {
					fmt.Printf("Failed to send Telegram message: %v\n", err)
					// Don't crash, just log and continue
				}
			}
		}

		// Sleep until the next configured run time (default 10:00 UTC)
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), runHour, runMinute, 0, 0, time.UTC)
		if !now.Before(next) {
			next = next.Add(24 * time.Hour)
		}
		sleep := time.Until(next)
		fmt.Printf("Sleeping for %v until next run at %02d:%02d UTC...\n", sleep, runHour, runMinute)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
	}
}

// formatTelegramTable wraps the table in Markdown code block for Telegram
func formatTelegramTable(table string) string {
	return "```\n" + table + "```"
}

func buildForecastTable(days []weather.ForecastDay) string {
	var b strings.Builder
	b.WriteString("Date       | Wind | Dir | East\n")
	b.WriteString("-----------+------+-----+-----\n")
	for _, day := range days {
		eastMarker := "   "
		if isEasterly(day.WindDirMean) {
			eastMarker = " ✈️"
		}
		b.WriteString(fmt.Sprintf("%s | %4.0f | %-3s |%s\n",
			day.Date.Format("Mon 02 Jan"),
			day.WindSpeedMax,
			degToCompass(day.WindDirMean),
			eastMarker,
		))
	}
	return b.String()
}

func buildPrompt(location string, _ []weather.ForecastDay, table string, easterlyAnalysis string) string {
	return fmt.Sprintf(`%s wind forecast. Easterly wind = planes overhead (✈️).

%s
%s
Summarize briefly: how many easterly days and when does wind change direction?`, location, easterlyAnalysis, table)
}

// degToCompass converts degrees to E or W (what matters for flight paths)
func degToCompass(deg float64) string {
	deg = float64(int(deg+360) % 360)
	// East: 0-180, West: 180-360
	if deg > 0 && deg < 180 {
		return "E"
	}
	return "W"
}

// isEasterly returns true if wind is from the east
func isEasterly(deg float64) bool {
	deg = float64(int(deg+360) % 360)
	return deg > 0 && deg < 180
}

// countEasterlyDays counts how many days have easterly winds
func countEasterlyDays(days []weather.ForecastDay) int {
	count := 0
	for _, d := range days {
		if isEasterly(d.WindDirMean) {
			count++
		}
	}
	return count
}

// buildEasterlyAnalysis creates a simple summary with dominant direction
func buildEasterlyAnalysis(days []weather.ForecastDay) string {
	eastCount := countEasterlyDays(days)
	westCount := len(days) - eastCount

	var dominant string
	if eastCount > westCount {
		dominant = "E ✈️"
	} else if westCount > eastCount {
		dominant = "W"
	} else {
		dominant = "Mixed"
	}

	return fmt.Sprintf("Dominant: %s | East: %d days | West: %d days\n", dominant, eastCount, westCount)
}

func fallbackLocation(name string) string {
	if strings.TrimSpace(name) == "" {
		return "the target location"
	}
	return name
}

// TelegramMessage is the payload for Telegram API
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func sendTelegramMessage(config *Config, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramToken)

	msg := TelegramMessage{
		ChatID:    config.TelegramChatID,
		Text:      message,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: close telegram response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
