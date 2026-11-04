package finder

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"slices"
)

// ErrRelativePath 表示传入 Find 的根不是绝对路径。只支持绝对路径。
var ErrRelativePath = errors.New("root must be an absolute path")

var vcsDirs = map[string]bool{
	".git": true,
	".hg":  true,
	".svn": true,
}

// Find 返回遍历 root 得到的序列（不包含根目录本身，默认包含文件和目录，
// 可用 OptionFiles / OptionDirectories 收窄）。root 只支持绝对路径，相对路径会产出 ErrRelativePath。
// 序列是惰性的，只在迭代时执行；调用方自行决定排序、去重、过滤（如按 .gitignore 规则）等后续处理。
//
// 产出的路径（Name/AbsPath/RelPath）一律是遍历到达时的逻辑路径：root 按传入值（仅做 Clean），
// 符号链接按链接自身的路径，不解析为目标路径；root 是符号链接时同样以传入值为前缀。
//
// 遇到的错误（目录不存在、不可读等）以 nil FileEntry 携带 error 产出到序列中，
// 产出一个错误后继续遍历，不会中断整个序列。
func Find(root string, opts ...Option) iter.Seq2[FileEntry, error] {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return func(yield func(FileEntry, error) bool) {
		find(root, &options, yield)
	}
}

// FindDirs 只产出目录，等价于 Find 追加 OptionDirectories()。
func FindDirs(root string, opts ...Option) iter.Seq2[FileEntry, error] {
	return Find(root, append(slices.Clone(opts), OptionDirectories())...)
}

// FindFiles 只产出文件，等价于 Find 追加 OptionFiles()。
func FindFiles(root string, opts ...Option) iter.Seq2[FileEntry, error] {
	return Find(root, append(slices.Clone(opts), OptionFiles())...)
}

func find(root string, opts *options, yield func(FileEntry, error) bool) bool {
	if !filepath.IsAbs(root) {
		return yield(nil, fmt.Errorf("%w: %s", ErrRelativePath, root))
	}
	root = filepath.Clean(root)

	// followLinks 下按解析后的真实路径记录当前递归路径上的目录，防止符号链接成环导致无限递归；
	// 仅禁止回到祖先，并行到达同一目录的不同路径都会正常遍历。真实路径不对外产出。
	var visited map[string]bool
	if opts.followLinks {
		visited = map[string]bool{}
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			visited[resolved] = true
		}
	}

	// walk 遍历 path 的直接子条目，返回是否继续。
	var walk func(path string) bool
	walk = func(path string) bool {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return yield(nil, err)
		}

		for _, d := range dirEntries {
			full := filepath.Join(path, d.Name())

			isDir := d.IsDir()
			// 跟随符号链接时条目按目标类型分类（目录则继续下钻），断链无法解析则按链接本身的条目产出；
			// 产出的路径始终是链接自身的路径。
			if opts.followLinks && d.Type()&fs.ModeSymlink != 0 {
				if info, statErr := os.Stat(full); statErr == nil {
					isDir = info.IsDir()
				}
			}

			rel, relErr := filepath.Rel(root, full)
			if relErr != nil {
				return yield(nil, relErr)
			}
			entry := makeEntry(full, rel, isDir)

			if !checkFilter(opts, entry) {
				// 目录被过滤时整棵子树一起跳过（不再下钻）。
				continue
			}
			if opts.kind.matches(entry) {
				if !yield(entry, nil) {
					return false
				}
			}

			if !isDir {
				continue
			}
			if !checkDepth(opts, entry.Depth()+1) {
				continue
			}
			if opts.followLinks {
				realPath, evalErr := filepath.EvalSymlinks(full)
				if evalErr != nil || visited[realPath] {
					continue
				}
				visited[realPath] = true
				if !walk(full) {
					return false
				}
				delete(visited, realPath)
			} else if !walk(full) {
				return false
			}
		}
		return true
	}

	return walk(root)
}

// 判断是否允许此深度
func checkDepth(opts *options, depth int) bool {
	return opts.maxDepth < 0 || opts.maxDepth >= depth
}

// 判断是否过滤，返回 true 为保留
func checkFilter(opts *options, e FileEntry) bool {
	// 过滤 vcs 目录
	if e.IsDir() && opts.ignoreVCS && vcsDirs[e.Name()] {
		return false
	}

	// 过滤深度过深
	if !checkDepth(opts, e.Depth()) {
		return false
	}

	// 使用自定义 filter 过滤
	for _, filter := range opts.filters {
		if !filter(e) {
			return false
		}
	}

	return true
}
