// Package tracing wires OpenTelemetry to the Jaeger collector via OTLP/gRPC.
//
// The TracerProvider is installed as the global provider so that any library
// calling `otel.Tracer(...)` (gin middleware, gorm plugin, otelhttp transport)
// will pick it up without further configuration.
package tracing

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Shutdown is returned by Init and should be called at process exit to
// flush any buffered spans to the collector.
type Shutdown func(context.Context) error

// Init sets up the global OpenTelemetry tracer provider.
//
// Configuration is read from environment variables:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT  – host:port of the OTLP gRPC receiver
//	                                (default: "jaeger:4317")
//	OTEL_SERVICE_NAME            – service name tag (default: "translator-checkin")
//
// If the collector is unreachable, Init still succeeds and logs a warning;
// the exporter will buffer and keep retrying in the background. We never want
// a missing tracing backend to crash the API.
//
// Tracing can also be disabled outright by setting OTEL_TRACES_EXPORTER=none,
// which is what the E2E docker-compose stack does (no jaeger in that env).
// In that case Init returns a no-op Shutdown — no exporter is created, no
// background retries spam the logs.
func Init(ctx context.Context) (Shutdown, error) {
	if os.Getenv("OTEL_TRACES_EXPORTER") == "none" {
		log.Println("[tracing] OTEL_TRACES_EXPORTER=none, tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "jaeger:4317"
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "translator-checkin"
	}

	// Build OTLP gRPC exporter. We use WithDialOption + insecure credentials
	// because the collector sits on the same Docker network behind nothing.
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithGRPCConn(conn),
	))
	if err != nil {
		return nil, err
	}

	// Build resource directly (no merge with resource.Default) because
	// Default() may use a newer semconv schema URL than our imported
	// semconv package, and Merge rejects conflicting schema URLs.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("p3-jaeger"),
		semconv.DeploymentEnvironment(getEnvOr("DEPLOY_ENV", "dev")),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		// Development default: sample every request. Switch to
		// TraceIDRatioBased once traffic ramps up.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("[tracing] initialized OTLP exporter service=%q endpoint=%q", serviceName, endpoint)

	return tp.Shutdown, nil
}

func getEnvOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
