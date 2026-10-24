package constants

type (
	RejectionCode string
)

const (
	RejectCodeUnknownTable         RejectionCode = "UNKNOWN_TABLE"
	RejectCodeUnknownField         RejectionCode = "UNKNOWN_FIELD"
	RejectCodeUnknownOperator      RejectionCode = "UNKNOWN_OPERATOR"
	RejectCodeOperatorTypeMismatch RejectionCode = "OPERATOR_TYPE_MISMATCH"
	RejectCodeInvalidValue         RejectionCode = "INVALID_VALUE"
	RejectCodeTooManyValues        RejectionCode = "TOO_MANY_VALUES"
	RejectCodeValueTypeMismatch    RejectionCode = "VALUE_TYPE_MISMATCH"
)
