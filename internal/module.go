package lib

import (
	"boilerplate-api/internal/auth"
	"boilerplate-api/internal/config"
	"boilerplate-api/internal/idempotency"
	"boilerplate-api/internal/middlewares"
	"boilerplate-api/internal/request_validator"
	"boilerplate-api/internal/router"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"internal",
	config.Module,
	middlewares.Module,
	idempotency.Module,
	fx.Options(
		fx.Provide(
			router.NewRouter,
			request_validator.NewValidator,
			auth.NewJWTAuthService,
		),
	),
)
