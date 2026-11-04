package finder

import (
	"path/filepath"
	"strings"
)

// FileEntry 表示遍历到的一个文件或目录：条目名、是否目录，以及相对遍历根的路径、绝对路径与深度。
// 路径一律是遍历到达时的逻辑路径：根按传入值，符号链接按链接自身的路径（见 OptionFollowLinks）。
type FileEntry interface {
	Name() string
	IsDir() bool
	AbsPath() string
	RelPath() string
	Depth() int
}

// fileEntry 是 FileEntry 的实现，包装遍历中拿到的 fs.DirEntry。
type fileEntry struct {
	name    string
	isDir   bool
	absPath string
	relPath string
	depth   int
}

var _ FileEntry = fileEntry{}

func makeEntry(path string, rel string, isDir bool) FileEntry {
	return fileEntry{
		name:    filepath.Base(path),
		isDir:   isDir,
		absPath: path,
		relPath: filepath.ToSlash(rel),
		depth:   strings.Count(rel, string(filepath.Separator)),
	}
}

func (e fileEntry) Name() string    { return e.name }
func (e fileEntry) IsDir() bool     { return e.isDir }
func (e fileEntry) AbsPath() string { return e.absPath }
func (e fileEntry) RelPath() string { return e.relPath }
func (e fileEntry) Depth() int      { return e.depth }
