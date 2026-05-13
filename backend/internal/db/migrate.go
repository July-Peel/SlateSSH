package db

import (
    "database/sql"
    "time"
)

func Migrate(db *sql.DB) error {
    statements := []string{
        `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, hashed_password TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);`,
        `CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);`,
        `CREATE TABLE IF NOT EXISTS connections (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, type TEXT NOT NULL DEFAULT 'SSH', host TEXT NOT NULL, port INTEGER NOT NULL, username TEXT NOT NULL, auth_method TEXT NOT NULL, encrypted_password TEXT, encrypted_private_key TEXT, encrypted_passphrase TEXT, notes TEXT, last_connected_at INTEGER, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);`,
        `CREATE TABLE IF NOT EXISTS transfer_tasks (id TEXT PRIMARY KEY NOT NULL, payload TEXT NOT NULL, status TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);`,
    }

    for _, statement := range statements {
        if _, err := db.Exec(statement); err != nil {
            return err
        }
    }

    now := time.Now().Unix()
    defaults := map[string]string{
        "language": "zh-CN",
        "autoCopyOnSelect": "true",
        "terminalEnableRightClickPaste": "true",
        "terminalScrollbackLimit": "5000",
        "showServerStatusWidget": "true",
    }

    for key, value := range defaults {
        if _, err := db.Exec(`INSERT OR IGNORE INTO settings (key, value, created_at, updated_at) VALUES (?, ?, ?, ?)`, key, value, now, now); err != nil {
            return err
        }
    }

    return nil
}
