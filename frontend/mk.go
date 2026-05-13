package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func main() {
    root := filepath.Join("frontend")
    _ = os.MkdirAll(filepath.Join(root, "assets", "js"), 0o755)
    _ = os.MkdirAll(filepath.Join(root, "assets", "css"), 0o755)
    fmt.Println(root)
}
