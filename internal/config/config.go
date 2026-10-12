package config

import (
	"flag"
	"os"
)

const (
	DefaultRunAddress  = "localhost:8080"
	DefaultDatabaseURI = ""
	DefaultAccrualAddr = "localhost:8081"
	DefaultJWTSecret   = "" // TODO: выбрать дефолт jwt

	EnvRunAddress     = "RUN_ADDRESS"
	EnvDatabaseURI    = "DATABASE_URI"
	EnvAccrualAddress = "ACCRUAL_SYSTEM_ADDRESS"
	EnvJWTSecret      = "JWT_SECRET"
)

// Config хранит настройки сервиса.
type Config struct {
	RunAddress     string
	DatabaseURI    string
	AccrualAddress string
	JWTSecret      string
}

// ParseFlags читает конфигурацию из флагов и переменных окружения.
// Приоритет: env > flag > default.
func ParseFlags() *Config {
	flagRun := flag.String("a", DefaultRunAddress, "address and port to run server")
	flagDB := flag.String("d", DefaultDatabaseURI, "database URI")
	flagAccrual := flag.String("r", DefaultAccrualAddr, "accrual system address")
	flagSecret := flag.String("jwt-secret", DefaultJWTSecret, "JWT signing secret")
	flag.Parse()

	cfg := &Config{
		RunAddress:     pick(os.Getenv(EnvRunAddress), *flagRun, DefaultRunAddress),
		DatabaseURI:    pick(os.Getenv(EnvDatabaseURI), *flagDB, DefaultDatabaseURI),
		AccrualAddress: pick(os.Getenv(EnvAccrualAddress), *flagAccrual, DefaultAccrualAddr),
		JWTSecret:      pick(os.Getenv(EnvJWTSecret), *flagSecret, DefaultJWTSecret),
	}
	return cfg
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
