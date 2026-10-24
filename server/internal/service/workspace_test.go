package service

import (
	"context"
	"testing"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

func TestUpdateConnectionLocalConnection(t *testing.T) {
	endpoint := "http://localhost:8081"
	req := types.UpdateConnectionRequest{
		ConnectionType: model.ConnectionTypeLocal,
		AWSRegion:      "us-east-1",
		EndpointURL:    &endpoint,
	}

	tests := []struct {
		name                 string
		allowLocalConnection bool
		wantCode             constants.QueryErrCode
	}{
		{name: "rejected outside development", allowLocalConnection: false, wantCode: constants.ErrCodeBadRequest},
		{name: "allowed in development", allowLocalConnection: true, wantCode: constants.ErrCodeNotImplemented},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewWorkspaceService(tc.allowLocalConnection)
			_, err := svc.UpdateConnection(context.Background(), "ws-acme", req)
			assertAppErrorCode(t, err, tc.wantCode)
		})
	}
}
