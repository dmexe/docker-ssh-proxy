package payloads

import (
	jwt "github.com/dgrijalva/jwt-go"
	"os"
)

// JwtParser is a parser implementation for JWT tokens
// known token keys
// * cid - container id identifier
// * env - container environment variable (eg. FOO=bar)
// * lab - container label
type JwtParser struct {
	secret string
}

const (
	jwtContainerID    = "cid"
	jwtContainerEnv   = "env"
	jwtContainerLabel = "lab"
)

// NewJwtParser constructs a new parser instance using given JWT secret
func NewJwtParser(secret string) (*JwtParser, error) {
	config := &JwtParser{
		secret: secret,
	}
	return config, nil
}

// NewJwtParserFromEnv construct a new parser instance using jwtSecret
// environment variable
func NewJwtParserFromEnv() (*JwtParser, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "none"
	}

	return NewJwtParser(secret)
}

// Parse given string to payload
func (p *JwtParser) Parse(token string) (Payload, error) {
	payload := Payload{}

	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.secret), nil
	})

	if err != nil {
		return payload, err
	}

	claims := parsed.Claims.(jwt.MapClaims)

	containerID := claims[jwtContainerID]
	containerEnv := claims[jwtContainerEnv]
	containerLabel := claims[jwtContainerLabel]

	if containerID != nil {
		payload.ContainerID = containerID.(string)
	}

	if containerEnv != nil {
		payload.ContainerEnv = containerEnv.(string)
	}

	if containerLabel != nil {
		payload.ContainerLabel = containerLabel.(string)
	}

	return payload, nil
}
