package apperror

import (
	"net/http"
)

const (
	BindingCode                 = "400001"
	ValidationCode              = "400002"
	InvalidCredentialsCode      = "400003"
	IncorrectPasswordCode       = "400007"
	InvalidPasswordCode         = "400009"
	InvalidImageCode            = "400010"
	DirAlreadyExistsCode        = "400011"
	StorageCapacityExceededCode = "400012"
	FileOnlyOperationCode       = "400013"
	DirectoryOnlyOperationCode  = "400014"
	UnauthorizedCode            = "401004"
	IdentityWasDisableCode      = "401009"
	ForbiddenCode               = "403005"
	RefreshTokenRequiredCode    = "403008"
	EntityNotFoundCode          = "404006"
	IdentityNotFoundCode        = "404007"
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

func ErrInvalidImage(err error) Error {
	return NewError(err, http.StatusBadRequest, InvalidImageCode, "Invalid image")
}

func ErrDirAlreadyExists(err error) Error {
	return NewError(err, http.StatusBadRequest, DirAlreadyExistsCode, "Directory already exists")
}

// 401 Unauthorized
func ErrInvalidCredentials(err error) Error {
	return NewError(err, http.StatusUnauthorized, InvalidCredentialsCode, "Invalid credentials")
}

func ErrUnauthorized(err error) Error {
	return NewError(err, http.StatusUnauthorized, UnauthorizedCode, "Unauthorized")
}

func ErrIdentityWasDisabled(err error) Error {
	return NewError(err, http.StatusUnauthorized, IdentityWasDisableCode, "Identity was disabled")
}

func ErrStorageCapacityExceeded() Error {
	return NewError(nil, http.StatusBadRequest, StorageCapacityExceededCode, "Storage capacity exceeded")
}

func ErrFileOnlyOperation() Error {
	return NewError(nil, http.StatusBadRequest, FileOnlyOperationCode, "This operation is only allowed for files")
}

func ErrDirectoryOnlyOperation() Error {
	return NewError(nil, http.StatusBadRequest, DirectoryOnlyOperationCode, "This operation is only allowed for directories")
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
