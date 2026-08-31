package livedemo

import (
	"context"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	edgedomain "github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	taskstatspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func Seed(ctx context.Context, gdb *gorm.DB) error {
	now := demoTime()

	if err := gdb.WithContext(ctx).Create([]topicpersist.TopicRow{
		{Key: "demo-render", Name: "渲染队列", Enabled: true, CreatedAt: now, UpdatedAt: now},
		{Key: "demo-review", Name: "人工复核", Enabled: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	hardware := mustJSON(edgedomain.Hardware{
		CPUModel: "Apple M2 Pro", CPUCores: 10, RAMBytes: 34359738368,
		GPUs:        []edgedomain.GPU{{Name: "Apple M2 Pro GPU", VRAMBytes: 17179869184}},
		CollectedAt: now,
	})
	startedAt := now.Add(-2 * time.Hour)
	if err := gdb.WithContext(ctx).Create([]instpersist.EdgeRow{
		{ID: "edge-demo-1", Name: "演示节点 A", Description: "可用图片生成节点", Enabled: true, CapabilitiesJSON: mustJSON([]string{"image"}), SubscribeTopicsJSON: mustJSON([]string{"demo-render"}), HardwareJSON: hardware, StartedAt: &startedAt, ComfyVersion: "0.3.40", CreatedAt: now, UpdatedAt: now},
		{ID: "edge-demo-2", Name: "演示节点 B", Description: "视频放大节点", Enabled: true, CapabilitiesJSON: mustJSON([]string{"image", "video"}), SubscribeTopicsJSON: mustJSON([]string{"demo-render"}), HardwareJSON: hardware, ComfyVersion: "0.3.40", CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	metricsJSON := mustJSON(map[string]any{
		"collected_at": now.Add(-time.Minute), "queue_depth": 2, "running": 1,
		"cpu_percent": 38.5, "memory_percent": 61.2, "gpu_percent": 84.0,
	})
	if err := gdb.WithContext(ctx).Create([]instpersist.MetricsRow{
		{EdgeID: "edge-demo-1", MetricsJSON: metricsJSON, CollectedAt: now.Add(-time.Minute)},
		{EdgeID: "edge-demo-2", MetricsJSON: metricsJSON, CollectedAt: now.Add(-2 * time.Minute)},
	}).Error; err != nil {
		return err
	}

	channelCredential, err := channeldomain.EncryptCredential(demoEncryptionKey(), channeldomain.Credential{BotToken: "demo-token"})
	if err != nil {
		return err
	}
	if err := gdb.WithContext(ctx).Create([]channelpersist.ChannelRow{
		{ID: "channel-demo", Platform: "telegram", Name: "演示 Telegram", CredentialCiphertext: channelCredential, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	email := "admin@example.com"
	operatorEmail := "operator@example.com"
	if err := gdb.WithContext(ctx).Create([]consolepersist.ConsoleUserRow{
		{ID: "console-admin", Username: "demo-admin", Email: &email, Nickname: "演示管理员", Role: "admin", Enabled: true, PasswordHash: "demo", CreatedAt: now, UpdatedAt: now},
		{ID: "console-operator", Username: "demo-operator", Email: &operatorEmail, Nickname: "演示运营", Role: "operator", Enabled: true, PasswordHash: "demo", CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	if err := gdb.WithContext(ctx).Create([]userpersist.UserRow{
		{ID: "user-demo-1", Username: "@alice", FirstName: "Alice", LanguageCode: "zh", LastSeenAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "user-demo-2", Username: "@bob", FirstName: "Bob", LanguageCode: "zh", LastSeenAt: now, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}
	if err := gdb.WithContext(ctx).Create([]userpersist.UserExternalIdentityRow{
		{ID: "identity-1", UserID: "user-demo-1", ChannelID: "channel-demo", ExternalUserID: "10001", ProfileJSON: mustJSON(map[string]any{"is_premium": true}), LastSeenAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "identity-2", UserID: "user-demo-2", ChannelID: "channel-demo", ExternalUserID: "10002", ProfileJSON: mustJSON(map[string]any{"is_premium": false}), LastSeenAt: now, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	document := catalogdomain.CaseDocument{
		ID:          1,
		Name:        "产品图精修",
		Description: "上传产品图后生成高质量精修结果。",
		Tags:        []string{"产品", "电商"},
		Categories:  []string{"图片生成"},
		Inputs: []catalogdomain.InputField{
			{Key: "prompt", Type: "string", Required: true, Description: "画面描述"},
			{Key: "image", Type: "image", Required: true, Description: "产品原图"},
		},
		Outputs:     []catalogdomain.OutputField{{Key: "image", Type: "image", Description: "精修图"}},
		InputSchema: map[string]any{"type": "object"},
	}
	if err := gdb.WithContext(ctx).Create([]casepersist.CaseRow{
		{ID: 1, Name: document.Name, TagsJSON: mustJSON(document.Tags), CatsJSON: mustJSON(document.Categories), DocJSON: mustJSON(document), Enabled: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	if err := gdb.WithContext(ctx).Create([]sesspersist.SessionRow{
		{ID: "session-demo-1", UserID: "user-demo-1", ChannelID: "channel-demo", ChatExternalID: "10001", Status: "submitted", CaseID: 1, InputKeysJSON: mustJSON([]string{"prompt", "image"}), DraftJSON: "{}", CreatedAt: now, UpdatedAt: now},
		{ID: "session-demo-2", UserID: "user-demo-2", ChannelID: "channel-demo", ChatExternalID: "10002", Status: "collecting", CaseID: 1, InputKeysJSON: mustJSON([]string{"prompt", "image"}), DraftJSON: "{}", CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	if err := gdb.WithContext(ctx).Create([]taskpersist.TaskRow{
		{ID: "task-demo-1", SessionID: "session-demo-1", CaseID: 1, Status: string(sharedkernel.TaskSucceeded), EdgeID: "edge-demo-1", DispatchTopic: "demo-render", PromptID: "prompt-demo-1", InputPrefix: "demo/1", OutputsJSON: "[]", StartedAt: now.Add(-time.Hour), CompletedAt: now.Add(-50 * time.Minute), CreatedAt: now.Add(-70 * time.Minute), UpdatedAt: now.Add(-50 * time.Minute)},
		{ID: "task-demo-2", SessionID: "session-demo-2", CaseID: 1, Status: string(sharedkernel.TaskRunning), EdgeID: "edge-demo-2", DispatchTopic: "demo-render", PromptID: "prompt-demo-2", InputPrefix: "demo/2", OutputsJSON: "[]", StartedAt: now.Add(-10 * time.Minute), CreatedAt: now.Add(-20 * time.Minute), UpdatedAt: now.Add(-10 * time.Minute)},
		{ID: "task-demo-3", SessionID: "session-demo-2", CaseID: 1, Status: string(sharedkernel.TaskPending), DispatchTopic: "demo-review", InputPrefix: "demo/3", OutputsJSON: "[]", CreatedAt: now.Add(-5 * time.Minute), UpdatedAt: now.Add(-5 * time.Minute)},
	}).Error; err != nil {
		return err
	}

	menu := mcdomain.Menu{
		ID: "demo-menu", Name: "演示主菜单", Columns: 2,
		Items: []mcdomain.MenuItem{
			{ID: "generate", Label: "开始生成", Action: mcdomain.Action{Type: "open_workflow", WorkflowID: "1"}},
			{ID: "tasks", Label: "我的任务", Action: mcdomain.Action{Type: "send_text", Text: "演示任务列表"}},
		},
	}
	if err := gdb.WithContext(ctx).Create([]mencardpersist.MainMenuRow{
		{ChannelID: "channel-demo", DocJSON: mustJSON(menu)},
	}).Error; err != nil {
		return err
	}

	statDate := "2026-08-31"
	if err := gdb.WithContext(ctx).Create([]taskstatspersist.DailyStatsRow{
		{StatDate: statDate, ProcessedCount: 128, SucceededCount: 118, FailedCount: 7, CancelledCount: 3, TotalDurationMS: 934000, TotalQueueMS: 184000, TotalExecMS: 750000, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}
	if err := gdb.WithContext(ctx).Create([]taskstatspersist.EdgeDailyStatsRow{
		{StatDate: statDate, EdgeID: "edge-demo-1", ProcessedCount: 71, SucceededCount: 67, FailedCount: 4, UpdatedAt: now},
		{StatDate: statDate, EdgeID: "edge-demo-2", ProcessedCount: 57, SucceededCount: 51, FailedCount: 3, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}
	if err := gdb.WithContext(ctx).Create([]taskstatspersist.ErrorDailyStatsRow{
		{StatDate: statDate, ErrorCode: "comfy_timeout", Count: 4, UpdatedAt: now},
		{StatDate: statDate, ErrorCode: "input_invalid", Count: 3, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}
	if err := gdb.WithContext(ctx).Create([]taskstatspersist.CaseDailyStatsRow{
		{StatDate: statDate, CaseID: 1, Count: 128, TotalDurationMS: 934000, UpdatedAt: now},
	}).Error; err != nil {
		return err
	}

	return nil
}
