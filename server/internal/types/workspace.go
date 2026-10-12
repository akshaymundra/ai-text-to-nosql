package types

import (
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type UpdateConnectionRequest struct {
	ConnectionType model.ConnectionType `json:"connection_type"`
	AWSRegion      string               `json:"aws_region"`
	EndpointURL    *string              `json:"endpoint_url,omitempty"`
	AWSCredentials *AWSCredentials      `json:"aws_credentials,omitempty"`
}

type AWSCredentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token"`
	Expiration      time.Time `json:"expiration"`
}
