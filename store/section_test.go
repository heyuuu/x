package store

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// newFile 返回该测试专属的配置文件路径（初始不存在）。
func newFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func TestLoadSection(t *testing.T) {
	tests := []struct {
		name    string
		content string // 空 = 文件不存在
		section string
		want    []string
	}{
		{"文件不存在", "", "items", nil},
		{"文件坏JSON", "{bad", "items", nil},
		{"节缺失", `{"other": 1}`, "items", nil},
		{"节格式坏", `{"items": "not-array"}`, "items", nil},
		{"节为null", `{"items": null}`, "items", nil},
		{"节为空数组", `{"items": []}`, "items", []string{}},
		{"正常读取", `{"items": ["a","b"], "other": 1}`, "items", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := newFile(t)
			if tt.content != "" {
				if err := os.WriteFile(file, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("准备测试文件失败: %v", err)
				}
			}
			got := LoadSection[[]string](file, tt.section)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSaveSection(t *testing.T) {
	t.Run("新建文件并创建父目录", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "sub", "config.json")
		if err := SaveSection(file, "items", []string{"a"}); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
		got := LoadSection[[]string](file, "items")
		if len(got) != 1 || got[0] != "a" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("其他节语义级透传", func(t *testing.T) {
		file := newFile(t)
		if err := os.WriteFile(file, []byte("{\"other\": {\"z\": 1, \"a\": [1, 2]}}"), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := SaveSection(file, "items", []string{"a"}); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取落盘文件失败: %v", err)
		}
		var d map[string]jsontext.Value
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatalf("落盘文件不是合法 JSON: %v\n%s", err, data)
		}
		// 字节级透传做不到（Unmarshal 落 jsontext.Value 会压缩空白），断言语义等价
		var other, wantOther map[string]any
		if err := json.Unmarshal(d["other"], &other); err != nil {
			t.Fatalf("other 节不是合法对象: %v", err)
		}
		if err := json.Unmarshal([]byte(`{"z": 1, "a": [1, 2]}`), &wantOther); err != nil {
			t.Fatalf("wantOther 构造失败: %v", err)
		}
		if !reflect.DeepEqual(other, wantOther) {
			t.Fatalf("other 节被改写: %s", d["other"])
		}
		var got []string
		if err := json.Unmarshal(d["items"], &got); err != nil {
			t.Fatalf("items 节不是合法数组: %v", err)
		}
		if len(got) != 1 || got[0] != "a" {
			t.Fatalf("items 节内容错误: %v", got)
		}
	})

	t.Run("替换已有节", func(t *testing.T) {
		file := newFile(t)
		if err := SaveSection(file, "items", []string{"a"}); err != nil {
			t.Fatalf("首次写入失败: %v", err)
		}
		if err := SaveSection(file, "items", []string{"b", "c"}); err != nil {
			t.Fatalf("替换写入失败: %v", err)
		}
		got := LoadSection[[]string](file, "items")
		if len(got) != 2 || got[0] != "b" || got[1] != "c" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("nil 删除节", func(t *testing.T) {
		file := newFile(t)
		if err := os.WriteFile(file, []byte(`{"items": ["a"], "other": 1}`), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		for _, src := range []any{nil, []string(nil), (*int)(nil), map[string]int(nil)} {
			if err := SaveSection(file, "items", src); err != nil {
				t.Fatalf("删节失败: %v", err)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("读取落盘文件失败: %v", err)
			}
			if strings.Contains(string(data), "items") {
				t.Fatalf("items 节应被删除 (src=%#v):\n%s", src, data)
			}
			if !strings.Contains(string(data), "other") {
				t.Fatalf("other 节不应受影响:\n%s", data)
			}
		}
	})

	t.Run("空集合不删节", func(t *testing.T) {
		file := newFile(t)
		if err := SaveSection(file, "items", []string{}); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
		got := LoadSection[[]string](file, "items")
		if got == nil || len(got) != 0 {
			t.Fatalf("空 slice 应落盘为 []，got %v", got)
		}
	})

	t.Run("坏文件拒写且不被改动", func(t *testing.T) {
		file := newFile(t)
		bad := "{bad json"
		if err := os.WriteFile(file, []byte(bad), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := SaveSection(file, "items", []string{"a"}); err == nil {
			t.Fatal("坏文件应拒绝写入")
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取落盘文件失败: %v", err)
		}
		if string(data) != bad {
			t.Fatalf("坏文件被改动: %s", data)
		}
	})

	t.Run("落盘两空格缩进", func(t *testing.T) {
		file := newFile(t)
		if err := SaveSection(file, "items", []string{"a"}); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取落盘文件失败: %v", err)
		}
		if !strings.Contains(string(data), "\n  \"items\"") {
			t.Fatalf("非两空格缩进:\n%s", data)
		}
	})
}

func TestLoadSectionOrDefault(t *testing.T) {
	defaultValue := []string{"d"}
	for _, tt := range []struct {
		name    string
		content string // 空 = 文件不存在
		want    []string
	}{
		{"文件不存在返回默认值", "", defaultValue},
		{"文件坏JSON返回默认值", "{bad", defaultValue},
		{"节缺失返回默认值", `{"other": 1}`, defaultValue},
		{"节格式坏返回默认值", `{"items": "not-array"}`, defaultValue}, {"节存在返回实际值", `{"items": ["a"]}`, []string{"a"}},
		{"节为空数组返回实际值", `{"items": []}`, []string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := newFile(t)
			if tt.content != "" {
				if err := os.WriteFile(file, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("准备测试文件失败: %v", err)
				}
			}
			got := LoadSectionOrDefault(file, "items", defaultValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteSection(t *testing.T) {
	t.Run("删除已有节", func(t *testing.T) {
		file := newFile(t)
		if err := os.WriteFile(file, []byte(`{"items": ["a"], "other": 1}`), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := DeleteSection(file, "items"); err != nil {
			t.Fatalf("删节失败: %v", err)
		}
		if got := LoadSection[[]string](file, "items"); got != nil {
			t.Fatalf("items 节应已删除: %v", got)
		}
		if got := LoadSection[int](file, "other"); got != 1 {
			t.Fatalf("other 节不应受影响: %v", got)
		}
	})

	t.Run("节或文件不存在也成功", func(t *testing.T) {
		file := newFile(t)
		if err := os.WriteFile(file, []byte(`{"other": 1}`), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := DeleteSection(file, "items"); err != nil {
			t.Fatalf("节不存在时应成功: %v", err)
		}
		if err := DeleteSection(newFile(t), "items"); err != nil {
			t.Fatalf("文件不存在时应成功: %v", err)
		}
	})

	t.Run("坏文件拒删", func(t *testing.T) {
		file := newFile(t)
		bad := "{bad json"
		if err := os.WriteFile(file, []byte(bad), 0o644); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := DeleteSection(file, "items"); err == nil {
			t.Fatal("坏文件应拒绝删除")
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取落盘文件失败: %v", err)
		}
		if string(data) != bad {
			t.Fatalf("坏文件被改动: %s", data)
		}
	})
}

func TestSaveSectionConcurrent(t *testing.T) {
	file := newFile(t)
	// 两个 goroutine 各写不同节，验收包级锁：最终两节并存、文件合法（-race 下无撕裂）
	var wg sync.WaitGroup
	for _, section := range []string{"alpha", "beta"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := SaveSection(file, section, []string{"v"}); err != nil {
				t.Errorf("并发写入失败: %v", err)
			}
		}()
	}
	wg.Wait()
	alpha := LoadSection[[]string](file, "alpha")
	beta := LoadSection[[]string](file, "beta")
	if len(alpha) != 1 || alpha[0] != "v" || len(beta) != 1 || beta[0] != "v" {
		t.Fatalf("并发写后两节应并存: alpha=%v beta=%v", alpha, beta)
	}
}
