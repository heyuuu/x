// Package finder 提供目录遍历，返回 iter.Seq2[FileEntry, error] 的惰性序列。
//
// 目录作为首个参数传入（只支持绝对路径），其余行为通过函数式选项配置：
//
//	for entry, err := range finder.Find([]string{absDir}, finder.OptionDepth(2)) {
//		if err != nil { ... }
//		fmt.Println(entry.RelPath)
//	}
package finder
