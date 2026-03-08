# TUI Guide

## Architecture

BubbleTea (Elm Architecture) ベースの TUI。
メイン画面（セッションリスト）は `model.go` が担当し、作成フォーム・ヘルプ・通知履歴は tmux popup で独立プロセスとして起動する。

```
internal/tui/
├─ model.go       ... メインModel（セッションリスト）, Update(), View() (~1430行)
├─ createform.go  ... セッション作成フォーム (popup用, ~540行)
├─ dirpicker.go   ... ディレクトリピッカー (createform内で使用, ~730行)
├─ notifyview.go  ... 通知履歴表示 (popup用, ~180行)
├─ helpview.go    ... ヘルプ表示 (popup用, ~100行)
└─ styles.go      ... lipglossスタイル定義 (Tokyo Night配色)

cmd/ccvalet/cmd/
├─ create_popup.go  ... ccvalet create-popup (Hidden) → CreateFormModel起動
├─ help_popup.go    ... ccvalet help-popup (Hidden)   → HelpModel起動
└─ notify_popup.go  ... ccvalet notify-popup (Hidden) → NotifyModel起動
```

## Model構造

`model.go` の `Model` がセッションリスト画面の状態を保持:
- セッション一覧 + カーソル位置 + ページネーション
- 検索モード（フィルタリング）
- 確認ダイアログ（Kill/Delete）
- daemon.Client（IPC通信用）
- tmux.Client（popup起動・ペイン制御用）
- ポーリングタイマー（tickMsg）

## Update/View パターン

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // updateListMode() に委譲
        // confirmDelete/confirmKill/searching 等のモード判定
    case tickMsg:
        // 定期ポーリング（daemon.Client.List()）
        // popup完了の検知（環境変数経由）
    case sessionsMsg:
        // セッション一覧更新
    }
}

func (m Model) View() string {
    // processingMsg != "" → renderProcessingView()
    // 通常 → renderListContent() + renderHelpLine()
}
```

## ビューモード

### メイン画面（model.go 内）

- **セッション一覧**: デフォルト画面、ステータス表示 + ページネーション
- **検索モード**: `/` キーで起動、セッション名のインクリメンタルフィルタリング
- **確認ダイアログ**: Kill/Delete時にヘルプラインに確認メッセージ表示

### Popup（tmux popup で独立プロセス起動）

- **作成フォーム**: `createform.go` の `CreateFormModel`（3ステップ: ホスト→WorkDir→名前）
- **ヘルプ**: `helpview.go` の `HelpModel`（キーバインド一覧）
- **通知履歴**: `notifyview.go` の `NotifyModel`（通知一覧 + セッション選択）

Popup完了後は環境変数（`CCVALET_CREATED_SESSION`, `CCVALET_NOTIFY_SESSION`）経由で親TUIに結果を返す。親TUIは tickMsg のポーリングで検知する。

## スタイリング

- `styles.go` で lipgloss スタイルを定義（Tokyo Night配色）
- 生のANSIコードは使わない
- カラーは lipgloss.Color() で指定

## 新規Popup追加手順

1. `internal/tui/` に新しい `.go` ファイルを作成し、独立した `tea.Model` を実装
2. `cmd/ccvalet/cmd/` に `xxx_popup.go` を作成（Hidden コマンドとして登録）
3. popup内で `tea.NewProgram()` を使い独立した BubbleTea プログラムとして実行
4. 結果を環境変数で親TUIに返す場合、`model.go` の `tick()` 内で検知ロジックを追加
5. `model.go` の `updateListMode()` に popup 起動のキーバインドを追加
6. 既存の create_popup.go / help_popup.go / notify_popup.go をパターン参考にする

## キーバインド

キーバインドは `config.GetKeybindings()` から取得される。
デフォルト値は `config.DefaultKeybindings()` で定義。
ユーザーは `~/.ccvalet/config.yaml` の `keybindings` セクションでカスタマイズ可能。
