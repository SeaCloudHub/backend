package postgrestore

import (
	"fmt"
	"strconv"

	"github.com/SeaCloudHub/backend/pkg/config"

	"github.com/jmoiron/sqlx"
)

type Options struct {
	DBName   string
	DBUser   string
	Password string
	Host     string
	Port     string
	SSLMode  bool
}

func ParseFromConfig(c *config.Config) Options {
	return Options{
		DBName:   c.DB.Name,
		DBUser:   c.DB.User,
		Password: c.DB.Pass,
		Host:     c.DB.Host,
		Port:     strconv.Itoa(c.DB.Port),
		SSLMode:  c.DB.EnableSSL,
	}
}

func NewConnection(opts Options) (*sqlx.DB, error) {
	sslmode := "disable"
	if opts.SSLMode {
		sslmode = "enable"
	}

	datasource := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		opts.Host, opts.Port, opts.DBUser, opts.Password, opts.DBName, sslmode,
	)

	return sqlx.Connect("postgres", datasource)
}
