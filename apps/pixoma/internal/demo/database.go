package demo

import (
	"encoding/json"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	consolepersist "github.com/Mr9esx/Pixoma/internal/adminusers/infrastructure/persistence"
	casepersist "github.com/Mr9esx/Pixoma/internal/cases/infrastructure/persistence"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	textpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	taskstatspersist "github.com/Mr9esx/Pixoma/internal/stats/infrastructure/persistence"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
	topicpersist "github.com/Mr9esx/Pixoma/internal/topics/infrastructure/persistence"
	userpersist "github.com/Mr9esx/Pixoma/internal/users/infrastructure/persistence"
)

func OpenDatabase() (*gorm.DB, func() error, error) {
	gdb, err := gorm.Open(sqlite.Open("file:pixoma_demo?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	models := []any{
		&casepersist.CaseRow{},
		&userpersist.UserRow{},
		&userpersist.UserExternalIdentityRow{},
		&consolepersist.ConsoleUserRow{},
		&sesspersist.SessionRow{},
		&taskpersist.TaskRow{},
		&taskstatspersist.DailyStatsRow{},
		&taskstatspersist.EdgeDailyStatsRow{},
		&taskstatspersist.ErrorDailyStatsRow{},
		&taskstatspersist.CaseDailyStatsRow{},
		&topicpersist.TopicRow{},
		&channelpersist.ChannelRow{},
		&textpersist.Row{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
		&instpersist.EdgeRow{},
		&instpersist.MetricsRow{},
	}
	if err := gdb.AutoMigrate(models...); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return gdb, func() error { return sqlDB.Close() }, nil
}

func demoTime() time.Time {
	return time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
}

func demoEncryptionKey() []byte {
	return []byte("01234567890123456789012345678901")
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
