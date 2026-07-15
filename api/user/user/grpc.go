package user

import (
	"context"
	"strconv"

	"boilerplate-api/lib/config"
	"boilerplate-api/lib/grpcserver"
	userv1 "boilerplate-api/proto/gen/go/user/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler serves the UserService gRPC contract by reusing the same
// domain Service that backs the HTTP handlers — one source of truth, two
// transports.
type GRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	logger  config.Logger
	service Service
}

// NewGRPCHandler builds the gRPC handler.
func NewGRPCHandler(logger config.Logger, service Service) *GRPCHandler {
	return &GRPCHandler{logger: logger, service: service}
}

// GetUser implements userv1.UserServiceServer.
func (h *GRPCHandler) GetUser(_ context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	u, err := h.service.GetOneUser(req.GetId())
	if err != nil {
		h.logger.Error("grpc GetUser: ", err.Error())
		return nil, status.Errorf(codes.NotFound, "user %s not found", req.GetId())
	}

	return &userv1.GetUserResponse{
		User: &userv1.User{
			Id:       strconv.FormatUint(uint64(u.ID), 10),
			Email:    u.Email,
			FullName: u.FullName,
		},
	}, nil
}

// RegisterGRPC attaches the handler to the shared gRPC server. Wired via
// fx.Invoke in the module.
func RegisterGRPC(srv *grpcserver.Server, h *GRPCHandler) {
	userv1.RegisterUserServiceServer(srv.Server, h)
}
