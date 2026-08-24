# GoServerless 评测说明

本项目是基于Go语言的现代无服务器（Serverless），旨在解决开发者只需在网页端写一段 Go 或 JavaScript 代码片段，点击部署，系统就能自动为其分配一个 HTTP 链接问题，使用了Go、React、Monaco Editor，功能有在线函数管理器、指标监控面版、动态代码编译/沙箱运行、冷启动与容器热池管理。

Go 模块位于 `backend/`。评测入口：在该目录执行 `go test ./...`。
