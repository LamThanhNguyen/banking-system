package api

import (
	"os"
	"testing"
	"time"

	db "github.com/LamThanhNguyen/banking-system/db/sqlc"
	"github.com/LamThanhNguyen/banking-system/util"
	"github.com/LamThanhNguyen/banking-system/worker"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	m, err := model.NewModelFromString(`
        [request_definition]
        r = sub, obj, act

        [policy_definition]
        p = sub, obj, act

        [policy_effect]
        e = some(where (p.eft == allow))

        [matchers]
        m = true
    `)
	require.NoError(t, err)

	e, err := casbin.NewEnforcer(m)
	require.NoError(t, err)
	return e
}

func newTestServer(
	t *testing.T,
	store db.Store,
	enforcer *casbin.Enforcer,
	taskDistributor worker.TaskDistributor,
) *Server {
	if enforcer == nil {
		enforcer = newTestEnforcer(t)
	}

	config := util.RuntimeConfig{
		Config: util.Config{
			TokenSymmetricKey:    util.RandomString(32),
			AllowedOrigins:       []string{"*"},
			Environment:          "test",
			AccessTokenDuration:  "1m",
			RefreshTokenDuration: "10m",
		},
		AccessTokenDurationParsed:  time.Minute,
		RefreshTokenDurationParsed: 10 * time.Minute,
	}

	server, err := NewServer(config, store, enforcer, taskDistributor)
	require.NoError(t, err)

	server.SetupRouter()

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}
