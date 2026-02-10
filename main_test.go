package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetPlansDir(t *testing.T) {
	t.Run("CLAUDE_CODE_TMPDIRが設定されている場合にplansパスを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		result, err := getPlansDir()
		if err != nil {
			t.Fatalf("getPlansDir() エラー = %v", err)
		}

		expected := filepath.Join(tmpDir, "plans")
		if result != expected {
			t.Errorf("getPlansDir() = %v, want %v", result, expected)
		}
	})

	t.Run("CLAUDE_CODE_TMPDIRが未設定の場合にエラーを返す", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_TMPDIR")

		_, err := getPlansDir()
		if err == nil {
			t.Error("getPlansDir() エラーが期待されましたが、nilが返されました")
		}
	})
}

// mockEsaPoster はテスト用のEsaPosterモック
type mockEsaPoster struct {
	createPostCalled bool
	createPostError  error
	response         *EsaPostResponse
	searchResults    []EsaSearchResult
	searchError      error
	updatePostCalled bool
	updatePostError  error
}

func (m *mockEsaPoster) CreatePost(post EsaPost) (*EsaPostResponse, error) {
	m.createPostCalled = true
	if m.createPostError != nil {
		return nil, m.createPostError
	}
	return m.response, nil
}

func (m *mockEsaPoster) SearchPosts(query string) ([]EsaSearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockEsaPoster) UpdatePost(postNumber int, post EsaPost) (*EsaPostResponse, error) {
	m.updatePostCalled = true
	if m.updatePostError != nil {
		return nil, m.updatePostError
	}
	return m.response, nil
}

func TestRun(t *testing.T) {
	t.Run("plansディレクトリが存在しない場合にノーオペで正常終了する", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// plansディレクトリを作成しない

		mock := &mockEsaPoster{}
		err := run("", false, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれましたが、呼ばれないはずです")
		}
	})

	t.Run("プランファイルがない場合にノーオペで正常終了する", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// plansディレクトリは作成するが、ファイルは置かない
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}

		mock := &mockEsaPoster{}
		err := run("", false, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれましたが、呼ばれないはずです")
		}
	})

	t.Run("プランファイルがある場合にAPIに投稿する", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// 設定ファイルを作成
		configDir := filepath.Join(tmpDir, "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("configディレクトリの作成に失敗: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Test/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		planContent := `# テストプラン

## 内容
テスト用のプランファイル
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		// 環境変数を設定
		os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer os.Unsetenv("ESA_ACCESS_TOKEN")

		mock := &mockEsaPoster{
			response: &EsaPostResponse{
				Number: 123,
				URL:    "https://test-team.esa.io/posts/123",
			},
		}

		err := run(configPath, false, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if !mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれませんでした")
		}
	})

	t.Run("dry-runモードでtoken取得・API呼び出しをスキップする", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// 設定ファイルを作成
		configDir := filepath.Join(tmpDir, "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("configディレクトリの作成に失敗: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Test/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		planContent := `# テストプラン

## 内容
dry-runテスト用のプランファイル
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		// 環境変数ESA_ACCESS_TOKENを設定（検索に必要）
		_ = os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer func() {
			_ = os.Unsetenv("ESA_ACCESS_TOKEN")
		}()

		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{}, // 既存記事なし
		}

		err := run(configPath, true, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.createPostCalled {
			t.Error("run() dry-runモードでCreatePostが呼ばれましたが、呼ばれないはずです")
		}

		if mock.updatePostCalled {
			t.Error("run() dry-runモードでUpdatePostが呼ばれましたが、呼ばれないはずです")
		}
	})

	t.Run("CLAUDE_CODE_TMPDIRが未設定の場合にエラーを返す", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_TMPDIR")

		mock := &mockEsaPoster{}
		err := run("", false, mock)

		if err == nil {
			t.Error("run() エラーが期待されましたが、nilが返されました")
		}

		if !strings.Contains(err.Error(), "CLAUDE_CODE_TMPDIR") {
			t.Errorf("run() エラーメッセージに 'CLAUDE_CODE_TMPDIR' が含まれていません: %v", err)
		}
	})

	t.Run("設定ファイルが不正な場合にエラーを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// 不正な設定ファイルを作成
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(configPath, []byte("invalid yaml {{{"), 0600); err != nil {
			t.Fatalf("不正な設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("# テスト\n内容"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		mock := &mockEsaPoster{}
		err := run(configPath, false, mock)

		if err == nil {
			t.Error("run() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("既存記事が見つかった場合に上書き更新する", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer func() {
			_ = os.Unsetenv("CLAUDE_CODE_TMPDIR")
		}()

		configPath := filepath.Join(tmpDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Test/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		planContent := `# 既存タイトル

## 内容
更新テスト用のプランファイル
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		_ = os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer func() {
			_ = os.Unsetenv("ESA_ACCESS_TOKEN")
		}()

		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "既存タイトル", Category: "Test/Plans/2026/02/11"},
			},
			response: &EsaPostResponse{
				Number: 999,
				Name:   "既存タイトル",
				URL:    "https://test-team.esa.io/posts/999",
			},
		}

		err := run(configPath, false, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれましたが、UpdatePostが呼ばれるはずです")
		}

		if !mock.updatePostCalled {
			t.Error("run() UpdatePostが呼ばれませんでした")
		}
	})

	t.Run("ESA_ACCESS_TOKENが未設定でdry-runでない場合にエラーを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// 設定ファイルを作成
		configDir := filepath.Join(tmpDir, "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("configディレクトリの作成に失敗: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Test/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("# テスト\n内容"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		// ESA_ACCESS_TOKENを未設定
		os.Unsetenv("ESA_ACCESS_TOKEN")

		mock := &mockEsaPoster{}
		err := run(configPath, false, mock)

		if err == nil {
			t.Error("run() エラーが期待されましたが、nilが返されました")
		}

		if !strings.Contains(err.Error(), "ESA_ACCESS_TOKEN") {
			t.Errorf("run() エラーメッセージに 'ESA_ACCESS_TOKEN' が含まれていません: %v", err)
		}
	})
}

func TestRunIntegration(t *testing.T) {
	t.Run("エンドツーエンドでプランファイルをesa.ioに投稿する", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer os.Unsetenv("CLAUDE_CODE_TMPDIR")

		// 設定ファイルを作成
		configDir := filepath.Join(tmpDir, "config")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("configディレクトリの作成に失敗: %v", err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Test/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		// プランファイルを作成（タイトルあり）
		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "integration-plan.md")
		planContent := `# 統合テストプラン

## 背景
統合テスト用のプランファイル

## 実装内容
- 機能A
- 機能B
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		// 環境変数を設定
		os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer os.Unsetenv("ESA_ACCESS_TOKEN")

		mock := &mockEsaPoster{
			response: &EsaPostResponse{
				Number: 456,
				URL:    "https://test-team.esa.io/posts/456",
			},
		}

		err := run(configPath, false, mock)

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if !mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれませんでした")
		}
	})
}
