# Blinky — 瞬きアニメーション・ジェネレーター

立ち絵に自然な瞬きを付けるための超軽量 Web ツールです。
通常顔・閉じ目・伏せ目（任意）をアップロードするだけで、**APNG** または **GIF** の瞬きアニメーションを出力します。

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

4. 瞬きのリズムと出力形式を選択して「出力」ボタンを押す

5. プレビューを確認してダウンロード

---

## 瞬きプリセット

アニメーションは常に **ぱち（単発）→ 待機 → ぱちぱち（二連）** の固定パターンです。

| プリセット | 説明 | ぱち前待機 | ぱちぱち前待機 | 連続間隔 |
|-----------|------|-----------|--------------|---------|
| **ふつう** | 自然な人間らしい瞬き | 3.5 秒 | 4.0 秒 | 0.8 秒 |
| **のんびり** | 眠そう・ゆったりとした瞬き | 6.0 秒 | 7.0 秒 | 1.1 秒 |
| **せわしない** | そわそわ・あわただしい瞬き | 1.6 秒 | 2.0 秒 | 0.5 秒 |

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

サーバーはファイルを保存せず、セッションも持ちません。リクエストごとにメモリ上で組み立てて即レスポンスします。

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
│   └── blink.go         # 瞬きフレーム構築・プリセット
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
