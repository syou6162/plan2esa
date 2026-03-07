package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	lastCreatedPost  EsaPost
	response         *EsaPostResponse
	searchResults    []EsaSearchResult
	searchError      error
	searchCalled     bool
	lastSearchQuery  string
	updatePostCalled bool
	updatePostError  error
	lastUpdatedPost  EsaPost
}

func (m *mockEsaPoster) CreatePost(post EsaPost) (*EsaPostResponse, error) {
	m.createPostCalled = true
	m.lastCreatedPost = post
	if m.createPostError != nil {
		return nil, m.createPostError
	}
	return m.response, nil
}

func (m *mockEsaPoster) SearchPosts(query string) ([]EsaSearchResult, error) {
	m.searchCalled = true
	m.lastSearchQuery = query
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockEsaPoster) UpdatePost(postNumber int, post EsaPost) (*EsaPostResponse, error) {
	m.updatePostCalled = true
	m.lastUpdatedPost = post
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
		err := run("", false, mock, "")

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
		err := run("", false, mock, "")

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
			searchResults: []EsaSearchResult{}, // 検索結果0件
		}

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		// SearchPostsが呼ばれたことを確認
		if !mock.searchCalled {
			t.Error("run() SearchPostsが呼ばれませんでした")
		}

		// 検索結果が0件なのでCreatePostが呼ばれることを確認
		if !mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれませんでした")
		}

		// 検索結果が0件なのでUpdatePostは呼ばれないことを確認
		if mock.updatePostCalled {
			t.Error("run() UpdatePostが呼ばれましたが、呼ばれないはずです")
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

		err := run(configPath, true, mock, "")

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

	t.Run("dry-runモードでローカルが古い場合にスキップ情報を表示する", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		t.Setenv("ESA_ACCESS_TOKEN", "test-token")

		configPath := filepath.Join(tmpDir, "config.yaml")
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n"
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("# dry-runスキップテスト\n本文"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		// ローカルファイルのModTimeを過去に設定
		pastTime := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
		if err := os.Chtimes(planFile, pastTime, pastTime); err != nil {
			t.Fatalf("os.Chtimes() エラー = %v", err)
		}

		esaUpdatedAt := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "dry-runスキップテスト", Category: expectedCategory, UpdatedAt: esaUpdatedAt},
			},
		}

		err := run(configPath, true, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}
		if mock.updatePostCalled {
			t.Error("dry-runモードでUpdatePostが呼ばれましたが、呼ばれないはずです")
		}
	})

	t.Run("CLAUDE_CODE_TMPDIRが未設定の場合にエラーを返す", func(t *testing.T) {
		os.Unsetenv("CLAUDE_CODE_TMPDIR")

		mock := &mockEsaPoster{}
		err := run("", false, mock, "")

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
		err := run(configPath, false, mock, "")

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

		// 実行時の日付に基づいてカテゴリを構築
		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)

		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "既存タイトル", Category: expectedCategory},
			},
			response: &EsaPostResponse{
				Number: 999,
				Name:   "既存タイトル",
				URL:    "https://test-team.esa.io/posts/999",
			},
		}

		err := run(configPath, false, mock, "")

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
		err := run(configPath, false, mock, "")

		if err == nil {
			t.Error("run() エラーが期待されましたが、nilが返されました")
		}

		if !strings.Contains(err.Error(), "ESA_ACCESS_TOKEN") {
			t.Errorf("run() エラーメッセージに 'ESA_ACCESS_TOKEN' が含まれていません: %v", err)
		}
	})
}

