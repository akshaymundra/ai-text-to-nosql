package types

import (
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type UpdateConnectionRequest struct {
	ConnectionType model.ConnectionType `json:"connection_type"           validate:"required,oneof=temporary_credentials local"`
	AWSRegion      string               `json:"aws_region"                validate:"required"`
	EndpointURL    *string              `json:"endpoint_url,omitempty"    validate:"required_if=ConnectionType local,excluded_if=ConnectionType temporary_credentials,omitnil,url,local_url"`
	AWSCredentials *AWSCredentials      `json:"aws_credentials,omitempty" validate:"required_if=ConnectionType temporary_credentials,excluded_if=ConnectionType local"`
}

func (r UpdateConnectionRequest) Validate() error {
	return validateStruct(r)
}

type AWSCredentials struct {
	AccessKeyID     string    `json:"access_key_id"     validate:"required"`
	SecretAccessKey string    `json:"secret_access_key" validate:"required"`
	SessionToken    string    `json:"session_token"     validate:"required"`
	Expiration      time.Time `json:"expiration"        validate:"required"`
}
