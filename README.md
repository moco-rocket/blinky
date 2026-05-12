# Blinky — 瞬きアニメーション生成ツール

立ち絵に自然な瞬きを付けるための超軽量 Web ツールです。
通常顔・閉じ目・伏せ目（任意）をアップロードするだけで、**APNG** または **GIF** の瞬きアニメーションを生成します。

---

## 使い方

1. サーバーを起動する

   ```bash
   go run .
   ```

2. ブラウザで `http://localhost:8080` を開く

3. 画像をアップロードする（ドラッグ＆ドロップ対応）

   | スロット | 内容 | 必須 |
   |---------|------|------|
   | 通常顔  | 目を開いた状態 | ✅ |
   | 閉じ目  | 完全に閉じた状態 | ✅ |
   | 伏せ目  | 半開き状態 | （任意） |

4. 瞬きのリズムと出力形式を選択して「生成」ボタンを押す

5. プレビューを確認してダウンロード

---

## 瞬きプリセット

| プリセット | 説明 | 間隔 | 二連瞬き |
|-----------|------|------|---------|
| **ふつう** | 自然な人間らしい瞬き | 3〜5 秒 | 25% |
| **のんびり** | 眠そう・ゆったりとした瞬き | 5〜8 秒 | 10% |
| **せわしない** | そわそわ・あわただしい瞬き | 1〜3 秒 | 45% |

各プリセットは生成のたびにランダムな揺らぎ（±20%程度）が加わります。

### 瞬きの非対称モーション

- **閉じる**：通常 → 閉じ目（1 フレームで一気に）
- **開く**：閉じ目 → 伏せ目 → 通常（段階的・ゆっくり）

完全対称モーションは使用しません。機械的に見えるためです。

---

## 出力形式

| 形式 | 説明 |
|------|------|
| **APNG** | 透過PNG対応・高品質（推奨） |
| **GIF** | 広く対応・透過は1ビット限定 |

---

## 対応入力画像

- PNG（透過 / α チャンネル対応）
- JPEG
- WebP

---

## API

```
POST /process
Content-Type: multipart/form-data

フィールド:
  normal   通常顔画像（必須）
  closed   閉じ目画像（必須）
  half     伏せ目画像（任意）
  preset   normal / relaxed / busy（省略時: normal）
  format   apng / gif（省略時: apng）

レスポンス:
  APNG → Content-Type: image/png
  GIF  → Content-Type: image/gif
```

サーバーはファイルを保存せず、セッションも持ちません。リクエストごとにメモリ上で生成して即レスポンスします。

---

## 開発

```bash
# 依存関係の取得
go mod tidy

# 起動
go run .

# ビルド確認
go build ./...
```

---

## Docker

```bash
docker build -t blinky .
docker run -p 8080:8080 blinky
```

---

## Cloud Run へのデプロイ

Blinky はステートレス設計のため、そのまま Cloud Run で動作します。

```bash
# Container Registry へプッシュ
gcloud builds submit --tag gcr.io/YOUR_PROJECT/blinky

# Cloud Run へデプロイ
gcloud run deploy blinky \
  --image gcr.io/YOUR_PROJECT/blinky \
  --platform managed \
  --allow-unauthenticated \
  --region asia-northeast1
```

`PORT` 環境変数は Cloud Run が自動で設定します。サーバー側でも `PORT` を参照しているため、特別な設定は不要です。

---

## ファイル構成

```
blinky/
├── main.go              # HTTP サーバー（/process エンドポイント）
├── blink/
│   └── blink.go         # 瞬きフレーム生成・プリセット
├── encoder/
│   ├── apng.go          # APNG エンコーダー（自前実装）
│   └── gif.go           # GIF エンコーダー
├── static/
│   └── index.html       # フロントエンド（1ファイル完結）
├── Dockerfile
├── .dockerignore
├── go.mod
└── go.sum
```