func TestRunTimestampSkip(t *testing.T) {
	setupTestEnv := func(t *testing.T) (tmpDir, configPath, planFile string) {
		t.Helper()
		tmpDir = t.TempDir()
		t.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		t.Setenv("ESA_ACCESS_TOKEN", "test-token")

		configPath = filepath.Join(tmpDir, "config.yaml")
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n"
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile = filepath.Join(plansDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("# スキップテスト\n本文"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}
		return
	}

	t.Run("ローカルファイルがesa側より古い場合にスキップする", func(t *testing.T) {
		_, configPath, planFile := setupTestEnv(t)

		// ローカルファイルのModTimeを過去に設定
		pastTime := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
		if err := os.Chtimes(planFile, pastTime, pastTime); err != nil {
			t.Fatalf("os.Chtimes() エラー = %v", err)
		}

		// esa側のUpdatedAtは未来
		esaUpdatedAt := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "スキップテスト", Category: expectedCategory, UpdatedAt: esaUpdatedAt},
			},
			response: &EsaPostResponse{Number: 999, URL: "https://test-team.esa.io/posts/999"},
		}

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}
		if mock.updatePostCalled {
			t.Error("UpdatePostが呼ばれましたが、スキップされるはずです")
		}
		if mock.createPostCalled {
			t.Error("CreatePostが呼ばれましたが、スキップされるはずです")
		}
	})

	t.Run("ローカルファイルがesa側より新しい場合に更新する", func(t *testing.T) {
		_, configPath, planFile := setupTestEnv(t)

		// esa側のUpdatedAtは過去
		esaUpdatedAt := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
		// ローカルファイルのModTimeを未来に設定
		futureTime := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
		if err := os.Chtimes(planFile, futureTime, futureTime); err != nil {
			t.Fatalf("os.Chtimes() エラー = %v", err)
		}

		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "スキップテスト", Category: expectedCategory, UpdatedAt: esaUpdatedAt},
			},
			response: &EsaPostResponse{Number: 999, URL: "https://test-team.esa.io/posts/999"},
		}

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}
		if !mock.updatePostCalled {
			t.Error("UpdatePostが呼ばれませんでした")
		}
	})

	t.Run("同一タイムスタンプの場合に更新する", func(t *testing.T) {
		_, configPath, planFile := setupTestEnv(t)

		sameTime := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
		if err := os.Chtimes(planFile, sameTime, sameTime); err != nil {
			t.Fatalf("os.Chtimes() エラー = %v", err)
		}

		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 999, Name: "スキップテスト", Category: expectedCategory, UpdatedAt: sameTime},
			},
			response: &EsaPostResponse{Number: 999, URL: "https://test-team.esa.io/posts/999"},
		}

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}
		if !mock.updatePostCalled {
			t.Error("UpdatePostが呼ばれませんでした（同一タイムスタンプは更新するはず）")
		}
	})

	t.Run("UpdatedAtがゼロ値の場合に従来通り更新する", func(t *testing.T) {
		_, configPath, _ := setupTestEnv(t)

		now := time.Now()
		expectedCategory := buildCategory("Test/Plans", now)
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				// UpdatedAtなし（ゼロ値）
				{Number: 999, Name: "スキップテスト", Category: expectedCategory},
			},
			response: &EsaPostResponse{Number: 999, URL: "https://test-team.esa.io/posts/999"},
		}

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}
		if !mock.updatePostCalled {
			t.Error("UpdatePostが呼ばれませんでした（UpdatedAtゼロ値は更新するはず）")
		}
	})

}

