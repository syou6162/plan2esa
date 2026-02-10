# plan2esa

Claude Codeのプランファイルをesa.ioに自動投稿するCLIツールです。

## 概要

Claude CodeのSessionEndフックから呼び出すことで、`$CLAUDE_CODE_TMPDIR/plans/`配下に生成されたプランファイル（Markdown）をesa.ioに自動投稿します。プランファイルはgitignore対象のため、ブランチ削除と共に失われてしまいますが、本ツールを使うことでナレッジとして蓄積できます。

## 特徴

- **自動投稿**: SessionEndフックからの自動実行
- **プランファイル検索**: `$CLAUDE_CODE_TMPDIR/plans/`から最新の`.md`ファイルを検索
- **タイトル抽出**: Markdownの最初の`# 見出し`をタイトルとして抽出（なければファイル名を使用）
- **タイトルサニタイズ**: esa.ioで特殊な意味を持つ文字（`# / \ | [ ] < > （ ） ：`）を`_`に置換
- **日付カテゴリ**: 設定したカテゴリに`/yyyy/mm/dd`を自動付与
- **Gitリポジトリタグ**: リポジトリ名を自動タグ付け
- **dry-runモード**: 投稿内容をプレビュー（実際には投稿しない）
- **セキュア設計**: TLS 1.2+、プロキシ無効、リダイレクト禁止、レスポンスサイズ制限

## インストール

```bash
go install github.com/syou6162/plan2esa@latest
```

または、ソースからビルド：

```bash
git clone https://github.com/syou6162/plan2esa.git
cd plan2esa
go build -o plan2esa .
```

## 設定

### 設定ファイル（YAML）

設定ファイルのパス解決順序：

1. `-config`フラグで明示指定された場合はそのパス
2. `$XDG_CONFIG_HOME/plan2esa/config.yaml`
3. `~/.config/plan2esa/config.yaml`

設定ファイルの例（`~/.config/plan2esa/config.yaml`）：

```yaml
esa:
  team_name: "your-team-name"  # あなたのesa.ioチーム名
post:
  category: "Claude Code/plans"  # カテゴリ（実際の投稿時は末尾に /yyyy/mm/dd が自動付与される）
```

### 環境変数

esa.ioのアクセストークンは環境変数で指定します：

```bash
export ESA_ACCESS_TOKEN="your_esa_access_token_here"
```

## 使い方

### 基本的な使い方

```bash
plan2esa
```

デフォルトの設定ファイルパス（`~/.config/plan2esa/config.yaml`）を使用し、`$CLAUDE_CODE_TMPDIR/plans/`配下の最新プランファイルをesa.ioに投稿します。

### 設定ファイルを指定

```bash
plan2esa -config /path/to/config.yaml
```

### dry-runモード

実際に投稿せず、投稿内容をプレビューします：

```bash
plan2esa -dry-run
```

dry-runモードでは環境変数`ESA_ACCESS_TOKEN`が不要です。

### Claude Code SessionEndフックとの連携

Claude Codeの設定ファイル（`~/.claude/settings.json`）にSessionEndフックを追加します：

```json
{
  "hooks": {
    "SessionEnd": {
      "command": "plan2esa",
      "blocking": false
    }
  }
}
```

これにより、Claude Codeのセッション終了時に自動的にプランファイルがesa.ioに投稿されます。

## 動作仕様

### プランファイルの処理順序

1. `$CLAUDE_CODE_TMPDIR/plans/`ディレクトリのパスを取得
   - 環境変数が未設定またはディレクトリが不在の場合は何もせず正常終了
2. 最新の`.md`ファイルを検索（更新日時順）
   - ファイルがない場合は何もせず正常終了
3. プランファイルが見つかった場合のみ、設定ファイルを読み込み
4. dry-runの場合：タイトル・本文・カテゴリを表示して終了
5. 通常モード：`ESA_ACCESS_TOKEN`を取得してesa.io APIに投稿

### タイトルの決定

1. Markdownの最初の`# 見出し`を抽出
2. サニタイズ（特殊文字を`_`に置換、制御文字を除去、255バイト上限）
3. `TrimSpace`を適用
4. 空文字になった場合はファイル名（`.md`拡張子を除く）を使用

### 本文の加工

最初の`# 見出し`行が本文から除去されます（タイトルとの重複を避けるため）。

### カテゴリ

設定ファイルで指定したカテゴリに、実行日の日付（`/yyyy/mm/dd`）が自動付与されます。

例：設定が`Claude Code/plans`の場合、2026年2月10日に実行すると`Claude Code/plans/2026/02/10`となります。

### タグ

Gitリポジトリ名が自動的にタグとして付与されます（`git config --get remote.origin.url`から抽出）。Gitリポジトリでない場合やremote.origin.urlが未設定の場合はタグなし（エラーにはしない）。

## 開発

### 依存関係のインストール

```bash
go mod download
```

### テストの実行

```bash
go test -v ./...
```

カバレッジ付き：

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### ビルド

```bash
go build -o plan2esa .
```

### pre-commit

pre-commitフックを使用して、コミット前に自動的にフォーマット・検証・テストを実行します：

```bash
# pre-commitのインストール（初回のみ）
brew install pre-commit  # macOS
# または
pip install pre-commit

# フックのインストール
pre-commit install

# 手動実行
pre-commit run --all-files
```

## ライセンス

MIT License - 詳細は[LICENSE](LICENSE)を参照してください。
