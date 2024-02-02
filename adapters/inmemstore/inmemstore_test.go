package inmemstore_test

import (
	"testing"

	"github.com/SeaCloudHub/backend/adapters/inmemstore"

	"github.com/stretchr/testify/assert"
)

func TestOpenConnection(t *testing.T) {
	t.Run("it should open new connection", func(t *testing.T) {
		db, err := inmemstore.NewConnection()

		assert.NoError(t, err)
		assert.NotNil(t, db)
		db.Close()
	})
}
