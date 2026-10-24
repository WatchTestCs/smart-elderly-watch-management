// Package config 从环境变量装配服务配置，连接凭据必须显式配置。
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"careold-go/internal/securestore"
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

// Validate 拒绝那些"进程能起来、但业务接口永远不可用且不会自愈"的配置。
//
// 这些错误原先发生在 cmd/server 的 watchDatabase 协程里：只打印一行日志就
// return，不再进入 3 秒重连循环；而就绪文件和 /healthz 早已成功，桌面端与
// 探针都认为服务正常。配置错误本身不会随等待消失（环境变量在进程生命周期内
// 不变），所以必须在启动阶段就说清楚，而不是让调用方永远看到"稍后重试"。
func (c Config) Validate() error {
	// 只要配置了 MySQL，就必须有数据加密密钥：缺少它时照护备注、通知收件人等
	// 敏感字段会明文落库，登录审计退化为可离线穷举的裸哈希——无论会话后端是什么。
	if strings.TrimSpace(c.MySQLDSN) != "" {
		if strings.TrimSpace(c.DataKey) == "" {
			return errors.New("配置 MYSQL_DSN 时必须设置 DATA_ENCRYPTION_KEY（32 字节随机数的 base64）；" +
				"缺少它时敏感字段会明文落库，业务接口会永久返回 503")
		}
		if _, err := securestore.New(c.DataKey); err != nil {
			return fmt.Errorf("DATA_ENCRYPTION_KEY 无效：%w", err)
		}
	}
	switch c.SessionBackend {
	case "", "memory":
		// 空值与 memory 都是单实例开发模式，会话保存在进程内存里。
		return nil
	case "database":
	default:
		return fmt.Errorf("SESSION_BACKEND=%q 无效，只能是 database 或 memory", c.SessionBackend)
	}
	if strings.TrimSpace(c.DataKey) == "" {
		return errors.New("SESSION_BACKEND=database 时必须设置 DATA_ENCRYPTION_KEY（32 字节随机数的 base64）；" +
			"缺少它时数据库会话无法加密，业务接口会永久返回 503")
	}
	if _, err := securestore.New(c.DataKey); err != nil {
		return fmt.Errorf("DATA_ENCRYPTION_KEY 无效：%w", err)
	}
	return nil
}

// SecureCookies 报告会话 Cookie 是否应携带 Secure 标志：凡是非回环监听地址，
// 都假定可能经网络（含明文代理）访问，Cookie 必须只在 TLS 上传输。
func (c Config) SecureCookies() bool {
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		host = c.Addr
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	switch strings.ToLower(host) {
	case "", "localhost", "127.0.0.1", "::1":
		return false
	}
	return true
}

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
