package httpserver_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SeaCloudHub/backend/adapters/httpserver"
	"github.com/SeaCloudHub/backend/domain/book"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type BookStore struct {
	mock.Mock
}

func (store *BookStore) Save(b *book.Book) error {
	args := store.Called(b)
	return args.Error(0)
}

func (store *BookStore) FindByISBN(isbn string) (*book.Book, error) {
	args := store.Called(isbn)
	return args.Get(0).(*book.Book), args.Error(1)
}

func TestCreateBook(t *testing.T) {
	bookTest := book.NewBook("9781617296277", "Unit Testing Principles, Practices, and Patterns")
	mockStore := new(BookStore)
	server := createBookServer(t, mockStore)

	t.Run("return status 201 after created bookTest", func(t *testing.T) {
		mockStore.On("Save", &bookTest).Return(nil).Times(1)

		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(newCreateBookRequest(bookTest), response)

		err := server.CreateBook(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, response.Code)
		mockStore.AssertExpectations(t)
	})

	t.Run("return status 400 when provide invalid input", func(t *testing.T) {
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(newCreateBookRequest(book.Book{}), response)

		err := server.CreateBook(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assertResponseError(t, response.Body, http.StatusText(http.StatusBadRequest))
		mockStore.AssertExpectations(t)
		mockStore.AssertNotCalled(t, "Save")
	})

	t.Run("return status 500 if unexpected error", func(t *testing.T) {
		mockStore.On("Save", mock.Anything).Return(errors.New("unexpected error")).Times(0)

		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(newCreateBookRequest(bookTest), response)

		err := server.CreateBook(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assertResponseError(t, response.Body, http.StatusText(http.StatusInternalServerError))
		mockStore.AssertExpectations(t)
	})
}

func TestGetBook(t *testing.T) {
	bookTest := book.NewBook("9781617296277", "Unit Testing Principles, Practices, and Patterns")
	mockStore := new(BookStore)
	server := createBookServer(t, mockStore)

	t.Run("return status 200 for existed book", func(t *testing.T) {
		mockStore.On("FindByISBN", bookTest.ISBN).Return(&bookTest, nil).Times(1)

		response := httptest.NewRecorder()
		ctx := newGetBookContext(bookTest, response)

		err := server.GetBook(ctx)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.Code)
		assertBookResponse(t, response.Body, bookTest)
		mockStore.AssertExpectations(t)
	})
}

func newGetBookContext(bookTest book.Book, response *httptest.ResponseRecorder) echo.Context {
	ctx := echo.New().NewContext(newGetBookRequest(bookTest.ISBN), response)
	ctx.SetParamNames("id")
	ctx.SetParamValues(bookTest.ISBN)
	return ctx
}

func assertBookResponse(t *testing.T, r io.Reader, bookTest book.Book) {
	t.Helper()

	var got book.Book
	err := json.NewDecoder(r).Decode(&got)
	assert.NoError(t, err)
	assert.Equal(t, bookTest.ISBN, got.ISBN)
	assert.Equal(t, bookTest.Name, got.Name)
}

func newGetBookRequest(id string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%s", id), nil)
	return request
}

func newCreateBookRequest(b book.Book) *http.Request {
	body := strings.NewReader(fmt.Sprintf(`{"isbn": "%s", "name": "%s"}`, b.ISBN, b.Name))
	request := httptest.NewRequest(http.MethodPost, "/api/books", body)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return request
}

func assertResponseError(t testing.TB, body io.Reader, message string) {
	t.Helper()

	var got map[string]string
	err := json.NewDecoder(body).Decode(&got)
	assert.NoError(t, err)
	assert.Equal(t, message, got["message"])
}

func createBookServer(t testing.TB, store book.Storage) *httpserver.Server {
	t.Helper()

	server, err := httpserver.New()
	assert.NoError(t, err)
	server.BookStore = store
	return server
}
