package api

import (
	"testing"

	db "github.com/LamThanhNguyen/future-bank/db/sqlc"
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/stretchr/testify/require"
)

func randomDistributorUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(10)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
		Role:           util.DepositorRole,
	}
	return
}

func randomBankerUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(10)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
		Role:           util.BankerRole,
	}
	return
}
