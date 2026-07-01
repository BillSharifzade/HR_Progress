package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv           string
	HTTPAddr         string
	DatabaseURL      string
	JWTSecret        []byte
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	BcryptCost       int
	LogLevel         string
	DefaultTimezone  string
	BootstrapAdminUsername string
	BootstrapAdminPassword string
	CORSAllowedOrigins string

	// BasePath is the URL path prefix the app is served under (e.g. "/progress"
	// when co-hosted on a shared domain). Empty means served at root.
	BasePath string

	// 1F (Первая форма) integration. Empty BaseURL disables the scheduler.
	OneFBaseURL      string
	OneFAuthToken    string
	OneFSyncInterval time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		AppEnv:           env("APP_ENV", "development"),
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		DatabaseURL:      env("DATABASE_URL", ""),
		LogLevel:         env("LOG_LEVEL", "info"),
		DefaultTimezone:  env("DEFAULT_TIMEZONE", "Europe/Moscow"),
		BootstrapAdminUsername: env("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPassword: env("BOOTSTRAP_ADMIN_PASSWORD", "admin"),
		CORSAllowedOrigins: env("CORS_ALLOWED_ORIGINS", "*"),
		BasePath:           env("BASE_PATH", ""),
	}

	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	secret := env("JWT_SECRET", "")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is required (generate with: openssl rand -hex 32)")
	}
	c.JWTSecret = []byte(secret)

	atTTL, err := time.ParseDuration(env("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	c.AccessTokenTTL = atTTL

	rtTTL, err := time.ParseDuration(env("REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}
	c.RefreshTokenTTL = rtTTL

	if v := env("BCRYPT_COST", ""); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("BCRYPT_COST: %w", err)
		}
		c.BcryptCost = n
	}

	c.OneFBaseURL = env("ONEF_BASE_URL", "")
	c.OneFAuthToken = env("ONEF_AUTH_TOKEN", "")
	oneFInterval, err := time.ParseDuration(env("ONEF_SYNC_INTERVAL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("ONEF_SYNC_INTERVAL: %w", err)
	}
	c.OneFSyncInterval = oneFInterval

	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
