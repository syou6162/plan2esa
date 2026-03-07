package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	dryRun := flag.Bool("dry-run", false, "Show what would be posted without actually posting")
	message := flag.String("message", "", "Message for the post")
	flag.Parse()

	// configPathが指定されていない場合はデフォルトパスを使用
	if *configPath == "" {
		defaultPath, err := getDefaultConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		*configPath = defaultPath
	}

	if err := run(*configPath, *dryRun, nil, *message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getPlansDir は$CLAUDE_CODE_TMPDIR/plansのパスを返します
func getPlansDir() (string, error) {
	tmpDir := os.Getenv("CLAUDE_CODE_TMPDIR")
	if tmpDir == "" {
		return "", fmt.Errorf("CLAUDE_CODE_TMPDIR environment variable is not set")
	}
	return filepath.Join(tmpDir, "plans"), nil
}

// run はメインロジックを実行します
func run(configPath string, dryRun bool, poster EsaPoster, cliMessage string) error {
	// 1. plansディレクトリのパスを取得
	plansDir, err := getPlansDir()
	if err != nil {
		return err
	}

	// plansディレクトリが存在するか確認
	if _, err := os.Stat(plansDir); os.IsNotExist(err) {
		// ディレクトリが存在しない場合はノーオペで正常終了
		return nil
	}

	// 2. 最新のプランファイルを検索
	planFile, err := findLatestPlanFile(plansDir)
	if err != nil {
		// プランファイルが見つからない場合はノーオペで正常終了
		if errors.Is(err, ErrNoPlanFiles) {
			return nil
		}
		return err
	}

	// 3. プランファイルが見つかった場合のみ、config読み込みに進む
	config, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	// プランファイルを読み込み
	content, err := os.ReadFile(planFile)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	// タイトルと本文を構築
	contentStr := string(content)
	filename := filepath.Base(planFile)
	postName, err := buildPostName(contentStr, filename)
	if err != nil {
		return fmt.Errorf("failed to build post name: %w", err)
	}
	bodyMd := removeTitle(contentStr)

	// カテゴリに日付を付与
	now := time.Now()
	category := buildCategory(config.Post.Category, now)

	// タグを取得
	tags := []string{}
	if repoName := getRepositoryName(); repoName != "" {
		tags = append(tags, repoName)
	}

	// message決定ロジック: CLIフラグ > config > 未送信
	resolvedMessage := cliMessage
	if resolvedMessage == "" {
		resolvedMessage = config.Post.Message
	}
	// 採用されたmessageが空白のみの場合はエラー
	if resolvedMessage != "" && strings.TrimSpace(resolvedMessage) == "" {
		return fmt.Errorf("message cannot be whitespace only")
	}

	post := EsaPost{
		Name:     postName,
		BodyMd:   bodyMd,
		Category: category,
		Wip:      false,
		Tags:     tags,
		Message:  resolvedMessage,
	}

	// 4. token取得（検索に必要）
	accessToken, err := getAccessToken()
	if err != nil {
		return err
	}

	// posterがnilの場合（main()から呼ばれた場合）は実際のEsaClientを作成
	if poster == nil {
		poster = NewEsaClient(config.Esa.TeamName, accessToken)
	}

	// 5. 既存記事を検索
	searchQuery := fmt.Sprintf(`name:"%s" in:"%s"`, postName, category)
	searchResults, err := poster.SearchPosts(searchQuery)
	if err != nil {
		return fmt.Errorf("failed to search posts: %w", err)
	}

	var existingPostNumber int
	var existingUpdatedAt time.Time
	for _, result := range searchResults {
		// 名前とカテゴリが完全一致する記事を探す
		if result.Name == postName && result.Category == category {
			existingPostNumber = result.Number
			existingUpdatedAt = result.UpdatedAt
			break
		}
	}

	// 6. dry-runの場合: タイトル・本文・カテゴリを表示して終了
	if dryRun {
		fmt.Println("=== Dry Run Mode ===")
		fmt.Printf("Title: %s\n", post.Name)
		fmt.Printf("Category: %s\n", post.Category)
		fmt.Printf("Tags: %v\n", post.Tags)
		fmt.Printf("WIP: %v\n", post.Wip)
		if post.Message != "" {
			fmt.Printf("Message: %s\n", post.Message)
		}
		if existingPostNumber > 0 {
			fileInfo, err := os.Stat(planFile)
			if err != nil {
				return fmt.Errorf("failed to stat plan file: %w", err)
			}
			if !existingUpdatedAt.IsZero() && fileInfo.ModTime().Before(existingUpdatedAt) {
				fmt.Printf("\n既存記事が見つかりました (Post #%d) - スキップします\n", existingPostNumber)
				fmt.Printf("  local:  %s\n", fileInfo.ModTime().Format(time.RFC3339))
				fmt.Printf("  esa:    %s\n", existingUpdatedAt.Format(time.RFC3339))
			} else {
				fmt.Printf("\n既存記事が見つかりました (Post #%d) - 上書き更新します\n", existingPostNumber)
			}
		} else {
			fmt.Println("\n既存記事が見つかりませんでした - 新規作成します")
		}
		fmt.Println("\n--- Body ---")
		fmt.Println(post.BodyMd)
		fmt.Println("------------")
		return nil
	}

	// 7. 投稿または更新
	var result *EsaPostResponse
	if existingPostNumber > 0 {
		// ローカルファイルのタイムスタンプをesa側と比較
		fileInfo, err := os.Stat(planFile)
		if err != nil {
			return fmt.Errorf("failed to stat plan file: %w", err)
		}
		if !existingUpdatedAt.IsZero() && fileInfo.ModTime().Before(existingUpdatedAt) {
			fmt.Printf("Skipped: local file (%s) is older than esa post #%d (%s)\n",
				fileInfo.ModTime().Format(time.RFC3339),
				existingPostNumber,
				existingUpdatedAt.Format(time.RFC3339))
			return nil
		}
		// 既存記事を更新
		result, err = poster.UpdatePost(existingPostNumber, post)
		if err != nil {
			return fmt.Errorf("failed to update post: %w", err)
		}
		fmt.Printf("Updated successfully!\n")
	} else {
		// 新規作成
		result, err = poster.CreatePost(post)
		if err != nil {
			return fmt.Errorf("failed to create post: %w", err)
		}
		fmt.Printf("Posted successfully!\n")
	}

	fmt.Printf("Number: %d\n", result.Number)
	fmt.Printf("URL: %s\n", result.URL)

	return nil
}
