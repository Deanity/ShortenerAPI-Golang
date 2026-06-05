package utils

import (
	"crypto/rand"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateSlug generates a cryptographically random base62 short code.
func GenerateSlug(length int) (string, error) {
	result := make([]byte, length)
	max := big.NewInt(int64(len(base62Chars)))

	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[n.Int64()]
	}

	return string(result), nil
}
