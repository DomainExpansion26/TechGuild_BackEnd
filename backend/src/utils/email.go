package utils

import (
	"fmt"
	"html"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"techguild-backend/src/config"

	"github.com/google/uuid"
)

// sanitizeHeader strips CR/LF to prevent SMTP header injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// sendEmail is the single internal function all email senders use.
func sendEmail(cfg *config.Config, toEmail, subject, htmlBody string) error {
	from := cfg.SMTPEmail
	password := cfg.SMTPPassword
	host := cfg.SMTPHost
	port := cfg.SMTPPort

	if from == "" || password == "" || host == "" || port == "" {
		return fmt.Errorf("smtp config missing: check SMTP_EMAIL/SMTP_PASSWORD/SMTP_HOST/SMTP_PORT")
	}

	toEmail = sanitizeHeader(toEmail)
	subject = sanitizeHeader(subject)

	auth := smtp.PlainAuth("", from, password, host)

	headers := fmt.Sprintf(
		"From: TechGuild <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Date: %s\r\n"+
			"Message-ID: <%s@techguild.com>\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		from, toEmail, subject, time.Now().Format(time.RFC1123Z), uuid.New().String(),
	)

	message := []byte(headers + htmlBody)

	return smtp.SendMail(host+":"+port, auth, from, []string{toEmail}, message)
}

func SendVerificationEmail(cfg *config.Config, toEmail string, token string) error {
	frontendURL := strings.TrimRight(cfg.FrontendURL, "/")
	verificationURL := fmt.Sprintf(
		"%s/verify-email?token=%s",
		frontendURL,
		url.QueryEscape(token),
	)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Welcome to TechGuild</h2>
			<p>Thank you for registering.</p>
			<p>Please click the button below to verify your email address.</p>
			<a href="%s"
			style="background:#2563eb;color:white;padding:12px 20px;text-decoration:none;border-radius:6px;">
				Verify Email
			</a>
			<br><br>
			<p>If you didn't create this account, you can safely ignore this email.</p>
			<br>
			<p>Regards,<br>TechGuild Team</p>
		</body>
		</html>
	`, verificationURL)

	return sendEmail(cfg, toEmail, "Verify your TechGuild Email", body)
}

func SendResetPasswordEmail(cfg *config.Config, toEmail string, token string) error {
	frontendURL := strings.TrimRight(cfg.FrontendURL, "/")
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, url.QueryEscape(token))

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Password Reset Requested</h2>
			<p>Click the link below to reset your password:</p>
			<a href="%s"
			style="background:#2563eb;color:white;padding:12px 20px;text-decoration:none;border-radius:6px;">
				Reset Password
			</a>
			<br><br>
			<p>This link is valid for 24 hours.</p>
			<p>If you did not request this, please ignore this email.</p>
			<br>
			<p>Regards,<br>TechGuild Team</p>
		</body>
		</html>
	`, resetLink)

	return sendEmail(cfg, toEmail, "TechGuild Password Reset", body)
}

func SendDataExportEmail(cfg *config.Config, toEmail string, firstName string, downloadURL string) error {
	safeName := html.EscapeString(firstName)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Your Data Export is Ready</h2>
			<p>Hi %s,</p>
			<p>Your TechGuild data export has been generated. Click the button below to download your data.</p>
			<a href="%s"
			style="background:#2563eb;color:white;padding:12px 20px;text-decoration:none;border-radius:6px;">
				Download My Data
			</a>
			<br><br>
			<p>This file contains all your personal data stored on TechGuild including your profile, account info, and activity.</p>
			<p>If you did not request this export, please contact support immediately.</p>
			<br>
			<p>Regards,<br>TechGuild Team</p>
		</body>
		</html>
	`, safeName, downloadURL)

	return sendEmail(cfg, toEmail, "Your TechGuild Data Export is Ready", body)
}
