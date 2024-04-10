package db

import (
        "crypto/rand"
)

const characters =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@"
const charactersLength = byte(len(characters) - 1)
const tokenLength = byte(128)

var tokens = map[string]Account{}

func newToken() (string, error) {
        var random [tokenLength]byte
        var b [tokenLength]byte
        _, err := rand.Read(random[:])
        if err != nil { return "", err }
        for i := range b {
                b[i] = characters[random[i] & charactersLength]
        }
        return string(b[:]), nil
}
