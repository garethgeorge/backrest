package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/auth"
	"github.com/garethgeorge/backrest/internal/config"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthenticationHandler struct {
	// v1connect.UnimplementedAuthenticationHandler
	authenticator *auth.Authenticator
	config        config.ConfigStore
}

var _ v1connect.AuthenticationHandler = &AuthenticationHandler{}

func NewAuthenticationHandler(authenticator *auth.Authenticator, config config.ConfigStore) *AuthenticationHandler {
	return &AuthenticationHandler{
		authenticator: authenticator,
		config:        config,
	}
}

func (s *AuthenticationHandler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	zap.L().Debug("login request", zap.String("username", req.Msg.Username))
	user, err := s.authenticator.Login(req.Msg.Username, req.Msg.Password)
	if err != nil {
		zap.L().Warn("failed login attempt", zap.Error(err))
		return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrInvalidPassword)
	}

	token, err := s.authenticator.CreateJWT(user)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.LoginResponse{
		Token: token,
	}), nil
}

func (s *AuthenticationHandler) GetAuthInfo(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.AuthInfo], error) {
	cfg, err := s.config.Get()
	if err != nil {
		return nil, err
	}
	driver := config.AuthDriverOf(cfg.GetAuth())
	info := &v1.AuthInfo{AuthDriver: driver}
	if driver == config.AuthDriverOIDC {
		info.OidcLoginUrl = "/auth/oidc/login"
	}
	return connect.NewResponse(info), nil
}

func (s *AuthenticationHandler) HashPassword(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[types.StringValue], error) {
	hash, err := auth.CreatePassword(req.Msg.Value)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&types.StringValue{Value: hash}), nil
}
