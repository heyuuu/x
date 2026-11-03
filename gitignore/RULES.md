# gitignore 规则参考

规则整理自 Git 官方文档 gitignore(5)，用于本包 pattern 匹配实现的依据。

来源（原始信息）：

- 官方 man page（HTML）：<https://git-scm.com/docs/gitignore>
- man page 源文件（AsciiDoc，随 Git 源码发布）：<https://git.kernel.org/pub/scm/git/git.git/plain/Documentation/gitignore.adoc>
- 通配符语义引用：fnmatch(3)，`FNM_PATHNAME` 模式（man page 中明确引用）

## 概述

> A gitignore file specifies intentionally untracked files that Git should ignore.
> Files already tracked by Git are not affected.

- gitignore 文件指定 Git 应忽略的**未跟踪**文件；**已被跟踪的文件不受影响**。
- 每一行是一个 pattern。
- 同一优先级内，**最后匹配的 pattern 决定结果**（last matching pattern decides the outcome）。

## 规则来源与优先级

优先级从高到低：

1. 命令行传入的 pattern（支持该方式的命令）。
2. 路径所在目录及各级父目录（直到工作树顶层）的 `.gitignore` 文件；
   下层目录文件中的 pattern 覆盖上层。pattern 相对于该 `.gitignore` 所在目录进行匹配。
3. `$GIT_COMMON_DIR/info/exclude`。
4. 配置变量 `core.excludesFile` 指定的文件。
   默认为 `$XDG_CONFIG_HOME/git/ignore`；若 `$XDG_CONFIG_HOME` 未设置或为空，则用 `$HOME/.config/git/ignore`。

工作树之外来源（第 3、4 级，如 `info/exclude`、`core.excludesFile`）的 pattern
**视为位于工作树根目录**（"treated as if they are specified at the root of the working tree"）。

## Pattern 格式

| 规则 | 说明 |
| --- | --- |
| 空行 | 不匹配任何文件，仅作可读性分隔 |
| `#` 开头 | 注释行；要匹配字面 `#` 开头的 pattern，需转义为 `\#` |
| 行尾空格 | 默认被忽略；用 `\` 转义的行尾空格保留 |
| `!` 前缀 | 否定模式（negate）：使被先前 pattern 排除的文件重新包含 |
| `/` | 目录分隔符，可出现在 pattern 的开头、中间或结尾 |
| `/` 在开头或中间 | pattern 锚定于 `.gitignore` 所在目录（只在该目录层级匹配） |
| 无 `/`，或 `/` 仅在结尾 | 可匹配 `.gitignore` 所在目录下**任意层级**的同名路径 |
| `/` 在结尾 | 只匹配目录；否则文件和目录都匹配 |
| `*` | 匹配除 `/` 外的任意内容 |
| `?` | 匹配单个非 `/` 字符 |
| `[a-zA-Z]` | 范围记法，匹配范围内单个字符（fnmatch(3) `FNM_PATHNAME` 语义） |
| `\` | 转义任意字符：`\*` 匹配字面 `*`，`\a` 匹配 `a` |

关键细节：

- **否定模式的限制**（原文：It is not possible to re-include a file if a parent
  directory of that file is excluded）：
  若某文件的父目录已被排除，则无法用 `!` 重新包含该文件。
  Git 出于性能不会列出（遍历）被排除的目录，因此其中任何文件上的 pattern 都不生效，
  无论 pattern 定义在何处。
- 字面 `!` 用 `\!` 转义，如 `\!important!.txt`。
- **以 `\` 结尾的 pattern 是非法 pattern，永不匹配任何内容**。
- 尾部 `/` 的匹配示例（原文示例）：
  `doc/frotz/` 匹配 `doc/frotz` 目录，但不匹配 `a/doc/frotz`（因为中间有 `/`，已锚定）；
  `frotz/` 可同时匹配 `frotz` 和 `a/frotz`（无开头/中间 `/`，任意层级）。

## `**` 双星号规则

对全路径名匹配时，连续两个星号 `**` 有特殊含义：

| 形式 | 含义 | 示例 |
| --- | --- | --- |
| `**/` 开头 | 在所有目录下匹配 | `**/foo` 等价于 `foo`，匹配任意位置的文件/目录 `foo`；`**/foo/bar` 匹配任意 `foo` 目录直接下一层的 `bar` |
| `/**` 结尾 | 匹配其内部所有内容（无限深度） | `abc/**` 匹配 `abc` 目录下所有文件 |
| `/**/` 中间 | 匹配零个或多个目录 | `a/**/b` 匹配 `a/b`、`a/x/b`、`a/x/y/b` |
| 其他连续 `*` | 按普通 `*` 处理 | 如 `***` 等同单个 `*` 的语义叠加 |
