package constants

type (
	QueryErrCode string
)

const (
	ErrCodeBadRequest          QueryErrCode = "BAD_REQUEST"
	ErrCodeUnauthorized        QueryErrCode = "UNAUTHORIZED"
	ErrCodeForbidden           QueryErrCode = "FORBIDDEN"
	ErrCodeNotFound            QueryErrCode = "NOT_FOUND"
	ErrCodeInternalServerError QueryErrCode = "INTERNAL_SERVER_ERROR"
)
