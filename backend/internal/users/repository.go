package users

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

// Count 用于统计数据记录数量。
// 输入参数：无。
// 输出参数：返回 int, error；error 表示执行失败原因。
func (r *Repository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// Create 用于创建新数据记录。
// 输入参数：username 表示用户名；hashedPassword 表示hashedPassword 参数。
// 输出参数：返回 *models.User, error；error 表示执行失败原因。
func (r *Repository) Create(username, hashedPassword string) (*models.User, error) {
	now := time.Now()
	result, err := r.db.Exec(`INSERT INTO users (username, hashed_password, created_at, updated_at) VALUES (?, ?, ?, ?)`, username, hashedPassword, now.Unix(), now.Unix())
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.User{ID: id, Username: username, HashedPassword: hashedPassword, CreatedAt: now, UpdatedAt: now}, nil
}

// FindByUsername 用于根据用户名查询用户记录。
// 输入参数：username 表示用户名。
// 输出参数：返回 *models.User, error；error 表示执行失败原因。
func (r *Repository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	var createdAt int64
	var updatedAt int64
	err := r.db.QueryRow(`SELECT id, username, hashed_password, created_at, updated_at FROM users WHERE username = ?`, username).Scan(&user.ID, &user.Username, &user.HashedPassword, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	return &user, nil
}

// FindByID 用于根据用户 ID 查询用户记录。
// 输入参数：id 表示标识符。
// 输出参数：返回 *models.User, error；error 表示执行失败原因。
func (r *Repository) FindByID(id int64) (*models.User, error) {
	var user models.User
	var createdAt int64
	var updatedAt int64
	err := r.db.QueryRow(`SELECT id, username, hashed_password, created_at, updated_at FROM users WHERE id = ?`, id).Scan(&user.ID, &user.Username, &user.HashedPassword, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	return &user, nil
}

// UpdatePassword 用于更新指定用户的密码哈希。
// 输入参数：id 表示标识符；hashedPassword 表示hashedPassword 参数。
// 输出参数：返回 error；error 表示执行失败原因。
func (r *Repository) UpdatePassword(id int64, hashedPassword string) error {
	_, err := r.db.Exec(`UPDATE users SET hashed_password = ?, updated_at = ? WHERE id = ?`, hashedPassword, time.Now().Unix(), id)
	return err
}
