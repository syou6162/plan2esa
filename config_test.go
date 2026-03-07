package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("有効なYAML設定ファイルを正しく読み込み、検証をパスする", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `esa:
  team_name: "yasuhisa"
post:
  category: "Claude Code/plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		config, err := loadConfig(configPath)
		if err != nil {
			t.Fatalf("loadConfig() エラー = %v", err)
		}

		if config.Esa.TeamName != "yasuhisa" {
			t.Errorf("team_name = %v, want %v", config.Esa.TeamName, "yasuhisa")
		}

		if config.Post.Category != "Claude Code/plans" {
			t.Errorf("category = %v, want %v", config.Post.Category, "Claude Code/plans")
		}
	})

	t.Run("messageが設定されている場合に読み込める", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `esa:
  team_name: "yasuhisa"
post:
  category: "Claude Code/plans"
  message: "plan2esaから投稿"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		config, err := loadConfig(configPath)
		if err != nil {
			t.Fatalf("loadConfig() エラー = %v", err)
		}

		if config.Post.Message != "plan2esaから投稿" {
			t.Errorf("message = %v, want plan2esaから投稿", config.Post.Message)
		}
	})

	t.Run("messageが未設定の場合は空文字になる", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `esa:
  team_name: "yasuhisa"
post:
  category: "Claude Code/plans"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		config, err := loadConfig(configPath)
		if err != nil {
			t.Fatalf("loadConfig() エラー = %v", err)
		}

		if config.Post.Message != "" {
			t.Errorf("message = %v, want 空文字", config.Post.Message)
		}
	})

	t.Run("設定ファイルが存在しない場合にエラーを返す", func(t *testing.T) {
		_, err := loadConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Error("loadConfig() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("不正なYAMLの場合にエラーを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		invalidYAML := `esa:
  team_name: "yasuhisa
  # 閉じ引用符がない不正なYAML
`
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
			t.Fatalf("設定ファイルの作成に失敗: %v", err)
		}

		_, err := loadConfig(configPath)
		if err == nil {
			t.Error("loadConfig() エラーが期待されましたが、nilが返されました")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("team_nameが空の場合にエラーを返す", func(t *testing.T) {
		config := &Config{}
		config.Esa.TeamName = ""

		err := validateConfig(config)
		if err == nil {
			t.Error("validateConfig() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("team_nameに不正な文字が含まれる場合にエラーを返す", func(t *testing.T) {
		testCases := []struct {
			name     string
			teamName string
		}{
			{"スラッシュ", "team/name"},
			{"スペース", "team name"},
			{"ドット", "team.name"},
			{"全角文字", "チーム名"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := &Config{}
				config.Esa.TeamName = tc.teamName

				err := validateConfig(config)
				if err == nil {
					t.Errorf("validateConfig() エラーが期待されましたが、nilが返されました (teamName=%v)", tc.teamName)
				}
			})
		}
	})

	t.Run("有効なteam_nameの場合にエラーを返さない", func(t *testing.T) {
		testCases := []string{
			"yasuhisa",
			"team-name",
			"team_name",
			"team123",
			"UPPERCASE",
		}

		for _, teamName := range testCases {
			t.Run(teamName, func(t *testing.T) {
				config := &Config{}
				config.Esa.TeamName = teamName
				config.Post.Category = "category"

				err := validateConfig(config)
				if err != nil {
					t.Errorf("validateConfig() エラー = %v, 期待 nil (teamName=%v)", err, teamName)
				}
			})
		}
	})

	t.Run("categoryが空の場合にエラーを返す", func(t *testing.T) {
		config := &Config{}
		config.Esa.TeamName = "valid-team"
		config.Post.Category = ""

		err := validateConfig(config)
		if err == nil {
			t.Error("validateConfig() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("categoryの末尾スラッシュが除去される", func(t *testing.T) {
		config := &Config{}
		config.Esa.TeamName = "valid-team"
		config.Post.Category = "Claude Code/plans/"

		err := validateConfig(config)
		if err != nil {
			t.Fatalf("validateConfig() エラー = %v", err)
		}

		if config.Post.Category != "Claude Code/plans" {
			t.Errorf("category = %v, want %v", config.Post.Category, "Claude Code/plans")
		}
	})
}

func TestGetAccessToken(t *testing.T) {
	t.Run("環境変数ESA_ACCESS_TOKENからaccess_tokenを取得できる", func(t *testing.T) {
		os.Setenv("ESA_ACCESS_TOKEN", "test-token-123")
		defer os.Unsetenv("ESA_ACCESS_TOKEN")

		token, err := getAccessToken()
		if err != nil {
			t.Fatalf("getAccessToken() エラー = %v", err)
		}

		if token != "test-token-123" {
			t.Errorf("getAccessToken() = %v, want %v", token, "test-token-123")
		}
	})

	t.Run("環境変数ESA_ACCESS_TOKENが未設定の場合にエラーを返す", func(t *testing.T) {
		os.Unsetenv("ESA_ACCESS_TOKEN")

		_, err := getAccessToken()
		if err == nil {
			t.Error("getAccessToken() エラーが期待されましたが、nilが返されました")
		}
	})
}

func TestBuildCategory(t *testing.T) {
	t.Run("カテゴリに実行日の日付が自動付与される", func(t *testing.T) {
		baseCategory := "Claude Code/plans"
		now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

		result := buildCategory(baseCategory, now)
		expected := "Claude Code/plans/2026/02/10"

		if result != expected {
			t.Errorf("buildCategory() = %v, want %v", result, expected)
		}
	})

	t.Run("一桁の月日も0埋めされる", func(t *testing.T) {
		baseCategory := "Test"
		now := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)

		result := buildCategory(baseCategory, now)
		expected := "Test/2026/01/05"

		if result != expected {
			t.Errorf("buildCategory() = %v, want %v", result, expected)
		}
	})
}

func TestGetRepositoryName(t *testing.T) {
	t.Run("Git リポジトリ名がタグとして自動付与される", func(t *testing.T) {
		// 実際のgit環境でテスト
		// このリポジトリは plan2esa
		repoName := getRepositoryName()

		// gitリポジトリでない、またはremoteが未設定の場合は空文字が返る
		// テストでは空文字か "plan2esa" のどちらかを許容
		if repoName != "" && repoName != "plan2esa" {
			t.Errorf("getRepositoryName() = %v, want \"\" or \"plan2esa\"", repoName)
		}
	})
}

func TestGetDefaultConfigPath(t *testing.T) {
	t.Run("XDG_CONFIG_HOMEが設定されている場合", func(t *testing.T) {
		os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
		defer os.Unsetenv("XDG_CONFIG_HOME")

		path, err := getDefaultConfigPath()
		if err != nil {
			t.Fatalf("getDefaultConfigPath() エラー = %v", err)
		}

		expected := "/tmp/xdg/plan2esa/config.yaml"

		if path != expected {
			t.Errorf("getDefaultConfigPath() = %v, want %v", path, expected)
		}
	})

	t.Run("XDG_CONFIG_HOMEが未設定の場合", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")

		path, err := getDefaultConfigPath()
		if err != nil {
			t.Fatalf("getDefaultConfigPath() エラー = %v", err)
		}

		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".config", "plan2esa", "config.yaml")

		if path != expected {
			t.Errorf("getDefaultConfigPath() = %v, want %v", path, expected)
		}
	})
}
