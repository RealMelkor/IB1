package util

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
)

const tokenLength = 32

func NewToken() (string, error) {
	var random [tokenLength]byte
	_, err := rand.Read(random[:])
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(random[:]), nil
}

func NewTextToken() (string, error) {
	var random [tokenLength]byte
	_, err := rand.Read(random[:])
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).
		EncodeToString(random[:]), nil
}
