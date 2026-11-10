// Package finder 提供目录遍历，返回 iter.Seq2[FileEntry, error] 的惰性序列。
//
// 遍历根作为首个参数传入（只支持绝对路径），其余行为通过函数式选项配置：
//
//	for entry, err := range finder.Find(absDir, finder.OptionDepth(2)) {
//		if err != nil { ... }
//		fmt.Println(entry.RelPath())
//	}
//
// FindFiles / FindDirs 是只产出文件 / 目录的快捷函数。
//
// # 按 .gitignore 过滤
//
// OptionIgnore 接受任意忽略匹配器，命中的目录整棵子树剪掉、不会读取。本包不依赖
// 具体的规则实现；github.com/heyuuu/x/gitignore 的匹配器与 IgnoreMatcher 形状一致，
// 可直接传入：
//
//	matcher := gitignore.NewAutoloadMatcher(gitignore.LocalFS{Root: absDir})
//	for entry, err := range finder.Find(absDir, finder.OptionIgnore(matcher)) { ... }
//
// 匹配器收到的是相对遍历根的路径，规则的判定根须与遍历根一致（如上例二者同为
// absDir），遍历根以上各级目录的 .gitignore 不参与判定；遍历根是工作树的子目录、
// 而要应用上层规则时，改用 OptionFilter 把 AbsPath 换算为相对判定根的路径。
// 过滤只依据规则本身，不查看 git 索引：已被 git 跟踪但命中规则的文件同样被跳过。
package finder
