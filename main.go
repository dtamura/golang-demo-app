package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type metrics struct {
	httpReqCounter   *prometheus.CounterVec
	temperatureGauge prometheus.Gauge
}

var (
	tracer            = otel.Tracer("test-tracer")
	metricsCollectors *metrics
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: log.FieldMap{
			log.FieldKeyTime:  "timestamp",
			log.FieldKeyLevel: "severity",
			log.FieldKeyMsg:   "msg",
		},
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)
}

func newMetrics(reg prometheus.Registerer) *metrics {
	httpReqCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "リクエスト数のトータル",
		},
		[]string{"code", "method"},
	)
	tempGage := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "app_temperature",
		Help: "温度",
	})
	m := &metrics{
		httpReqCounter:   httpReqCounter,
		temperatureGauge: tempGage,
	}
	reg.MustRegister(m.httpReqCounter)
	reg.MustRegister(m.temperatureGauge)

	return m
}

func setupRouter() *gin.Engine {

	r := gin.New()

	r.Use(gin.Recovery()) // panic時に500エラーを返却
	r.Use(otelgin.Middleware("", otelgin.WithFilter(ignoreTracingRequest)))
	r.Use(loggingHandler)

	// Ping test
	r.GET("/ping", gin.WrapF(pingHandler))
	r.GET("/healthz", gin.WrapF(healthzHandler))
	r.GET("/sendmail", gin.WrapF(sendmailHandler))

	// Metrics
	// r.GET("/metrics", gin.WrapF(promhttp.Handler().ServeHTTP))
	reg := prometheus.NewRegistry()
	metricsCollectors = newMetrics(reg)
	r.GET("/metrics", gin.WrapF(promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}).ServeHTTP))

	return r
}

// ロギング・トレース対象外
func ignoreTracingRequest(r *http.Request) bool {
	return !(r.RequestURI == "/healthz" || strings.HasPrefix(r.RequestURI, "/static"))
}

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

	// Gin mode:
	//  - using env:   export GIN_MODE=release
	//  - using code:  gin.SetMode(gin.ReleaseMode)
	gin.SetMode(gin.ReleaseMode)

	router := setupRouter()
	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:3000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Info("Start Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	log.Info("start shutdown")
	wait := time.Second * 15
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	if err := shutdown(ctx); err != nil { // Otel
		log.Fatal("failed to shutdown TracerProvider: %w", err)
	}

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("finish shutdown")
	os.Exit(0)
}
