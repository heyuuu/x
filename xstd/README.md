# xstd

标准库扩展：收编那些"应该在标准库提供、但还没提供"的通用快捷函数。

不是 polyfill / shim——Go 是编译期静态语言，没有运行时垫片一说；这里所有函数都是纯增量扩展。按标准库的领域结构组织为若干子包，各自扩充对应的标准库包：

| 目录 | 包 | 扩充 |
| --- | --- | --- |
| `xslices/` | `xslices` | `slices` |
| （根包） | `xstd` | 领域无关的通用助手（`Ptr`、`Must`…），及暂未独立成包的杂项 |

单模块、单版本线，按子包单独引用即可：

```bash
go get github.com/heyuuu/x/xstd/xslices
```

```go
import (
	"slices"

	"github.com/heyuuu/x/xstd/xslices"
)

slices.Sort(ids)        // 标准库
uniq := xslices.Unique(ids)
```

## 命名原则

- 机械规则：标准库包 `P` 对应的扩展包命名为 `xP`——`slices` → `xslices`、`maps` → `xmaps`、`strings` → `xstrings`，无需猜测。根包 `xstd` 即该规则应用于 `P = std`。
- 用前缀而非后缀（`xslices` 而非 `slicesx`）：人眼阅读锚定词首，`xslices.Map` 与 `slices.Map` 在第一个字符就分叉，一眼可辨；`slicesx` 与 `slices` 词形几乎相同，扫读时极易被当成标准库调用。
- 函数名不重复包名已携带的信息：`xslices.Map`，而非 `xslices.SliceMap`。
- API 风格向标准库看齐：命名习惯、错误处理、迭代器风格（`iter.Seq`）等，如无必要不自造术语。

## 收录原则

- 只收"标准库级别"的通用函数；领域特定、业务相关的工具不收。
- 零第三方依赖，只用标准库，保证随时可被无负担地引入。
- 候补标准库：新增函数前先确认当前 Go 版本确实没有等价能力；标准库将来补齐后，以 `// Deprecated:` 指回标准库实现，参照 `golang.org/x/exp` 的玩法。
- 优先采用较新的语言特性与标准库设施（泛型、`iter.Seq` 等）。

## 组织原则

- 单模块多子包：所有领域共用 `go.mod` 与一条版本线，避免每个领域独立成模块造成版本碎片化。
- 不预建空包：某领域函数还很少时先放根包，攒够了再独立成 `xP` 子包。

## 版本策略

见[仓库根 README](../README.md)。
