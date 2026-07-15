package telemetry

import "go.uber.org/fx"

// Module provides the tracer and metrics collectors to the fx container.
var Module = fx.Options(
	fx.Provide(NewTracer),
	fx.Provide(NewMetrics),
)
