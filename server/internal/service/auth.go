package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

// Compared against on unknown names so login takes as long as a wrong password and does not reveal which names exist.
var dummyPasswordHash = []byte("$2a$10$66hH8GMnMBLMFlJR1i7Ym.TXZfRk6kCWTnojXpa9kqTDvaWCWaDOa")

func (s *AuthService) Register(ctx context.Context, req types.RegisterRequest) (types.TokenResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return types.TokenResponse{}, fmt.Errorf("hashing password: %w", err)
	}

	ws, err := s.workspaces.Create(ctx, model.Workspace{
		Name:           strings.ToLower(req.Name),
		PasswordHash:   string(hash),
		ConnectionType: req.ConnectionType,
		AWSRegion:      req.AWSRegion,
		EndpointURL:    req.EndpointURL,
	})
	if errors.Is(err, repo.ErrAlreadyExists) {
		return types.TokenResponse{}, apperror.Conflict("workspace name is already taken")
	}
	if err != nil {
		return types.TokenResponse{}, err
	}

	return s.issueToken(ws.ID)
}

func (s *AuthService) Login(ctx context.Context, req types.LoginRequest) (types.TokenResponse, error) {
	ws, err := s.workspaces.GetByName(ctx, strings.ToLower(req.Name))
	if errors.Is(err, repo.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(req.Password))
		return types.TokenResponse{}, invalidCredentials()
	}
	if err != nil {
		return types.TokenResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(ws.PasswordHash), []byte(req.Password)); err != nil {
		return types.TokenResponse{}, invalidCredentials()
	}

	return s.issueToken(ws.ID)
}

func (s *AuthService) VerifyToken(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(*jwt.Token) (any, error) { return s.jwtSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return "", apperror.Unauthorized("invalid or expired token")
	}
	return claims.Subject, nil
}

func (s *AuthService) issueToken(workspaceID string) (types.TokenResponse, error) {
	now := time.Now()
	// JWT timestamps have second precision; truncate so expires_at matches the token's exp claim.
	expiresAt := now.Add(s.tokenTTL).Truncate(time.Second)

	claims := jwt.RegisteredClaims{
		Subject:   workspaceID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return types.TokenResponse{}, fmt.Errorf("signing token: %w", err)
	}

	return types.TokenResponse{Token: token, ExpiresAt: expiresAt}, nil
}

func invalidCredentials() error {
	return apperror.Unauthorized("invalid name or password")
}
