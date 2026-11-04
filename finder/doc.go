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
package finder
