package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/sirupsen/logrus"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "ping-handler")
	defer span.Finish()

	msg := ping(ctx)
	log.WithFields(log.Fields{}).Info(msg)
	span.SetTag("ping", msg)

	// Response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"msg": msg})
}

func ping(ctx context.Context) string {
	span, childCtx := opentracing.StartSpanFromContext(ctx, "ping")
	defer span.Finish()

	// create http request
	client := &http.Client{}

	target := os.Getenv("PING_TARGET_URL")
	req, err := http.NewRequestWithContext(childCtx, "GET", target+"/ping", nil)
	if err != nil {
		log.WithFields(log.Fields{}).Error(err)
		return ""
	}
	req.Header.Add("Content-Type", "application/json")

	tracer := opentracing.GlobalTracer()
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	err = tracer.Inject(span.Context(), opentracing.HTTPHeaders, carrier)
	if err != nil {
		log.Errorf("Error non-nil %v", err)
		span.SetTag(string(ext.Error), true)
		span.SetTag("msg", err)
	}

	// Start Request
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{}).Error(err)
		span.SetTag(string(ext.Error), true)
		span.SetTag("msg", err)
		return ""
	}
	defer resp.Body.Close()

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.WithFields(log.Fields{}).Error(err)
		span.SetTag(string(ext.Error), true)
		span.SetTag("msg", err)
		return ""
	}
	span.SetTag("error", err)

	return data["msg"]
}
