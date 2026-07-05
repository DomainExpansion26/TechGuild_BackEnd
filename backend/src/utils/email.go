package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendOTPEmail(toEmail string, otp string) error {

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	subject := "Subject: TechGuild Email Verification\r\n"

	body := fmt.Sprintf(
		"Hello,\r\n\r\n"+
			"Your TechGuild verification code is:\r\n\r\n"+
			"%s\r\n\r\n"+
			"This OTP is valid for 10 minutes.\r\n\r\n"+
			"If you did not register, please ignore this email.\r\n\r\n"+
			"Regards,\r\n"+
			"TechGuild Team",
		otp,
	)

	message := []byte(subject + "\r\n" + body)

	err := smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{toEmail},
		message,
	)

	if err != nil {
		return err
	}

	return nil
}