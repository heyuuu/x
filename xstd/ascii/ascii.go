package ascii

// MaxASCII 是 ASCII 字符的最大值 0x7f。
const (
	MaxASCII = 0x7f
)

// IsASCII 判断字符是否在 ASCII 范围内（0x00-0x7f）。
func IsASCII[T byte | rune](c T) bool {
	return 0 <= c && c <= MaxASCII
}

// IsLower 判断是否为小写字母 a-z。
func IsLower[T byte | rune](c T) bool {
	return 'a' <= c && c <= 'z'
}

// IsUpper 判断是否为大写字母 A-Z。
func IsUpper[T byte | rune](c T) bool {
	return 'A' <= c && c <= 'Z'
}

// IsAlpha 判断是否为字母 a-z 或 A-Z。
func IsAlpha[T byte | rune](c T) bool {
	return IsLower(c) || IsUpper(c)
}

// IsDigit 判断是否为数字 0-9。
func IsDigit[T byte | rune](c T) bool {
	return '0' <= c && c <= '9'
}

// IsAlphaNum 判断是否为字母或数字。
func IsAlphaNum[T byte | rune](c T) bool {
	return IsAlpha(c) || IsDigit(c)
}

// IsXDigit 判断是否为十六进制数字 0-9、a-f、A-F。
func IsXDigit[T byte | rune](c T) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// ParseXDigit 解析十六进制数字为 0-15 的数值，非十六进制数字返回 (0, false)。
func ParseXDigit[T byte | rune](c T) (byte, bool) {
	if c >= '0' && c <= '9' {
		return byte(c - '0'), true
	} else if c >= 'A' && c <= 'F' {
		return byte(c - 'A' + 10), true
	} else if c >= 'a' && c <= 'f' {
		return byte(c - 'a' + 10), true
	}

	return 0, false
}

// IsControl 判断是否为控制字符 0x00-0x1f 或 0x7f。
func IsControl[T byte | rune](c T) bool {
	return (0 <= c && c <= 0x1f) || c == 0x7f
}

// IsSpace 判断是否为空白字符：空格、\t、\n、\v、\f、\r。
func IsSpace[T byte | rune](c T) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

// ToLower 转为小写；非 ASCII 大写字母原样返回。
func ToLower[T byte | rune](c T) T {
	if IsUpper(c) {
		return c - 'A' + 'a'
	}
	return c
}

// ToUpper 转为大写；非 ASCII 小写字母原样返回。
func ToUpper[T byte | rune](c T) T {
	if IsLower(c) {
		return c - 'a' + 'A'
	}
	return c
}
