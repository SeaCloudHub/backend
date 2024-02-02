package postgrestore_test

import (
	"testing"

	"github.com/SeaCloudHub/backend/adapters/testutil"

	"github.com/stretchr/testify/assert"
)

type Info struct {
	CurrentUser string `db:"current_user"`
}

func TestConnection(t *testing.T) {
	dbName, dbUser, dbPass := "test1", "test1", "123456"
	db := testutil.CreateConnection(t, dbName, dbUser, dbPass)
	testutil.MigrateTestDatabase(t, db, "../../migrations")

	var info Info
	err := db.Get(&info, "SELECT current_user")
	assert.NoError(t, err)
	assert.Equal(t, dbUser, info.CurrentUser)
}
