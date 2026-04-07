package server

import (
	"crypto/subtle"
	"os"
)

type Auth struct {
	token string
}

func NewAuth() *Auth {
	return &Auth{token: os.Getenv("PROXY_AUTH_TOKEN")}
}

func (a *Auth) Validate(token string) bool {
	if a.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a.token), []byte(token)) == 1
}
