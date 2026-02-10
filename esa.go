package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// EsaPost はesa.io投稿リクエスト
type EsaPost struct {
	Name     string   `json:"name"`
	BodyMd   string   `json:"body_md"`
	Category string   `json:"category"`
	Wip      bool     `json:"wip"`
	Tags     []string `json:"tags,omitempty"`
}

// EsaPostResponse はesa.io投稿レスポンス
type EsaPostResponse struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	URL    string `json:"url"`
}

// EsaPoster はesa.io投稿インターフェース
type EsaPoster interface {
	CreatePost(post EsaPost) (*EsaPostResponse, error)
}

// EsaClient はesa.io APIクライアント
type EsaClient struct {
	TeamName    string
	AccessToken string
	HTTPClient  *http.Client
}

// NewEsaClient は新しいEsaClientを作成します
func NewEsaClient(teamName, accessToken string) *EsaClient {
	// セキュアなHTTPクライアント設定
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // TLS 1.2以上を要求
		},
		Proxy: nil, // HTTPプロキシを無効化
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// リダイレクトを禁止
			return http.ErrUseLastResponse
		},
	}

	return &EsaClient{
		TeamName:    teamName,
		AccessToken: accessToken,
		HTTPClient:  client,
	}
}

// CreatePost は新規記事を作成します
func (c *EsaClient) CreatePost(post EsaPost) (*EsaPostResponse, error) {
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts", c.TeamName)

	// esa.io APIは {"post": {...}} 形式を要求
	wrapped := map[string]interface{}{
		"post": post,
	}

	jsonData, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを制限付きで読み込み（10MB上限+1バイトで超過検知）
	const maxResponseSize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// レスポンスサイズが上限を超えている場合はエラー
	if len(respBody) > maxResponseSize {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseSize)
	}

	// ステータスコードチェック
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// エラーメッセージをサニタイズ（最大500文字、制御文字除去）
		errMsg := sanitizeErrorMessage(string(respBody))
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errMsg)
	}

	// レスポンスをパース
	var result EsaPostResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// sanitizeErrorMessage はエラーメッセージをサニタイズします
func sanitizeErrorMessage(msg string) string {
	// 最大500文字に制限
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}

	// 制御文字を除去
	var sb strings.Builder
	for _, r := range msg {
		if !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
