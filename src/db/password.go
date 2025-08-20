package db

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

var errInvalidCredential = errors.New("invalid credentials")

func comparePassword(password, hash string) error {

	parts := strings.Split(hash, "$")
	if len(parts) < 4 {
		return errors.New("invalid hash")
	}
	var time uint32
	var memory uint32
	var threads uint8

	_, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&memory, &time, &threads,
	)
	if err != nil {
		return err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}
	keyLen := uint32(len(decodedHash))

	comparisonHash := argon2.IDKey(
		[]byte(password), salt, time,
		memory, threads, keyLen,
	)

	if subtle.ConstantTimeCompare(decodedHash, comparisonHash) != 1 {
		return errInvalidCredential
	}

	return nil
}

const passwordTime = 1
const passwordMemory = 64 * 1024
const passwordThreads = 4
const passwordKeyLen = 32

const maxPassword = 128
const minPassword = 5

func isPasswordValid(password string) error {
	if len(password) > maxPassword {
		return errors.New("the password is too long")
	}
	if len(password) < minPassword {
		return errors.New("the password is too short")
	}
	return nil
}

func hashPassword(password string) (string, error) {

	if err := isPasswordValid(password); err != nil {
		return "", err
	}

	// Generate a Salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password), salt, passwordTime,
		passwordMemory, passwordThreads, passwordKeyLen,
	)

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	full := fmt.Sprintf(
		format, argon2.Version, passwordMemory,
		passwordTime, passwordThreads, b64Salt, b64Hash,
	)
	return full, nil
}
