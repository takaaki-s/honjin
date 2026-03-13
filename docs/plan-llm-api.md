# CLI LLM操作対応 — 実装計画

ccvaletをClaude Code等のCLI LLMから操作可能にするための段階的実装計画。
各ステップは独立したセッション（my-01 → my-05のサイクル）で完結する。

---

## Step 1: `--json` グローバルフラグ + `session list --json`

**目的**: LLMが出力をパースできる基盤を作る

### スコープ
- rootCmdに `--json` persistent flagを追加
- `session list` コマンドでJSON出力に対応（IPC層は既にJSONなのでそのまま流す）
- JSON出力用のヘルパー関数を用意（他コマンドで再利用）

### 成果物
```
ccvalet session list --json
→ [{"id":"...","name":"...","status":"idle","work_dir":"/path",...}]
```

### 見積り
小（CLIフラグ追加 + 出力フォーマッタ）

---

## Step 2: 主要コマンドのJSON出力対応

**目的**: session操作系コマンドすべてでJSON出力をサポート

### スコープ
- `session new --json` → 作成したセッション情報をJSON出力
- `session kill --json` → `{"success":true,"id":"...","name":"..."}`
- `session delete --json` → 同上
- `daemon status --json` → `{"running":true,"pid":1234}`

### 前提
- Step 1の `--json` フラグ基盤

### 見積り
小（既存コマンドへの分岐追加）

---

## Step 3: `session info` コマンド

**目的**: 特定セッションの詳細情報を取得

### スコープ
- `ccvalet session info <name>` コマンド追加
- デフォルトで人間可読テーブル、`--json` でJSON出力
- `session.Info` の全フィールドを出力（status, workdir, branch, last messages等）

### 成果物
```
ccvalet session info my-session --json
→ {"id":"...","name":"my-session","status":"idle","work_dir":"/path","current_branch":"feat/xxx","last_user_message":"...","last_assistant_message":"..."}
```

### 見積り
小（listの結果をフィルタ + フォーマット）

---

## Step 4: `session send` コマンド（プロンプト送信）

**目的**: LLMが別セッションにタスクを投げられるようにする

### スコープ
- IPCアクション `send` を追加（daemon server.go + client.go）
- `ccvalet session send <name> <prompt>` コマンド追加
- 内部実装: `tmux.SendKeysLiteral` でプロンプトを入力 → Enter送信
- セッションが `idle` 状態でない場合はエラー返却
- `--json` 対応

### 注意点
- tmuxのSendKeysは文字列長制限がある → 長文は一時ファイル経由などの検討
- 改行を含むプロンプトの扱い
- stdinからの読み取り対応（パイプ入力 `echo "prompt" | ccvalet session send name -`）

### 成果物
```
ccvalet session send my-session "Fix the bug in auth.go"
→ Sent prompt to session: my-session

ccvalet session send my-session --json < prompt.txt
→ {"success":true,"session":"my-session","status":"thinking"}
```

### 見積り
中（IPC追加 + tmux連携 + エッジケース処理）

---

## Step 5: `session wait` コマンド（状態待機）

**目的**: セッションが特定のステータスになるまでブロッキング待機

### スコープ
- IPCアクション `get` を追加（単一セッション取得、Step 3のinfo用にも使える）
- `ccvalet session wait <name> --status idle --timeout 300` コマンド追加
- ポーリング間隔: 2秒（内部でlistを繰り返し呼ぶ）
- タイムアウト時は exit code 4 で終了
- `--json` 対応（待機完了時のセッション情報を出力）

### 成果物
```
ccvalet session wait my-session --status idle --timeout 60
→ (ブロック... セッションがidleになったら)
→ Session my-session is now idle

ccvalet session wait my-session --status idle --timeout 60 --json
→ {"id":"...","name":"my-session","status":"idle",...}
```

### 見積り
中（ポーリングループ + タイムアウト + シグナル処理）

---

## Step 6: `session output` コマンド（出力取得）

**目的**: セッションの直近の会話内容を取得

### スコープ
- `ccvalet session output <name>` — 最後のアシスタントメッセージを出力
- `--last N` フラグ — 直近N往復の会話を出力
- `--json` 対応
- 内部: transcript readerを活用（既存の `GetLastMessages` を拡張）

### 成果物
```
ccvalet session output my-session
→ (最後のアシスタントメッセージのテキスト)

ccvalet session output my-session --last 3 --json
→ [{"role":"user","content":"..."},{"role":"assistant","content":"..."},...]
```

### 見積り
中（transcript reader拡張 + CLIフォーマット）

---

## Step 7: 終了コード体系の整備

**目的**: LLMがexit codeで条件分岐できるようにする

### スコープ
- 終了コード定数を定義（`internal/exitcode/exitcode.go`）
  - 0: 成功
  - 1: 一般エラー
  - 2: セッション未発見
  - 3: デーモン未起動
  - 4: タイムアウト
- 既存コマンドにexit code適用
- `--json` エラー出力に `exit_code` フィールド追加

### 見積り
小（定数定義 + 既存エラーの分類）

---

## 実装順序と依存関係

```
Step 1 ──→ Step 2 ──→ Step 3
  │                      │
  │                      ↓
  └──────────────→ Step 4 ──→ Step 5
                              │
                              ↓
                         Step 6
                              │
                              ↓
                         Step 7
```

- Step 1 は全ての基盤（最優先）
- Step 2, 3 は Step 1 の `--json` 基盤に依存
- Step 4（send）は独立して着手可能だが、Step 3 の `session info` があると検証しやすい
- Step 5（wait）は Step 4 と組み合わせて初めて真価を発揮
- Step 6（output）は Step 5 の後が自然（send → wait → output の流れ）
- Step 7 は最後にまとめて適用

---

## 各ステップの my-* ワークフロー

各ステップで以下のサイクルを回す：

1. **`/my-01-spec`** — このドキュメントの該当Stepをベースに仕様整理
2. **`/my-02-design`** — 対象ファイルの特定、変更箇所の設計、TODO作成
3. **`/my-03-work`** — 実装
4. **`/my-04-review`** — コードレビュー
5. **`/my-05-verify`** — テスト実行・動作確認

---

## LLMからの典型的な利用フロー（完成形イメージ）

```bash
# 1. セッション作成
ID=$(ccvalet session new -d /path/to/project --json | jq -r '.id')

# 2. プロンプト送信
ccvalet session send "$ID" "Fix the authentication bug in login.go"

# 3. 完了待機
ccvalet session wait "$ID" --status idle --timeout 300

# 4. 結果取得
RESULT=$(ccvalet session output "$ID" --json | jq -r '.[-1].content')

# 5. 必要に応じて追加指示
ccvalet session send "$ID" "Now add tests for the fix"
ccvalet session wait "$ID" --status idle --timeout 300

# 6. セッション削除
ccvalet session delete "$ID"
```
