package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindLatestPlanFile(t *testing.T) {
	t.Run("1つのmdファイルがある場合にそのパスを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		planFile := filepath.Join(tmpDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("test"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		result, err := findLatestPlanFile(tmpDir)
		if err != nil {
			t.Fatalf("findLatestPlanFile() エラー = %v", err)
		}

		if result != planFile {
			t.Errorf("findLatestPlanFile() = %v, want %v", result, planFile)
		}
	})

	t.Run("複数mdファイルがある場合に最新のものを返す", func(t *testing.T) {
		tmpDir := t.TempDir()

		// 古いファイル
		oldFile := filepath.Join(tmpDir, "old.md")
		if err := os.WriteFile(oldFile, []byte("old"), 0600); err != nil {
			t.Fatalf("古いファイルの作成に失敗: %v", err)
		}

		// 少し待ってから新しいファイルを作成
		time.Sleep(10 * time.Millisecond)

		newFile := filepath.Join(tmpDir, "new.md")
		if err := os.WriteFile(newFile, []byte("new"), 0600); err != nil {
			t.Fatalf("新しいファイルの作成に失敗: %v", err)
		}

		result, err := findLatestPlanFile(tmpDir)
		if err != nil {
			t.Fatalf("findLatestPlanFile() エラー = %v", err)
		}

		if result != newFile {
			t.Errorf("findLatestPlanFile() = %v, want %v", result, newFile)
		}
	})

	t.Run("plansディレクトリが存在しない場合のハンドリング", func(t *testing.T) {
		_, err := findLatestPlanFile("/nonexistent/plans")
		if err == nil {
			t.Error("findLatestPlanFile() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("mdファイルがない場合にエラーを返す", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := findLatestPlanFile(tmpDir)
		if err == nil {
			t.Error("findLatestPlanFile() エラーが期待されましたが、nilが返されました")
		}
	})
}

func TestExtractTitle(t *testing.T) {
	t.Run("最初の#_見出しがある場合にタイトルを抽出できる", func(t *testing.T) {
		content := `# プランタイトル

## セクション1
内容`
		result := extractTitle(content)
		expected := "プランタイトル"

		if result != expected {
			t.Errorf("extractTitle() = %v, want %v", result, expected)
		}
	})

	t.Run("見出しがない場合に空文字を返す", func(t *testing.T) {
		content := `内容のみ`
		result := extractTitle(content)

		if result != "" {
			t.Errorf("extractTitle() = %v, want empty string", result)
		}
	})

	t.Run("複数の見出しがある場合に最初のものを返す", func(t *testing.T) {
		content := `# 最初のタイトル

## セクション

# 2番目のタイトル`
		result := extractTitle(content)
		expected := "最初のタイトル"

		if result != expected {
			t.Errorf("extractTitle() = %v, want %v", result, expected)
		}
	})
}

func TestRemoveTitle(t *testing.T) {
	t.Run("#_見出しがある場合に本文からその行が除去される", func(t *testing.T) {
		content := `# タイトル

## セクション1
内容`
		result := removeTitle(content)
		expected := `
## セクション1
内容`

		if result != expected {
			t.Errorf("removeTitle() = %v, want %v", result, expected)
		}
	})

	t.Run("見出しがない場合に元の内容を返す", func(t *testing.T) {
		content := `## セクション
内容`
		result := removeTitle(content)

		if result != content {
			t.Errorf("removeTitle() = %v, want %v", result, content)
		}
	})
}

func TestSanitizePostName(t *testing.T) {
	t.Run("タイトル中の特殊文字が_に置換される", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"#を置換", "タイトル#タグ", "タイトル_タグ"},
			{"/を置換", "カテゴリ/サブ", "カテゴリ_サブ"},
			{"\\を置換", "パス\\ファイル", "パス_ファイル"},
			{"|を置換", "A|B", "A_B"},
			{"[]を置換", "[リンク]", "_リンク_"},
			{"<>を置換", "<タグ>", "_タグ_"},
			{"全角括弧を置換", "（注釈）", "_注釈_"},
			{"全角コロンを置換", "タイトル：サブ", "タイトル_サブ"},
			{"複数置換", "A#B/C\\D", "A_B_C_D"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := sanitizePostName(tc.input)
				if result != tc.expected {
					t.Errorf("sanitizePostName(%v) = %v, want %v", tc.input, result, tc.expected)
				}
			})
		}
	})

	t.Run("タイトル中の制御文字が除去される", func(t *testing.T) {
		input := "タイトル\nサブ\tタブ"
		result := sanitizePostName(input)
		expected := "タイトルサブタブ"

		if result != expected {
			t.Errorf("sanitizePostName(%v) = %v, want %v", input, result, expected)
		}
	})

	t.Run("タイトルが255バイトを超える場合にUTF-8ルーン境界でトランケートされる", func(t *testing.T) {
		// 255バイトを超える文字列（日本語3バイト文字で構成）
		input := strings.Repeat("あ", 100) // 300バイト

		result := sanitizePostName(input)

		if len(result) > 255 {
			t.Errorf("sanitizePostName() length = %v, want <= 255", len(result))
		}

		// UTF-8として有効であることを確認
		if !isValidUTF8(result) {
			t.Error("sanitizePostName() 結果がUTF-8として不正です")
		}
	})
}

