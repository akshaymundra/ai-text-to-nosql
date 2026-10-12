package apperror

import (
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
)

type AppError struct {
	Code       constants.QueryErrCode `json:"code"`
	Message    string                 `json:"message"`
	HTTPStatus int                    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeBadRequest, Message: msg, HTTPStatus: http.StatusBadRequest}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeUnauthorized, Message: msg, HTTPStatus: http.StatusUnauthorized}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeForbidden, Message: msg, HTTPStatus: http.StatusForbidden}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeNotFound, Message: msg, HTTPStatus: http.StatusNotFound}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeConflict, Message: msg, HTTPStatus: http.StatusConflict}
}

func Internal(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeInternalServerError, Message: msg, HTTPStatus: http.StatusInternalServerError}
}

func NotImplemented(msg string) *AppError {
	return &AppError{Code: constants.ErrCodeNotImplemented, Message: msg, HTTPStatus: http.StatusNotImplemented}
}
