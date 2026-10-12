package types

import (
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type RegisterRequest struct {
	Name           string               `json:"name"                   validate:"workspace_name"`
	Password       string               `json:"password"               validate:"min=8,maxbytes=72"`
	ConnectionType model.ConnectionType `json:"connection_type"        validate:"required,oneof=temporary_credentials local"`
	AWSRegion      string               `json:"aws_region"             validate:"required"`
	EndpointURL    *string              `json:"endpoint_url,omitempty" validate:"required_if=ConnectionType local,excluded_if=ConnectionType temporary_credentials,omitnil,url"`
}

func (r RegisterRequest) Validate() error {
	return validateStruct(r)
}

type LoginRequest struct {
	Name     string `json:"name"     validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (r LoginRequest) Validate() error {
	return validateStruct(r)
}

type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
