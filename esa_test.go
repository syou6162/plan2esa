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
