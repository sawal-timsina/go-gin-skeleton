package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Env has environment stored
type Env struct {
	HOST        string `mapstructure:"HOST"`
	TimeZone    string `mapstructure:"TZ"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	Environment string `mapstructure:"ENVIRONMENT" validate:"required,oneof=local development production test"`
	LogOutput   string `mapstructure:"LOG_OUTPUT"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`

	// --- Microservice identity & topology ---
	// ServiceName identifies this service in traces, metrics and event
	// metadata. Defaults to "gin-skeleton" when unset.
	ServiceName    string `mapstructure:"SERVICE_NAME"`
	ServiceVersion string `mapstructure:"SERVICE_VERSION"`

	// GRPCPort is the port the gRPC server listens on. Empty disables gRPC.
	GRPCPort string `mapstructure:"GRPC_PORT"`

	// ServiceEndpoints is a comma-separated registry of peer services in the
	// form "name=host:port,other=host:port". Parsed into Endpoints for the
	// gRPC client factory to dial. This is the 12-factor, config-driven
	// alternative to a discovery server; swap for Consul/etcd if needed.
	ServiceEndpoints string            `mapstructure:"SERVICE_ENDPOINTS"`
	Endpoints        map[string]string `mapstructure:"-"`

	// --- Telemetry ---
	// OtelExporterEndpoint is the OTLP/gRPC collector address (e.g.
	// "otel-collector:4317"). Empty disables trace export.
	OtelExporterEndpoint string  `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OtelTraceSampleRatio float64 `mapstructure:"OTEL_TRACE_SAMPLE_RATIO"`
	MetricsEnabled       bool    `mapstructure:"METRICS_ENABLED"`

	// --- Async messaging ---
	// NatsURL points at the NATS server (e.g. "nats://nats:4222"). Empty
	// selects the no-op broker so the service runs without a broker present.
	NatsURL            string `mapstructure:"NATS_URL"`
	NatsStreamName     string `mapstructure:"NATS_STREAM_NAME"`
	EventSubjectPrefix string `mapstructure:"EVENT_SUBJECT_PREFIX"`

	DBType     string `mapstructure:"DB_TYPE" validate:"required"`
	DBUsername string `mapstructure:"DB_USERNAME" validate:"required"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBHost     string `mapstructure:"DB_HOST" validate:"required"`
	DBPort     string `mapstructure:"DB_PORT" validate:"required"`
	DBName     string `mapstructure:"DB_NAME" validate:"required"`

	DBMaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`

	SentryDSN string `mapstructure:"SENTRY_DSN"`

	CorsAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	StorageBucketName string `mapstructure:"STORAGE_BUCKET_NAME"`
	ServiceAccountKey string `mapstructure:"SERVICE_ACCOUNT_KEY"`

	AdminEmail string `mapstructure:"ADMIN_EMAIL"`
	AdminPass  string `mapstructure:"ADMIN_PASS"`
	AdminName  string `mapstructure:"ADMIN_NAME"`

	MailClientID     string `mapstructure:"MAIL_CLIENT_ID"`
	MailClientSecret string `mapstructure:"MAIL_CLIENT_SECRET"`
	MailAccesstoken  string `mapstructure:"MAIL_ACCESS_TOKEN"`
	MailRefreshToken string `mapstructure:"MAIL_REFRESH_TOKEN"`

	AwsS3Region  string `mapstructure:"AWS_S3_REGION"`
	AwsS3Bucket  string `mapstructure:"AWS_S3_BUCKET"`
	AwsAccessKey string `mapstructure:"AWS_ACCESS_KEY"`
	AwsSecretKey string `mapstructure:"AWS_SECRET_KEY"`

	TwilioBaseURL            string `mapstructure:"TWILIO_BASE_URL"`
	TwilioSID                string `mapstructure:"TWILIO_SID"`
	TwilioAuthToken          string `mapstructure:"TWILIO_AUTH_TOKEN"`
	TwilioSMSFrom            string `mapstructure:"TWILIO_SMS_FROM"`
	JwtAccessSecret          string `mapstructure:"JWT_ACCESS_SECRET" validate:"required,min=16"`
	JwtRefreshSecret         string `mapstructure:"JWT_REFRESH_SECRET" validate:"required,min=16"`
	JwtAccessTokenExpiresAt  int    `mapstructure:"JWT_ACCESS_TOKEN_EXPIRES_AT" validate:"required,min=1"`
	JwtRefreshTokenExpiresAt int    `mapstructure:"JWT_REFRESH_TOKEN_EXPIRES_AT" validate:"required,min=1"`

	IdempotencyStore string        `mapstructure:"IDEMPOTENCY_STORE" validate:"omitempty,oneof=none mysql redis"`
	IdempotencyTTL   time.Duration `mapstructure:"IDEMPOTENCY_TTL"`
	RedisAddr        string        `mapstructure:"REDIS_ADDR"`
	RedisPassword    string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB          int           `mapstructure:"REDIS_DB"`

	RateLimitPeriod   time.Duration `mapstructure:"RATE_LIMIT_PERIOD"`
	RateLimitRequests int64         `mapstructure:"RATE_LIMIT_REQUESTS"`

	ProjectName       string `mapstructure:"PROJECT_NAME"`
	BillingAccountId  string `mapstructure:"BILLING_ACCOUNT_ID"`
	BudgetDisplayName string `mapstructure:"BUDGET_DISPLAY_NAME"`
	BudgetAmount      int64  `mapstructure:"BUDGET_AMOUNT"`
	SetBudget         int    `mapstructure:"SET_BUDGET"`

	StripeSecretKey   string `mapstructure:"STRIPE_SECRET_KEY"`
	StripeProductID   string `mapstructure:"STRIPE_PRODUCT_ID"`
	StripeWebhookKey  string `mapstructure:"STRIPE_WEBHOOK_KEY"`
	StripeRedirectUrl string `mapstructure:"STRIPE_REDIRECT_URL"`
}

