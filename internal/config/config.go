// Package config читает конфигурацию сервиса из переменных окружения
// и флагов командной строки.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log/slog"
	"os"
)

const (
	DefaultRunAddress  = "localhost:8080"
	DefaultDatabaseURI = ""
	DefaultAccrualAddr = "localhost:8081"

	EnvRunAddress     = "RUN_ADDRESS"
	EnvDatabaseURI    = "DATABASE_URI"
	EnvAccrualAddress = "ACCRUAL_SYSTEM_ADDRESS"
	EnvJWTSecret      = "JWT_SECRET"
)

// Config — конфигурация сервиса: адрес, БД, accrual, JWT.
type Config struct {
	RunAddress     string
	DatabaseURI    string
	AccrualAddress string
	JWTSecret      string
}

// ParseFlags читает конфигурацию из os.Args: приоритет env > flag > default.
//
// Если JWT_SECRET не задан ни через env, ни через флаг — генерируется
// случайный секрет на время жизни процесса. Это безопаснее хардкод-дефолта:
// при рестарте приложения старые токены становятся невалидными.
func ParseFlags() (*Config, error) {
	return parseFlags(os.Args[1:])
}

func parseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("gophermart", flag.ContinueOnError)

	flagRun := fs.String("a", DefaultRunAddress, "address and port to run server")
	flagDB := fs.String("d", DefaultDatabaseURI, "database URI")
	flagAccrual := fs.String("r", DefaultAccrualAddr, "accrual system address")
	flagSecret := fs.String("jwt-secret", "", "JWT signing secret")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg := &Config{
		RunAddress:     pick(os.Getenv(EnvRunAddress), *flagRun, DefaultRunAddress),
		DatabaseURI:    pick(os.Getenv(EnvDatabaseURI), *flagDB, DefaultDatabaseURI),
		AccrualAddress: pick(os.Getenv(EnvAccrualAddress), *flagAccrual, DefaultAccrualAddr),
		JWTSecret:      pick(os.Getenv(EnvJWTSecret), *flagSecret, ""),
	}

	if cfg.JWTSecret == "" {
		secret, err := randomSecret()
		if err != nil {
			return nil, err
		}
		cfg.JWTSecret = secret
		slog.Warn("JWT_SECRET is not set, generated random secret for this run; " +
			"tokens will be invalidated on restart")
	}
	return cfg, nil
}

// randomSecret генерирует 32-байтный случайный секрет в hex.
func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func pick(env, flagVal, def string) string {
	if env != "" {
		return env
	}
	if flagVal != "" {
		return flagVal
	}
	return def
}
