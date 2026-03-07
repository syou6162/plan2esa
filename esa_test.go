package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// mockTransport はHTTPリクエストをモックするためのhttp.RoundTripper
type mockTransport struct {
	response    *http.Response
	err         error
	lastRequest *http.Request
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestEsaClientCreatePost(t *testing.T) {
	t.Run("esa.io APIに正しいリクエストを送信できる", func(t *testing.T) {
		responseBody := `{"number": 123, "url": "https://yasuhisa.esa.io/posts/123"}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 201,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  mockClient,
		}

		post := EsaPost{
			Name:     "テスト投稿",
			BodyMd:   "本文",
			Category: "Test/2026/02/10",
			Wip:      false,
			Tags:     []string{"plan2esa"},
		}

		result, err := client.CreatePost(post)
		if err != nil {
			t.Fatalf("CreatePost() エラー = %v", err)
		}

		if result.Number != 123 {
			t.Errorf("Number = %v, want 123", result.Number)
		}

		if result.URL != "https://yasuhisa.esa.io/posts/123" {
			t.Errorf("URL = %v, want https://yasuhisa.esa.io/posts/123", result.URL)
		}
	})

	t.Run("201レスポンスを正しくパースできる", func(t *testing.T) {
		responseBody := `{"number": 456, "url": "https://test.esa.io/posts/456", "name": "投稿タイトル"}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 201,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "test",
			AccessToken: "token",
			HTTPClient:  mockClient,
		}

		post := EsaPost{
			Name:     "タイトル",
			BodyMd:   "本文",
			Category: "Test",
			Wip:      false,
		}

		result, err := client.CreatePost(post)
		if err != nil {
			t.Fatalf("CreatePost() エラー = %v", err)
		}

		if result.Number != 456 {
			t.Errorf("Number = %v, want 456", result.Number)
		}
	})

	t.Run("esa.io APIがエラーを返した場合にエラーを返す", func(t *testing.T) {
		errorBody := `{"error": "Unauthorized"}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(bytes.NewBufferString(errorBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "test",
			AccessToken: "invalid-token",
			HTTPClient:  mockClient,
		}

		post := EsaPost{
			Name:   "テスト",
			BodyMd: "本文",
		}

		_, err := client.CreatePost(post)
		if err == nil {
			t.Error("CreatePost() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("messageが指定された場合にリクエストボディに含まれる", func(t *testing.T) {
		responseBody := `{"number": 123, "url": "https://yasuhisa.esa.io/posts/123"}`

		transport := &mockTransport{
			response: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				Header:     make(http.Header),
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  &http.Client{Transport: transport},
		}

		post := EsaPost{
			Name:     "テスト投稿",
			BodyMd:   "本文",
			Category: "Test",
			Wip:      false,
			Message:  "テスト用変更メモ",
		}

		_, err := client.CreatePost(post)
		if err != nil {
			t.Fatalf("CreatePost() エラー = %v", err)
		}

		// リクエストボディを読み取って検証
		if transport.lastRequest == nil {
			t.Fatal("lastRequest が nil です")
		}
		body, _ := io.ReadAll(transport.lastRequest.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("リクエストボディのパースエラー = %v", err)
		}
		if reqBody["post"]["message"] != "テスト用変更メモ" {
			t.Errorf("message = %v, want テスト用変更メモ", reqBody["post"]["message"])
		}
	})

	t.Run("messageが未指定の場合にリクエストボディに含まれない", func(t *testing.T) {
		responseBody := `{"number": 123, "url": "https://yasuhisa.esa.io/posts/123"}`

		transport := &mockTransport{
			response: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				Header:     make(http.Header),
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  &http.Client{Transport: transport},
		}

		post := EsaPost{
			Name:     "テスト投稿",
			BodyMd:   "本文",
			Category: "Test",
			Wip:      false,
		}

		_, err := client.CreatePost(post)
		if err != nil {
			t.Fatalf("CreatePost() エラー = %v", err)
		}

		body, _ := io.ReadAll(transport.lastRequest.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("リクエストボディのパースエラー = %v", err)
		}
		if _, ok := reqBody["post"]["message"]; ok {
			t.Error("messageが未指定なのにリクエストボディに含まれています")
		}
	})
}

func TestEsaClientSearchPosts(t *testing.T) {
	t.Run("記事を検索できる", func(t *testing.T) {
		responseBody := `{"posts": [{"number": 123, "name": "テスト記事", "category": "Test/2026/02/11"}]}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  mockClient,
		}

		results, err := client.SearchPosts(`name:"テスト記事" in:Test/2026/02/11`)
		if err != nil {
			t.Fatalf("SearchPosts() エラー = %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("SearchPosts() 結果数 = %v, want 1", len(results))
		}

		if results[0].Number != 123 {
			t.Errorf("Number = %v, want 123", results[0].Number)
		}

		if results[0].Name != "テスト記事" {
			t.Errorf("Name = %v, want テスト記事", results[0].Name)
		}
	})

	t.Run("検索結果が空の場合に空配列を返す", func(t *testing.T) {
		responseBody := `{"posts": []}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "test",
			AccessToken: "token",
			HTTPClient:  mockClient,
		}

		results, err := client.SearchPosts("存在しない記事")
		if err != nil {
			t.Fatalf("SearchPosts() エラー = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("SearchPosts() 結果数 = %v, want 0", len(results))
		}
	})

	t.Run("検索クエリがURLエンコードされる", func(t *testing.T) {
		responseBody := `{"posts": []}`

		mockTransport := &mockTransport{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				Header:     make(http.Header),
			},
		}

		mockClient := &http.Client{
			Transport: mockTransport,
		}

		client := &EsaClient{
			TeamName:    "test",
			AccessToken: "token",
			HTTPClient:  mockClient,
		}

		// スペース、ダブルクォート、スラッシュ、コロン、日本語を含むクエリ
		query := `name:"テスト記事" in:Test/2026/02/11`
		_, err := client.SearchPosts(query)
		if err != nil {
			t.Fatalf("SearchPosts() エラー = %v", err)
		}

		// リクエストURLがキャプチャされていることを確認
		if mockTransport.lastRequest == nil {
			t.Fatal("lastRequest が nil です")
		}

		// RawQueryを取得
		actualRawQuery := mockTransport.lastRequest.URL.RawQuery

		// 期待値を生成（url.Valuesを使って同じ方法でエンコード）
		expectedParams := url.Values{}
		expectedParams.Set("q", query)
		expectedRawQuery := expectedParams.Encode()

		// RawQueryを比較（url.Valuesはスペースを+にエンコードする）
		if actualRawQuery != expectedRawQuery {
			t.Errorf("RawQuery = %v, want %v", actualRawQuery, expectedRawQuery)
		}

		// 重要な文字が正しくエンコードされていることを個別確認
		if actualRawQuery == "" {
			t.Error("RawQuery が空です")
		}

		// クエリパラメータをデコードして確認
		parsedQuery := mockTransport.lastRequest.URL.Query()
		actualQuery := parsedQuery.Get("q")
		if actualQuery != query {
			t.Errorf("デコードされたクエリ = %v, want %v", actualQuery, query)
		}
	})
}

func TestEsaSearchResultUpdatedAt(t *testing.T) {
	t.Run("updated_atをJSONからtime.Timeにデシリアライズできる", func(t *testing.T) {
		jsonStr := `{"posts": [{"number": 1, "name": "テスト", "category": "Test", "updated_at": "2026-03-07T12:00:00+09:00"}]}`
		var resp EsaSearchResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			t.Fatalf("Unmarshal() エラー = %v", err)
		}
		if len(resp.Posts) != 1 {
			t.Fatalf("Posts 件数 = %d, want 1", len(resp.Posts))
		}
		expected, _ := time.Parse(time.RFC3339, "2026-03-07T12:00:00+09:00")
		if !resp.Posts[0].UpdatedAt.Equal(expected) {
			t.Errorf("UpdatedAt = %v, want %v", resp.Posts[0].UpdatedAt, expected)
		}
	})

	t.Run("updated_atがない場合はゼロ値になる", func(t *testing.T) {
		jsonStr := `{"posts": [{"number": 1, "name": "テスト", "category": "Test"}]}`
		var resp EsaSearchResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			t.Fatalf("Unmarshal() エラー = %v", err)
		}
		if !resp.Posts[0].UpdatedAt.IsZero() {
			t.Errorf("UpdatedAt = %v, want zero value", resp.Posts[0].UpdatedAt)
		}
	})
}

func TestEsaClientUpdatePost(t *testing.T) {
	t.Run("既存記事を更新できる", func(t *testing.T) {
		responseBody := `{"number": 999, "url": "https://yasuhisa.esa.io/posts/999", "name": "更新後タイトル"}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  mockClient,
		}

		post := EsaPost{
			Name:     "更新後タイトル",
			BodyMd:   "更新後本文",
			Category: "Test/2026/02/11",
			Wip:      false,
			Tags:     []string{"plan2esa"},
		}

		result, err := client.UpdatePost(999, post)
		if err != nil {
			t.Fatalf("UpdatePost() エラー = %v", err)
		}

		if result.Number != 999 {
			t.Errorf("Number = %v, want 999", result.Number)
		}

		if result.URL != "https://yasuhisa.esa.io/posts/999" {
			t.Errorf("URL = %v, want https://yasuhisa.esa.io/posts/999", result.URL)
		}
	})

	t.Run("esa.io APIがエラーを返した場合にエラーを返す", func(t *testing.T) {
		errorBody := `{"error": "Not Found"}`

		mockClient := &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString(errorBody)),
					Header:     make(http.Header),
				},
			},
		}

		client := &EsaClient{
			TeamName:    "test",
			AccessToken: "token",
			HTTPClient:  mockClient,
		}

		post := EsaPost{
			Name:   "テスト",
			BodyMd: "本文",
		}

		_, err := client.UpdatePost(999, post)
		if err == nil {
			t.Error("UpdatePost() エラーが期待されましたが、nilが返されました")
		}
	})

	t.Run("messageが指定された場合にリクエストボディに含まれる", func(t *testing.T) {
		responseBody := `{"number": 999, "url": "https://yasuhisa.esa.io/posts/999", "name": "更新後タイトル"}`

		transport := &mockTransport{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				Header:     make(http.Header),
			},
		}

		client := &EsaClient{
			TeamName:    "yasuhisa",
			AccessToken: "test-token",
			HTTPClient:  &http.Client{Transport: transport},
		}

		post := EsaPost{
			Name:     "更新後タイトル",
			BodyMd:   "更新後本文",
			Category: "Test",
			Wip:      false,
			Message:  "更新メモ",
		}

		_, err := client.UpdatePost(999, post)
		if err != nil {
			t.Fatalf("UpdatePost() エラー = %v", err)
		}

		body, _ := io.ReadAll(transport.lastRequest.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("リクエストボディのパースエラー = %v", err)
		}
		if reqBody["post"]["message"] != "更新メモ" {
			t.Errorf("message = %v, want 更新メモ", reqBody["post"]["message"])
		}
	})
}
