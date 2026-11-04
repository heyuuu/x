package finder

// OptionFiles 只匹配文件。
func OptionFiles() Option {
	return func(o *options) {
		o.kind = entryKindFiles
	}
}

// OptionDirectories 只匹配目录。
func OptionDirectories() Option {
	return func(o *options) {
		o.kind = entryKindDirectories
	}
}

// OptionDepth 设置相对每个根目录的最大深度，0 表示仅根目录下的条目。默认不限制（-1）。
func OptionDepth(maxDepth int) Option {
	return func(o *options) {
		o.maxDepth = maxDepth
	}
}

// OptionIgnoreVCS 设置是否跳过 VCS 目录（.git/.hg/.svn），默认 true。
func OptionIgnoreVCS(ignore bool) Option {
	return func(o *options) {
		o.ignoreVCS = ignore
	}
}

// OptionFollowLinks 设置跟随符号链接：条目按目标类型产出（IsDir 以目标为准），指向目录的链接下钻遍历，
// 断链按链接自身条目产出。产出的路径始终是链接自身的路径（逻辑路径），即链接下的条目以链接路径为前缀
// （dirlink -> dir 时产出 dirlink/child 而非 dir/child），指向根外的链接 RelPath 也不会出现 ".."。
// 成环检测在内部按解析后的真实路径进行，仅沿当前递归路径禁止回到祖先，
// 并行到达同一目录的不同路径都会正常遍历。默认不跟随。
func OptionFollowLinks() Option {
	return func(o *options) {
		o.followLinks = true
	}
}

// OptionFilter 增加一个自定义过滤函数，返回 false 的条目被跳过；目录被跳过时其子树整棵剪掉。
// 可多次调用，串联执行：任一函数不满足即跳过。
// 注意 OptionFiles/OptionDirectories 的类型筛选不剪枝——OptionFiles 仍会深入子目录找文件。
func OptionFilter(filter func(FileEntry) bool) Option {
	return func(o *options) {
		o.filters = append(o.filters, filter)
	}
}

// entryKind 表示匹配的条目类型。
type entryKind int

const (
	entryKindAll entryKind = iota
	entryKindFiles
	entryKindDirectories
)

func (k entryKind) matches(entry FileEntry) bool {
	switch k {
	case entryKindFiles:
		return !entry.IsDir()
	case entryKindDirectories:
		return entry.IsDir()
	default:
		return true
	}
}

type options struct {
	kind        entryKind
	maxDepth    int
	ignoreVCS   bool
	followLinks bool
	filters     []func(FileEntry) bool
}

// Option 是配置 Find 行为的函数式选项。
type Option func(*options)

func defaultOptions() options {
	return options{
		kind:      entryKindAll,
		maxDepth:  -1,
		ignoreVCS: true,
	}
}
