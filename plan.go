package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// findLatestPlanFile は最新のプランファイルを検索します
func findLatestPlanFile(plansDir string) (string, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return "", fmt.Errorf("failed to read plans directory: %w", err)
	}

	var latestFile string
	var latestTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(plansDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > latestTime {
			latestTime = info.ModTime().Unix()
			latestFile = fullPath
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("no .md files found in %s", plansDir)
	}

	return latestFile, nil
}

// extractTitle はMarkdownの最初の # 見出しを抽出します
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}

// removeTitle は最初の # 見出し行を本文から除去します
func removeTitle(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			// この行を除去して残りを結合
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return content
}

// sanitizePostName はタイトルをサニタイズします
func sanitizePostName(name string) string {
	// 置換対象文字を _に置換
	replacements := map[rune]rune{
		'#':  '_',
		'/':  '_',
		'\\': '_',
		'|':  '_',
		'[':  '_',
		']':  '_',
		'<':  '_',
		'>':  '_',
		'（':  '_',
		'）':  '_',
		'：':  '_',
	}

	var result strings.Builder
	for _, r := range name {
		if replacement, ok := replacements[r]; ok {
			result.WriteRune(replacement)
		} else if unicode.IsControl(r) {
			// 制御文字は除去
			continue
		} else {
			result.WriteRune(r)
		}
	}

	sanitized := result.String()

	// 255バイト上限でトランケート（UTF-8ルーン境界）
	if len(sanitized) > 255 {
		// 255バイトでトランケートし、UTF-8として有効になるまで削る
		sanitized = sanitized[:255]
		for len(sanitized) > 0 && !utf8.ValidString(sanitized) {
			sanitized = sanitized[:len(sanitized)-1]
		}
	}

	return sanitized
}

// buildPostName はタイトルを決定します
func buildPostName(content, filename string) string {
	// まず見出しから抽出を試みる
	title := extractTitle(content)

	// サニタイズ
	title = sanitizePostName(title)

	// TrimSpace 適用
	title = strings.TrimSpace(title)

	// 空文字ならファイル名にフォールバック
	if title == "" {
		title = strings.TrimSuffix(filename, ".md")
	}

	return title
}
