package apperror

import (
	"net/http"
)

const (
	BindingCode            = "400001"
	ValidationCode         = "400002"
	InvalidCredentialsCode = "401003"
	UnauthorizedCode       = "401004"
	ForbiddenCode          = "403005"
	EntityNotFoundCode     = "404006"
)

// 400 Bad Request
func ErrInvalidRequest(err error) Error {
	return NewError(err, http.StatusBadRequest, BindingCode, "Invalid request")
}

func ErrInvalidParam(err error) Error {
	return NewError(err, http.StatusBadRequest, ValidationCode, "Invalid param")
}

// 401 Unauthorized
func ErrInvalidCredentials(err error) Error {
	return NewError(err, http.StatusUnauthorized, InvalidCredentialsCode, "Invalid credentials")
}

func ErrUnauthorized(err error) Error {
	return NewError(err, http.StatusUnauthorized, UnauthorizedCode, "Unauthorized")
}

// 403 Forbidden
func ErrForbidden(err error) Error {
	return NewError(err, http.StatusForbidden, ForbiddenCode, "You don't have permission to access this resource")
}

// 404 Not Found
func ErrEntityNotFound(err error) Error {
	return NewError(err, http.StatusNotFound, EntityNotFoundCode, "No such file or directory")
}