type EnvPath string

func (p EnvPath) ToString() string {
	return string(p)
}

// NewEnv creates a new environment.
//
// Configuration is loaded from the env file when present (local development)
// and always overlaid with real environment variables (12-factor container
// deploys). A missing env file is NOT fatal — in Kubernetes the file is absent
// and every value arrives via the environment; only a malformed file aborts
// startup. Required values are still enforced by validateEnv below.
func NewEnv(envPath EnvPath) Env {
	env := Env{}
	_ = godotenv.Load(envPath.ToString())

	viper.AutomaticEnv()
	bindEnvVars(env)

	if _, statErr := os.Stat(envPath.ToString()); statErr == nil {
		viper.SetConfigFile(envPath.ToString())
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("☠️ Env config file error: %+v", err)
		}
	} else {
		log.Printf("ℹ️ no env file at %q; reading configuration from environment", envPath.ToString())
	}

	if err := viper.Unmarshal(&env); err != nil {
		log.Fatalf("☠️ environment can't be loaded: %+v", err)
	}

	if env.TimeZone == "" {
		env.TimeZone = "UTC"
	}

	if env.ServiceName == "" {
		env.ServiceName = "gin-skeleton"
	}
	if env.ServiceVersion == "" {
		env.ServiceVersion = "dev"
	}
	if env.EventSubjectPrefix == "" {
		env.EventSubjectPrefix = env.ServiceName
	}
	if env.NatsStreamName == "" {
		env.NatsStreamName = "EVENTS"
	}
	// Default to full sampling in non-production so local traces are complete;
	// production should lower this via OTEL_TRACE_SAMPLE_RATIO.
	if env.OtelTraceSampleRatio == 0 {
		if env.Environment == "production" {
			env.OtelTraceSampleRatio = 0.1
		} else {
			env.OtelTraceSampleRatio = 1.0
		}
	}

	env.Endpoints = parseEndpoints(env.ServiceEndpoints)

	if err := validateEnv(&env); err != nil {
		log.Fatalf("☠️ environment validation failed:\n%s", err.Error())
	}

	return env
}

// bindEnvVars registers every mapstructure key with viper so that
// AutomaticEnv values are picked up by Unmarshal even when no config file is
// present (viper only merges env vars for keys it knows about).
func bindEnvVars(iface any) {
	t := reflect.TypeOf(iface)
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		_ = viper.BindEnv(tag)
	}
}

// parseEndpoints turns a "name=host:port,other=host:port" string into a map.
// Malformed pairs (missing '=') are skipped so a typo can't crash startup.
func parseEndpoints(s string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, addr, ok := strings.Cut(pair, "=")
		name, addr = strings.TrimSpace(name), strings.TrimSpace(addr)
		if !ok || name == "" || addr == "" {
			continue
		}
		out[name] = addr
	}
	return out
}

// validateEnv runs struct validation and aggregates all issues into a single
// error message so missing/invalid vars are reported together at startup.
func validateEnv(env *Env) error {
	v := validator.New()
	err := v.Struct(env)
	if err == nil {
		return nil
	}

	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var b strings.Builder
	for _, fe := range verrs {
		fmt.Fprintf(&b, "  - %s failed %q (got %q)\n", fe.Field(), fe.Tag(), fmt.Sprint(fe.Value()))
	}
	return fmt.Errorf("%s", b.String())
}
