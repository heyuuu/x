package xstrings

// 本文件内是处理字符串大小写相关的函数。
// 这里有几个通用约定:
// - 大小写转换只处理 ASCII 范围内的字符，对之外的字符保持不变。与标准库函数(strings.ToLower/strings.ToUpper等)逻辑不同。
// - 分词时，ASCII 中非字母数字的字符皆为分隔符并丢弃，非 ASCII 字符作为单独类型，不同 ASCII 间不做区分。
//   e.g. "v2.0.1" 分词为 "v" + "2" + "0" + "1"；"用户のID" 分词为 "用户の" + "ID"

import (
	"unsafe"

	"github.com/heyuuu/x/xstd/ascii"
)

type byteKind uint8

// 字节在分词中的类别
const (
	kindOther     byteKind = iota // 非 ASCII 字节，连续出现归为一个词
	kindSeparator                 // 分隔符：ASCII 中非字母数字的字符
	kindLower                     // 小写字母
	kindUpper                     // 大写字母
	kindDigit                     // 数字
)

// kindOf 判断字节在分词中的类别
func kindOf(c byte) byteKind {
	switch {
	case c >= 0x80:
		return kindOther
	case 'a' <= c && c <= 'z':
		return kindLower
	case 'A' <= c && c <= 'Z':
		return kindUpper
	case '0' <= c && c <= '9':
		return kindDigit
	default:
		return kindSeparator
	}
}

// unsafeBytesToString 字符切片直接转string，要求调用方保证字符切片不再修改
func unsafeBytesToString(s []byte) string {
	return unsafe.String(unsafe.SliceData(s), len(s))
}

// Capitalize 首字母大写，其他字母小写
func Capitalize(s string) string {
	var buf []byte
	for i, c := range []byte(s) {
		var rc byte
		if i == 0 {
			rc = ascii.ToUpper(c)
		} else {
			rc = ascii.ToLower(c)
		}

		if rc != c {
			// 懒初始化，只在字符变更时才构建新字符串
			if buf == nil {
				buf = []byte(s)
			}
			buf[i] = rc
		}
	}
	if buf == nil {
		return s
	}
	return unsafeBytesToString(buf)
}

// UpperFirst 首字母大写
func UpperFirst(s string) string {
	if s == "" || !ascii.IsLower(s[0]) {
		return s
	}

	buf := []byte(s)
	buf[0] = ascii.ToUpper(buf[0])
	return unsafeBytesToString(buf)
}

// LowerFirst 首字母小写
func LowerFirst(s string) string {
	if s == "" || !ascii.IsUpper(s[0]) {
		return s
	}

	buf := []byte(s)
	buf[0] = ascii.ToLower(buf[0])
	return unsafeBytesToString(buf)
}

// splitWords 分词，返回字符串切分后的单词列表。
// 分隔符不出现在结果中；连续的非 ASCII 字符归为一个词；
// 连续大写字母后跟小写字母时从最后一个大写字母处分词，e.g. "HTTPServer" → ["HTTP", "Server"]
func splitWords(s string) []string {
	var result []string
	var state = kindSeparator
	var wordStart = 0
	for i, c := range []byte(s) {
		nextState := kindOf(c)
		if state == nextState {
			continue
		}

		// AAAaa 形式会拆分为 "AA" + "Aaa"
		if state == kindUpper && nextState == kindLower {
			if wordStart < i-1 {
				result = append(result, s[wordStart:i-1])
			}

			state = nextState
			wordStart = i - 1
		} else {
			if state != kindSeparator {
				result = append(result, s[wordStart:i])
			}

			state = nextState
			wordStart = i
		}
	}
	if state != kindSeparator {
		result = append(result, s[wordStart:])
	}
	return result
}

// commonCase 通用的命名法转换函数：将 s 分词后以 sep 连接，
// 逐词应用 caseHandler 做大小写处理（i 为单词序号，word 为结果缓冲区内该单词的切片）
func commonCase(s string, sep string, caseHandler func(i int, word []byte)) string {
	// 分词
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}

	// 预计算结果字符串尺寸
	size := 0
	for i, word := range words {
		if i > 0 {
			size += len(sep)
		}
		size += len(word)
	}

	buf := make([]byte, 0, size)
	for i, word := range words {
		// 添加分隔符
		if i > 0 {
			buf = append(buf, sep...)
		}

		// 非英文单词不处理大小写
		if !ascii.IsAlpha(word[0]) {
			buf = append(buf, word...)
			continue
		}

		// 英文单词处理大小写
		wordStart := len(buf)
		buf = append(buf, word...)
		caseHandler(i, buf[wordStart:])
	}
	return unsafeBytesToString(buf)
}

// CamelCase 驼峰命名法(又称小驼峰命名法)，e.g. "userName"
func CamelCase(s string) string {
	return commonCase(s, "", func(wordIndex int, word []byte) {
		for charIndex, c := range word {
			if charIndex == 0 && wordIndex > 0 {
				word[charIndex] = ascii.ToUpper(c)
			} else {
				word[charIndex] = ascii.ToLower(c)
			}
		}
	})
}

// PascalCase 帕斯卡命名法(又称大驼峰命名法)，e.g. "UserName"
func PascalCase(s string) string {
	return commonCase(s, "", func(_ int, word []byte) {
		for charIndex, c := range word {
			if charIndex == 0 {
				word[charIndex] = ascii.ToUpper(c)
			} else {
				word[charIndex] = ascii.ToLower(c)
			}
		}
	})
}

// SnakeCase 蛇型命名法，e.g. "user_name"
func SnakeCase(s string) string {
	return commonCase(s, "_", func(_ int, word []byte) {
		for charIndex, c := range word {
			word[charIndex] = ascii.ToLower(c)
		}
	})
}

// ScreamingSnakeCase 大蛇型命名法，e.g. "USER_NAME"
func ScreamingSnakeCase(s string) string {
	return commonCase(s, "_", func(_ int, word []byte) {
		for charIndex, c := range word {
			word[charIndex] = ascii.ToUpper(c)
		}
	})
}

// KebabCase 烤串式命名法，e.g. "user-name"
func KebabCase(s string) string {
	return commonCase(s, "-", func(_ int, word []byte) {
		for charIndex, c := range word {
			word[charIndex] = ascii.ToLower(c)
		}
	})
}

// ScreamingKebabCase 大烤串式命名法，e.g. "USER-NAME"
func ScreamingKebabCase(s string) string {
	return commonCase(s, "-", func(_ int, word []byte) {
		for charIndex, c := range word {
			word[charIndex] = ascii.ToUpper(c)
		}
	})
}
