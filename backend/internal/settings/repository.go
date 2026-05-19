package settings

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

// NewRepository 用于执行 NewRepository 相关后端逻辑。
// 输入参数：db 表示数据库连接。
// 输出参数：返回 *Repository。
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// All 用于读取全部设置项。
// 输入参数：无。
// 输出参数：返回 map[string]string, error；error 表示执行失败原因。
func (r *Repository) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

// Get 用于根据请求参数返回单条数据。
// 输入参数：key 表示键名。
// 输出参数：返回 string, error；error 表示执行失败原因。
func (r *Repository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// Set 用于写入单个设置项。
// 输入参数：key 表示键名；value 表示输入值。
// 输出参数：返回 error；error 表示执行失败原因。
func (r *Repository) Set(key, value string) error {
	now := time.Now().Unix()
	_, err := r.db.Exec(`INSERT INTO settings (key, value, created_at, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, key, value, now, now)
	return err
}

// SetMany 用于批量写入设置项。
// 输入参数：values 表示values 参数。
// 输出参数：返回 error；error 表示执行失败原因。
func (r *Repository) SetMany(values map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	for key, value := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value, created_at, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, key, value, now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
