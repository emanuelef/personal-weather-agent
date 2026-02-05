package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/emanuelefumagalli/test-agent/internal/agent"
	"github.com/emanuelefumagalli/test-agent/internal/ollama"
	"github.com/emanuelefumagalli/test-agent/internal/weather"
)

const (
	// London Heathrow (wind check)
	heathrowLatitude  = 51.47
	heathrowLongitude = -0.4543

	// Twickenham (rain check)
	twickenhamLatitude  = 51.449
	twickenhamLongitude = -0.337
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	ag := agent.New(agent.Config{
		// Wind check at 10am UTC
		WindLocation: "London Heathrow",
		WindDays:     15,
		WindHour:     10,
		WindWeather: &weather.OpenMeteoClient{
			Latitude:  heathrowLatitude,
			Longitude: heathrowLongitude,
		},

		// Rain check at 7:30am London time
		RainLocation: "Twickenham",
		RainDays:     7,
		RainHour:     7,
		RainWeather: &weather.OpenMeteoClient{
			Latitude:  twickenhamLatitude,
			Longitude: twickenhamLongitude,
		},

		Ollama: &ollama.Client{
			Host:  envOrDefault("OLLAMA_HOST", "http://127.0.0.1:11434"),
			Model: envOrDefault("OLLAMA_MODEL", "llama3.1"),
		},
		TelegramToken:  os.Getenv("TELEGRAM_TOKEN"),
		TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
	})

	if err := ag.Run(ctx); err != nil {
		log.Fatalf("agent failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
