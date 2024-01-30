package api

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/auth"
	"go.uber.org/zap"
)

type AuthenticationHandler struct {
	// v1connect.UnimplementedAuthenticationHandler
	authenticator *auth.Authenticator
}

var _ v1connect.AuthenticationHandler = &AuthenticationHandler{}

func NewAuthenticationHandler(authenticator *auth.Authenticator) *AuthenticationHandler {
	return &AuthenticationHandler{
		authenticator: authenticator,
	}
}

func (s *AuthenticationHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	zap.L().Debug("login request", zap.String("username", req.Msg.Username))
	user, err := s.authenticator.Login(req.Msg.Username, req.Msg.Password)
	if err != nil {
		zap.L().Warn("failed login attempt", zap.Error(err))
		return nil, auth.ErrInvalidPassword
	}

	token, err := s.authenticator.CreateJWT(user)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.LoginResponse{
		Token: token,
	}), nil
}
