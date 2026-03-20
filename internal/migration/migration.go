// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Run 执行数据库迁移
// 对于新数据库：运行所有迁移（建表+种子数据）
// 对于已有数据库：自动 baseline 后只运行增量迁移
func Run(dsn string) error {
	dsn = normalizeMigrationDSN(dsn)

	// 打开原生 database/sql 连接（golang-migrate 需要）
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("打开数据库连接失败: %w", err)
	}
	defer db.Close()

	// 处理 baseline：已有旧数据库但没有 schema_migrations 表
	if err := handleBaseline(db); err != nil {
		return fmt.Errorf("baseline 处理失败: %w", err)
	}

	// 创建迁移源（从 embed.FS 读取）
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("创建迁移源失败: %w", err)
	}

	// 创建 MySQL 驱动
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("创建数据库驱动失败: %w", err)
	}

	// 创建迁移实例
	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return fmt.Errorf("创建迁移实例失败: %w", err)
	}

	// 执行所有未执行的迁移
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %w", err)
	}

	version, dirty, _ := m.Version()
	fmt.Printf("[迁移] 数据库版本: %d, dirty: %v\n", version, dirty)

	return nil
}

// normalizeMigrationDSN 为迁移连接补充 TiDB 兼容参数。
// golang-migrate 的 mysql 驱动在 SetVersion 时会以 SERIALIZABLE 开事务，
// TiDB 默认会拒绝该隔离级别，因此这里仅对迁移连接追加跳过检查参数。
func normalizeMigrationDSN(dsn string) string {
	if strings.Contains(dsn, "tidb_skip_isolation_level_check=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "tidb_skip_isolation_level_check=1"
}

// handleBaseline 处理已有数据库的 baseline
// 如果检测到 sys_user 表存在但 schema_migrations 不存在，
// 说明是从旧版本升级，自动标记为 v1（跳过初始建表迁移）
func handleBaseline(db *sql.DB) error {
	// 检查 schema_migrations 表是否存在
	var smCount int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'schema_migrations'").Scan(&smCount)
	if err != nil {
		return err
	}
	if smCount > 0 {
		return nil // schema_migrations 已存在，不需要 baseline
	}

	// 检查 sys_user 表是否存在（判断是否是旧数据库）
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'sys_user'").Scan(&userCount)
	if err != nil {
		return err
	}
	if userCount == 0 {
		return nil // 全新数据库，不需要 baseline
	}

	// 旧数据库：创建 schema_migrations 并标记 v1 已完成
	fmt.Println("[迁移] 检测到已有数据库，执行 baseline（标记 v1 已完成）")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `schema_migrations` (`version` bigint NOT NULL PRIMARY KEY, `dirty` boolean NOT NULL)")
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO `schema_migrations` (`version`, `dirty`) VALUES (1, false)")
	if err != nil {
		return err
	}

	return nil
}
