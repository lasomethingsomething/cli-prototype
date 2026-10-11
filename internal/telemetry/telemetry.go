package telemetry

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
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

// isValidHostPort checks if s is a valid host[:port] with numeric port in 1-65535.
// Returns the normalized form (appends :4318 if no port) and true if valid.
func isValidHostPort(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		// No port, validate as plain host
		if isValidHost(s) {
			return s + ":4318", true
		}
		return s, false
	}
	if !isValidHost(host) {
		return s, false
	}
	// Port must be numeric 1-65535
	if port == "" {
		return s, false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return s, false
		}
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return s, false
	}
	return s, true
}

// isValidHost checks if s is a valid hostname or IP (no port).
func isValidHost(s string) bool {
	if s == "" {
		return false
	}
	// IPv6 bracketed
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return net.ParseIP(strings.TrimPrefix(strings.TrimSuffix(s, "]"), "[")) != nil
	}
	// IPv4 or hostname
	if net.ParseIP(s) != nil {
		return true
	}
	// Hostname: alphanum, dots, hyphens; no leading/trailing dot/hyphen
	if len(s) == 0 {
		return false
	}
	if s[0] == '.' || s[0] == '-' || s[len(s)-1] == '.' || s[len(s)-1] == '-' {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

// normalizeEndpoint normalizes endpoint: strips http://https:// from bare host[:port],
// keeps full URLs with paths, validates format. Returns normalized endpoint and ok.
func normalizeEndpoint(endpoint string) (string, bool) {
	if strings.TrimSpace(endpoint) == "" {
		return endpoint, false
	}

	// Parse as URL
	u, err := url.Parse(endpoint)
	if err == nil && u.Host != "" {
		if u.Scheme == "http" || u.Scheme == "https" {
			if u.Path == "" || u.Path == "/" {
				// scheme + host[:port], strip scheme
				if normalized, ok := isValidHostPort(u.Host); ok {
					return normalized, true
				}
				return endpoint, false
			}
			// Full URL with path, keep as-is
			return endpoint, true
		}
		// URL with other scheme or relative, keep as-is
		return endpoint, true
	}

	// Strip http:// or https:// prefix if present
	normalized := endpoint
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimPrefix(normalized, prefix)
			break
		}
	}

	if normalized, ok := isValidHostPort(normalized); ok {
		return normalized, true
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

	// Track endpoint and export errors for reporting in Shutdown
	t := &Telemetry{endpoint: endpoint}
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		if err != nil {
			t.lastExportErr = err
		}
	}))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "model-cli"),
			attribute.String("service.version", version),
		),
	)
	if err != nil {
		return &Telemetry{}
	}

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
	// endpoint and lastExportErr for error reporting in Shutdown
	endpoint      string
	lastExportErr error
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
	// Warn if export failed
	if t.lastExportErr != nil && t.endpoint != "" {
		fmt.Fprintf(os.Stderr, "model-cli: telemetry export to %s failed: %v\n", t.endpoint, t.lastExportErr)
	}
}
