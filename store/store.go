package store

import "errors"

// ErrFileMissing 是 LoadJSON / LoadJsonl 对「文件不存在」的可区分哨兵错误，
// 调用方据此时降级（区别于文件存在但读失败/解析失败——那类错误同样原样上抛，
// 但不应被当作空数据）。
var ErrFileMissing = errors.New("文件不存在")
