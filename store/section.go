package store

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"sync"
)

// doc 文档内存形态：节名 → 原始 JSON。非目标节语义级透传——值原样保留不解析，
// 但 Unmarshal 落 jsontext.Value 时会压缩空白，存盘经统一缩进，其他节的格式会被规范化。
type doc map[string]jsontext.Value

// fileMu 串行化单进程内的读-改-写。
// 仅保证单次 SaveSection 原子；调用方跨两次调用的组合竞态不在保障范围（单写者假设）。
var fileMu sync.Mutex

// SaveSection 锁内「读全文档 → 替换/删除 section 节 → 原子写回」。
// 文件为坏 JSON 时拒绝写入，防止静默覆盖掉手工搞坏的其他节。
// src 为 nil 值（untyped nil、nil 指针/slice/map）表示删除该节；
// 想写空集合请传初始化过的值（[]string{} / map[string]any{}）。
func SaveSection(path string, section string, v any) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	d, err := LoadJSON[doc](path)
	if err != nil {
		// 文件不存在是正常逻辑，会直接创建存储文件
		if IsStoreMissing(err) {
			d = doc{}
		} else {
			return err
		}
	}

	if isNilValue(v) {
		delete(d, section)
	} else {
		raw, err := json.Marshal(v)
		if err != nil {
			return sectionMarshalFailed(path, section, err)
		}
		d[section] = raw
	}

	return SaveJSON(path, d)
}

// DeleteSection 删除 section 节并原子写回（文件或节不存在视为成功，文件为坏 JSON 时拒写）。
func DeleteSection(path string, section string) error {
	return SaveSection(path, section, nil)
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

// LoadSection 从 file 读出 section 节并反序列化为 T。
// 失败形态经哨兵错误上抛，降级策略归调用方：
// 文件缺失或节缺失 → ErrStoreMissing；读取失败 → ErrStoreLoadFailed；
// 坏 JSON / 节格式坏 → ErrUnmarshalFailed。
// 不关心失败原因的调用方用 LoadSectionOrDefault。
// 注意：Go 无返回值类型推断，调用方需显式写 LoadSection[[]string](...)。
func LoadSection[T any](file string, section string) (T, error) {
	var zero T
	d, err := LoadJSON[doc](file)
	if err != nil {
		// 存储不存在的错误改为 section 的存储错误
		if IsStoreMissing(err) {
			return zero, sectionStoreMissingErr(file, section)
		}
		return zero, err
	}

	raw, ok := d[section]
	if !ok {
		return zero, sectionStoreMissingErr(file, section)
	}

	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return zero, sectionUnmarshalFailed(file, section, err)
	}

	return v, nil
}

// LoadSectionOrDefault 行为同 LoadSection，但一切读失败形态
// （文件不存在 / 坏 JSON / 节缺失 / 节格式坏）返回 defaultValue 而非报错——
// 选此变体即显式声明不关心失败原因。
func LoadSectionOrDefault[T any](file string, section string, defaultValue T) T {
	v, err := LoadSection[T](file, section)
	if err != nil {
		return defaultValue
	}
	return v
}
