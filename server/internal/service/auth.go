package service

import (
	"context"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

func (s *AuthService) Register(ctx context.Context, req types.RegisterRequest) (types.TokenResponse, error) {
	return types.TokenResponse{}, apperror.NotImplemented("register is not implemented")
}

func (s *AuthService) Login(ctx context.Context, req types.LoginRequest) (types.TokenResponse, error) {
	return types.TokenResponse{}, apperror.NotImplemented("login is not implemented")
}
