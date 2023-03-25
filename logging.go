package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func loggingHandler(c *gin.Context) {

	if ignoreGinTracingRequest(c) {
		c.Next()
		return
	}

	// Start timer
	start := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery

	span, _ := tracer.SpanFromContext(c.Request.Context())
	ddLog := getDDLogFields(span)

	c.Next()

	now := time.Now()
	latency := float64(now.Sub(start).Nanoseconds()) / 1000000.0
	clientIP := c.ClientIP()
	method := c.Request.Method
	statusCode := c.Writer.Status()
	errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
	userAgent := c.Request.UserAgent()
	proto := c.Request.Proto

	log.WithFields(log.Fields{
		"http": log.Fields{
			"status":     statusCode,
			"client":     clientIP,
			"method":     method,
			"path":       path,
			"query":      raw,
			"user-agent": userAgent,
			"proto":      proto,
			"headers":    headersFromRequest(c.Request),
			"latency":    latency,
			"error":      errorMessage,
		},
		"dd": ddLog,
	}).Info()

}

func getDDLogFields(span ddtrace.Span) log.Fields {
	return log.Fields{
		"service":  os.Getenv("DD_SERVICE"),
		"version":  os.Getenv("DD_VERSION"),
		"env":      os.Getenv("DD_ENV"),
		"trace_id": span.Context().TraceID(),
		"span_id":  span.Context().SpanID(),
	}
}

func headersFromRequest(r *http.Request) log.Fields {
	ipHeaders := []string{
		"x-forwarded-for",
		"x-real-ip",
		"x-client-ip",
		"x-forwarded",
		"x-cluster-client-ip",
		"forwarded-for",
		"forwarded",
		"via",
		"true-client-ip",
	}
	var headers []string
	var ips []string
	for _, hdr := range ipHeaders {
		if v := r.Header.Get(hdr); v != "" {
			headers = append(headers, hdr)
			ips = append(ips, v)
		}
	}

	result := log.Fields{}
	for i := range headers {
		result[headers[i]] = ips[i]
	}
	return result
}
