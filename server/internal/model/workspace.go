package model

import "time"

type ConnectionType string

const (
	ConnectionTypeTemporaryCredentials ConnectionType = "temporary_credentials"
	ConnectionTypeLocal                ConnectionType = "local"
)

type Workspace struct {
	ID                      string         `db:"id"                         json:"id"`
	Name                    string         `db:"name"                       json:"name"`
	PasswordHash            string         `db:"password_hash"              json:"-"`
	ConnectionType          ConnectionType `db:"connection_type"            json:"connection_type"`
	AWSRegion               string         `db:"aws_region"                 json:"aws_region"`
	EndpointURL             *string        `db:"endpoint_url"               json:"endpoint_url"`
	AWSCredentialsEnc       []byte         `db:"aws_credentials_enc"        json:"-"`
	AWSCredentialsExpiresAt *time.Time     `db:"aws_credentials_expires_at" json:"aws_credentials_expires_at"`
	CreatedAt               time.Time      `db:"created_at"                 json:"created_at"`
	UpdatedAt               time.Time      `db:"updated_at"                 json:"updated_at"`
}
