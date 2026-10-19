// Package config 从环境变量装配服务配置，全部有合理默认值。
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr         string        // 默认只监听回环地址，远程部署必须显式配置
	DatabaseURL  string        // PostgreSQL 连接串
	SessionTTL   time.Duration // 登录 session 有效期，默认 12h
	ImageDir     string        // 图片上传落盘目录，默认 ./data/images
	DesktopToken string        // X-Suining-Desktop 桌面端 token
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	ttlHours, err := strconv.Atoi(getenv("SESSION_TTL_HOURS", "12"))
	if err != nil || ttlHours <= 0 {
		ttlHours = 12
	}
	return Config{
		Addr:         getenv("ADDR", "127.0.0.1:8080"),
		DatabaseURL:  getenv("DATABASE_URL", ""),
		SessionTTL:   time.Duration(ttlHours) * time.Hour,
		ImageDir:     getenv("IMAGE_DIR", "./data/images"),
		DesktopToken: getenv("DESKTOP_TOKEN", getenv("SUINING_DESKTOP_TOKEN", "")),
	}
}
