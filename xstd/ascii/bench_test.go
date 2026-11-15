package ascii

// IsLower 各类实现方式的性能对比，每次操作扫描 4096 字节：
//
//	Range      区间比较 'a' <= c && c <= 'z'（IsLower 当前实现，泛型）
//	RangeByte  同 Range，非泛型版本，用于对比泛型开销
//	Bool       [256]bool 查表（xstrings.byteIsAlpha 风格）
//	Kind       [256]uint8 枚举查表（xstrings.byteKinds 风格）
//	Bitmap     [32]byte 位图（unicode 标准库风格）
//
// 数据分布（影响分支预测）：
//
//	mixed  0..255 均匀随机，约 10% 为小写
//	lower  全小写，判断恒为真
//	other  全大写+数字，判断恒为假
//
// 测试结论见 BENCHMARK.md。

import (
	"testing"
)

// 查表数据

var (
	benchBool   [256]bool
	benchKind   [256]uint8
	benchBitmap [32]byte
)

const (
	benchKindOther = iota
	benchKindLower
	benchKindUpper
)

func init() {
	for c := byte('a'); c <= 'z'; c++ {
		benchBool[c] = true
		benchKind[c] = benchKindLower
		benchBitmap[c>>3] |= 1 << (c & 7)
	}
	for c := byte('A'); c <= 'Z'; c++ {
		benchKind[c] = benchKindUpper
	}
}

func lookupBool(c byte) bool   { return benchBool[c] }
func lookupKind(c byte) bool   { return benchKind[c] == benchKindLower }
func lookupBitmap(c byte) bool { return benchBitmap[c>>3]&(1<<(c&7)) != 0 }

// 测试数据

var (
	benchMixed = randBytes(4096, 0x9e3779b97f4a7c15)
	benchLower = repeatPattern("abcdefghijklmnopqrstuvwxyz", 4096)
	benchOther = repeatPattern("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 4096)
)

// randBytes xorshift64* 生成确定性伪随机字节
func randBytes(size int, seed uint64) []byte {
	data := make([]byte, size)
	x := seed
	for i := range data {
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		data[i] = byte(x * 2685821657736338717 >> 33)
	}
	return data
}

// repeatPattern 循环填充固定模式
func repeatPattern(pattern string, size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}

// benchSink 接收结果，防止循环被优化掉
var benchSink int

func runBenchmarks(b *testing.B, benchFunc func(b *testing.B, data []byte)) {
	b.Run("mixed", func(b *testing.B) { benchFunc(b, benchMixed) })
	b.Run("lower", func(b *testing.B) { benchFunc(b, benchLower) })
	b.Run("other", func(b *testing.B) { benchFunc(b, benchOther) })
}

// 各实现，直接调用保证可内联

func benchIsLowerRange(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		n := 0
		for _, c := range data {
			if IsLower(c) {
				n++
			}
		}
		benchSink = n
	}
}

func benchIsLowerRangeByte(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		n := 0
		for _, c := range data {
			if 'a' <= c && c <= 'z' {
				n++
			}
		}
		benchSink = n
	}
}

func benchIsLowerBool(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		n := 0
		for _, c := range data {
			if lookupBool(c) {
				n++
			}
		}
		benchSink = n
	}
}

func benchIsLowerKind(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		n := 0
		for _, c := range data {
			if lookupKind(c) {
				n++
			}
		}
		benchSink = n
	}
}

func benchIsLowerBitmap(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		n := 0
		for _, c := range data {
			if lookupBitmap(c) {
				n++
			}
		}
		benchSink = n
	}
}

func BenchmarkIsLowerRange(b *testing.B) {
	runBenchmarks(b, benchIsLowerRange)
}

func BenchmarkIsLowerRangeByte(b *testing.B) {
	runBenchmarks(b, benchIsLowerRangeByte)
}

func BenchmarkIsLowerBool(b *testing.B) {
	runBenchmarks(b, benchIsLowerBool)
}

func BenchmarkIsLowerKind(b *testing.B) {
	runBenchmarks(b, benchIsLowerKind)
}

func BenchmarkIsLowerBitmap(b *testing.B) {
	runBenchmarks(b, benchIsLowerBitmap)
}
