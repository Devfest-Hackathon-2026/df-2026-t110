# Robot Field クライアント

開発とテストには Node.js 24 以上を使います。

`/` はARを読み込まないホームです。`/tutorials` から教材詳細（`/tutorials/:tutorialId`）へ進めます。`/tutorials/demo` は従来のステージ表示用プレビューです。`/rooms` と `/account` は、対戦とログインの準備状況を示すクライアント内のページです。認証・マッチングAPIにはまだ接続していません。直接URLを開く配信環境では、各パスを `index.html` に戻す設定が必要です。

## 教材詳細

`/tutorials/:tutorialId` は OpenAPI の `GET /api/tutorial/{tutorialId}` と `GET /api/field/{fieldId}` を読み、取得したフィールドを3D・カメラ表示します。別のAPIオリジンを使う場合は `VITE_API_BASE_URL` を設定してください。教材APIの説明文形式は未定なので、`first-step`、`more-steps`、`turn`、`break-rock` の4件にはローカルの案内文と盤面を用意しています。APIが使えない場合は「サンプル教材」として表示します。案内役は同じ `robot.glb` の胴体と腕を実行時にオレンジへ変えたものです。

教材画面の「プログラムを実行」は、実行直前に候補配置を選び、`window` に `robot-field:tutorial-run` を発行します。`detail` は `{ tutorialId, layout, candidateIndex }` です。「リセット」はロボットの表示を初期位置に戻し、`robot-field:tutorial-reset` を `{ tutorialId }` とともに発行します。教育用エディタとゲームエンジンはこのイベントに接続する予定で、現時点では命令の実行・クリア判定は行いません。

iPad横向きのような横長画面では盤面とエディタ領域を左右に並べます。幅980px以下で縦向きに近い画面、または幅700px以下では上下に並べます。課題説明はエディタの下にスクロールでき、実行・リセットはパネル下部に固定します。エディタ領域は `TutorialPage` の `program-editor` スロットで差し替えられます。現在は接続位置を示す枠だけです。

## ARフィールド表示（issues #1–3）

1つのフィールドに属する配置候補をすべて見られる3D/カメラビュー、実行前の抽選、Blender製GLBロボットとイベント別アニメーションを含みます。

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
  "id": "crystal-route",
  "title": "結晶の道",
  "template": ["S?????G", "???????"],
  "rockCounts": [1, 2],
  "candidates": [
    { "id": "layout-a", "map": ["S..#..G", "._O.X.."] },
    { "id": "layout-b", "map": ["S._#..G", "..O.X.."] },
    { "id": "layout-c", "map": ["S..##.G", ".O._X.."] },
    { "id": "layout-d", "map": ["S.#..#G", "..O_.X."] }
  ]
}
```

`template` はフィールドで固定するマスを示します。`?` は配置ごとに変わるマスで、`candidates` を列挙する場合は陸・岩・海・荷物・置き場にできます。`rockCounts` は固定の `#` も含めた岩の総数候補です。例は盤面サイズが共通で、岩が1個または2個、海や荷物も異なる4配置をすべて表示します。候補のリストがそのフィールドの全候補です。

`candidates` がない簡易モードでは、`?` を陸または岩として `rockCounts` に一致する全組み合わせを生成します。海・荷物なども変わる場合は、サーバーから具体的な `candidates` を渡してください。

記号は `S` スタート、`G` ゴール、`.` 陸、`#` 岩、`_` 海、`O` 荷物、`X` 置き場、`?` 可変マスです。行幅は同一、`S` は1個です。配置候補の `map` に `?` は使いません。

APIが用意されたら画面のフィールドIDから `GET /field/{id}` を呼びます。別オリジンなら `.env.local` に `VITE_API_BASE_URL=https://...` を設定してください。レスポンスの形式が確定したら [stage.ts](src/features/stage/stage.ts) の `normalizeField` で変換します。表示は [StageScene.vue](src/features/stage/StageScene.vue) の `stage` prop に依存し、API通信には依存しません。

## 抽選と動作イベント

配置を切り替えて、そのフィールドの全候補を確認できます。サンプルと貼り付けJSONはローカルで配置を抽選します。APIから取得したフィールドでは、`POST /field/{id}/match` のレスポンスにある0始まりの `candidateIndex` または `selectedLayoutId` で結果を決めます。クライアントはその結果へルーレットを止めます。実行開始時に `window` の `robot-field:run` イベントを発行し、`detail` は `{ fieldId, layout, candidateIndex, match }` です。ゲームエンジンとの接続はまだないため、現時点の「実行」は配置を決めてイベントを発行するところまでです。

`StageScene` の `animationEvent` prop に `{ id, type, to? }` を渡すと、ロボットの動作を発火できます。`id` はイベントごとに増やしてください。`type` は `move`（`to: {x,y}` が必要）、`drill`（目のビーム）、`get` / `put`（腕を待機姿勢から Rx +30°）、`clear`（ジャンプと一回転）です。GLB内にはアニメーションクリップがないため、`left_arm` / `right_arm` と `left_sholder` / `right_sholder` から実行時に肩の回転軸を作ります。命中や岩の破壊、箱の移動、クリア判定は今後のゲームエンジンが状態を更新します。

海には前のモックと同じ波・光の回路・波紋を描くシェーダーを使用します。カメラ表示は現時点ではカメラ映像への重ね合わせです。平面検出と空間固定は今後の作業です。

## 接続時に確定する項目

- `GET /field/{id}` が固定マスを示す `template`、岩の個数候補 `rockCounts`、全配置の `candidates` をどう返すか。海・荷物なども変わるため、具体的な候補マップが必要です。
- `POST /field/{id}/match` のレスポンス形式。現状は0始まりの `candidateIndex`、または候補の `id` と一致する `selectedLayoutId` を受け取ります。対戦ではサーバーの選択を唯一の結果として扱います。
- ゲームエンジンからの動作イベントに含めるロボットID、tick、座標、向き、および対戦相手への配信方法。現在は描画確認用の仮形式です。

```bash
pnpm test
pnpm type-check
pnpm lint
pnpm format:check
pnpm build
```

## Client の構成

- `src/app`: 画面の入口とルート別の表示
- `src/features/stage`: 盤面データ、3D表示、ロボットの動作
- `src/features/tutorial`: 教材の画面、ローカル教材、API
- `src/styles`: サイト共通のスタイル

`pnpm format` はClient内のソースと設定をPrettierで整形します。ESLintとTypeScriptの設定もClient内で完結しています。
