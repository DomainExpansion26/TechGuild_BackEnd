package utils

import (
	"fmt"
	"net/smtp"
	"os"
	"time"

	"github.com/google/uuid"
)

func SendVerificationEmail(toEmail string, token string) error {

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	frontendURL := os.Getenv("FRONTEND_URL")

	verificationURL := fmt.Sprintf(
		"%s/verify-email?token=%s",
		frontendURL,
		token,
	)

	headers := fmt.Sprintf(
		"From: TechGuild <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: Verify your TechGuild Email\r\n"+
			"Date: %s\r\n"+
			"Message-ID: <%s@techguild.com>\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		from, toEmail, time.Now().Format(time.RFC1123Z), uuid.New().String(),
	)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Welcome to TechGuild</h2>

			<p>Thank you for registering.</p>

			<p>Please click the button below to verify your email address.</p>

			<a href="%s"
			style="
				background:#2563eb;
				color:white;
				padding:12px 20px;
				text-decoration:none;
				border-radius:6px;">
				Verify Email
			</a>

			<br><br>

			<p>If you didn't create this account, you can safely ignore this email.</p>

			<br>

			<p>Regards,<br>TechGuild Team</p>

		</body>
		</html>
	`, verificationURL)

	message := []byte(headers + body)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{toEmail},
		message,
	)
}
func SendResetPasswordEmail(toEmail string, token string) error {

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	resetLink := os.Getenv("FRONTEND_URL") + "/reset-password?token=" + token

	headers := fmt.Sprintf(
		"From: TechGuild <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: TechGuild Password Reset\r\n"+
			"Date: %s\r\n"+
			"Message-ID: <%s@techguild.com>\r\n\r\n",
		from, toEmail, time.Now().Format(time.RFC1123Z), uuid.New().String(),
	)
	body := fmt.Sprintf(
		"Hello,\r\n\r\n"+
			"You requested to reset your password.\r\n\r\n"+
			"Click the link below to reset your password:\r\n\r\n"+
			"%s\r\n\r\n"+
			"This link is valid for 24 hours.\r\n\r\n"+
			"If you did not request this, please ignore this email.\r\n\r\n"+
			"Regards,\r\n"+
			"TechGuild Team",
		resetLink,
	)

	message := []byte(headers + body)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{toEmail},
		message,
	)
}

func SendDataExportEmail(toEmail string, firstName string, downloadURL string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	headers := fmt.Sprintf(
		"From: TechGuild <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: Your TechGuild Data Export is Ready\r\n"+
			"Date: %s\r\n"+
			"Message-ID: <%s@techguild.com>\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		from, toEmail, time.Now().Format(time.RFC1123Z), uuid.New().String(),
	)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Your Data Export is Ready</h2>
			<p>Hi %s,</p>
			<p>Your TechGuild data export has been generated. Click the button below to download your data.</p>
			<a href="%s"
			style="
				background:#2563eb;
				color:white;
				padding:12px 20px;
				text-decoration:none;
				border-radius:6px;">
				Download My Data
			</a>
			<br><br>
			<p>This file contains all your personal data stored on TechGuild including your profile, account info, and activity.</p>
			<p>If you did not request this export, please contact support immediately.</p>
			<br>
			<p>Regards,<br>TechGuild Team</p>
		</body>
		</html>
	`, firstName, downloadURL)

	message := []byte(headers + body)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{toEmail},
		message,
	)
}
