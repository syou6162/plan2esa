package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// mockTransport はHTTPリクエストをモックするためのhttp.RoundTripper
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
}
