package main

import (
	jwt "github.com/dgrijalva/jwt-go"
	"os"
)

type JwtPayloadParser struct {
	secret string
}

func NewJwtPayloadParser() (*JwtPayloadParser, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "none"
	}

	config := &JwtPayloadParser{
		secret: secret,
	}

	return config, nil
}

func (p *JwtPayloadParser) Parse(token string) (*Payload, error) {
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.secret), nil
	})

	if err != nil {
		return nil, err
	}

	payload := parsed.Claims.(jwt.MapClaims)

	containerId := payload["container.id"]
	containerEnv := payload["container.env"]
	containerLabel := payload["container.label"]

	filter := &Payload{}

	if containerId != nil {
		filter.ContainerId = containerId.(string)
	}

	if containerEnv != nil {
		filter.ContainerEnv = containerEnv.(string)
	}

	if containerLabel != nil {
		filter.ContainerLabel = containerLabel.(string)
	}

	return filter, nil
}
