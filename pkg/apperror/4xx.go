package apperror

import (
	"net/http"
)

const (
	BindingCode              = "400001"
	ValidationCode           = "400002"
	InvalidCredentialsCode   = "400003"
	IncorrectPasswordCode    = "400007"
	InvalidPasswordCode      = "400009"
	UnauthorizedCode         = "401004"
	ForbiddenCode            = "403005"
	RefreshTokenRequiredCode = "403008"
	EntityNotFoundCode       = "404006"
	IdentityNotFoundCode     = "404007"
)

// 400 Bad Request
func ErrInvalidRequest(err error) Error {
	return NewError(err, http.StatusBadRequest, BindingCode, "Invalid request")
}

func ErrInvalidParam(err error) Error {
	return NewError(err, http.StatusBadRequest, ValidationCode, "Invalid param")
}

func ErrIncorrectPassword(err error) Error {
	return NewError(err, http.StatusBadRequest, InvalidCredentialsCode, "Incorrect old password")
}

func ErrInvalidPassword(err error) Error {
	return NewError(err, http.StatusBadRequest, InvalidPasswordCode, "Invalid new password, please use a different one")
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

func ErrSessionRefreshRequired(err error) Error {
	return NewError(err, http.StatusForbidden, RefreshTokenRequiredCode, "The login session is too old and thus not allowed to update these fields. Please re-authenticate.")
}

// 404 Not Found
func ErrEntityNotFound(err error) Error {
	return NewError(err, http.StatusNotFound, EntityNotFoundCode, "No such file or directory")
}

func ErrIdentityNotFound(err error) Error {
	return NewError(err, http.StatusNotFound, IdentityNotFoundCode, "Identity not found")
}
