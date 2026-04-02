package seeds

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadDemoDataIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seeds.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.Channel{},
		&model.Customer{},
		&model.Conversation{},
		&model.Message{},
		&model.Ticket{},
	))

	require.NoError(t, LoadDemoData(context.Background(), db))
	require.NoError(t, LoadDemoData(context.Background(), db))

	requireCount(t, db, &model.Tenant{}, 2)
	requireCount(t, db, &model.User{}, 4)
	requireCount(t, db, &model.Channel{}, 4)
	requireCount(t, db, &model.Customer{}, 2)
	requireCount(t, db, &model.Conversation{}, 2)
	requireCount(t, db, &model.Message{}, 2)
	requireCount(t, db, &model.Ticket{}, 1)

	var ticket model.Ticket
	require.NoError(t, db.First(&ticket, "id = ?", "81111111-1111-1111-1111-111111111111").Error)
	require.Equal(t, model.TicketStatusInProgress, ticket.Status)
	require.Equal(t, model.TicketPriorityHigh, ticket.Priority)

	var conversation model.Conversation
	require.NoError(t, db.First(&conversation, "id = ?", "62222222-2222-2222-2222-222222222222").Error)
	require.Equal(t, model.ConversationStatusAssigned, conversation.Status)
	require.NotNil(t, conversation.AssignedAgentID)
}

func TestLoadDemoDataReturnsDatabaseError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seeds-error.db")), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	err = LoadDemoData(context.Background(), db)
	require.Error(t, err)
}

func TestLoadDemoDataReturnsIntermediateUpsertErrors(t *testing.T) {
	t.Run("missing users table", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seeds-users-missing.db")), &gorm.Config{})
		require.NoError(t, err)
		require.NoError(t, db.AutoMigrate(&model.Tenant{}))

		err = LoadDemoData(context.Background(), db)
		require.Error(t, err)
	})

	t.Run("missing tickets table", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seeds-tickets-missing.db")), &gorm.Config{})
		require.NoError(t, err)
		require.NoError(t, db.AutoMigrate(
			&model.Tenant{},
			&model.User{},
			&model.Channel{},
			&model.Customer{},
			&model.Conversation{},
			&model.Message{},
		))

		err = LoadDemoData(context.Background(), db)
		require.Error(t, err)
	})
}

func requireCount(t *testing.T, db *gorm.DB, value any, expected int64) {
	t.Helper()

	var count int64
	require.NoError(t, db.Model(value).Count(&count).Error)
	require.Equal(t, expected, count)
}
