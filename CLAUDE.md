# Claude Code プロジェクト固有の指示

## プロジェクト概要

`plan2esa`は、Claude Codeのプランファイル（`$CLAUDE_CODE_TMPDIR/plans/*.md`）をesa.ioに自動投稿するCLIツールです。SessionEndフックから呼び出され、プランファイルをナレッジとして蓄積します。

## 開発時の注意事項

### コーディング規約

- **パッケージ構成**: 単一の`package main`を使用し、ファイル分割で責務を分離する
- **エラーハンドリング**: すべてのエラーは`fmt.Errorf`でラップし、コンテキストを付与する
- **命名規則**:
  - 公開関数: PascalCase（例: `CreatePost`）
  - 非公開関数: camelCase（例: `getPlansDir`）
  - 構造体: PascalCase（例: `EsaClient`）
  - インターフェース: PascalCase + 動詞または名詞（例: `EsaPoster`）
- **コメント**: 公開API・インターフェース・構造体には必ずドキュメントコメントを付ける

### テスト方針

- **TDD（テスト駆動開発）**: t_wada式を採用
  1. TODOリストを作成し、小さなステップに分解
  2. Red-Green-Refactorサイクルを回す
  3. 一度に一つのことに集中する
- **テストデータ**: `t.TempDir()`を使用して一時ディレクトリを生成し、自動クリーンアップさせる
- **モック**: インターフェース（`EsaPoster`）を使ったDIパターンで、テスト時はモックを注入する
- **網羅性**: 正常系・異常系・境界値を必ずテストする

### テストカバレッジの目標

- 全体カバレッジ: 80%以上
- 主要関数（`run`, `CreatePost`, `findLatestPlanFile`など）: 90%以上

### コミット規約

- **Conventional Commits形式**を使用:
  - `feat`: 新機能
  - `fix`: バグ修正
  - `refactor`: リファクタリング
  - `test`: テスト追加・修正
  - `docs`: ドキュメント
  - `build`: ビルド関連
  - `ci`: CI設定
- **コミットメッセージ**:
  - 1行目: 短い要約（50文字以内）
  - 2行目: 空行
  - 3行目以降: 詳細説明（必要に応じて）
- **コミット単位**: 意味のある最小単位に分割する（`git-sequential-stage`を使用）

### セキュリティ要件

- **HTTPクライアント**:
  - TLS 1.2以上を強制
  - HTTPプロキシを無効化
  - リダイレクトを禁止
  - タイムアウトを設定（30秒）
- **レスポンス処理**:
  - `io.LimitReader`でサイズ制限（10MB）
  - エラーメッセージのサニタイズ（制御文字除去、500文字上限）
- **入力検証**:
  - team_nameは`[A-Za-z0-9_-]`のみ許可
  - タイトルは255バイト上限（UTF-8ルーン境界）

### 依存関係

- **標準ライブラリのみ**: 外部依存は`gopkg.in/yaml.v3`のみ
- **最小限の依存**: 必要最小限のライブラリのみを使用し、定期的に見直す

### CI/CD

- **GitHub Actions**でfmt, vet, staticcheck, test, buildを自動実行
- **pre-commit**でコミット前にfmt, vet, staticcheck, testを自動実行
- **アクションのSHAピン留め**: セキュリティのためバージョンタグではなくSHAでピン留めする

## 関連ファイル

### 設定ファイル

- `go.mod`, `go.sum`: Go モジュール定義
- `.gitignore`: Git無視ファイル

### ソースコード

- `main.go`: エントリポイント、メインロジック
  - `main()`: フラグ解析、エントリポイント
  - `getPlansDir()`: `$CLAUDE_CODE_TMPDIR/plans`のパス取得
  - `run()`: メインロジック実行
- `config.go`: 設定ファイル読み込み、検証
  - `loadConfig()`: YAML読み込み
  - `validateConfig()`: 設定検証
  - `getDefaultConfigPath()`: デフォルト設定パス取得
  - `getAccessToken()`: 環境変数からトークン取得
  - `buildCategory()`: カテゴリに日付付与
  - `getRepositoryName()`: Gitリポジトリ名取得
- `plan.go`: プランファイル処理
  - `findLatestPlanFile()`: 最新プランファイル検索
  - `extractTitle()`: タイトル抽出
  - `removeTitle()`: タイトル行除去
  - `sanitizePostName()`: タイトルサニタイズ
  - `buildPostName()`: タイトル決定ロジック
- `esa.go`: esa.io APIクライアント
  - `EsaPoster` interface: DI用インターフェース
  - `EsaClient`: esa.io APIクライアント実装
  - `NewEsaClient()`: クライアント生成
  - `CreatePost()`: 記事投稿
  - `sanitizeErrorMessage()`: エラーメッセージサニタイズ

### テスト

- `*_test.go`: 各ファイルに対応するテストファイル
- テストは`t.TempDir()`で一時ファイル・ディレクトリを生成

### ドキュメント

- `README.md`: ユーザー向けドキュメント
- `CLAUDE.md`（本ファイル）: Claude Code向け開発ガイド
- `LICENSE`: MITライセンス

### CI/CD設定

- `.github/workflows/ci.yaml`: GitHub Actions CI設定
- `.pre-commit-config.yaml`: pre-commit設定

## 参考情報

- esa.io API公式ドキュメント: https://docs.esa.io/posts/102
- Claude Code Hooksドキュメント: https://code.claude.com/docs/en/hooks
- 参考実装: `/Users/yasuhisa.yoshida/work/esa-llm-scoped-guard/`
  - セキュリティ設計パターン: `internal/esa/client.go`
  - YAML設定読み込みパターン: `config.go`
