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

// SetupLogger memasang slog JSON ke stdout sebagai logger default.
// Tiap log yang dibuat dengan *Context (mis. slog.InfoContext) otomatis menyertakan trace_id.
func SetupLogger() {
	base := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(traceHandler{Handler: base}))
}

// traceHandler menyisipkan trace_id dari context ke tiap record.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	// trace_id disisipkan otomatis dari context supaya setiap baris log bisa
	// dikorelasikan dengan span/trace di sistem tracing (satu request → satu trace_id).
	// Tanpa ini, developer harus menempelkan trace_id manual di tiap call log.
	// Bila context tidak punya trace aktif, atribut dilewati agar tidak ada field kosong.
	if id := TraceID(ctx); id != "" {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// InitTracer menyiapkan TracerProvider OTLP. Bila endpoint kosong, tracing dimatikan
// (provider global tetap no-op) dan shutdown yang dikembalikan tidak melakukan apa-apa.
func InitTracer(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	// Endpoint kosong = tracing dimatikan. Kita kembalikan shutdown no-op (bukan nil)
	// supaya pemanggil bisa selalu memanggil defer shutdown() tanpa cek nil — tracing
	// jadi opsional tanpa membebani kode pemanggil dengan percabangan.
	if endpoint == "" {
		slog.Info("OTLP endpoint kosong, tracing dimatikan")
		return func(context.Context) error { return nil }, nil
	}
	// Exporter mengirim span via OTLP/HTTP ke collector di endpoint.
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("membuat OTLP exporter: %w", err)
	}
	// Resource menandai semua span dengan service.name agar trace bisa difilter
	// per-service di backend (Jaeger/Tempo/dll).
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("membuat resource: %w", err)
	}
	// WithBatcher: span di-buffer dan dikirim per-batch, bukan satu-per-satu, agar
	// export tidak menambah latensi pada jalur request dan hemat koneksi ke collector.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	// Set sebagai provider global supaya otel.Tracer(...) di mana pun memakai provider ini.
	otel.SetTracerProvider(tp)
	// tp.Shutdown mem-flush span yang masih tertahan di buffer; WAJIB dipanggil saat
	// shutdown agar trace terakhir tidak hilang.
	return tp.Shutdown, nil
}

// TraceID mengambil trace id aktif dari context (kosong bila tidak ada).
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}
