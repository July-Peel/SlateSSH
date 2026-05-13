package users

import (
    "database/sql"
    "time"

    "slatessh/backend/internal/models"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Count() (int, error) {
    var count int
    err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
    return count, err
}

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

func (r *Repository) UpdatePassword(id int64, hashedPassword string) error {
    _, err := r.db.Exec(`UPDATE users SET hashed_password = ?, updated_at = ? WHERE id = ?`, hashedPassword, time.Now().Unix(), id)
    return err
}
