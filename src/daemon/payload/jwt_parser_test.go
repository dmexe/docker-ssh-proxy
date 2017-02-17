package payload

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_JwtParser_shouldSuccessfullyParseValidToken(t *testing.T) {
	token := NewTestJwtToken(t, jwt.MapClaims{
		"cid":    "cid",
		"env":   "cenv",
		"lab": "clabel",
	})
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse(token)

	require.NoError(t, err)
	require.NotNil(t, parsed)

	require.Equal(t, parsed.ContainerId, "cid")
	require.Equal(t, parsed.ContainerLabel, "clabel")
	require.Equal(t, parsed.ContainerEnv, "cenv")
}

func Test_JwtParser_shouldFailOnInvalidToken(t *testing.T) {
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse("")

	require.Error(t, err)
	require.Nil(t, parsed)
}

func Test_JwtParser_shouldFailOnExpiredToken(t *testing.T) {
	token := NewTestJwtToken(t, jwt.MapClaims{
		"exp": time.Now().Add(-1 * time.Second).Unix(),
	})
	parser := NewTestJwtParser(t)
	parsed, err := parser.Parse(token)

	require.Error(t, err)
	require.Nil(t, parsed)
}

func NewTestJwtParser(t *testing.T) *JwtParser {
	secret := "secret"
	parser, err := NewJwtParser(secret)

	require.NoError(t, err)
	require.NotNil(t, parser)

	return parser
}

func NewTestJwtToken(t *testing.T, claims jwt.Claims) string {
	secret := "secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))

	require.NoError(t, err)
	require.NotEmpty(t, signed)

	return signed
}
