package types

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

func validRegister() RegisterRequest {
	return RegisterRequest{
		Name:           "acme-prod",
		Password:       "correct-horse",
		ConnectionType: model.ConnectionTypeTemporaryCredentials,
		AWSRegion:      "ap-south-1",
	}
}

func strPtr(s string) *string {
	return &s
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(r *RegisterRequest)
		wantErr bool
	}{
		{name: "valid temporary credentials", mutate: func(r *RegisterRequest) {}},
		{name: "uppercase name allowed", mutate: func(r *RegisterRequest) { r.Name = "Acme" }},
		{
			name: "valid local with endpoint",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("http://localhost:8081")
			},
		},
		{name: "empty name", mutate: func(r *RegisterRequest) { r.Name = "" }, wantErr: true},
		{name: "name too short", mutate: func(r *RegisterRequest) { r.Name = "ab" }, wantErr: true},
		{name: "name too long", mutate: func(r *RegisterRequest) { r.Name = strings.Repeat("a", 64) }, wantErr: true},
		{name: "name starts with hyphen", mutate: func(r *RegisterRequest) { r.Name = "-acme" }, wantErr: true},
		{name: "name with space", mutate: func(r *RegisterRequest) { r.Name = "acme prod" }, wantErr: true},
		{name: "password too short", mutate: func(r *RegisterRequest) { r.Password = "short" }, wantErr: true},
		{
			name:    "password over bcrypt limit",
			mutate:  func(r *RegisterRequest) { r.Password = strings.Repeat("p", 73) },
			wantErr: true,
		},
		{
			name:    "multi-byte password over 72 bytes",
			mutate:  func(r *RegisterRequest) { r.Password = strings.Repeat("é", 40) },
			wantErr: true,
		},
		{
			name: "local with invalid endpoint url",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("not a url")
			},
			wantErr: true,
		},
		{
			name: "local with loopback IPv4 endpoint",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("http://127.0.0.1:8081")
			},
		},
		{
			name: "local with loopback IPv6 endpoint",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("http://[::1]:8081")
			},
		},
		{
			name: "local with cloud metadata endpoint",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("http://169.254.169.254/latest/meta-data")
			},
			wantErr: true,
		},
		{
			name: "local with host that only starts with localhost",
			mutate: func(r *RegisterRequest) {
				r.ConnectionType = model.ConnectionTypeLocal
				r.EndpointURL = strPtr("http://localhost.evil.com:8081")
			},
			wantErr: true,
		},
		{name: "unknown connection type", mutate: func(r *RegisterRequest) { r.ConnectionType = "ssh" }, wantErr: true},
		{name: "missing region", mutate: func(r *RegisterRequest) { r.AWSRegion = "" }, wantErr: true},
		{
			name:    "local without endpoint",
			mutate:  func(r *RegisterRequest) { r.ConnectionType = model.ConnectionTypeLocal },
			wantErr: true,
		},
		{
			name:    "temporary credentials with endpoint",
			mutate:  func(r *RegisterRequest) { r.EndpointURL = strPtr("http://localhost:8081") },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validRegister()
			tc.mutate(&req)

			assertBadRequest(t, req.Validate(), tc.wantErr)
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{name: "valid", req: LoginRequest{Name: "acme", Password: "x"}},
		{name: "missing name", req: LoginRequest{Password: "x"}, wantErr: true},
		{name: "missing password", req: LoginRequest{Name: "acme"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertBadRequest(t, tc.req.Validate(), tc.wantErr)
		})
	}
}

func assertBadRequest(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if !wantErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.ErrCodeBadRequest {
		t.Fatalf("want BAD_REQUEST AppError, got %v", err)
	}
}

func TestUpdateConnection(t *testing.T) {
	creds := &AWSCredentials{
		AccessKeyID:     "ASIAEXAMPLE",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(time.Hour),
	}

	tests := []struct {
		name    string
		req     UpdateConnectionRequest
		wantErr bool
	}{
		{
			name: "temporary credentials",
			req: UpdateConnectionRequest{
				ConnectionType: model.ConnectionTypeTemporaryCredentials,
				AWSRegion:      "ap-south-1",
				AWSCredentials: creds,
			},
		},
		{
			name: "temporary credentials missing",
			req: UpdateConnectionRequest{
				ConnectionType: model.ConnectionTypeTemporaryCredentials,
				AWSRegion:      "ap-south-1",
			},
			wantErr: true,
		},
		{
			name: "credentials missing session token",
			req: UpdateConnectionRequest{
				ConnectionType: model.ConnectionTypeTemporaryCredentials,
				AWSRegion:      "ap-south-1",
				AWSCredentials: &AWSCredentials{AccessKeyID: "ASIA", SecretAccessKey: "s", Expiration: time.Now()},
			},
			wantErr: true,
		},
		{
			name: "local with remote endpoint",
			req: UpdateConnectionRequest{
				ConnectionType: model.ConnectionTypeLocal,
				AWSRegion:      "us-east-1",
				EndpointURL:    strPtr("http://10.0.0.5:8081"),
			},
			wantErr: true,
		},
		{
			name: "local with credentials",
			req: UpdateConnectionRequest{
				ConnectionType: model.ConnectionTypeLocal,
				AWSRegion:      "us-east-1",
				EndpointURL:    strPtr("http://localhost:8081"),
				AWSCredentials: creds,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertBadRequest(t, tc.req.Validate(), tc.wantErr)
		})
	}
}
