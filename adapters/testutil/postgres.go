package testutil

import (
	"context"
	"testing"

	"github.com/SeaCloudHub/backend/adapters/postgrestore"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
)

func MigrateTestDatabase(t testing.TB, db *sqlx.DB, migrationPath string) {
	t.Helper()

	migrations := &migrate.FileMigrationSource{
		Dir: migrationPath,
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	assert.NoError(t, err)
}

func CreateConnection(t testing.TB, dbName string, dbUser string, dbPass string) *sqlx.DB {
	cont := SetupPostgresContainer(t, dbName, dbUser, dbPass)
	host, _ := cont.Host(context.Background())
	port, _ := cont.MappedPort(context.Background(), "5432")

	db, err := postgrestore.NewConnection(postgrestore.Options{
		DBName:   dbName,
		DBUser:   dbUser,
		Password: dbPass,
		Host:     host,
		Port:     port.Port(),
	})
	assert.NoError(t, err)

	return db
}
