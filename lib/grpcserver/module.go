package grpcserver

import (
	"context"
	"net"

	"boilerplate-api/lib/config"
	"boilerplate-api/lib/utils"

	"go.uber.org/fx"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Module provides the gRPC server and registers its serve/shutdown lifecycle.
//
// Ordering note: fx runs all fx.Provide/fx.Invoke (where feature modules
// attach their handlers) before any OnStart hook fires, so every handler is
// registered by the time Serve is called below.
var Module = fx.Options(
	fx.Provide(NewServer),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, srv *Server, env config.Env, logger config.Logger) {
	if utils.IsCli() {
		return
	}
	if env.GRPCPort == "" {
		logger.Info("gRPC server disabled (GRPC_PORT empty)")
		return
	}

	var lis net.Listener
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			var err error
			lis, err = net.Listen("tcp", ":"+env.GRPCPort)
			if err != nil {
				return err
			}
			srv.Health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
			logger.Info("gRPC server listening on :", env.GRPCPort)
			go func() {
				if err := srv.Serve(lis); err != nil {
					logger.Error("gRPC serve error: ", err.Error())
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Shutting down gRPC server...")
			srv.Health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			srv.GracefulStop()
			return nil
		},
	})
}
