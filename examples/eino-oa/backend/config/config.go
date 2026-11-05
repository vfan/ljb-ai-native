package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort    string
	DBPath        string
	DeepSeekKey   string
	DeepSeekModel string
	DeepSeekBase  string

	// AdminUsername / AdminPassword 来自 backend/.env，每次启动同步到数据库。
	// AdminPassword 必填。
	AdminUsername string
	AdminPassword string
	SessionTTL    time.Duration
}

// Load 从 .env 文件和环境变量中加载配置。
// 配置文件固定为 backend/.env（与 go.mod 同级），从仓库根目录或 backend 目录启动都能找到。
// 环境变量始终优先于 .env 文件中的值。
func Load() *Config {
	tryLoadEnvFile()

	dbPath := getEnv("DB_PATH", "data/eino-oa.db")
	if !filepath.IsAbs(dbPath) {
		// 相对路径统一按 backend/ 解析，避免从仓库根目录启动二进制时
		// 跑到另一份 data/eino-oa.db。
		dbPath = filepath.Join(backendDir(), dbPath)
	}

	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBPath:        dbPath,
		DeepSeekKey:   getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekModel: getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekBase:  getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		SessionTTL:    time.Duration(getEnvInt("SESSION_TTL_HOURS", 168)) * time.Hour,
	}
}

func tryLoadEnvFile() {
	path := envFilePath()

	if _, err := os.Stat(path); err != nil {
		log.Printf("未找到 %s，仅使用环境变量", path)
		return
	}
	if err := godotenv.Load(path); err != nil {
		log.Printf("加载 %s 失败: %v", path, err)
		return
	}
	log.Printf("已加载配置文件: %s", path)
}

// envFilePath 返回 backend/.env 的路径。
func envFilePath() string {
	return filepath.Join(backendDir(), ".env")
}

// backendDir 返回 backend/ 相对于当前工作目录的路径：
//   - 在 backend/ 目录执行：.
//   - 在仓库根目录执行：backend
//   - 在 backend/cmd/server 执行：../..
func backendDir() string {
	switch {
	case fileExists("go.mod") && isDir("cmd"):
		return "."
	case fileExists(filepath.Join("backend", "go.mod")):
		return "backend"
	case fileExists(filepath.Join("..", "go.mod")):
		return ".."
	case fileExists(filepath.Join("..", "..", "go.mod")):
		return filepath.Join("..", "..")
	default:
		return "."
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("环境变量 %s=%q 不是有效正整数，使用默认值 %d", key, v, fallback)
		return fallback
	}
	return n
}
