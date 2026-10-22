// Package config 从环境变量装配服务配置，连接凭据必须显式配置。
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string // 默认只监听回环地址，远程部署必须显式配置
	MySQLDSN         string
	PGAnalyticsURL   string
	SessionTTL       time.Duration // 登录 session 有效期，默认 12h
	ImageDir         string        // 图片上传落盘目录，默认 ./data/images
	DesktopToken     string        // X-Suining-Desktop 桌面端 token
	DataKey          string
	SessionBackend   string
	HealthCacheBytes int64 // 健康统计内存缓存预算；0 禁用
}

func getenv(key, def string) string {
	if path := os.Getenv(key + "_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func Secret(key string) string { return getenv(key, "") }

func Load() Config {
	ttlHours, err := strconv.Atoi(getenv("SESSION_TTL_HOURS", "12"))
	if err != nil || ttlHours <= 0 {
		ttlHours = 12
	}
	cacheMiB, err := strconv.ParseInt(getenv("HEALTH_HOT_CACHE_MIB", "16"), 10, 64)
	if err != nil || cacheMiB < 0 || cacheMiB > 256 {
		cacheMiB = 16
	}
	return Config{
		MySQLDSN:         getenv("MYSQL_DSN", ""),
		PGAnalyticsURL:   getenv("PG_ANALYTICS_URL", ""),
		Addr:             getenv("ADDR", "127.0.0.1:8080"),
		SessionTTL:       time.Duration(ttlHours) * time.Hour,
		ImageDir:         getenv("IMAGE_DIR", "./data/images"),
		DesktopToken:     getenv("DESKTOP_TOKEN", getenv("SUINING_DESKTOP_TOKEN", "")),
		DataKey:          getenv("DATA_ENCRYPTION_KEY", ""),
		SessionBackend:   strings.ToLower(strings.TrimSpace(getenv("SESSION_BACKEND", "database"))),
		HealthCacheBytes: cacheMiB << 20,
	}
}
