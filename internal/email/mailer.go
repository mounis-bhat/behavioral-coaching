package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
)

const (
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPPort = "587"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, textBody, htmlBody string) error
}

type GmailMailer struct {
	from     string
	username string
	password string
}

func NewGmailMailer(from, appPassword string) (*GmailMailer, error) {
	from = strings.TrimSpace(from)
	appPassword = strings.TrimSpace(appPassword)
	if from == "" || appPassword == "" {
		return nil, errors.New("missing gmail credentials")
	}
	return &GmailMailer{
		from:     from,
		username: from,
		password: appPassword,
	}, nil
}

func (m *GmailMailer) Send(_ context.Context, to, subject, textBody, htmlBody string) error {
	if m == nil {
		return errors.New("mailer not configured")
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("missing recipient")
	}
	if textBody == "" && htmlBody == "" {
		return errors.New("missing email body")
	}

	msg := buildMessage(m.from, to, subject, textBody, htmlBody)
	addr := net.JoinHostPort(gmailSMTPHost, gmailSMTPPort)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, gmailSMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("smtp server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: gmailSMTPHost}); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", m.username, m.password, gmailSMTPHost)
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildMessage(from, to, subject, textBody, htmlBody string) string {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", encodeHeader(subject)),
		"MIME-Version: 1.0",
	}

	if htmlBody == "" {
		headers = append(headers, "Content-Type: text/plain; charset=UTF-8")
		return strings.Join(headers, "\r\n") + "\r\n\r\n" + textBody
	}

	boundary := "boundary_alt_9b2b0a1d"
	headers = append(headers, fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s", boundary))

	var body strings.Builder
	body.WriteString(strings.Join(headers, "\r\n"))
	body.WriteString("\r\n\r\n")
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	body.WriteString(textBody)
	body.WriteString("\r\n")
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	body.WriteString(htmlBody)
	body.WriteString("\r\n--" + boundary + "--\r\n")
	return body.String()
}

func encodeHeader(value string) string {
	if value == "" {
		return ""
	}
	clean := strings.NewReplacer("\r", "", "\n", "").Replace(value)
	return mime.QEncoding.Encode("utf-8", clean)
}
