package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"boilerplate-api/lib/config"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// natsBroker is a JetStream-backed Broker providing durable, at-least-once
// delivery.
type natsBroker struct {
	nc            *nats.Conn
	js            nats.JetStreamContext
	logger        config.Logger
	source        string // service name, stamped on every event
	subjectPrefix string
}

// NewNatsBroker connects to NATS, ensures the events stream exists, and
// returns a JetStream-backed Broker.
func NewNatsBroker(env config.Env, logger config.Logger) (Broker, error) {
	nc, err := nats.Connect(
		env.NatsURL,
		nats.Name(env.ServiceName),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("events: connect nats: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("events: jetstream context: %w", err)
	}

	// Ensure a stream capturing every subject under the prefix exists. Created
	// once; subsequent boots find it and move on.
	streamSubjects := env.EventSubjectPrefix + ".>"
	if _, err := js.StreamInfo(env.NatsStreamName); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name:      env.NatsStreamName,
			Subjects:  []string{streamSubjects},
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
			MaxAge:    7 * 24 * time.Hour,
		}); err != nil {
			nc.Close()
			return nil, fmt.Errorf("events: create stream %q: %w", env.NatsStreamName, err)
		}
		logger.Info("events: created JetStream stream ", env.NatsStreamName, " for ", streamSubjects)
	}

	logger.Info("events: connected to NATS at ", env.NatsURL)
	return &natsBroker{
		nc:            nc,
		js:            js,
		logger:        logger,
		source:        env.ServiceName,
		subjectPrefix: env.EventSubjectPrefix,
	}, nil
}

func (b *natsBroker) Publish(ctx context.Context, eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("events: marshal payload: %w", err)
	}

	// Carry the active trace so consumers continue the same distributed trace.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	e := Event{
		ID:          uuid.NewString(),
		Type:        eventType,
		Source:      b.source,
		Time:        time.Now().UTC(),
		TraceParent: carrier["traceparent"],
		Data:        payload,
	}
	envelope, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("events: marshal envelope: %w", err)
	}

	subject := b.subjectPrefix + "." + eventType
	if _, err := b.js.Publish(subject, envelope, nats.Context(ctx)); err != nil {
		return fmt.Errorf("events: publish %q: %w", subject, err)
	}
	return nil
}

func (b *natsBroker) Subscribe(ctx context.Context, subjectPattern, durable string, handler Handler) (Subscription, error) {
	cb := func(msg *nats.Msg) {
		var e Event
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			b.logger.Error("events: undecodable message on ", subjectPattern, ": ", err.Error())
			// Poison message — terminate so it is not redelivered forever.
			_ = msg.Term()
			return
		}

		// Rebuild the producer's trace context for the handler.
		msgCtx := ctx
		if e.TraceParent != "" {
			msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": e.TraceParent})
		}

		if err := handler(msgCtx, e); err != nil {
			b.logger.Error("events: handler failed for ", e.Type, " (", e.ID, "): ", err.Error())
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	}

	// Durable push consumer as a queue group so replicas share the load and
	// redeliveries survive restarts.
	sub, err := b.js.QueueSubscribe(
		subjectPattern,
		durable,
		cb,
		nats.Durable(durable),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.DeliverAll(),
	)
	if err != nil {
		return nil, fmt.Errorf("events: subscribe %q: %w", subjectPattern, err)
	}
	b.logger.Info("events: subscribed to ", subjectPattern, " (durable ", durable, ")")
	return &natsSubscription{sub: sub}, nil
}

func (b *natsBroker) Close() error {
	if err := b.nc.Drain(); err != nil {
		return err
	}
	return nil
}

type natsSubscription struct {
	sub *nats.Subscription
}

func (s *natsSubscription) Close() error {
	return s.sub.Drain()
}