func TestRunMessage(t *testing.T) {
	setupEnv := func(t *testing.T, configContent string) (configPath string) {
		t.Helper()
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		t.Setenv("ESA_ACCESS_TOKEN", "test-token")

		configPath = filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "plan.md")
		if err := os.WriteFile(planFile, []byte("# messageテスト\n本文"), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}
		return
	}

	t.Run("CLIフラグのmessageが使われる", func(t *testing.T) {
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n  message: \"configのメモ\"\n"
		configPath := setupEnv(t, configContent)

		mock := &mockEsaPoster{
			response:      &EsaPostResponse{Number: 1, URL: "https://test.esa.io/posts/1"},
			searchResults: []EsaSearchResult{},
		}

		err := run(configPath, false, mock, "CLIのメモ")
		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.lastCreatedPost.Message != "CLIのメモ" {
			t.Errorf("Message = %v, want CLIのメモ", mock.lastCreatedPost.Message)
		}
	})

	t.Run("CLIフラグがない場合はconfigのmessageが使われる", func(t *testing.T) {
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n  message: \"configのメモ\"\n"
		configPath := setupEnv(t, configContent)

		mock := &mockEsaPoster{
			response:      &EsaPostResponse{Number: 1, URL: "https://test.esa.io/posts/1"},
			searchResults: []EsaSearchResult{},
		}

		err := run(configPath, false, mock, "")
		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.lastCreatedPost.Message != "configのメモ" {
			t.Errorf("Message = %v, want configのメモ", mock.lastCreatedPost.Message)
		}
	})

	t.Run("どちらも未指定の場合はmessageが空文字でAPIに送信されない", func(t *testing.T) {
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n"
		configPath := setupEnv(t, configContent)

		mock := &mockEsaPoster{
			response:      &EsaPostResponse{Number: 1, URL: "https://test.esa.io/posts/1"},
			searchResults: []EsaSearchResult{},
		}

		err := run(configPath, false, mock, "")
		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if mock.lastCreatedPost.Message != "" {
			t.Errorf("Message = %v, want 空文字", mock.lastCreatedPost.Message)
		}
	})

	t.Run("空白のみのmessageはエラーになる", func(t *testing.T) {
		configContent := "esa:\n  team_name: \"test-team\"\npost:\n  category: \"Test/Plans\"\n"
		configPath := setupEnv(t, configContent)

		mock := &mockEsaPoster{}

		err := run(configPath, false, mock, "   ")
		if err == nil {
			t.Error("run() エラーが期待されましたが、nilが返されました")
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

		err := run(configPath, false, mock, "")

		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		if !mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれませんでした")
		}
	})

	t.Run("スペースを含むカテゴリで検索クエリにダブルクォートが付く", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("CLAUDE_CODE_TMPDIR", tmpDir)
		defer func() {
			_ = os.Unsetenv("CLAUDE_CODE_TMPDIR")
		}()

		configPath := filepath.Join(tmpDir, "config.yaml")
		configContent := `esa:
  team_name: "test-team"
post:
  category: "Claude Code/Plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "test.md")
		planContent := `# テストプラン

## 内容
スペース含むカテゴリのテスト
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		_ = os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer func() {
			_ = os.Unsetenv("ESA_ACCESS_TOKEN")
		}()

		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{},
			response: &EsaPostResponse{
				Number: 100,
				URL:    "https://test-team.esa.io/posts/100",
			},
		}

		err := run(configPath, false, mock, "")
		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		// 検索クエリにin:"が含まれることを確認
		if !strings.Contains(mock.lastSearchQuery, `in:"`) {
			t.Errorf("検索クエリに in:\" が含まれていません: %s", mock.lastSearchQuery)
		}

		// カテゴリ全体がクォートで囲まれていることを確認（例: in:"Claude Code/Plans/2026/02/11"）
		expectedCategory := "Claude Code/Plans"
		if !strings.Contains(mock.lastSearchQuery, expectedCategory) {
			t.Errorf("検索クエリにカテゴリ %s が含まれていません: %s", expectedCategory, mock.lastSearchQuery)
		}
	})

	t.Run("検索結果にName/Categoryが異なる記事が含まれる場合は無視してCreatePostする", func(t *testing.T) {
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

		plansDir := filepath.Join(tmpDir, "plans")
		if err := os.MkdirAll(plansDir, 0755); err != nil {
			t.Fatalf("plansディレクトリの作成に失敗: %v", err)
		}
		planFile := filepath.Join(plansDir, "test.md")
		planContent := `# マッチテスト

## 内容
厳密一致のテスト
`
		if err := os.WriteFile(planFile, []byte(planContent), 0600); err != nil {
			t.Fatalf("プランファイルの作成に失敗: %v", err)
		}

		_ = os.Setenv("ESA_ACCESS_TOKEN", "test-token")
		defer func() {
			_ = os.Unsetenv("ESA_ACCESS_TOKEN")
		}()

		// 検索結果にはサフィックス付きの記事のみが返る（Name/Categoryが完全一致しない）
		mock := &mockEsaPoster{
			searchResults: []EsaSearchResult{
				{Number: 200, Name: "マッチテスト (1)", Category: "Test/Plans/2026/02/11"},
				{Number: 201, Name: "マッチテスト (2)", Category: "Test/Plans/2026/02/11"},
			},
			response: &EsaPostResponse{
				Number: 300,
				URL:    "https://test-team.esa.io/posts/300",
			},
		}

		err := run(configPath, false, mock, "")
		if err != nil {
			t.Fatalf("run() エラー = %v", err)
		}

		// Name/Categoryが完全一致しないので、CreatePostが呼ばれる
		if !mock.createPostCalled {
			t.Error("run() CreatePostが呼ばれませんでした")
		}

		// UpdatePostは呼ばれない
		if mock.updatePostCalled {
			t.Error("run() UpdatePostが呼ばれましたが、呼ばれないはずです")
		}
	})
}
