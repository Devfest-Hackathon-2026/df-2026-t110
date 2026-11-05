# ARステージ表示（issue #2）

ステージデータを受け取り、ロボットを含まない立体的な盤面を表示する最初の移植です。

```bash
cd Client
pnpm install
pnpm dev
```

3dプレビュー:カメラなしで動くデバッグ用
- カメラに重ねる:localhost または HTTPS とカメラ許可が必要

画面をタップすると盤面の表示位置を変えられます。現時点ではカメラ映像への重ね合わせで、平面検出・空間固定は次の段階です。

## ステージデータ
いったん仮ですすめる、jsonが確定したらそっちに合わせる
```json
{
  "stages": [
    { "id": "example", "title": "岩の道", "map": ["S..#..G"] },
    { "id": "sea", "title": "海の道", "map": ["S.._..G"] }
  ]
}
```

`stages`、`candidates`、配列、または単一ステージを受け取れます。各ステージの `map` は行ごとの文字列配列、または改行区切りの文字列です。記号は `S` スタート、`G` ゴール、`.` 陸、`#` 岩、`_` 海、`O` 荷物、`X` 置き場です。行幅は同一、`S` は1個です。

APIが用意されたら画面のフィールドIDから `GET /field/{id}` を呼びます。別オリジンなら `.env.local` に `VITE_API_BASE_URL=https://...` を設定してください。レスポンスの形式が確定したら [src/stage.js](src/stage.js) の `normalizeStages` で変換します。表示は [StageScene.vue](src/StageScene.vue) の `stage` prop に依存し、API通信には依存しません。

```bash
pnpm test
pnpm build
```
