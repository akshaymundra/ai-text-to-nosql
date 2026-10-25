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
	RejectCodeInvalidRange         RejectionCode = "INVALID_RANGE"
	RejectCodeInvalidDateFormat    RejectionCode = "INVALID_DATE_FORMAT"

	RejectCodeNoSchemas        RejectionCode = "NO_SCHEMAS"
	RejectCodeExtractionFailed RejectionCode = "EXTRACTION_FAILED"

	RejectCodeScanRequired          RejectionCode = "SCAN_REQUIRED"
	RejectCodeInvalidKeyCondition   RejectionCode = "INVALID_KEY_CONDITION"
	RejectCodeMultipleKeyConditions RejectionCode = "MULTIPLE_KEY_CONDITIONS"
	RejectCodeEmptyKeyValue         RejectionCode = "EMPTY_KEY_VALUE"
)
