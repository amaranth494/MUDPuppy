package email

import (
	"crypto/tls"
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

	// Connect to SMTP server with TLS
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName:         s.host,
		InsecureSkipVerify: false,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Printf("Failed to connect to SMTP server: %v", err)
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		log.Printf("Failed to create SMTP client: %v", err)
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err := client.Auth(auth); err != nil {
		log.Printf("Failed to authenticate: %v", err)
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Set sender
	if err := client.Mail(s.fromAddress); err != nil {
		log.Printf("Failed to set sender: %v", err)
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(recipient); err != nil {
		log.Printf("Failed to set recipient: %v", err)
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Write message
	w, err := client.Data()
	if err != nil {
		log.Printf("Failed to get data writer: %v", err)
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte(msg.String())); err != nil {
		log.Printf("Failed to write message: %v", err)
		return fmt.Errorf("failed to write message: %w", err)
	}

	log.Printf("Email sent successfully to %s", recipient)
	return nil
}

// IsConfigured checks if the SMTP sender is properly configured
func (s *Sender) IsConfigured() bool {
	return s.host != "" && s.username != "" && s.password != "" && s.fromAddress != ""
}
