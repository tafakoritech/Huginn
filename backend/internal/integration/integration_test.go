package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

func mustConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func ts() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func TestAuthLifecycle(t *testing.T) {
	db := setupDB(t)
	cfg := mustConfig(t)

	repo := users.NewRepository(db)
	svc := auth.NewAuthService(auth.NewManager(cfg.Auth), repo)

	hash, _ := passwd.HashPassword("S3cret!pass")
	u, err := repo.Create(context.Background(), &users.User{
		Name: "IT User", Username: "it_auth_" + ts(), Role: rbac.RoleSalesAgent, IsActive: true, PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// login
	res, err := svc.Login(context.Background(), auth.LoginParams{Login: u.Username, Password: "S3cret!pass"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Tokens.AccessToken == "" || res.Tokens.RefreshToken == "" {
		t.Fatal("missing tokens")
	}

	// access token parses and carries the right role
	claims, err := svc.Manager().ParseAccessToken(res.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if claims.UserID != u.ID || claims.Role != rbac.RoleSalesAgent {
		t.Fatalf("bad claims: %+v", claims)
	}

	// refresh rotates
	res2, err := svc.Refresh(context.Background(), res.Tokens.RefreshToken, "", "", "")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if res2.Tokens.RefreshToken == res.Tokens.RefreshToken {
		t.Fatal("refresh token must rotate")
	}

	// old refresh token is dead after rotation
	if _, err := svc.Refresh(context.Background(), res.Tokens.RefreshToken, "", "", ""); err == nil {
		t.Fatal("reused refresh token must be rejected")
	}

	// logout revokes
	if err := svc.Logout(context.Background(), res2.Tokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), res2.Tokens.RefreshToken, "", "", ""); err == nil {
		t.Fatal("logged-out refresh token must be rejected")
	}

	// wrong password
	if _, err := svc.Login(context.Background(), auth.LoginParams{Login: u.Username, Password: "wrong"}); err == nil {
		t.Fatal("wrong password must fail")
	}
}

func TestLeadStateMachinePersistence(t *testing.T) {
	db := setupDB(t)
	svc := leads.NewService(leads.NewRepository(db))

	lead, err := svc.Create(context.Background(), leads.CreateParams{CompanyName: "IT Company " + ts()})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if lead.Status != leads.StatusNew {
		t.Fatalf("expected NEW, got %s", lead.Status)
	}

	if _, err := svc.ChangeStatus(context.Background(), lead.ID, leads.StatusReady, "", ""); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if _, err := svc.ChangeStatus(context.Background(), lead.ID, leads.StatusAssigned, "", ""); err != nil {
		t.Fatalf("assigned: %v", err)
	}

	// invalid transitions must be rejected server-side
	lead2, _ := svc.Create(context.Background(), leads.CreateParams{CompanyName: "IT Company 2 " + ts()})
	if _, err := svc.ChangeStatus(context.Background(), lead2.ID, leads.StatusQualified, "", ""); err == nil {
		t.Fatal("invalid transition NEW->QUALIFIED must be rejected")
	}
}

func TestInteractionMandatoryReport(t *testing.T) {
	db := setupDB(t)
	leadRepo := leads.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	userRepo := users.NewRepository(db)

	agent, err := userRepo.Create(context.Background(), &users.User{
		Name: "IT Agent", Username: "it_agent_" + ts(), Role: rbac.RoleSalesAgent, IsActive: true,
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	lead, err := leadRepo.Create(context.Background(), &leads.Lead{CompanyName: "IT Contact " + ts(), Status: leads.StatusNew})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	inter, err := interRepo.Create(context.Background(), &interactions.Interaction{
		LeadID: lead.ID, UserID: agent.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCreated,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction: %v", err)
	}
	if inter.ReportStatus != interactions.ReportPending {
		t.Fatalf("expected PENDING report, got %s", inter.ReportStatus)
	}
}

func TestMinioRoundtrip(t *testing.T) {
	store := setupMinIO(t)
	ctx := context.Background()

	content := "hello huginn storage"
	obj, err := store.Put(ctx, strings.NewReader(content), int64(len(content)), "text/plain", "test/integration_"+ts()+".txt")
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if obj.Size == 0 || obj.Checksum == "" {
		t.Fatal("missing object metadata")
	}
	rc, size, err := store.Get(ctx, obj.Key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	buf := make([]byte, size)
	n, _ := rc.Read(buf)
	if string(buf[:n]) != content {
		t.Fatalf("roundtrip mismatch: %q", string(buf[:n]))
	}
	url, err := store.PresignedGetURL(ctx, obj.Key)
	if err != nil || url == "" {
		t.Fatalf("presign: %v url=%q", err, url)
	}
}

func TestRedisCache(t *testing.T) {
	c := setupRedis(t)
	ctx := context.Background()
	if err := c.Set(ctx, "it:key", "value", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := c.Get(ctx, "it:key")
	if err != nil || v != "value" {
		t.Fatalf("get: %v %q", err, v)
	}
}
