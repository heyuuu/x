// Package ascii 提供 ASCII 字符的判断与转换函数，功能类似 C 标准库 ctype.h。
// 通过泛型同时支持 byte 与 rune；对 ASCII 范围之外的字符，判断函数返回 false，
// 转换函数原样返回。
package ascii
