package jsonl

import (
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eachUser struct {
	Name string `json:"name"`
}

func writeLines(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "log.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func collectEach[T any](t *testing.T, seq iter.Seq2[T, error]) (values []T, errs []error) {
	t.Helper()
	for value, err := range seq {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		values = append(values, value)
	}
	return values, errs
}

func names(t *testing.T, users []eachUser) []string {
	t.Helper()
	var result []string
	for _, user := range users {
		result = append(result, user.Name)
	}
	return result
}

func TestEachOrder(t *testing.T) {
	path := writeLines(t, `{"name":"a"}`+"\n"+`{"name":"b"}`+"\n")

	users, errs := collectEach(t, Each[eachUser](path))
	require.Empty(t, errs)
	assert.Equal(t, []string{"a", "b"}, names(t, users))
}

func TestReverseEachOrder(t *testing.T) {
	path := writeLines(t, `{"name":"a"}`+"\n"+`{"name":"b"}`+"\n"+`{"name":"c"}`) // 结尾无换行

	users, errs := collectEach(t, ReverseEach[eachUser](path))
	require.Empty(t, errs)
	assert.Equal(t, []string{"c", "b", "a"}, names(t, users), "应从最后一行向前，且结尾无换行不丢行")
}

// 正向与反向对同一文件必须严格互逆。
func TestReverseEachMirrorsEach(t *testing.T) {
	for name, content := range map[string]string{
		"结尾有换行":    `{"name":"a"}` + "\n" + `{"name":"b"}` + "\n" + `{"name":"c"}` + "\n",
		"结尾无换行":    `{"name":"a"}` + "\n" + `{"name":"b"}`,
		"空文件":      "",
		"行跨越多个读取块": strings.Repeat(`{"name":"x"}`+"\n", 30_000),
	} {
		t.Run(name, func(t *testing.T) {
			path := writeLines(t, content)

			forward, forwardErrs := collectEach(t, Each[eachUser](path))
			reverse, reverseErrs := collectEach(t, ReverseEach[eachUser](path))
			require.Empty(t, forwardErrs)
			require.Empty(t, reverseErrs)

			slices.Reverse(forward)
			assert.Equal(t, forward, reverse)
		})
	}
}

func TestEachBadLineContinues(t *testing.T) {
	path := writeLines(t, `{"name":"a"}`+"\n"+"坏行\n"+`{"name":"c"}`)

	users, errs := collectEach(t, Each[eachUser](path))
	require.Len(t, errs, 1)
	assert.True(t, IsParse(errs[0]))
	assert.ErrorIs(t, errs[0], ErrParse)
	assert.Contains(t, errs[0].Error(), "第 2 行")
	assert.Equal(t, []string{"a", "c"}, names(t, users), "坏行之后迭代应继续")
}

func TestReverseEachBadLineMessage(t *testing.T) {
	path := writeLines(t, `{"name":"a"}`+"\n"+`{"name":"b"}`+"\n"+"坏行") // 坏行是最后一行

	users, errs := collectEach(t, ReverseEach[eachUser](path))
	require.Len(t, errs, 1)
	assert.True(t, IsParse(errs[0]))
	assert.Contains(t, errs[0].Error(), "倒数第 1 行")
	assert.Equal(t, []string{"b", "a"}, names(t, users), "坏行在最前，其后继续")
}

// 空行不是合法 JSON，按坏行处理。
func TestEachBlankLine(t *testing.T) {
	path := writeLines(t, "\n")

	users, errs := collectEach(t, Each[eachUser](path))
	assert.Empty(t, users)
	require.Len(t, errs, 1)
	assert.True(t, IsParse(errs[0]))
}

func TestEachOpenFailed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.jsonl")

	users, errs := collectEach(t, Each[eachUser](path))
	assert.Empty(t, users)
	require.Len(t, errs, 1, "打开失败只产出一次错误")
	assert.False(t, IsParse(errs[0]), "打开失败不是坏行错误")
	assert.ErrorIs(t, errs[0], os.ErrNotExist, "底层错误应可识别")
}

func TestEachLongLine(t *testing.T) {
	long := strings.Repeat("x", 200_000) // 远超常见 64 KiB 的单行上限
	path := writeLines(t, `{"name":"`+long+`"}`+"\n")

	for _, seq := range []iter.Seq2[eachUser, error]{Each[eachUser](path), ReverseEach[eachUser](path)} {
		users, errs := collectEach(t, seq)
		require.Empty(t, errs)
		require.Len(t, users, 1)
		assert.Len(t, users[0].Name, 200_000)
	}
}

// 迭代中提前 break 应正常结束，不阻塞、不泄漏（文件在序列结束时关闭）。
func TestEachEarlyBreak(t *testing.T) {
	path := writeLines(t, strings.Repeat(`{"name":"a"}`+"\n", 100))

	count := 0
	for _, err := range Each[eachUser](path) {
		require.NoError(t, err)
		count++
		break
	}
	assert.Equal(t, 1, count)
}
