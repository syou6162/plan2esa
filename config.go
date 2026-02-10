package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション設定
type Config struct {
	Esa struct {
		TeamName string `yaml:"team_name"`
	} `yaml:"esa"`
	Post struct {
		Category string `yaml:"category"`
	} `yaml:"post"`
}

// loadConfig は設定ファイルを読み込み、検証します
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig は設定の妥当性を検証します
func validateConfig(config *Config) error {
	// team_nameの検証
	if config.Esa.TeamName == "" {
		return fmt.Errorf("team_name cannot be empty")
	}

	// team_nameは [A-Za-z0-9_-] のみ許可
	for _, c := range config.Esa.TeamName {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return fmt.Errorf("team_name contains invalid characters (only A-Z, a-z, 0-9, _, - allowed): %s", config.Esa.TeamName)
		}
	}

	return nil
}

// getDefaultConfigPath はデフォルトの設定ファイルパスを返します
func getDefaultConfigPath() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "plan2esa", "config.yaml")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// エラーの場合は相対パスにフォールバック
		return ".config/plan2esa/config.yaml"
	}

	return filepath.Join(homeDir, ".config", "plan2esa", "config.yaml")
}

// getAccessToken は環境変数からアクセストークンを取得します
func getAccessToken() (string, error) {
	token := os.Getenv("ESA_ACCESS_TOKEN")
	if token == "" {
		return "", fmt.Errorf("ESA_ACCESS_TOKEN environment variable is not set")
	}
	return token, nil
}

// buildCategory はカテゴリに日付を付与します
func buildCategory(baseCategory string, now time.Time) string {
	return fmt.Sprintf("%s/%04d/%02d/%02d", baseCategory, now.Year(), now.Month(), now.Day())
}

// getRepositoryName はGitリポジトリ名を取得します
func getRepositoryName() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		// gitリポジトリでない場合やremote未設定の場合は空文字
		return ""
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return ""
	}

	// URLからリポジトリ名を抽出
	// 例: https://github.com/user/repo.git → repo
	// 例: git@github.com:user/repo.git → repo
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}

	repoName := parts[len(parts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	if repoName == "" {
		return ""
	}

	return repoName
}
