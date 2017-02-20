package payloads

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_JwtParser(t *testing.T) {

	t.Run("should parse valid token", func(t *testing.T) {
		token := newTestJwtToken(t, jwt.MapClaims{
			"cid": "cid",
			"env": "cenv",
			"lab": "clabel",
		})
		parser := newTestJwtParser(t)
		payload, err := parser.Parse(token)

		require.NoError(t, err)
		require.NotNil(t, payload)

		require.Equal(t, payload.ContainerID, "cid")
		require.Equal(t, payload.ContainerLabel, "clabel")
		require.Equal(t, payload.ContainerEnv, "cenv")
	})

	t.Run("fail on invalid token", func(t *testing.T) {
		parser := newTestJwtParser(t)
		_, err := parser.Parse("")

		require.Error(t, err)
	})

	t.Run("fail on expired token", func(t *testing.T) {
		token := newTestJwtToken(t, jwt.MapClaims{
			"exp": time.Now().Add(-1 * time.Second).Unix(),
		})
		parser := newTestJwtParser(t)
		_, err := parser.Parse(token)

		require.Error(t, err)
	})
}

func newTestJwtParser(t *testing.T) *JwtParser {
	secret := "secret"
	parser, err := NewJwtParser(secret)

	require.NoError(t, err)
	require.NotNil(t, parser)

	return parser
}

func newTestJwtToken(t *testing.T, claims jwt.Claims) string {
	secret := "secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))

	require.NoError(t, err)
	require.NotEmpty(t, signed)

	return signed
}
