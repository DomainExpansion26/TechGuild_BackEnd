package utils

import (
	"fmt"
	"net/smtp"
	"os"
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

	subject := "Subject: Verify your TechGuild Email\r\n"
	mime := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n"

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

	message := []byte(subject + mime + body)

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

	subject := "Subject: TechGuild Password Reset\r\n"

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

	message := []byte(subject + "\r\n" + body)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{toEmail},
		message,
	)
}
