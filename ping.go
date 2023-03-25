package main

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {

	span, _ := tracer.StartSpanFromContext(r.Context(), "ping-handler")

	msg := ping(span)

	log.WithFields(log.Fields{
		"dd": getDDLogFields(span),
	}).Info(msg)

	span.SetTag("ping", msg)

	span.Finish()

	// Response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"msg": msg})
}

func ping(parent ddtrace.Span) string {
	span := tracer.StartSpan("ping", tracer.ChildOf(parent.Context()))
	defer span.Finish()

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
