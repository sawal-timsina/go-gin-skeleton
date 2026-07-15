package user

import (
	"context"

	"boilerplate-api/lib/config"
	"boilerplate-api/lib/events"

	"go.uber.org/fx"
)

// RegisterSubscribers wires event consumers for the user module. In a real
// deployment a *peer* service would subscribe to another service's events;
// here the service consumes its own user.created event to demonstrate the
// end-to-end publish → deliver round trip with trace propagation.
//
// With the no-op broker (NATS_URL unset) Subscribe is inert, so this is safe
// to leave registered in every environment.
func RegisterSubscribers(
	lc fx.Lifecycle,
	broker events.Broker,
	logger config.Logger,
	env config.Env,
) {
	var sub events.Subscription

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			pattern := env.EventSubjectPrefix + "." + EventUserCreated
			s, err := broker.Subscribe(
				context.Background(),
				pattern,
				env.ServiceName+"-user-created",
				func(ctx context.Context, e events.Event) error {
					var payload UserCreatedEvent
					if err := e.UnmarshalData(&payload); err != nil {
						return err
					}
					logger.Info("event received: ", e.Type, " id=", e.ID, " email=", payload.Email)
					return nil
				},
			)
			if err != nil {
				return err
			}
			sub = s
			return nil
		},
		OnStop: func(context.Context) error {
			if sub != nil {
				return sub.Close()
			}
			return nil
		},
	})
}
