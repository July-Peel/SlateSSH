package settings

import (
    "database/sql"
    "time"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

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

func (r *Repository) Get(key string) (string, error) {
    var value string
    err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
    if err == sql.ErrNoRows {
        return "", nil
    }
    return value, err
}

func (r *Repository) Set(key, value string) error {
    now := time.Now().Unix()
    _, err := r.db.Exec(`INSERT INTO settings (key, value, created_at, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, key, value, now, now)
    return err
}

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
