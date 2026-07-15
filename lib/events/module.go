package events

import (
	"context"

	"boilerplate-api/lib/config"

	"go.uber.org/fx"
)

// Module provides a Broker, selecting the NATS backend when NATS_URL is set
// and the no-op backend otherwise. The broker is closed on shutdown.
var Module = fx.Options(
	fx.Provide(NewBroker),
)

// NewBroker chooses the concrete Broker from configuration and registers its
// lifecycle so the connection drains cleanly on shutdown.
func NewBroker(lc fx.Lifecycle, env config.Env, logger config.Logger) (Broker, error) {
	var (
		broker Broker
		err    error
	)
	if env.NatsURL == "" {
		broker = NewNoopBroker(logger)
	} else {
		broker, err = NewNatsBroker(env, logger)
		if err != nil {
			return nil, err
		}
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return broker.Close()
		},
	})
	return broker, nil
}
