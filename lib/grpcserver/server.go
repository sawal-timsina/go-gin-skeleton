// Package grpcserver provides a shared gRPC server wired with tracing, panic
// recovery, health checking and reflection. Feature modules register their
// service handlers onto the *Server via fx.Invoke; the server is started and
// gracefully stopped by the fx lifecycle (see Module).
package grpcserver

import (
	"context"

	"boilerplate-api/lib/config"
	"boilerplate-api/lib/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server wraps *grpc.Server plus the health service so the lifecycle can flip
// serving status during startup/shutdown.
type Server struct {
	*grpc.Server
	Health *health.Server
}

// NewServer builds the gRPC server. The telemetry.Tracer dependency is taken
// (though unused directly) to guarantee the global OTel propagator/provider is
// installed before otelgrpc's handler reads it.
func NewServer(logger config.Logger, _ telemetry.Tracer) *Server {
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(recoveryUnaryInterceptor(logger)),
	)

	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, hs)
	reflection.Register(srv)

	return &Server{Server: srv, Health: hs}
}

// recoveryUnaryInterceptor converts a handler panic into an Internal error so
// one bad request cannot crash the process.
func recoveryUnaryInterceptor(logger config.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("grpc panic recovered in ", info.FullMethod, ": ", r)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}
