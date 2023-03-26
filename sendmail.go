package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Mail struct {
	Sender  string
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Body    bytes.Buffer
}

func BuildMessage(mail Mail) string {
	msg := ""
	msg += fmt.Sprintf("From: %s\r\n", mail.Sender)
	if len(mail.To) > 0 {
		msg += fmt.Sprintf("To: %s\r\n", mail.To[0])
	}
	if len(mail.Cc) > 0 {
		msg += fmt.Sprintf("Cc: %s\r\n", strings.Join(mail.Cc, ";"))
	}
	msg += fmt.Sprintf("Subject: %s\r\n", mail.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", mail.Body.String())

	return msg
}

func sendmail() error {
	sender := "dtamura@example.com"

	// my_user := ""
	// my_password := ""
	addr := "mailhog:1025"
	// host := "mailhog"

	subject := "こんにちは！"

	var template_data = `
    Dear {{ .Name }}, your debt amount is`

	t := template.Must(template.New("template_data").Parse(template_data))
	var body bytes.Buffer
	if err := t.Execute(&body, struct{ Name string }{"Daiki Tamura"}); err != nil {
		log.Fatal(err)
		return err
	}

	request := Mail{
		Sender:  sender,
		To:      []string{"bob@example.com"},
		Cc:      []string{"alice@example.com"},
		Subject: subject,
		Body:    body,
	}

	msg := BuildMessage(request)
	// auth := smtp.PlainAuth("", my_user, my_password, host)
	if err := smtp.SendMail(addr, nil, sender, []string{"bob@example.com"}, []byte(msg)); err != nil {
		log.Fatal(err)
		return err
	}

	return nil

}

func sendmailHandler(w http.ResponseWriter, r *http.Request) {

	span := trace.SpanFromContext(r.Context())
	log.WithFields(log.Fields{
		"dd": getDDLogFields(span),
	}).Info("sending email")
	span.SetAttributes(attribute.String("sendmail", "hello"))

	if err := sendmail(); err != nil {
		log.WithFields(log.Fields{
			"dd": getDDLogFields(span),
		}).Error(err)
		// Response
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("%v", err)})
		return
	}

	// Response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"msg": "Success Send Mail"})
}
