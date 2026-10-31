package finder

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

// FileEntry 表示遍历到的一个文件或目录：组合了 fs.DirEntry（Name/IsDir/Type/Info），
// 并附加了相对遍历根与绝对路径。
type FileEntry interface {
	fs.DirEntry
	AbsPath() string
	RelPath() string
	Depth() int
}

// fileEntry 是 FileEntry 的实现，包装遍历中拿到的 fs.DirEntry。
type fileEntry struct {
	fs.DirEntry
	absPath string
	relPath string
	depth   int
}

func (e fileEntry) AbsPath() string { return e.absPath }
func (e fileEntry) RelPath() string { return e.relPath }
func (e fileEntry) Depth() int      { return e.depth }

var _ FileEntry = fileEntry{}

// ErrRelativePath 表示传入 Find 的目录不是绝对路径。只支持绝对路径。
var ErrRelativePath = errors.New("dir must be an absolute path")

var vcsDirs = map[string]bool{
	".git": true,
	".hg":  true,
	".svn": true,
}

// Find 返回遍历 dirs 得到的序列（不包含根目录本身，默认包含文件和目录，
// 可用 OptionFiles / OptionDirectories 收窄）。dirs 只支持绝对路径，相对路径会产出 ErrRelativePath。
// 序列是惰性的，只在迭代时执行；调用方自行决定排序、去重、过滤（如按 .gitignore 规则）等后续处理。
//
// 遇到的错误（目录不存在、不可读等）以 nil FileEntry 携带 error 产出到序列中，
// 产出一个错误后继续遍历后续目录，不会中断整个序列。
func Find(dirs []string, opts ...Option) iter.Seq2[FileEntry, error] {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return func(yield func(FileEntry, error) bool) {
		for _, dir := range dirs {
			if !findDir(dir, &options, yield) {
				return
			}
		}
	}
}

func findDir(dir string, opts *options, yield func(FileEntry, error) bool) bool {
	if !filepath.IsAbs(dir) {
		return yield(nil, fmt.Errorf("%w: %s", ErrRelativePath, dir))
	}

	// 符号链接作为遍历根时先解析到目标，避免按 Lstat 判断根不是目录而静默得到空结果。
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return yield(nil, err)
	}
	dir = resolved

	// TrimSuffix 会把 "/" 截成空串，需保留文件系统根。
	root := strings.TrimSuffix(dir, string(filepath.Separator))
	if root == "" {
		root = string(filepath.Separator)
	}

	// followLinks 下按真实路径记录当前递归路径上的目录，防止符号链接成环导致无限递归；
	// 仅禁止回到祖先，并行到达同一目录的不同路径都会正常遍历。
	visited := map[string]bool{dir: true}

	// walk 遍历 path 的直接子条目，返回是否继续。
	var walk func(path string) bool
	walk = func(path string) bool {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return yield(nil, err)
		}

		for _, d := range dirEntries {
			full := filepath.Join(path, d.Name())

			// 跟随符号链接时按目标类型处理；断链无法解析则按链接本身的条目产出。
			if opts.followLinks && d.Type()&fs.ModeSymlink != 0 {
				if info, statErr := os.Stat(full); statErr == nil {
					d = statDirEntry{info: info}
				}
			}

			rel, relErr := filepath.Rel(root, full)
			if relErr != nil {
				return yield(nil, relErr)
			}
			entry := makeEntry(d, full, rel)

			if !checkFilter(opts, entry) {
				// 目录被过滤时整棵子树一起跳过（不再下钻）。
				continue
			}
			if opts.kind.matches(entry) {
				if !yield(entry, nil) {
					return false
				}
			}

			if !d.IsDir() {
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

	return walk(dir)
}

// statDirEntry 把 os.Stat 的 FileInfo 适配为 fs.DirEntry，用于 followLinks 时以目标类型描述条目。
type statDirEntry struct {
	info fs.FileInfo
}

func (e statDirEntry) Name() string               { return e.info.Name() }
func (e statDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e statDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e statDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

var _ fs.DirEntry = statDirEntry{}

func makeEntry(d fs.DirEntry, path string, rel string) fileEntry {
	return fileEntry{
		DirEntry: d,
		absPath:  path,
		relPath:  filepath.ToSlash(rel),
		depth:    strings.Count(rel, string(filepath.Separator)),
	}
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
