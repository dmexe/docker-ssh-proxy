package payload

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

func (p *JwtParser) Parse(token string) (*Request, error) {
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims := parsed.Claims.(jwt.MapClaims)

	containerId := claims["cid"]
	containerEnv := claims["env"]
	containerLabel := claims["lab"]

	filter := &Request{}

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
