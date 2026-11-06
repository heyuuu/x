# x

Go 实验包总仓库。

仓库里每个顶层目录（`cmd/` 除外）都是一个独立的 Go 模块，各自拥有自己的 `go.mod`，互不依赖，可以单独引用（`xstd` 例外地为单模块多子包）：

| 目录 | 包 | 说明 |
| --- | --- | --- |
| `di` | `di` | 依赖注入 |
| `store` | `store` | 文件存储通用原语：原子写、JSON 整文件 / 分节读写、Store 接口 |
| `jsonl` | `jsonl` | JSONL 行式读写：Append 追加写、Write 原子覆盖写、Each / ReverseEach 逐行读 |
| `finder` | `finder` | 目录遍历，返回惰性 `iter.Seq2[FileEntry, error]` |
| `gitignore` | `gitignore` | .gitignore 规则解析与匹配（静态 / 自动加载 Matcher） |
| `xstd` | `xstd/xmaps` 等 | 标准库扩展，单模块多子包，详见 [xstd/README](./xstd/README.md) |

## 版本策略

- 包版本只会是 `v0.x.y`，永远不会到 `v1`。一个包一旦成熟，就应该离开本仓库，成为独立的包。
- `patch` 版本保证向后兼容。
- `minor` 版本不保证向后兼容，升级前请查看变更。

## 引用方式

各包为独立模块，直接按目录路径引用即可，例如：

```bash
go get github.com/heyuuu/x/store
```
