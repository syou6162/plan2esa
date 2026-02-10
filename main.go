package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	dryRun := flag.Bool("dry-run", false, "Show what would be posted without actually posting")
	flag.Parse()

	// configPathが指定されていない場合はデフォルトパスを使用
	if *configPath == "" {
		*configPath = getDefaultConfigPath()
	}

	// 実際のEsaClientを作成（run内部で使用）
	// ただし、run()はインターフェースを受け取るので、ここでは作成しない
	// run()の中でdryRunでない場合のみEsaClientを作成する
	if err := run(*configPath, *dryRun, nil); err != nil {
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
func run(configPath string, dryRun bool, poster EsaPoster) error {
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

	post := EsaPost{
		Name:     postName,
		BodyMd:   bodyMd,
		Category: category,
		Wip:      false,
		Tags:     tags,
	}

	// 4. dry-runの場合: タイトル・本文・カテゴリを表示して終了
	if dryRun {
		fmt.Println("=== Dry Run Mode ===")
		fmt.Printf("Title: %s\n", post.Name)
		fmt.Printf("Category: %s\n", post.Category)
		fmt.Printf("Tags: %v\n", post.Tags)
		fmt.Printf("WIP: %v\n", post.Wip)
		fmt.Println("\n--- Body ---")
		fmt.Println(post.BodyMd)
		fmt.Println("------------")
		return nil
	}

	// 5. dry-runでない場合: token取得 → API呼び出し
	accessToken, err := getAccessToken()
	if err != nil {
		return err
	}

	// posterがnilの場合（main()から呼ばれた場合）は実際のEsaClientを作成
	if poster == nil {
		poster = NewEsaClient(config.Esa.TeamName, accessToken)
	}

	result, err := poster.CreatePost(post)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	fmt.Printf("Posted successfully!\n")
	fmt.Printf("Number: %d\n", result.Number)
	fmt.Printf("URL: %s\n", result.URL)

	return nil
}
