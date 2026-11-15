package xstrings

import (
	"slices"
	"testing"
)

func TestCapitalize(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"", ""},
		{"Simple WORD", "Simple word"},
		{"word", "Word"},
		{"WORD", "Word"},
		{"_wORD", "_word"},
		{"用户のiD", "用户のid"},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := Capitalize(tt.arg); got != tt.want {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestUpperFirst(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"", ""},
		{"Simple WORD", "Simple WORD"},
		{"word", "Word"},
		{"WORD", "WORD"},
		{"_wORD", "_wORD"},
		{"用户のiD", "用户のiD"},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := UpperFirst(tt.arg); got != tt.want {
				t.Errorf("UpperFirst(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"", ""},
		{"Simple WORD", "simple WORD"},
		{"word", "word"},
		{"WORD", "wORD"},
		{"_wORD", "_wORD"},
		{"用户のiD", "用户のiD"},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := LowerFirst(tt.arg); got != tt.want {
				t.Errorf("LowerFirst(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func Test_splitWords(t *testing.T) {
	tests := []struct {
		arg  string
		want []string
	}{
		{"", nil},
		{" word ", []string{"word"}},
		{"word", []string{"word"}},
		{"Simple word", []string{"Simple", "word"}},
		{"Simple-word", []string{"Simple", "word"}},
		{"Simple_word", []string{"Simple", "word"}},
		{"with01number", []string{"with", "01", "number"}},
		{"HTTPServer", []string{"HTTP", "Server"}},
		{"IDs", []string{"I", "Ds"}},
		{"HTTPServer2Go", []string{"HTTP", "Server", "2", "Go"}},
		{"用户のId", []string{"用户の", "Id"}},
		{"a.b", []string{"a", "b"}},
		{"a\tb", []string{"a", "b"}},
		{"a--b", []string{"a", "b"}},
		{"v2.0.1", []string{"v", "2", "0", "1"}},
		{"用户の.ID", []string{"用户の", "ID"}},
		{" foo_bar-baz ", []string{"foo", "bar", "baz"}},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := splitWords(tt.arg); !slices.Equal(got, tt.want) {
				t.Errorf("splitWords(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestAllCases(t *testing.T) {
	type want struct {
		CamelCase          string
		PascalCase         string
		SnakeCase          string
		ScreamingSnakeCase string
		KebabCase          string
		ScreamingKebabCase string
	}

	tests := []struct {
		arg  string
		want want
	}{
		{"", want{}},
		{" ", want{}},
		{"Simple word", want{
			CamelCase:          "simpleWord",
			PascalCase:         "SimpleWord",
			SnakeCase:          "simple_word",
			ScreamingSnakeCase: "SIMPLE_WORD",
			KebabCase:          "simple-word",
			ScreamingKebabCase: "SIMPLE-WORD",
		}},
		{"wordWith01number", want{
			CamelCase:          "wordWith01Number",
			PascalCase:         "WordWith01Number",
			SnakeCase:          "word_with_01_number",
			ScreamingSnakeCase: "WORD_WITH_01_NUMBER",
			KebabCase:          "word-with-01-number",
			ScreamingKebabCase: "WORD-WITH-01-NUMBER",
		}},
		{"HTTPServer", want{
			CamelCase:          "httpServer",
			PascalCase:         "HttpServer",
			SnakeCase:          "http_server",
			ScreamingSnakeCase: "HTTP_SERVER",
			KebabCase:          "http-server",
			ScreamingKebabCase: "HTTP-SERVER",
		}},
		{"IDs", want{
			CamelCase:          "iDs",
			PascalCase:         "IDs",
			SnakeCase:          "i_ds",
			ScreamingSnakeCase: "I_DS",
			KebabCase:          "i-ds",
			ScreamingKebabCase: "I-DS",
		}},
		{"用户のid", want{
			CamelCase:          "用户のId",
			PascalCase:         "用户のId",
			SnakeCase:          "用户の_id",
			ScreamingSnakeCase: "用户の_ID",
			KebabCase:          "用户の-id",
			ScreamingKebabCase: "用户の-ID",
		}},
		{"a.b", want{
			CamelCase:          "aB",
			PascalCase:         "AB",
			SnakeCase:          "a_b",
			ScreamingSnakeCase: "A_B",
			KebabCase:          "a-b",
			ScreamingKebabCase: "A-B",
		}},
		{"v2.0.1", want{
			CamelCase:          "v201",
			PascalCase:         "V201",
			SnakeCase:          "v_2_0_1",
			ScreamingSnakeCase: "V_2_0_1",
			KebabCase:          "v-2-0-1",
			ScreamingKebabCase: "V-2-0-1",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			checks := []struct {
				name string
				got  string
				want string
			}{
				{"CamelCase", CamelCase(tt.arg), tt.want.CamelCase},
				{"PascalCase", PascalCase(tt.arg), tt.want.PascalCase},
				{"SnakeCase", SnakeCase(tt.arg), tt.want.SnakeCase},
				{"ScreamingSnakeCase", ScreamingSnakeCase(tt.arg), tt.want.ScreamingSnakeCase},
				{"KebabCase", KebabCase(tt.arg), tt.want.KebabCase},
				{"ScreamingKebabCase", ScreamingKebabCase(tt.arg), tt.want.ScreamingKebabCase},
			}
			for _, check := range checks {
				if check.got != check.want {
					t.Errorf("%s(%q) = %q, want %q", check.name, tt.arg, check.got, check.want)
				}
			}
		})
	}
}