func TestBuildPostName(t *testing.T) {
	t.Run("見出しがある場合にタイトルを使用", func(t *testing.T) {
		content := `# プランタイトル

内容`
		filename := "plan.md"

		result := buildPostName(content, filename)
		expected := "プランタイトル"

		if result != expected {
			t.Errorf("buildPostName() = %v, want %v", result, expected)
		}
	})

	t.Run("見出しがない場合にファイル名（.md拡張子除去済み）をタイトルに使う", func(t *testing.T) {
		content := `内容のみ`
		filename := "my-plan.md"

		result := buildPostName(content, filename)
		expected := "my-plan"

		if result != expected {
			t.Errorf("buildPostName() = %v, want %v", result, expected)
		}
	})

	t.Run("見出しのみ空白の場合にファイル名にフォールバックする", func(t *testing.T) {
		content := `#

内容`
		filename := "fallback.md"

		result := buildPostName(content, filename)
		expected := "fallback"

		if result != expected {
			t.Errorf("buildPostName() = %v, want %v", result, expected)
		}
	})

	t.Run("先頭・末尾の空白がトリムされる", func(t *testing.T) {
		content := `#   タイトル

内容`
		filename := "plan.md"

		result := buildPostName(content, filename)
		expected := "タイトル"

		if result != expected {
			t.Errorf("buildPostName() = %v, want %v", result, expected)
		}
	})
}

// isValidUTF8 はUTF-8として有効かチェックするヘルパー
func isValidUTF8(s string) bool {
	for len(s) > 0 {
		r, size := decodeRune(s)
		if r == -1 {
			return false
		}
		s = s[size:]
	}
	return true
}

// decodeRune は先頭のruneをデコードする（簡易版）
func decodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return -1, 0
	}

	// 1バイト文字
	if s[0] < 0x80 {
		return rune(s[0]), 1
	}

	// 2バイト文字
	if len(s) >= 2 && s[0]&0xE0 == 0xC0 {
		return rune(s[0]&0x1F)<<6 | rune(s[1]&0x3F), 2
	}

	// 3バイト文字
	if len(s) >= 3 && s[0]&0xF0 == 0xE0 {
		return rune(s[0]&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F), 3
	}

	// 4バイト文字
	if len(s) >= 4 && s[0]&0xF8 == 0xF0 {
		return rune(s[0]&0x07)<<18 | rune(s[1]&0x3F)<<12 | rune(s[2]&0x3F)<<6 | rune(s[3]&0x3F), 4
	}

	return -1, 0
}
