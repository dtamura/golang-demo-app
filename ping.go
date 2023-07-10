package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {

	span := trace.SpanFromContext(r.Context())

	email := r.Header.Get("X-Goog-Authenticated-User-Email")
	if email != "" {
		span.SetAttributes((attribute.String("X-Goog-Authenticated-User-Email", email)))
	}

	msg := ping(r.Context())

	log.WithFields(log.Fields{
		"dd": getDDLogFields(span),
	}).Info(msg)
	span.SetAttributes(attribute.String("ping", msg))

	// Metrics
	metricsCollectors.httpReqCounter.With(prometheus.Labels{"code": "200", "method": "GET"}).Inc()

	temperature := .0
	if temp := r.URL.Query().Get("temp"); temp != "" {
		if t, err := strconv.ParseFloat(temp, 64); err == nil {
			temperature = t
		}
	}
	metricsCollectors.mu.Lock()
	metricsCollectors.temp = temperature
	metricsCollectors.temperatureGauge.Set(temperature)
	metricsCollectors.mu.Unlock()

	// Response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"msg": msg})
}

func ping(ctx context.Context) string {
	_, span := tracer.Start(ctx, "ping")
	defer span.End()

	// // create http request
	// client := &http.Client{}

	// target := os.Getenv("PING_TARGET_URL")
	// req, err := http.NewRequest("GET", target+"/ping", nil)
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"dd": getDDLogFields(span),
	// 	}).Error(err)
	// 	return ""
	// }
	// err = tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header))
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"dd": getDDLogFields(span),
	// 	}).Warn(err)
	// }
	// req.Header.Add("Content-Type", "application/json")

	// // Start Request
	// resp, err := client.Do(req)
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"dd": getDDLogFields(span),
	// 	}).Error(err)
	// 	span.Finish(tracer.WithError(err))
	// 	return ""
	// }
	// defer resp.Body.Close()

	// var data map[string]string
	// if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
	// 	log.WithFields(log.Fields{
	// 		"dd": getDDLogFields(span),
	// 	}).Error(err)
	// 	span.Finish(tracer.WithError(err))
	// 	return ""
	// }
	// span.Finish(tracer.WithError(err))

	return "ping"
}
