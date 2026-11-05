package eval

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	Timeout     time.Duration
	Concurrency int
	PriceIn     float64
	PriceOut    float64
	Mode        string // live | replay
	Host        string
	Port        string
	DataDir     string
}

func LoadConfig() Config {
	loadDotEnv(".env")

	cfg := Config{
		APIKey:      strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		BaseURL:     strings.TrimRight(envOr("DEEPSEEK_BASE_URL", "https://api.deepseek.com"), "/"),
		Model:       envOr("EVAL_MODEL", "deepseek-flash"),
		Temperature: envFloat("EVAL_TEMPERATURE", 0),
		Timeout:     time.Duration(envInt("EVAL_REQUEST_TIMEOUT", 90)) * time.Second,
		Concurrency: max(1, envInt("EVAL_CONCURRENCY", 3)),
		PriceIn:     envFloat("EVAL_PRICE_INPUT_PER_M", 2.0),
		PriceOut:    envFloat("EVAL_PRICE_OUTPUT_PER_M", 8.0),
		Host:        envOr("EVAL_HOST", "127.0.0.1"),
		Port:        envOr("EVAL_PORT", "8765"),
		DataDir:     envOr("EVAL_DATA_DIR", "data"),
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("EVAL_MODE"))) {
	case "live":
		cfg.Mode = "live"
	case "replay":
		cfg.Mode = "replay"
	default:
		if cfg.APIKey != "" {
			cfg.Mode = "live"
		} else {
			cfg.Mode = "replay"
		}
	}
	return cfg
}

func (c Config) Cost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)/1_000_000*c.PriceIn + float64(completionTokens)/1_000_000*c.PriceOut
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64)
	if err != nil {
		return fallback
	}
	return value
}

// loadDotEnv 读取项目根目录 .env，不覆盖已有环境变量。
func loadDotEnv(name string) {
	file, err := os.Open(name)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}
