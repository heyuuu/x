// Package gitignore 提供 .gitignore 规则的解析与匹配，
// 规则依据 gitignore(5) 与 fnmatch(3)，整理后的规则说明见包内 RULES.md。
//
// 规则以文本传入：NewStaticMatcher 接受固定规则文本；NewAutoloadMatcher
// 在判定时经 FS 按需加载路径各级祖先目录的 .gitignore 并缓存。
// 规则解析与匹配均为包内实现，不导出中间表示。
//
// # 路径表示
//
// 判定用路径统一为相对忽略判定根（如工作树根）的 "/" 分隔字符串，根为空串。
// Match 入口统一规范化：无前导 "./" 或 "/"、无尾随 "/"、无空段；前导 "/"
// 视为相对根。越过判定根的路径（如 "../x"）不在判定范围内，直接返回不忽略
// （git check-ignore 对仓库外路径同样拒绝）。不转换反斜杠——gitignore 语法中
// "\" 是转义符。FS 实现收到的均为规范路径（LocalFS 在 Windows 上以
// filepath.FromSlash 翻译给操作系统）。
//
// # 匹配语义
//
// AutoloadMatcher 的 Match 会先自根向叶逐级判定祖先目录（均按目录语义），
// 任一祖先被排除即整体忽略，等价 git check-ignore：父目录被排除后，! 否定模式
// 无法重新包含其子孙。每一级只对该级路径自身求值，因此 ! 只重新包含命中的路径
// 自身、不连带其子孙（如 "*" 加 "!*/"：目录被重新包含，其中的文件仍被 "*" 排除）。
// StaticMatcher 只判定路径本身，祖先目录的忽略需调用方自行逐级判定；作为近似，
// pattern 命中路径的祖先段时视为整棵子树命中。
//
// # 用法示例
//
//	static := gitignore.NewStaticMatcher("", "*.log\n!important.log\n")
//	auto := gitignore.NewAutoloadMatcher(gitignore.LocalFS{Root: worktreeRoot}, globalIgnores)
//	if auto.Match("cmd/x/main.go", false) {
//		// 被忽略
//	}
package gitignore
