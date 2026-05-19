package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Open 用于打开 SQLite 数据库连接。
// 输入参数：path 表示路径。
// 输出参数：返回 *sql.DB, error；error 表示执行失败原因。
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", path))
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Minute)
	return db, nil
}
