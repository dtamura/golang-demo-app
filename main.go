package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("test-tracer")
)

func initProvider() (func(context.Context) error, error) {
	ctx := context.Background()

	resource, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName("runqslower"),
			semconv.DeploymentEnvironment("uat"),
			semconv.ServiceVersion("v0.1.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}

func main() {
	// init Otel
	shutdown, err := initProvider()
	if err != nil {
		log.Fatal(err)
	}

	// main
	t1, _ := time.Parse(time.RFC3339Nano, "2023-08-29T07:30:00.999999999+09:00")
	ctx, rootSpan := tracer.Start(context.Background(), "root", trace.WithTimestamp(t1))
	span := trace.SpanFromContext(ctx)
	ctx, childSpan1 := tracer.Start(ctx, "child")
	_, childChildSpan1 := tracer.Start(ctx, "chichi1")
	span.SetAttributes((attribute.String("owner", "tamtam")))
	childChildSpan1.End()
	childSpan1.End()
	t2, _ := time.Parse(time.RFC3339Nano, "2023-08-29T07:31:00.999999999+09:00")
	rootSpan.End(trace.WithTimestamp(t2))

	// shutdown
	if err := shutdown(ctx); err != nil { // Otel
		log.Fatal("failed to shutdown TracerProvider: %w", err)
	}

	log.Println("finish shutdown")
	os.Exit(0)
}
