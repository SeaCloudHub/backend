package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeaCloudHub/backend/adapters/httpserver"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	server, err := httpserver.New()
	assert.NoError(t, err)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	server.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "OK!!!", response.Body.String())
}
