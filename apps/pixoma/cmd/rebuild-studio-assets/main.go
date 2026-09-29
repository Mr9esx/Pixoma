package main

import (
	"flag"
	"fmt"
	"os"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

var studioDataTables = []string{
	"studio_project_asset_tags",
	"studio_asset_version_palettes",
	"studio_asset_library_preferences",
	"studio_blob_write_intents",
	"studio_session_asset_usages",
	"studio_project_assets",
	"studio_asset_categories",
	"studio_asset_tags",
	"studio_library_assets",
	"studio_library_categories",
	"studio_library_folders",
	"studio_flow_edges",
	"studio_flow_nodes",
	"studio_asset_versions",
	"studio_assets",
	"studio_workflow_executions",
	"studio_context_events",
	"studio_context_requests",
	"studio_run_checkpoints",
	"studio_run_progress",
	"studio_events",
	"studio_approvals",
	"studio_clarifications",
	"studio_runs",
	"studio_messages",
	"studio_sessions",
	"studio_projects",
}

func main() {
	driver := flag.String("driver", "", "数据库驱动：sqlite、mysql 或 postgres")
	dsn := flag.String("dsn", "", "业务数据库连接信息")
	confirm := flag.Bool("confirm-clear-studio-data", false, "确认清除 Studio 项目、对话和资产数据")
	flag.Parse()
	if !*confirm || *dsn == "" {
		fmt.Fprintln(os.Stderr, "需要 --dsn 和 --confirm-clear-studio-data；执行前请停止服务并备份数据库")
		os.Exit(2)
	}
	gdb, err := db.Open(db.Options{Driver: *driver, DSN: *dsn})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer sqlDB.Close()
	if err := rebuildStudioData(gdb); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Studio 项目、对话和资产数据结构已重建")
}

func rebuildStudioData(gdb *gorm.DB) error {
	for _, table := range studioDataTables {
		if gdb.Migrator().HasTable(table) {
			if err := gdb.Migrator().DropTable(table); err != nil {
				return fmt.Errorf("删除 %s: %w", table, err)
			}
		}
	}
	models := []any{
		&persistence.ProjectRow{}, &persistence.SessionRow{}, &persistence.MessageRow{},
		&persistence.RunRow{}, &persistence.RunProgressRow{}, &persistence.CheckpointRow{},
		&persistence.EventRow{}, &persistence.ContextRequestRow{}, &persistence.ContextEventRow{},
		&persistence.ApprovalRow{}, &persistence.ClarificationRow{}, &persistence.WorkflowExecutionRow{},
		&persistence.AssetRow{}, &persistence.AssetVersionRow{}, &persistence.ProjectAssetRow{},
		&persistence.SessionAssetUsageRow{}, &persistence.AssetCategoryRow{}, &persistence.AssetTagRow{},
		&persistence.ProjectAssetTagRow{}, &persistence.AssetVersionPaletteRow{},
		&persistence.AssetLibraryPreferencesRow{}, &persistence.BlobWriteIntentRow{},
		&persistence.FlowNodeRow{}, &persistence.FlowEdgeRow{},
	}
	return db.AutoMigrate(gdb, models...)
}
