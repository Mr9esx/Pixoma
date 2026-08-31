package livedemo

import (
	"encoding/json"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	taskstatspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
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
		&text.Row{},
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
