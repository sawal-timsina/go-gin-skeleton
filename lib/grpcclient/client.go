// Package grpcclient provides a factory for dialling peer microservices over
// gRPC. Endpoints are resolved from the SERVICE_ENDPOINTS registry (see
// config.Env.Endpoints), connections are shared and lazily created, and every
// client is instrumented with OpenTelemetry plus a conservative retry policy.
package grpcclient

import (
	"context"
	"fmt"
	"sync"

	"boilerplate-api/lib/config"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// retryPolicy retries idempotent-safe UNAVAILABLE failures with capped
// exponential backoff. It applies to all methods ("name":[{}]); tighten per
// method in real services that have non-retryable RPCs.
const retryPolicy = `{
  "methodConfig": [{
    "name": [{}],
    "retryPolicy": {
      "maxAttempts": 3,
      "initialBackoff": "0.1s",
      "maxBackoff": "1s",
      "backoffMultiplier": 2.0,
      "retryableStatusCodes": ["UNAVAILABLE"]
    }
  }]
}`

// Factory hands out shared *grpc.ClientConn instances keyed by service name.
type Factory struct {
	env    config.Env
	logger config.Logger

	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
}

// NewFactory constructs the factory and registers connection cleanup on stop.
func NewFactory(lc fx.Lifecycle, env config.Env, logger config.Logger) *Factory {
	f := &Factory{
		env:    env,
		logger: logger,
		conns:  make(map[string]*grpc.ClientConn),
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			for name, c := range f.conns {
				if err := c.Close(); err != nil {
					f.logger.Warn("grpcclient: closing ", name, ": ", err.Error())
				}
			}
			return nil
		},
	})
	return f
}

// Dial returns a shared connection to the named peer service, creating it on
// first use. The address comes from SERVICE_ENDPOINTS (name=host:port). The
// connection is lazy — grpc.NewClient does not block on a live server — so a
// missing peer only surfaces when an RPC is actually made.
func (f *Factory) Dial(service string) (*grpc.ClientConn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if c, ok := f.conns[service]; ok {
		return c, nil
	}

	addr, ok := f.env.Endpoints[service]
	if !ok {
		return nil, fmt.Errorf("grpcclient: no endpoint for %q (add it to SERVICE_ENDPOINTS)", service)
	}

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: dial %q at %s: %w", service, addr, err)
	}

	f.conns[service] = conn
	f.logger.Info("grpcclient: connected to ", service, " at ", addr)
	return conn, nil
}
