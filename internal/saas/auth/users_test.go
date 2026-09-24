package auth

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testUserDB(t *testing.T) *store.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	return &store.DB{DB: gdb}
}

func TestRegisterLoginAgentLogin(t *testing.T) {
	db := testUserDB(t)
	authSvc := NewService(&config.Config{JWT: config.JWTConfig{Secret: "test-secret-xyz", ExpireHour: 1}})
	us := NewUserService(db, authSvc)

	u, err := us.Register(RegisterRequest{Email: "User@Example.com", Password: "secret12", DisplayName: "U"})
	require.NoError(t, err)
	require.Equal(t, "user@example.com", u.Email)
	require.Equal(t, "user", u.Role)
	require.NotEqual(t, "secret12", u.PasswordHash)

	_, err = us.Register(RegisterRequest{Email: "user@example.com", Password: "secret12"})
	require.ErrorIs(t, err, ErrUserExists)

	resp, err := us.Login(LoginRequest{Email: "user@example.com", Password: "secret12"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	claims, err := authSvc.ParseToken(resp.Token)
	require.NoError(t, err)
	require.Equal(t, u.ID, claims.UserID)
	require.Equal(t, "user", claims.Role)

	_, err = us.Login(LoginRequest{Email: "user@example.com", Password: "wrong"})
	require.ErrorIs(t, err, ErrInvalidCreds)

	agent, err := us.Register(RegisterRequest{Email: "agent@x.com", Password: "agentpw", Role: "agent"})
	require.NoError(t, err)
	require.Equal(t, "agent", agent.Role)
	aresp, err := us.AgentLogin(LoginRequest{Email: "agent@x.com", Password: "agentpw"})
	require.NoError(t, err)
	require.NotEmpty(t, aresp.Token)

	_, err = us.AgentLogin(LoginRequest{Email: "user@example.com", Password: "secret12"})
	require.ErrorIs(t, err, ErrRoleNotAllowed)
}

func TestHashPassword(t *testing.T) {
	_, err := HashPassword("short")
	require.Error(t, err)
	h, err := HashPassword("longenough")
	require.NoError(t, err)
	require.True(t, CheckPassword(h, "longenough"))
	require.False(t, CheckPassword(h, "nope"))
}
