package grpcclient

import "go.uber.org/fx"

// Module provides the gRPC client Factory to the fx container.
var Module = fx.Options(
	fx.Provide(NewFactory),
)
