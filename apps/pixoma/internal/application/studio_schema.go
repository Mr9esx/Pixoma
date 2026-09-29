package application

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

func requireStudioAssetSchema(_ context.Context, gdb *gorm.DB) error {
	if gdb.Migrator().HasTable("studio_library_assets") ||
		gdb.Migrator().HasTable("studio_library_categories") ||
		gdb.Migrator().HasTable("studio_library_folders") ||
		gdb.Migrator().HasColumn("studio_assets", "session_id") {
		return fmt.Errorf("Studio 资产结构需要重建：停止服务并备份业务数据库后，运行 rebuild-studio-assets 命令")
	}
	return nil
}
