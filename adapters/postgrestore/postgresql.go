package postgrestore

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SeaCloudHub/backend/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Options struct {
	DBName   string
	DBUser   string
	Password string
	Host     string
	Port     string
	SSLMode  bool
	Debug    bool
}

func ParseFromConfig(c *config.Config) Options {
	return Options{
		DBName:   c.DB.Name,
		DBUser:   c.DB.User,
		Password: c.DB.Pass,
		Host:     c.DB.Host,
		Port:     strconv.Itoa(c.DB.Port),
		SSLMode:  c.DB.EnableSSL,
		Debug:    c.Debug,
	}
}

func NewConnection(opts Options) (*gorm.DB, error) {
	sslmode := "disable"
	if opts.SSLMode {
		sslmode = "enable"
	}

	connectionString := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		opts.DBUser, opts.Password, opts.Host, opts.Port, opts.DBName, sslmode,
	)

	db, err := gorm.Open(postgres.New(
		postgres.Config{DSN: connectionString, PreferSimpleProtocol: true}),
		&gorm.Config{TranslateError: true},
	)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if opts.Debug {
		db = db.Debug()
	}

	rawDB, _ := db.DB()
	// rawDB.SetConnMaxIdleTime(time.Hour)
	// rawDB.SetMaxIdleConns(cfg.PostgreSQL.DBMaxIdleConns)
	// rawDB.SetMaxOpenConns(cfg.PostgreSQL.DBMaxOpenConns)
	rawDB.SetConnMaxLifetime(time.Minute * 5)

	err = rawDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
