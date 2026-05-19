package connections

import (
	"database/sql"
	"time"

	"slatessh/backend/internal/models"
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

// List 用于返回列表数据。
// 输入参数：无。
// 输出参数：返回 []models.Connection, error；error 表示执行失败原因。
func (r *Repository) List() ([]models.Connection, error) {
	rows, err := r.db.Query(`SELECT id, name, type, host, port, username, auth_method, encrypted_password, encrypted_private_key, encrypted_passphrase, notes, last_connected_at, created_at, updated_at FROM connections ORDER BY COALESCE(last_connected_at, 0) DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Connection
	for rows.Next() {
		item, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// Find 用于根据标识查询数据记录。
// 输入参数：id 表示标识符。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
func (r *Repository) Find(id int64) (*models.Connection, error) {
	row := r.db.QueryRow(`SELECT id, name, type, host, port, username, auth_method, encrypted_password, encrypted_private_key, encrypted_passphrase, notes, last_connected_at, created_at, updated_at FROM connections WHERE id = ?`, id)
	item, err := scanConnection(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Create 用于创建新数据记录。
// 输入参数：input 表示连接输入参数。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
func (r *Repository) Create(input models.Connection) (*models.Connection, error) {
	now := time.Now().Unix()
	result, err := r.db.Exec(`INSERT INTO connections (name, type, host, port, username, auth_method, encrypted_password, encrypted_private_key, encrypted_passphrase, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, input.Name, input.Type, input.Host, input.Port, input.Username, input.AuthMethod, input.EncryptedPassword, input.EncryptedKey, input.EncryptedPhrase, input.Notes, now, now)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.Find(id)
}

// Update 用于更新已有数据记录。
// 输入参数：id 表示标识符；input 表示连接输入参数。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
func (r *Repository) Update(id int64, input models.Connection) (*models.Connection, error) {
	_, err := r.db.Exec(`UPDATE connections SET name = ?, type = ?, host = ?, port = ?, username = ?, auth_method = ?, encrypted_password = ?, encrypted_private_key = ?, encrypted_passphrase = ?, notes = ?, updated_at = ? WHERE id = ?`, input.Name, input.Type, input.Host, input.Port, input.Username, input.AuthMethod, input.EncryptedPassword, input.EncryptedKey, input.EncryptedPhrase, input.Notes, time.Now().Unix(), id)
	if err != nil {
		return nil, err
	}
	return r.Find(id)
}

// Delete 用于删除指定数据记录。
// 输入参数：id 表示标识符。
// 输出参数：返回 error；error 表示执行失败原因。
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM connections WHERE id = ?`, id)
	return err
}

// TouchLastConnected 用于更新连接最近使用时间。
// 输入参数：id 表示标识符。
// 输出参数：返回 error；error 表示执行失败原因。
func (r *Repository) TouchLastConnected(id int64) error {
	_, err := r.db.Exec(`UPDATE connections SET last_connected_at = ?, updated_at = ? WHERE id = ?`, time.Now().Unix(), time.Now().Unix(), id)
	return err
}

type scanner interface{ Scan(dest ...any) error }

// scanConnection 用于从数据库行扫描连接记录。
// 输入参数：row 表示row 参数。
// 输出参数：返回 models.Connection, error；error 表示执行失败原因。
func scanConnection(row scanner) (models.Connection, error) {
	var item models.Connection
	var createdAt int64
	var updatedAt int64
	var lastConnected sql.NullInt64
	err := row.Scan(&item.ID, &item.Name, &item.Type, &item.Host, &item.Port, &item.Username, &item.AuthMethod, &item.EncryptedPassword, &item.EncryptedKey, &item.EncryptedPhrase, &item.Notes, &lastConnected, &createdAt, &updatedAt)
	if err != nil {
		return models.Connection{}, err
	}
	item.CreatedAt = time.Unix(createdAt, 0)
	item.UpdatedAt = time.Unix(updatedAt, 0)
	if lastConnected.Valid {
		ts := time.Unix(lastConnected.Int64, 0)
		item.LastConnectedAt = &ts
	}
	return item, nil
}
