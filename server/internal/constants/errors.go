package constants

type (
	QueryErrCode string
)

const (
	ErrCodeBadRequest          QueryErrCode = "BAD_REQUEST"
	ErrCodeUnauthorized        QueryErrCode = "UNAUTHORIZED"
	ErrCodeForbidden           QueryErrCode = "FORBIDDEN"
	ErrCodeNotFound            QueryErrCode = "NOT_FOUND"
	ErrCodeConflict            QueryErrCode = "CONFLICT"
	ErrCodeInternalServerError QueryErrCode = "INTERNAL_SERVER_ERROR"
	ErrCodeNotImplemented      QueryErrCode = "NOT_IMPLEMENTED"
)
