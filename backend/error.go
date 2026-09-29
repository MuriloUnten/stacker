package main

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorKind string
const (
	Invalid         ErrorKind = "invalid"
	IO              ErrorKind = "io"
	Validation      ErrorKind = "validation"
	Exist           ErrorKind = "exist"
	NotExist        ErrorKind = "not_exist"
	Internal        ErrorKind = "internal"
	Database        ErrorKind = "database"
	// TODO add this back whenever authentication is implemented
	// Unauthenticated ErrorKind = "unauthenticated"
	// Forbidden       ErrorKind = "forbidden"
)

type AppError struct {
	Kind  ErrorKind
	Field string
	Inner error
}

func (e AppError) Error() string {
	return e.Inner.Error()
}

func NewAppError(kind ErrorKind, inner error) AppError {
	return AppError{
		Kind:  kind,
		Inner: inner,
	}
}

func NewAppErrorWithField(kind ErrorKind, inner error, field string) AppError {
	return AppError{
		Kind:  kind,
		Inner: inner,
		Field: field,
	}
}

func newErrorResponse(err *AppError) ErrorResponse {
	const msg string = "internal server error"

	switch err.Kind {
	case IO, Internal, Database:
		return ErrorResponse{
			Error: APIError{
				Kind:    Internal,
				Message: msg,
			},
		}
	default:
		return ErrorResponse{
			Error: APIError{
				Kind:    err.Kind,
				Field:   err.Field,
				Message: err.Error(),
			},
		}
	}
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Kind    ErrorKind `json:"kind"`
	Message string    `json:"msg"`
	// NOTE: (Murilo) AFAIK Field is only used when Kind == Validation
	Field   string    `json:"field,omitempty"`
}

func (e APIError) Error() string {
	if e.Kind == Validation {
		return fmt.Sprintf("api error (%s): %s - %s", e.Kind, e.Field, e.Message)
	}
	return fmt.Sprintf("api error (%s): %s", e.Kind, e.Message)
}

func httpStatusCodeFromErrorKind(kind ErrorKind) int {
	switch kind {
	case Invalid, Validation:
		return http.StatusBadRequest
	case NotExist:
		return http.StatusNotFound
	case Exist:
		return http.StatusConflict
	case IO, Internal, Database:
		return http.StatusInternalServerError
	// TODO add these when implementing authentication
	// case Unauthenticated:
	// 	return http.StatusUnauthorized,
	// case Forbidden:
	// 	return http.StatusForbidden,
	default:
		return http.StatusInternalServerError
	}
}

func NewAPIError(err AppError) APIError {
	switch err.Kind {
	case IO, Internal, Database:
		return APIError{
			Kind:    err.Kind,
			Message: "internal server error",
		}
	default:
		return APIError {
			Kind:    err.Kind,
			Message: err.Error(),
			Field:   err.Field,
		}
	}
}

func InternalError(err error) AppError {
	return NewAppError(Internal, err)
}

// TODO review parameter type. This was the old one
// func InvalidJSONRequestData(errors map[string]string) APIError {
func InvalidJSONRequestData(field string, msg string) AppError {
	return NewAppErrorWithField(Validation, errors.New(msg), field)
}

func NotImplemented() AppError {
	return NewAppError(Invalid, errors.New("endpoint not implemented"))
}
