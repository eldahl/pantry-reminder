package main

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendEmail(to []string, subject, body string) error {
	from := os.Getenv("GMAIL_USER")
	password := os.Getenv("GMAIL_PASSWORD")

	if from == "" || password == "" {
		return fmt.Errorf("GMAIL_USER or GMAIL_PASSWORD not set")
	}

	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Message.
	message := []byte("To: " + to[0] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		return err
	}
	return nil
}
