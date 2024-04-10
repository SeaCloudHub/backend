package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var Empty = new(Config)

type Config struct {
	AppEnv       string `envconfig:"APP_ENV"`
	Debug        bool   `envconfig:"DEBUG"`
	Port         int    `envconfig:"PORT"`
	SentryDSN    string `envconfig:"SENTRY_DSN"`
	AllowOrigins string `envconfig:"ALLOW_ORIGINS"`

	DB struct {
		Name      string `envconfig:"DB_NAME"`
		Host      string `envconfig:"DB_HOST"`
		Port      int    `envconfig:"DB_PORT"`
		User      string `envconfig:"DB_USER"`
		Pass      string `envconfig:"DB_PASS"`
		EnableSSL bool   `envconfig:"ENABLE_SSL"`
	}

	SeaweedFS struct {
		MasterServer string `envconfig:"MASTER_SERVER"`
		FilerServer  string `envconfig:"FILER_SERVER"`
	}

	Kratos struct {
		AdminURL  string `envconfig:"KRATOS_ADMIN_URL"`
		PublicURL string `envconfig:"KRATOS_PUBLIC_URL"`
	}

	Keto struct {
		ReadURL  string `envconfig:"KETO_READ_URL"`
		WriteURL string `envconfig:"KETO_WRITE_URL"`
	}
}

func LoadConfig() (*Config, error) {
	// load default .env file, ignore the error
	_ = godotenv.Load()

	cfg := new(Config)
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, fmt.Errorf("load config error: %v", err)
	}

	return cfg, nil
}
