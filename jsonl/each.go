package jsonl

import (
	"bufio"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"slices"
	"strings"
)

// ErrParse 坏行（不是合法 JSON）的哨兵错误
var ErrParse = errors.New("行解析失败")

// IsParse 报告 err 是否为坏行错误。坏行错误不会终止迭代，调用方可跳过或就此中断。
func IsParse(err error) bool {
	return errors.Is(err, ErrParse)
}

// Each 返回按 path 正序逐行解码的惰性序列，每行反序列化为 T。
//
// 错误约定：
//   - 坏行产出包装 ErrParse 的错误（消息含行号），迭代不终止，调用方可跳过或中断；
//   - 其余错误（打开、读取文件）意味着序列终止，作为最后一次迭代产出，此时 T 为零值；
//     底层错误已用 %w 包装，可用 errors.Is / errors.As 识别。
//
// 行长无上限；序列是单次使用的。
func Each[T any](path string) iter.Seq2[T, error] {
	return decode[T](forwardLines(path))
}

// ReverseEach 同 Each，但从最后一行向前迭代，坏行错误中的行号为「倒数第 n 行」。
// 从文件尾部按块倒读，内存占用只与最长行相关，与文件大小无关。
func ReverseEach[T any](path string) iter.Seq2[T, error] {
	return decode[T](reverseLines(path))
}

// decode 在行序列上逐行反序列化：坏行继续，其余错误终止。
func decode[T any](lines iter.Seq2[fileLine, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for line, err := range lines {
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			var value T
			if parseErr := json.Unmarshal([]byte(line.text), &value); parseErr != nil {
				var zero T
				if !yield(zero, parseFailed(line, parseErr)) {
					return
				}
				continue
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func parseFailed(line fileLine, err error) error {
	return fmt.Errorf("%w: %s, err=%w", ErrParse, line.describe(), err)
}

// readBlockSize 是行读取的内部缓冲块大小，只影响 I/O 次数，不是行长上限。
const readBlockSize = 64 * 1024

// fileLine 是行生产层产出的一行原始文本：text 不含换行符与行尾 '\r'；
// number 为行号（正向从文件头数、反向从文件尾数，均从 1 起），countFromEnd 标记计数方向。
type fileLine struct {
	text         string
	number       int
	countFromEnd bool
}

// describe 返回用于错误消息的行定位描述。
func (l fileLine) describe() string {
	if l.countFromEnd {
		return fmt.Sprintf("倒数第 %d 行", l.number)
	}
	return fmt.Sprintf("第 %d 行", l.number)
}

// forwardLines 按 path 正序产出原始行，只有打开、读取错误（终止序列）。
func forwardLines(path string) iter.Seq2[fileLine, error] {
	return func(yield func(fileLine, error) bool) {
		file, err := os.Open(path)
		if err != nil {
			yield(fileLine{}, fmt.Errorf("打开 JSONL 文件失败: %w", err))
			return
		}
		defer func() { _ = file.Close() }()

		// ReadBytes 对超长行自动累积，行长天然无上限。
		reader := bufio.NewReaderSize(file, readBlockSize)
		number := 0
		for {
			raw, readErr := reader.ReadBytes('\n')
			if len(raw) > 0 {
				number++
				text := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
				if !yield(fileLine{text: text, number: number}, nil) {
					return
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					yield(fileLine{}, fmt.Errorf("读取 JSONL 文件失败: %w", readErr))
				}
				return
			}
		}
	}
}

// reverseLines 按 path 逆序产出原始行（最后一行在最前），只有打开、读取错误（终止序列）。
// 从文件尾部按块向前扫描，行语义与正向一致：去掉 '\n' 与行尾 '\r'，结尾的 '\n' 不产生空行。
func reverseLines(path string) iter.Seq2[fileLine, error] {
	return func(yield func(fileLine, error) bool) {
		file, err := os.Open(path)
		if err != nil {
			yield(fileLine{}, fmt.Errorf("打开 JSONL 文件失败: %w", err))
			return
		}
		defer func() { _ = file.Close() }()

		info, err := file.Stat()
		if err != nil {
			yield(fileLine{}, fmt.Errorf("读取 JSONL 文件信息失败: %w", err))
			return
		}
		size := info.Size()

		// carry 按文件逆序累积当前行的字节，凑成完整行后翻转即得正序内容。
		var carry []byte
		number := 0
		emitLine := func() bool {
			slices.Reverse(carry)
			text := strings.TrimSuffix(string(carry), "\r")
			carry = carry[:0]
			number++
			return yield(fileLine{text: text, number: number, countFromEnd: true}, nil)
		}

		// 文件以 '\n' 结尾时，末尾的 '\n' 只是最后一行的终结符，跳过它，避免反向多读出一个空行。
		position := size
		if position > 0 {
			var last [1]byte
			if _, err := file.ReadAt(last[:], position-1); err != nil {
				yield(fileLine{}, fmt.Errorf("检查 JSONL 文件末尾失败: %w", err))
				return
			}
			if last[0] == '\n' {
				position--
			}
		}

		buffer := make([]byte, readBlockSize)
		for position > 0 {
			length := min(readBlockSize, int(position))
			position -= int64(length)
			if _, err := file.ReadAt(buffer[:length], position); err != nil && !errors.Is(err, io.EOF) {
				yield(fileLine{}, fmt.Errorf("读取 JSONL 文件块失败: %w", err))
				return
			}
			for index := length - 1; index >= 0; index-- {
				if buffer[index] == '\n' {
					if !emitLine() {
						return
					}
					continue
				}
				carry = append(carry, buffer[index])
			}
		}
		// 文件第一行（可能为空，例如文件以 '\n' 开头）。
		if size > 0 {
			emitLine()
		}
	}
}
