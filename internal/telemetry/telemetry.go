package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Enabled reports whether telemetry was turned on by the environment.
func Enabled() bool {
	return os.Getenv("MODEL_CLI_OTEL_ENDPOINT") != ""
}

// normalizeEndpoint strips optional http:// or https:// prefix from a bare
// host:port, or returns the input as-is if it is a valid URL with a path.
// Returns the normalized endpoint and true if it appears valid, or the original
// string and false if it cannot be parsed.
func normalizeEndpoint(endpoint string) (string, bool) {
	if endpoint == "" {
		return "", false
	}

	// Try to parse as a URL first
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		// If it's http:// or https:// with no path (or just /), strip the scheme
		// and return just the host:port
		if (u.Scheme == "http" || u.Scheme == "https") && (u.Path == "" || u.Path == "/") {
			return u.Host, true
		}
		// It's a valid URL with a non-trivial path - use it as-is
		return endpoint, true
	}

	// Strip optional scheme prefix if present (for malformed but fixable input)
	normalized := endpoint
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimPrefix(normalized, prefix)
			break
		}
	}

	// Validate that what remains looks like a valid endpoint
	// It should parse as a URL when prefixed with http:// and have a non-empty host
	// that doesn't start with a colon
	if normalized != "" {
		if u, err := url.Parse("http://" + normalized); err == nil && u.Host != "" && !strings.HasPrefix(u.Host, ":") {
			return normalized, true
		}
	}

	return endpoint, false
}

// Start initializes OTel when MODEL_CLI_OTEL_ENDPOINT is set; otherwise it
// returns a zero-value Telemetry whose methods are all safe no-ops.
func Start(version string) *Telemetry {
	endpoint := os.Getenv("MODEL_CLI_OTEL_ENDPOINT")
	if endpoint == "" {
		return &Telemetry{}
	}

	normalized, ok := normalizeEndpoint(endpoint)
	if !ok {
		fmt.Fprintf(os.Stderr, "model-cli: ignoring MODEL_CLI_OTEL_ENDPOINT=%s (could not parse endpoint)\n", endpoint)
		return &Telemetry{}
	}

	endpoint = normalized

	ctx := context.Background()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "model-cli"),
			attribute.String("service.version", version),
		),
	)
	if err != nil {
		return &Telemetry{}
	}

	t := &Telemetry{}

	if exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // local collector speaks plaintext HTTP
	); err == nil {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		t.tracerProvider = tp
		t.tracer = tp.Tracer("model-cli")
	}

	if exp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	); err == nil {
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp,
				sdkmetric.WithInterval(time.Second))),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		t.meterProvider = mp
		meter := mp.Meter("model-cli")
		t.runs, _ = meter.Int64Counter("modelcli.command.runs",
			metric.WithDescription("Number of model-cli command runs"))
		t.duration, _ = meter.Float64Histogram("modelcli.command.duration",
			metric.WithDescription("Command duration"),
			metric.WithUnit("s"))
	}

	return t
}

// Telemetry is safe to use in its zero state: every method no-ops.
type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	runs           metric.Int64Counter
	duration       metric.Float64Histogram
}

// StartSpan begins a span for one command run.
func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if t == nil || t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, "model-cli "+name)
}

// Record notes the outcome of the run. exit is "ok" or "error".
func (t *Telemetry) Record(ctx context.Context, command, exit string, tookSeconds float64) {
	if t == nil || t.runs == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("command", command),
		attribute.String("exit", exit),
	)
	t.runs.Add(ctx, 1, attrs)
	t.duration.Record(ctx, tookSeconds, attrs)
}

// Shutdown flushes everything. A CLI exits right after this — without it,
// batched spans/metrics are silently lost.
func (t *Telemetry) Shutdown(ctx context.Context) {
	if t == nil {
		return
	}
	if t.tracerProvider != nil {
		_ = t.tracerProvider.Shutdown(ctx)
	}
	if t.meterProvider != nil {
		_ = t.meterProvider.Shutdown(ctx)
	}
}
