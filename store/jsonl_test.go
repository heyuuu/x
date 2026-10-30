package store

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendAndLoadJsonl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.jsonl")

	type record struct {
		Path string `json:"path"`
		Seq  int    `json:"seq"`
	}

	_, err := LoadJsonl[record](path)
	require.ErrorIs(t, err, ErrFileMissing, "缺失文件应返回 ErrFileMissing")

	want := []record{{Path: "a", Seq: 1}, {Path: "b", Seq: 2}, {Path: "a", Seq: 3}}
	for _, r := range want {
		require.NoError(t, AppendJsonl(path, r), "AppendJsonl 失败")
	}

	// 追加一条含换行敏感字符的记录，确认序列化后仍是单行
	require.NoError(t, AppendJsonl(path, record{Path: "x\ny", Seq: 4}))

	// 注入坏行：中间坏行被跳过，好行保留
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("\n{broken\n") // 一个空行 + 一个坏行
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got, err := LoadJsonl[record](path)
	require.NoError(t, err, "LoadJsonl 失败")
	require.Len(t, got, len(want)+1, "坏行跳过后条数不对")
	assert.Equal(t, want, got[:len(want)])
	assert.Equal(t, "x\ny", got[len(got)-1].Path, "含换行记录解析错误")

	// 空文件：返回 (nil, nil) 而非 ErrFileMissing
	empty := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(empty, nil, 0644))
	got, err = LoadJsonl[record](empty)
	require.NoError(t, err, "空文件不应报错")
	assert.Empty(t, got, "空文件应读出 0 条")
}

func TestWriteJsonl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a/b/usage.jsonl") // 父目录不存在，验证自动创建

	items := []map[string]int{{"x": 1}, {"x": 2}}
	require.NoError(t, WriteJsonl(path, items), "WriteJsonl 失败")
	got, err := LoadJsonl[map[string]int](path)
	require.NoError(t, err, "LoadJsonl 失败")
	assert.Equal(t, items, got, "写读不一致")

	// 整体重写语义：旧内容被替换而非追加
	require.NoError(t, WriteJsonl(path, []map[string]int{{"y": 9}}), "WriteJsonl 重写失败")
	got, _ = LoadJsonl[map[string]int](path)
	assert.Equal(t, []map[string]int{{"y": 9}}, got, "重写后应只剩新内容")

	// 空切片：整文件清空（0 行）
	require.NoError(t, WriteJsonl[map[string]int](path, nil), "WriteJsonl 空切片失败")
	got, _ = LoadJsonl[map[string]int](path)
	assert.Empty(t, got, "空切片应清空文件")
}

func TestIterJsonl(t *testing.T) {
	dir := t.TempDir()

	type record struct {
		Seq int `json:"seq"`
	}

	// 缺失文件：静默结束，不 panic
	n := 0
	for range IterJsonl[record](filepath.Join(dir, "none.jsonl")) {
		n++
	}
	assert.Zero(t, n, "缺失文件应产出 0 条")

	// 坏行占行号、空行占行号
	path := filepath.Join(dir, "it.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"seq":1}`+"\n\n"+`{bad`+"\n"+`{"seq":2}`+"\n"), 0644))

	var seqs []int
	var lineNos []int
	for lineNo, r := range IterJsonl[record](path) {
		seqs = append(seqs, r.Seq)
		lineNos = append(lineNos, lineNo)
	}
	assert.Equal(t, []int{1, 2}, seqs, "正向迭代记录不对")
	assert.Equal(t, []int{1, 4}, lineNos, "物理行号不对")

	// 提前停止：只取第一条
	var firstSeq int
	for _, r := range IterJsonl[record](path) {
		firstSeq = r.Seq
		break
	}
	assert.Equal(t, 1, firstSeq, "提前停止取首条失败")
}

func TestIterJsonlReverse(t *testing.T) {
	dir := t.TempDir()

	type record struct {
		Seq int `json:"seq"`
	}

	// 缺失文件：静默结束
	n := 0
	for range IterJsonlReverse[record](filepath.Join(dir, "none.jsonl")) {
		n++
	}
	assert.Zero(t, n, "缺失文件应产出 0 条")

	// 多行 + 坏行 + 空行；文件以 '\n' 结尾
	buildLines := func(trailingNL bool) (string, []byte) {
		lines := `{"seq":1}` + "\n" +
			"\n" +
			`{bad` + "\n" +
			`{"seq":2}` + "\n" +
			`{"seq":3}` + "\n"
		if !trailingNL {
			lines = strings.TrimSuffix(lines, "\n")
		}
		return lines, []byte(lines)
	}

	for _, trailingNL := range []bool{true, false} {
		_, data := buildLines(trailingNL)
		path := filepath.Join(dir, "rev.jsonl")
		require.NoError(t, os.WriteFile(path, data, 0644))

		var seqs []int
		var lineNos []int
		for lineNo, r := range IterJsonlReverse[record](path) {
			seqs = append(seqs, r.Seq)
			lineNos = append(lineNos, lineNo)
		}
		require.Equal(t, []int{3, 2, 1}, seqs, "逆向迭代记录不对 (trailingNL=%v)", trailingNL)
		require.Equal(t, []int{5, 4, 1}, lineNos, "物理行号不对 (trailingNL=%v)", trailingNL)

		// 提前停止：只取最后一条
		var lastSeq int
		for _, r := range IterJsonlReverse[record](path) {
			lastSeq = r.Seq
			break
		}
		assert.Equal(t, 3, lastSeq, "提前停止取末条失败 (trailingNL=%v)", trailingNL)
	}

	// 空文件：静默结束
	empty := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(empty, nil, 0644))
	n = 0
	for range IterJsonlReverse[record](empty) {
		n++
	}
	assert.Zero(t, n, "空文件应产出 0 条")

	// 单行且无结尾换行：行号 1，记录正常产出
	single := filepath.Join(dir, "single.jsonl")
	require.NoError(t, os.WriteFile(single, []byte(`{"seq":7}`), 0644))
	var singleSeq int
	var singleLine int
	for lineNo, r := range IterJsonlReverse[record](single) {
		singleSeq, singleLine = r.Seq, lineNo
	}
	assert.Equal(t, 7, singleSeq, "单行无换行记录不对")
	assert.Equal(t, 1, singleLine, "单行无换行行号不对")

	// 超过分块大小（>64KB）的单行：整行跨多个块拼在 pending 里
	long := filepath.Join(dir, "long.jsonl")
	longLine := `{"seq":9,"pad":"` + strings.Repeat("y", 3*64*1024) + `"}`
	require.NoError(t, os.WriteFile(long, []byte(longLine), 0644))
	n = 0
	for _, r := range IterJsonlReverse[record](long) {
		assert.Equal(t, 9, r.Seq, "超长单行记录不对")
		n++
	}
	assert.Equal(t, 1, n, "超长单行应产出 1 条")

	// 超过分块大小（>64KB）：验证跨块拼接
	path := filepath.Join(dir, "big.jsonl")
	var sb strings.Builder
	want := 2000
	for i := range want {
		sb.WriteString(`{"seq":` + strconv.Itoa(i) + `,"pad":"` + strings.Repeat("x", 64) + `"}` + "\n")
	}
	require.NoError(t, os.WriteFile(path, []byte(sb.String()), 0644))

	var got []int
	for _, r := range IterJsonlReverse[record](path) {
		got = append(got, r.Seq)
	}
	require.Len(t, got, want, "跨块总条数不对")
	for i, seq := range got {
		require.Equal(t, want-1-i, seq, "跨块逆序错位: 位置 %d", i)
	}
}
