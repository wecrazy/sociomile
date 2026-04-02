package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type serviceFixture struct {
	ctx                 context.Context
	store               *repository.Store
	authService         *AuthService
	userService         *UserService
	conversationService *ConversationService
	ticketService       *TicketService
	tenantA             model.Tenant
	tenantB             model.Tenant
	adminA              model.User
	agentA              model.User
	adminB              model.User
	agentB              model.User
	channelA            model.Channel
	channelB            model.Channel
	password            string
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "service-test.db")), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.Channel{},
		&model.Customer{},
		&model.Conversation{},
		&model.Message{},
		&model.Ticket{},
		&model.ActivityLog{},
		&model.OutboxEvent{},
	)
	require.NoError(t, err)

	password := "Password123!"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	fixture := &serviceFixture{
		ctx:      context.Background(),
		store:    repository.NewStore(db),
		password: password,
		tenantA:  model.Tenant{BaseModel: model.BaseModel{ID: "tenant-a"}, Name: "Tenant A", Slug: "tenant-a"},
		tenantB:  model.Tenant{BaseModel: model.BaseModel{ID: "tenant-b"}, Name: "Tenant B", Slug: "tenant-b"},
		adminA:   model.User{BaseModel: model.BaseModel{ID: "admin-a"}, TenantID: "tenant-a", Name: "Admin A", Email: "admin-a@example.com", PasswordHash: string(passwordHash), Role: model.RoleAdmin, Active: true},
		agentA:   model.User{BaseModel: model.BaseModel{ID: "agent-a"}, TenantID: "tenant-a", Name: "Agent A", Email: "agent-a@example.com", PasswordHash: string(passwordHash), Role: model.RoleAgent, Active: true},
		adminB:   model.User{BaseModel: model.BaseModel{ID: "admin-b"}, TenantID: "tenant-b", Name: "Admin B", Email: "admin-b@example.com", PasswordHash: string(passwordHash), Role: model.RoleAdmin, Active: true},
		agentB:   model.User{BaseModel: model.BaseModel{ID: "agent-b"}, TenantID: "tenant-b", Name: "Agent B", Email: "agent-b@example.com", PasswordHash: string(passwordHash), Role: model.RoleAgent, Active: true},
		channelA: model.Channel{BaseModel: model.BaseModel{ID: "channel-a"}, TenantID: "tenant-a", Key: "whatsapp", Name: "WhatsApp"},
		channelB: model.Channel{BaseModel: model.BaseModel{ID: "channel-b"}, TenantID: "tenant-b", Key: "whatsapp", Name: "WhatsApp"},
	}

	require.NoError(t, db.Create(&fixture.tenantA).Error)
	require.NoError(t, db.Create(&fixture.tenantB).Error)
	require.NoError(t, db.Create(&fixture.adminA).Error)
	require.NoError(t, db.Create(&fixture.agentA).Error)
	require.NoError(t, db.Create(&fixture.adminB).Error)
	require.NoError(t, db.Create(&fixture.agentB).Error)
	require.NoError(t, db.Create(&fixture.channelA).Error)
	require.NoError(t, db.Create(&fixture.channelB).Error)

	cacheClient := cache.New(nil)
	fixture.authService = NewAuthService(fixture.store, "test-secret", 0)
	fixture.userService = NewUserService(fixture.store)
	fixture.conversationService = NewConversationService(fixture.store, cacheClient)
	fixture.ticketService = NewTicketService(fixture.store, cacheClient)

	return fixture
}
