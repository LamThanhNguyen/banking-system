package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/LamThanhNguyen/future-bank/token"
	"github.com/stretchr/testify/require"
)

func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	role string,
	duration time.Duration,
) {
	token, payload, err := tokenMaker.CreateToken(username, role, duration, token.TokenTypeAccessToken)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}
