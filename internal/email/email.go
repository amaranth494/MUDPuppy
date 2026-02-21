package email

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Sender handles sending emails via SMTP
type Sender struct {
	host        string
	port        int
	username    string
	password    string
	fromAddress string
}

// NewSender creates a new email sender
func NewSender(host string, port int, username, password, fromAddress string) *Sender {
	return &Sender{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		fromAddress: fromAddress,
	}
}

// SendOTP sends an OTP email to the recipient
func (s *Sender) SendOTP(recipientEmail, otpCode string) error {
	subject := "Your MUDPuppy Verification Code"
	body := fmt.Sprintf(`Your verification code is: %s

This code will expire in 15 minutes.

If you didn't request this code, please ignore this email.`, otpCode)

	return s.sendEmail(recipientEmail, subject, body)
}

// sendEmail sends an email with the given subject and body
func (s *Sender) sendEmail(recipient, subject, body string) error {
	// Build email headers
	headers := make(map[string]string)
	headers["From"] = s.fromAddress
	headers["To"] = recipient
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	// Construct email message
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// Connect to SMTP server and send
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// Use TLS
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	err := smtp.SendMail(addr, auth, s.fromAddress, []string{recipient}, []byte(msg.String()))
	if err != nil {
		log.Printf("Failed to send email to %s: %v", recipient, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully to %s", recipient)
	return nil
}

// IsConfigured checks if the SMTP sender is properly configured
func (s *Sender) IsConfigured() bool {
	return s.host != "" && s.username != "" && s.password != "" && s.fromAddress != ""
}
