package payload

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_JwtParser_shouldSuccessfullyParseValidToken(t *testing.T) {
	token := NewTestJwtToken(t, jwt.MapClaims{
		"container.id":    "cid",
		"container.env":   "cenv",
		"container.label": "clabel",
	})
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse(token)

	require.Nil(t, err)
	require.NotNil(t, parsed)

	require.Equal(t, parsed.ContainerId, "cid")
	require.Equal(t, parsed.ContainerLabel, "clabel")
	require.Equal(t, parsed.ContainerEnv, "cenv")
}

func Test_JwtParser_shouldFailOnInvalidToken(t *testing.T) {
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse("")

	require.NotNil(t, err)
	require.Nil(t, parsed)
}

func Test_JwtParser_shouldFailOnExpiredToken(t *testing.T) {
	token := NewTestJwtToken(t, jwt.MapClaims{
		"exp": time.Now().Add(-1 * time.Second).Unix(),
	})
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse(token)

	require.NotNil(t, err)
	require.Nil(t, parsed)
}

func NewTestJwtParser(t *testing.T) (*JwtParser) {
	secret := "secret"
	parser, err := NewJwtParser(secret)

	require.Nil(t, err)
	require.NotNil(t, parser)

	return parser
}

func NewTestJwtToken(t *testing.T, claims jwt.Claims) string {
	secret := "secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))

	require.Nil(t, err)
	require.NotEmpty(t, signed)

	return signed
}
