package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	_ "github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testAPIServer(t *testing.T) *Server {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	db := &store.DB{DB: gdb}

	cfg := &config.Config{
		AppRole: config.RoleDev,
		JWT:     config.JWTConfig{Secret: "api-test-secret", ExpireHour: 1},
		Server:  config.ServerConfig{HTTPAddr: ":0"},
	}
	authSvc := auth.NewService(cfg)
	userSvc := auth.NewUserService(db, authSvc)
	instSvc := instance.NewService(db)
	require.NoError(t, instSvc.EnsureTemplates())
	hub := ws.NewHub(ws.NewFillBridge(instSvc, zap.NewNop()), zap.NewNop())
	t.Cleanup(hub.Close)

	return &Server{Users: userSvc, Auth: authSvc, Inst: instSvc, Hub: hub, Cfg: cfg, Log: zap.NewNop()}
}

func TestAuthAndInstanceCRUD(t *testing.T) {
	srv := testAPIServer(t)
	h := srv.Routes()

	body := `{"email":"a@b.com","password":"secret12","display_name":"A"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"a@b.com","password":"secret12"}`))
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var login auth.TokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &login))
	require.NotEmpty(t, login.Token)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(`{"template_id":"lunar","symbol":"510300","capital_quota":100000}`))
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var inst store.StrategyInstance
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &inst))
	require.Equal(t, store.InstanceStopped, inst.Status)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+itoa(inst.ID)+"/start", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+itoa(inst.ID)+"/portfolio", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+itoa(inst.ID)+"/trades",
		bytes.NewBufferString(`{"side":"BUY","engine":"MICRO","qty":100}`))
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+itoa(inst.ID)+"/stop", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAgentLoginEndpoint(t *testing.T) {
	srv := testAPIServer(t)
	h := srv.Routes()

	_, err := srv.Users.Register(auth.RegisterRequest{Email: "ag@t.com", Password: "agentpw", Role: "agent"})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/login", bytes.NewBufferString(`{"email":"ag@t.com","password":"agentpw"}`))
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var out map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	require.NotEmpty(t, out["token"])
}

func itoa(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
