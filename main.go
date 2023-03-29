package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/opentracing-contrib/go-gin/ginhttp"

	opentracing "github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
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

func setupRouter() *gin.Engine {

	r := gin.New()

	r.Use(gin.Recovery())                                 // panic時に500エラーを返却
	r.Use(ginhttp.Middleware(opentracing.GlobalTracer())) // TODO: 無視の設定は？
	r.Use(loggingHandler)

	// Ping test
	r.GET("/ping", gin.WrapF(pingHandler))
	r.GET("/healthz", gin.WrapF(healthzHandler))

	return r
}

// ロギング・トレース対象外
func ignoreTracingRequest(r *http.Request) bool {
	return !(r.RequestURI == "/healthz" || strings.HasPrefix(r.RequestURI, "/static"))
}

func initTracer() io.Closer {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return nil
	}
	cfg.Sampler.Param = 1

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return nil
	}

	opentracing.SetGlobalTracer(tracer)

	return closer
}

func main() {
	// init Tracer
	closer := initTracer()
	if closer == nil {
		log.Fatal("Unable to init Tracer")
	}
	defer closer.Close()

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

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("finish shutdown")
	os.Exit(0)
}
