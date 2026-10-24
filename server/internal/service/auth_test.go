package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

const testSecret = "test-secret-at-least-32-bytes-long!!"

type fakeWorkspaceStore struct {
	byName    map[string]model.Workspace
	createErr error
}

func newFakeWorkspaceStore() *fakeWorkspaceStore {
	return &fakeWorkspaceStore{byName: map[string]model.Workspace{}}
}

func (f *fakeWorkspaceStore) Create(ctx context.Context, ws model.Workspace) (model.Workspace, error) {
	if f.createErr != nil {
		return model.Workspace{}, f.createErr
	}
	if _, ok := f.byName[ws.Name]; ok {
		return model.Workspace{}, repo.ErrAlreadyExists
	}
	ws.ID = "ws-" + ws.Name
	f.byName[ws.Name] = ws
	return ws, nil
}

func (f *fakeWorkspaceStore) GetByName(ctx context.Context, name string) (model.Workspace, error) {
	ws, ok := f.byName[name]
	if !ok {
		return model.Workspace{}, repo.ErrNotFound
	}
	return ws, nil
}

func registerRequest(name, password string) types.RegisterRequest {
	return types.RegisterRequest{
		Name:           name,
		Password:       password,
		ConnectionType: model.ConnectionTypeTemporaryCredentials,
		AWSRegion:      "ap-south-1",
	}
}

func TestRegister(t *testing.T) {
	store := newFakeWorkspaceStore()
	svc := NewAuthService(store, testSecret, time.Hour, false)

	resp, err := svc.Register(context.Background(), registerRequest("Acme", "correct-horse"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	stored, ok := store.byName["acme"]
	if !ok {
		t.Fatal("workspace not stored under lowercased name")
	}
	if stored.PasswordHash == "correct-horse" {
		t.Fatal("password stored in plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("correct-horse")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}

	workspaceID, err := svc.VerifyToken(resp.Token)
	if err != nil {
		t.Fatalf("verify issued token: %v", err)
	}
	if workspaceID != stored.ID {
		t.Fatalf("token subject = %q, want %q", workspaceID, stored.ID)
	}
	if !resp.ExpiresAt.After(time.Now()) {
		t.Fatalf("expires_at %v is not in the future", resp.ExpiresAt)
	}
}

func TestRegisterErrors(t *testing.T) {
	storeErr := errors.New("connection refused")

	tests := []struct {
		name     string
		setup    func(store *fakeWorkspaceStore, svc *AuthService)
		wantCode constants.QueryErrCode
		wantErr  error
	}{
		{
			name: "name taken, case-insensitive",
			setup: func(store *fakeWorkspaceStore, svc *AuthService) {
				if _, err := svc.Register(context.Background(), registerRequest("acme", "correct-horse")); err != nil {
					t.Fatalf("seed register: %v", err)
				}
			},
			wantCode: constants.ErrCodeConflict,
		},
		{
			name:    "store failure passes through",
			setup:   func(store *fakeWorkspaceStore, svc *AuthService) { store.createErr = storeErr },
			wantErr: storeErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeWorkspaceStore()
			svc := NewAuthService(store, testSecret, time.Hour, false)
			tc.setup(store, svc)

			_, err := svc.Register(context.Background(), registerRequest("ACME", "correct-horse"))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			assertAppErrorCode(t, err, tc.wantCode)
		})
	}
}

func TestRegisterLocalConnection(t *testing.T) {
	req := registerRequest("acme", "correct-horse")
	req.ConnectionType = model.ConnectionTypeLocal
	endpoint := "http://localhost:8081"
	req.EndpointURL = &endpoint

	t.Run("rejected outside development", func(t *testing.T) {
		svc := NewAuthService(newFakeWorkspaceStore(), testSecret, time.Hour, false)
		_, err := svc.Register(context.Background(), req)
		assertAppErrorCode(t, err, constants.ErrCodeBadRequest)
	})

	t.Run("allowed in development", func(t *testing.T) {
		svc := NewAuthService(newFakeWorkspaceStore(), testSecret, time.Hour, true)
		if _, err := svc.Register(context.Background(), req); err != nil {
			t.Fatalf("register: %v", err)
		}
	})
}

func TestLogin(t *testing.T) {
	store := newFakeWorkspaceStore()
	svc := NewAuthService(store, testSecret, time.Hour, false)
	if _, err := svc.Register(context.Background(), registerRequest("acme", "correct-horse")); err != nil {
		t.Fatalf("seed register: %v", err)
	}

	tests := []struct {
		name     string
		req      types.LoginRequest
		wantCode constants.QueryErrCode
	}{
		{name: "valid, name case-insensitive", req: types.LoginRequest{Name: "ACME", Password: "correct-horse"}},
		{
			name:     "wrong password",
			req:      types.LoginRequest{Name: "acme", Password: "wrong-horse"},
			wantCode: constants.ErrCodeUnauthorized,
		},
		{
			name:     "unknown name",
			req:      types.LoginRequest{Name: "globex", Password: "correct-horse"},
			wantCode: constants.ErrCodeUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.Login(context.Background(), tc.req)
			if tc.wantCode != "" {
				assertAppErrorCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("login: %v", err)
			}
			workspaceID, err := svc.VerifyToken(resp.Token)
			if err != nil || workspaceID != "ws-acme" {
				t.Fatalf("VerifyToken = %q, %v; want ws-acme", workspaceID, err)
			}
		})
	}
}

func TestVerifyTokenRejects(t *testing.T) {
	svc := NewAuthService(newFakeWorkspaceStore(), testSecret, time.Hour, false)

	expired, err := NewAuthService(newFakeWorkspaceStore(), testSecret, -time.Minute, false).issueToken("ws-acme")
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}
	otherSecret, err := NewAuthService(newFakeWorkspaceStore(), "another-secret-also-32-bytes-long!", time.Hour, false).
		issueToken("ws-acme")
	if err != nil {
		t.Fatalf("issue token with other secret: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "garbage", token: "not-a-jwt"},
		{name: "expired", token: expired.Token},
		{name: "signed with another secret", token: otherSecret.Token},
		{name: "unsigned alg none", token: signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, "ws-acme")},
		{name: "HS512 instead of HS256", token: signToken(t, jwt.SigningMethodHS512, []byte(testSecret), "ws-acme")},
		{name: "missing subject", token: signToken(t, jwt.SigningMethodHS256, []byte(testSecret), "")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.VerifyToken(tc.token)
			assertAppErrorCode(t, err, constants.ErrCodeUnauthorized)
		})
	}
}

func TestDummyHashIsValidBcrypt(t *testing.T) {
	if _, err := bcrypt.Cost(dummyPasswordHash); err != nil {
		t.Fatalf("dummyPasswordHash is not a bcrypt hash: %v", err)
	}
}

func signToken(t *testing.T, method jwt.SigningMethod, key any, subject string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{Subject: subject, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return token
}

func assertAppErrorCode(t *testing.T, err error, want constants.QueryErrCode) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != want {
		t.Fatalf("err = %v, want AppError %s", err, want)
	}
}
