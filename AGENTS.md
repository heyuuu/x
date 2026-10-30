# AGENTS.md

Go 实验包总仓库（多模块）。开始工作前请先阅读 [README.md](./README.md)。

## 仓库结构

- 每个目录（`cmd/` 除外）都是一个独立的 Go 模块，各自有 `go.mod`，互不依赖，可单独引用。
  - `di/`：依赖注入
  - `store/`：文件存储原语（原子写、JSON 整文件读写、JSONL 行式读写）
- `cmd/x/`：仓库管理 CLI 工具 `x`（由 justfile 封装调用），用于初始化包、批量执行命令、打 tag。
- 根目录 `justfile` 封装所有常用命令。

## 常用命令

- `just lint {包名}`：对指定包执行 `go vet` + `golangci-lint run`
- `just test {包名}` / `just test-all`：跑测试（带 `-race`）
- `just fmt {包名}` / `just fix {包名}`：格式化 / 自动修复 lint 问题
- `just run {包名} <cmd>` / `just run-all <cmd>`：在指定包/所有包目录下执行任意命令
- `just tag {包名}` / `just tag-all`：打 patch 版本 tag

## 约束（必须遵守）

- 代码修改后必须通过 `just lint {包名}`。
- 提交前必须 `just test-all`，所有包测试通过。

## Golang 代码规范

- 项目当前 Go 版本为 1.27.1，尽量使用较新的语言特性和标准库包；例如 JSON 统一使用 `encoding/json/v2`，而非 `encoding/json`。
- 包注释使用独立的 `doc.go` 文件，不写在其他文件的注释里。

## 版本策略

- 包版本只到 `v0.x.y`，不到 v1；包成熟后应迁出本仓库独立。
- `patch` 保证向后兼容，`minor` 不保证。
