package web

import (
	"crypto/rand"
	"encoding/base64"
)

const tokenLength = 32

func newToken() (string, error) {
	var random [tokenLength]byte
	_, err := rand.Read(random[:])
	if err != nil { return "", err }
	return base64.RawStdEncoding.EncodeToString(random[:]), nil
}
