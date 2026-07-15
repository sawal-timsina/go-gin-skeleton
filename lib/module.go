package lib

import (
	"boilerplate-api/lib/auth"
	"boilerplate-api/lib/config"
	"boilerplate-api/lib/events"
	"boilerplate-api/lib/grpcclient"
	"boilerplate-api/lib/grpcserver"
	"boilerplate-api/lib/idempotency"
	"boilerplate-api/lib/middlewares"
	"boilerplate-api/lib/request_validator"
	"boilerplate-api/lib/router"
	"boilerplate-api/lib/telemetry"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"internal",
	config.Module,
	middlewares.Module,
	idempotency.Module,
	telemetry.Module,
	events.Module,
	grpcserver.Module,
	grpcclient.Module,
	fx.Options(
		fx.Provide(
			router.NewRouter,
			request_validator.NewValidator,
			auth.NewJWTAuthService,
		),
	),
)
