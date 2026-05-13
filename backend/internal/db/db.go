package db

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", path))
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(1)
    db.SetConnMaxLifetime(time.Minute)
    return db, nil
}
