# Robot Field クライアント

開発とテストには Node.js 24 以上を使います。依存関係の確認には pnpm 12.17.0 を使用しました。

`/` はARを読み込まないホームです。`/tutorials` から教材詳細（`/tutorials/:tutorialId`）へ進めます。`/tutorials/demo` は従来のステージ表示用プレビューです。`/rooms` はクライアント内で部屋作成とマッチングを試せる画面、`/account` はログインの準備状況を示すページです。認証・マッチングAPIにはまだ接続していません。直接URLを開く配信環境では、各パスを `index.html` に戻す設定が必要です。

## 対戦のローカルプレビュー

`/rooms` で部屋を作成し、6文字のコードまたは共有リンクから `/rooms/:roomId` に参加できます。Piniaで画面状態を管理し、`localStorage` の変更通知で同じブラウザの別タブに反映します。参加者が2人になると待機画面はマッチング成立を表示します。プレイヤーIDはタブごとの `sessionStorage` に保存するため、確認時は複製タブではなく新しいタブを開いてください。別端末との同期、サーバーへの保存、対戦画面への遷移はまだありません。

## 教材詳細

`/tutorials/:tutorialId` は OpenAPI の `GET /api/tutorial/{tutorialId}` が返す `stage.variants` を3D・カメラ表示します。旧APIの `fieldId` が返る場合は `GET /api/field/{fieldId}` も読みます。別のAPIオリジンを使う場合は `VITE_API_BASE_URL` を設定してください。`first-step`、`more-steps`、`turn`、`break-rock` の4件にはローカルの案内文と盤面を用意しています。APIが使えない場合は「サンプル教材」として表示します。案内役は同じ `robot.glb` の胴体と腕を実行時にオレンジへ変えたものです。

教材画面の「実行」は、実行直前に候補配置を選び、`window` に `robot-field:tutorial-run` を発行します。`detail` は `{ tutorialId, layout, candidateIndex, program }` で、`program` は `Schema/picolang/toplevel.schema.yaml` に沿った Picolang v0 のソースです。続いてクライアント内の実行器が AST を評価し、ロボットの動作を1秒単位で `StageScene` のアニメーションへ渡します。前進・後退・回転・待機・拾う・置く・岩破壊、条件・繰り返し・関数呼び出し、算術・論理・vector演算に対応します。岩や荷物の変化は動作後に盤面へ反映し、ゴールに着くとクリア演出を再生します。「リセット」は実行を中断して盤面・ロボットを初期状態に戻し、`robot-field:tutorial-reset` を `{ tutorialId }` とともに発行します。サーバーのVM・型チェックとはまだ接続していません。ローカル実行器には最大10000ステップ、関数呼び出し深さ64の制限があります。旧記号マップにドリル制限が無い場合は99回とします。

iPad横向きのような横長画面では、左にPicolangのカテゴリ・命令パレット・作業領域、右上にスクロールできる課題説明、右下にARを置きます。パレットをタップすると関数の末尾のreturnの手前へ命令を追加し、ドラッグすると制御構文の内側にも追加できます。各ブロックからも命令の追加・並べ替え・削除ができ、式は入れ子で編集できます。関数と引数、型（入れ子のvectorを含む）も編集できます。教材ごとの下書きはブラウザのlocalStorageへ保存し、JSONの確認・コピー・読み込みもできます。JSONの読み込み時はノード形状と重複する `hintId` を検査します。サーバーの意味解析・型チェックはまだ接続していません。左右の境界はドラッグまたはフォーカス後の左右矢印キー、説明とARの境界はドラッグまたは上下矢印キーで調整できます。「全画面」はブラウザの全画面表示を試し、対応していない環境では画面全体にARを重ねます。「ARを隠す」後は説明欄の「ARを表示」から戻せます。幅980px以下で縦向きに近い画面、または幅700px以下では上下に並べます。

## ARフィールド表示（issues #1–3）

1つのフィールドに属する配置候補をすべて見られる3D/カメラビュー、実行前の抽選、Blender製GLBロボットとイベント別アニメーションを含みます。盤面には `public/stage-assets` の床・岩・ゴール・チェックポイント・荷物・スタート・旧デモ用置き場・海の枠を使います。Blenderから書き出したGLBには画像テクスチャが埋め込まれ、編集用PNGは `public/stage-assets/textures`、開いて編集できる `.blend` 原本は `assets/blender` にあります。再生成スクリプトは `scripts/build_stage_assets.py` です。再生成後の `.blend` は `scripts/finalize_stage_blend.py` を Blender のバックグラウンド実行で各ファイルへ適用します。

