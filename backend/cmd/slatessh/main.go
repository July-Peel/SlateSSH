package main

import (
	"log"

	"slatessh/backend/internal/app"
)

// main 用于作为程序入口启动 SlateSSH 服务。
// 输入参数：无。
// 输出参数：无。
func main() {
	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
