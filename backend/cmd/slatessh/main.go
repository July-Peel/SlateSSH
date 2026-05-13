package main

import (
    "log"

    "slatessh/backend/internal/app"
)

func main() {
    application, err := app.New()
    if err != nil {
        log.Fatal(err)
    }

    if err := application.Run(); err != nil {
        log.Fatal(err)
    }
}
