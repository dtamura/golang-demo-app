package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	ctx, rootSpan := tracer.Start(context.Background(), "root")
	span := trace.SpanFromContext(ctx)
	span.SetAttributes((attribute.String("owner", "tamtam")))
	rootSpan.End()

	// shutdown
	if err := shutdown(ctx); err != nil { // Otel
		log.Fatal("failed to shutdown TracerProvider: %w", err)
	}

	log.Println("finish shutdown")
	os.Exit(0)
}
