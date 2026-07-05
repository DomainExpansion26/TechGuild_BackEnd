package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateOTP() (string, error) {

	const digits = "0123456789"

	otp := make([]byte, 6)

	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}

		otp[i] = digits[num.Int64()]
	}

	return string(otp), nil
}