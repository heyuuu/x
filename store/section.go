package store

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

// doc 文档内存形态：节名 → 原始 JSON。非目标节语义级透传——值原样保留不解析，
// 但 Unmarshal 落 jsontext.Value 时会压缩空白，存盘经统一缩进，其他节的格式会被规范化。
type doc map[string]jsontext.Value

// fileMu 串行化单进程内的读-改-写。
// 仅保证单次 SaveSection 原子；调用方跨两次调用的组合竞态不在保障范围（单写者假设）。
var fileMu sync.Mutex

// LoadSection 从 file 读出 section 节并反序列化为 T。
// 一切读失败形态（文件不存在 / 坏 JSON / 节缺失 / 节格式坏）统一降级：
// 记日志、返回 T 的零值、不返回 error——读路径不向调用方报错。
// 注意：Go 无返回值类型推断，调用方需显式写 LoadSection[[]string](...)。
func LoadSection[T any](file, section string) T {
	var zero T
	return LoadSectionOrDefault(file, section, zero)
}

// LoadSectionOrDefault 行为同 LoadSection，但一切读失败形态
// （文件不存在 / 坏 JSON / 节缺失 / 节格式坏）返回 defaultValue 而非零值。
func LoadSectionOrDefault[T any](file, section string, defaultValue T) T {
	var zero T
	d, err := readDoc(file)
	if err != nil {
		slog.Warn("读取 JSON 配置文件失败，按默认配置继续", "file", file, "err", err)
		return defaultValue
	}
	raw, ok := d[section]
	if !ok {
		return defaultValue
	}
	if err := json.Unmarshal(raw, &zero); err != nil {
		slog.Warn("配置节格式错误，该节按默认配置继续", "file", file, "section", section, "err", err)
		return defaultValue
	}
	return zero
}

// SaveSection 锁内「读全文档 → 替换/删除 section 节 → 原子写回」。
// 文件为坏 JSON 时拒绝写入，防止静默覆盖掉手工搞坏的其他节。
// src 为 nil 值（untyped nil、nil 指针/slice/map）表示删除该节；
// 想写空集合请传初始化过的值（[]string{} / map[string]any{}）。
func SaveSection(file, section string, src any) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	d, err := readDoc(file)
	if err != nil {
		return fmt.Errorf("文件不可解析，拒绝写入以免覆盖其他节，请手工修复: file=%s err=%w", file, err)
	}
	if isNilValue(src) {
		delete(d, section)
	} else {
		raw, err := json.Marshal(src)
		if err != nil {
			return fmt.Errorf("序列化节失败: section=%s err=%w", section, err)
		}
		d[section] = raw
	}
	return saveDoc(file, d)
}

// DeleteSection 删除 section 节并原子写回（文件或节不存在视为成功，文件为坏 JSON 时拒写）。
func DeleteSection(file, section string) error {
	return SaveSection(file, section, nil)
}

// isNilValue 判定 any 里的值是否为 nil——untyped nil，
// 以及装进接口后 == nil 不再成立的类型化 nil（nil 指针 / slice / map / interface）。
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}

// readDoc 读入并解析文档（LoadJSON）。文件不存在是正常态（首次运行）→ 空 doc；
// 坏 JSON 是异常态 → 返回错误（读侧由 LoadSection 降级，写侧用于拒写）。
func readDoc(file string) (doc, error) {
	d, err := LoadJSON[doc](file)
	if errors.Is(err, ErrFileMissing) {
		return doc{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取/解析文件失败: file=%s err=%w", file, err)
	}
	return d, nil
}

// saveDoc 原子写入（SaveJSON：缩进 + tmp/rename）。
// 顶层键序由 json/v2 的 Deterministic 保证——稳定且 diff 友好，不引入 OrderedMap。
func saveDoc(file string, d doc) error {
	if err := SaveJSON(file, d); err != nil {
		return fmt.Errorf("写入文件失败: file=%s err=%w", file, err)
	}
	return nil
}
