package store

import (
	"errors"
	"fmt"
)

// ErrStoreMissing 存储不存在的哨兵错误
var ErrStoreMissing = errors.New("存储不存在")

// ErrStoreLoadFailed 存储存在但读取失败的哨兵错误
var ErrStoreLoadFailed = errors.New("存储读取失败")

// ErrMarshalFailed 序列化数据失败的哨兵错误
var ErrMarshalFailed = errors.New("序列化数据失败")

// ErrUnmarshalFailed 反序列化数据失败的哨兵错误
var ErrUnmarshalFailed = errors.New("反序列化数据失败")

// IsStoreMissing 报告 err 是否为存储不存在（含节缺失）。
func IsStoreMissing(err error) bool {
	return errors.Is(err, ErrStoreMissing)
}

// IsStoreLoadFailed 报告 err 是否为存储存在但读取失败。
func IsStoreLoadFailed(err error) bool {
	return errors.Is(err, ErrStoreLoadFailed)
}

// IsMarshalFailed 报告 err 是否为序列化数据失败。
func IsMarshalFailed(err error) bool {
	return errors.Is(err, ErrMarshalFailed)
}

// IsUnmarshalFailed 报告 err 是否为反序列化数据失败。
func IsUnmarshalFailed(err error) bool {
	return errors.Is(err, ErrUnmarshalFailed)
}

/* json */

func jsonStoreMissing(path string) error {
	return fmt.Errorf("%w: path=%s", ErrStoreMissing, path)
}

func jsonStoreLoadFailed(path string) error {
	return fmt.Errorf("%w: path=%s", ErrStoreLoadFailed, path)
}

func jsonMarshalFailed(path string, marshalErr error) error {
	return fmt.Errorf("%w: path=%s, err=%w", ErrMarshalFailed, path, marshalErr)
}

func jsonUnmarshalFailed(path string, unmarshalErr error) error {
	return fmt.Errorf("%w: path=%s, err=%w", ErrUnmarshalFailed, path, unmarshalErr)
}

/* section */

func sectionStoreMissingErr(path string, section string) error {
	return fmt.Errorf("%w: path=%s, section=%s", ErrStoreMissing, path, section)
}

func sectionMarshalFailed(path string, section string, marshalErr error) error {
	return fmt.Errorf("%w: path=%s, section=%s, err=%w", ErrMarshalFailed, path, section, marshalErr)
}

func sectionUnmarshalFailed(path string, section string, unmarshalErr error) error {
	return fmt.Errorf("%w: path=%s, section=%s, err=%w", ErrUnmarshalFailed, path, section, unmarshalErr)
}