```bash
cd Client
pnpm install
pnpm dev
```

Vite 8 の標準ビルドは Rolldown と Oxc を使用します。`esbuild` は互換性のための任意の依存関係としてロックファイルに含まれ、リポジトリのルートにある `pnpm-workspace.yaml` でそのインストール時スクリプトを許可しています。

3dプレビュー:カメラなしで動くデバッグ用

- カメラに重ねる:localhost または HTTPS とカメラ許可が必要

画面をタップすると盤面の表示位置を変えられます。現時点ではカメラ映像への重ね合わせで、平面検出・空間固定は次の段階です。

## ステージデータ

以下は旧デモ用の記号マップです。現在の `Schema/field/field.schema.yaml` と `stage.schema.yaml` の形式も受け付けます。

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

現行schemaでは `terrain[y][x]` が `land / rock / sea`、`objects` が `goal / checkpoint / luggage`、`robot` が初期座標と方向です。`Stage.variants` の各 `Field` と単独の `Field` を [stage.ts](src/features/stage/stage.ts) の `normalizeField` で表示用に変換します。チェックポイントの `order` は盤面上の番号として表示します。別オリジンなら `.env.local` に `VITE_API_BASE_URL=https://...` を設定してください。表示は [StageScene.vue](src/features/stage/StageScene.vue) の `stage` prop に依存し、API通信には依存しません。

## 抽選と動作イベント

配置を切り替えて、そのフィールドの全候補を確認できます。開発用プレビューの抽選は、サンプル・貼り付けJSON・APIから取得したフィールドのいずれもローカルで行い、対戦作成APIは呼びません。実行開始時に `window` の `robot-field:run` イベントを発行し、`detail` は `{ fieldId, layout, candidateIndex }` です。ゲームエンジンとの接続はまだないため、現時点の「実行」は配置を決めてイベントを発行するところまでです。

`StageScene` の `animationEvent` prop に `{ id, type, to?, direction?, ticks?, success? }` を渡すとロボットの動作を発火できます。`id` はイベントごとに増やしてください。`type` は `forward` / `back`（現在の向きに1マス進む）、`rotate`（`direction: 'left' | 'right'` で90°回る）、`wait`（`ticks` 単位時間待つ）、`pick_up` / `release`（念動力）、`drill`（目のビーム）、`clear`（クリア演出）です。これらの基本動作は1単位時間を1秒として描画します。移動先が岩・海・盤外なら進まず、`success: false` を渡した動作も失敗として演出します。旧形式の `move`（`to: {x,y}`）、`get`、`put` も受け付けます。GLB内にはアニメーションクリップがないため、`left_arm` / `right_arm` と `left_sholder` / `right_sholder` から実行時に肩の回転軸を作り、念動力の瞬間は腕の絶対角度 Rx=-90°にします。命中や岩の破壊、箱の移動、クリア判定は今後のゲームエンジンが状態を更新します。

海には前のモックと同じ波・光の回路・波紋を描くシェーダーを使用し、Blender製の立体的な枠を重ねます。カメラ表示は現時点ではカメラ映像への重ね合わせです。平面検出と空間固定は今後の作業です。

## 接続時に確定する項目

- 現行schemaの `Stage.variants` に一意の配置IDを持たせるか。現時点ではIDが無い場合、表示側で `layout-1` などを付けます。
- 対戦作成時の `POST /field/{id}/match` のレスポンス形式。対戦ではサーバーが選んだ配置と結果を唯一の正とし、開発用プレビューのローカル抽選とは分けて扱います。
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

## 印刷マーカーの読み取り

`/rooms` の「ARマーカーを読み取る」から、Hiro（A）とKanji（B）の検出状態を確認できます。確認専用ページはありません。`src/features/rooms/roomMarkerTracking.ts` の `mountRoomMarkerTracker(container, callbacks)` は個別の検出状態、カメラ準備完了、カメラエラーをコールバックで通知します。戻り値の `markers` から各マーカーの座標を取得でき、`stop()` でカメラを停止します。印刷した黒い正方形の一辺が14 cmの場合、`size="0.14"` です。異なる倍率で印刷した場合は `MARKER_SIZE_METERS` を実測値（メートル）へ変更してください。AR.jsとA-Frame本体は配布済みJavaScriptとして `public/vendor`、パターンは `public/markers` に置いています。2点からの盤面配置やルーム作成には未接続です。
