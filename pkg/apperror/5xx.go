package apperror

import "net/http"

const (
	InternalCode = "500001"
)

// 500 Internal Server Error
func ErrInternalServer(err error) Error {
	return NewError(err, http.StatusInternalServerError, InternalCode, "Internal Server Error")
}
