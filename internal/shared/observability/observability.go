// Package observability menyiapkan tracing OTLP dan structured logging JSON.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// SetupLogger masang slog JSON ke stdout sebagai logger default.
// Tiap log yang dibuat lewat varian *Context (mis. slog.InfoContext) otomatis nyertain trace_id.
func SetupLogger() {
	base := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(traceHandler{Handler: base}))
}

// traceHandler nyisipin trace_id dari context ke tiap record log.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	// trace_id kita sisipin otomatis dari context biar tiap baris log bisa
	// dikorelasiin sama span/trace di sistem tracing (satu request → satu trace_id).
	// Tanpa ini, developer harus nempelin trace_id manual di tiap pemanggilan log.
	// Kalau context-nya nggak punya trace aktif, atributnya kita skip biar nggak ada field kosong.
	if id := TraceID(ctx); id != "" {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// InitTracer nyiapin TracerProvider OTLP. Kalau endpoint-nya kosong, tracing dimatiin
// (provider global-nya tetap no-op) dan fungsi shutdown yang dibalikin nggak ngapa-ngapain.
func InitTracer(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	// Endpoint kosong berarti tracing dimatiin. Kita balikin shutdown no-op, bukan nil,
	// biar pemanggil bisa selalu manggil defer shutdown() tanpa harus cek nil dulu —
	// jadi tracing bisa opsional tanpa bikin kode pemanggil penuh percabangan.
	if endpoint == "" {
		slog.Info("OTLP endpoint kosong, tracing dimatikan")
		return func(context.Context) error { return nil }, nil
	}
	// Exporter tugasnya ngirim span lewat OTLP/HTTP ke collector yang ada di endpoint.
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("membuat OTLP exporter: %w", err)
	}
	// Resource nandain semua span dengan service.name biar trace-nya bisa difilter
	// per-service di backend (Jaeger/Tempo/dll).
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("membuat resource: %w", err)
	}
	// WithBatcher: span-nya di-buffer dulu lalu dikirim per-batch, bukan satu-satu,
	// biar proses export nggak nambah latensi di jalur request sekaligus hemat koneksi
	// ke collector.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	// Kita set sebagai provider global biar otel.Tracer(...) di mana pun pakai provider ini.
	otel.SetTracerProvider(tp)
	// tp.Shutdown bakal nge-flush span yang masih nyangkut di buffer; WAJIB dipanggil
	// pas shutdown biar trace terakhir nggak ilang.
	return tp.Shutdown, nil
}

// TraceID ngambil trace id yang aktif dari context (balik string kosong kalau nggak ada).
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}
