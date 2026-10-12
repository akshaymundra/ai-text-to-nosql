package types

import (
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type RegisterRequest struct {
	Name           string               `json:"name"`
	Password       string               `json:"password"`
	ConnectionType model.ConnectionType `json:"connection_type"`
	AWSRegion      string               `json:"aws_region"`
	EndpointURL    *string              `json:"endpoint_url,omitempty"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
