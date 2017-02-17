package payloads

import (
	jwt "github.com/dgrijalva/jwt-go"
	"os"
)

type JwtParser struct {
	secret string
}

func NewJwtParser(secret string) (*JwtParser, error) {
	config := &JwtParser{
		secret: secret,
	}
	return config, nil
}

func NewJwtParserFromEnv() (*JwtParser, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "none"
	}

	return NewJwtParser(secret)
}

func (p *JwtParser) Parse(token string) (*Payload, error) {
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims := parsed.Claims.(jwt.MapClaims)

	containerID := claims["cid"]
	containerEnv := claims["env"]
	containerLabel := claims["lab"]

	payload := &Payload{}

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
