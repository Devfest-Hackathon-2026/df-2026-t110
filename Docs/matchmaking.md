# 対戦マッチングの設計メモ

マッチングは同時接続を前提にしない非同期方式とする

## 前提

[対戦モードのルール](battle-mode.md) にあるとおり、試合はプログラムを事前設定して自動実行するものである。
同期的に行われるものではない。

## 相手の選び方

- (仮実装) 表示名と3台分の対戦用プログラムを登録している自分以外のユーザーからランダムに1人選び、登録済みのプログラムをロボット順に使う。プログラムIDの重複は保持する
  - レートを考慮した相手選択は [#156](https://github.com/dvpico/Programming-Learning-AR/issues/156) で対応する (下記「レート差による重み付きランダム選択」)
  - ランクが近い人だけに絞る場合、似た組み合わせの対戦になりやすい
  - プログラム単位で選ぶ場合、多く提出したユーザーほど選ばれやすくなる

### レート差による重み付きランダム選択

[#156](https://github.com/dvpico/Programming-Learning-AR/issues/156) で採用する方式。レートの計算は [rating.md](rating.md) を参照。

- 自分以外の公開entityから、**レート差による重み付きランダム選択**を行う。
- レート差が小さい相手ほど選ばれやすくする。
- ハードなレート差制限は設けない。
- 各候補の選択重みは次のガウス関数で計算する。

\[
w(d)=\exp\left(-\frac{d^2}{2\sigma^2}\right)
\]

- \(d\)：相手のレート − 挑戦者のレート
- \(\sigma=150\)：暫定値
- 各候補の選択確率は、重みを全候補の重みの合計で割って算出する。
- ユーザーによる対戦相手の指定・再抽選はできない。
- 同じ相手との連戦は制限しない。
- 対戦候補が存在しない場合はマッチング不可とする。

### エラーの扱い

- 相手候補がいない場合
  - マッチングが開始できないため `POST /match` は `409` を返す

## 対戦の流れ

シミュレーションはサーバーで実行し、クライアントは結果を再生するだけである

1. `POST /match` に `[{"programId":"a"},{"programId":"b"},{"programId":"a"}]` のように自分の対戦用プログラムを3つ指定し、対戦を作成する。申込側にも表示名が必要で、保存済みの対戦用プログラム設定は変更しない
2. サーバーアプリ内で同期実行し、結果と対戦情報をまとめて保存して `201` を返す。レスポンスの `status` は `finished`
3. レスポンスの `runId` を使い、`GET /run/{runId}/events` の SSE で結果を受け取る。両参加者だけが購読できる
  - 対戦作成という副作用を POST に限定
  - GET は再接続しても安全に
    - これにより、通信の不具合で切断されても結果を取り直せる

## 対戦結果

- (仮実装) サーバーのDBで無期限に保持する
  - `GET /run/{runId}/events` により後からも同じデータを参照できるようにすることで、エラー発生時も改めて読み込みやすく

既存DBには、サーバー更新前に `Server/infra/repo/sql/migrations/20261109_matches.sql`、`20261109_ratings.sql`、`20261110_field_id.sql` の順で未適用のものを適用する。新規DBでは `Server/infra/repo/sql/schema/schema.sql` を使う。

### DBスキーマの適用方法

`schema.sql` は新規DBを作るためのDDLで、`sqlc` はこのファイルとクエリからGoコードを生成するだけである。コード生成やサーバー起動ではDBのテーブルは作成・更新されない。

| 実行環境 | 現在の適用方法 |
| --- | --- |
| ローカルの Docker Compose | `sql/schema` を `/docker-entrypoint-initdb.d` にマウントし、PostgreSQLのデータ領域が空の初回起動時だけ実行する。既存ボリュームの再起動・再ビルド時には再実行しない |
| DB統合テスト | テストごとに一時スキーマを作り、`schema.sql` を適用する |
| Cloud SQL | TerraformはDBインスタンス・DB・ユーザーを作成する。テーブルDDLを適用する処理はない |
| サーバーのデプロイCI | イメージのビルド・配布・Cloud Runの更新を行う。DBマイグレーションを実行する処理はない |

`20261109_matches.sql` は `matches` の追加と、対戦では使わない `runs.tutorial_id` / `runs.program_id` のNULL許可を含む。`20261109_ratings.sql` は `users.rating` と、`matches` の勝敗・レート列を追加する。レート導入前の対戦は保存済みのスコアから勝敗だけを復元し、レート列は NULL のままとする。既存DBのデータを維持してこの構造に更新するために移行SQLが必要となる。`20261110_field_id.sql` は、対戦の `selected_layout_id` を `runs.field_id` に移してから重複列を削除する。チュートリアルの `field_id` と保存済みリプレイは変更しない。`sql/migrations` は現時点では手動適用用のSQLを置くディレクトリであり、適用履歴の管理や自動実行の仕組みはない。

Composeの既存DBでは、リポジトリルートから未適用のものを順に一度ずつ実行する。

```sh
docker compose exec -T postgres psql -X -v ON_ERROR_STOP=1 -U app -d app < Server/infra/repo/sql/migrations/20261109_matches.sql
docker compose exec -T postgres psql -X -v ON_ERROR_STOP=1 -U app -d app < Server/infra/repo/sql/migrations/20261109_ratings.sql
docker compose exec -T postgres psql -X -v ON_ERROR_STOP=1 -U app -d app < Server/infra/repo/sql/migrations/20261110_field_id.sql
```

Cloud SQLでは接続を用意したうえで同じ移行SQLを手動適用する。新規の空DBには移行SQLではなく `schema.sql` を適用する。

## 仮の盤面設定

- 配置候補は2種類で、フィールドIDは `battle-1-layout-1` / `battle-1-layout-2`。9×7マスで、左右の端1列を各チームのテリトリーとする
- 各チーム3台、掘削上限3回、規定200ターン、中央の荷物3個、荷物単価1点、投棄ペナルティ1点
- 初期位置・向きとテリトリーは固定し、岩の配置が異なる左右対称な候補から抽選する。地形の各行・荷物・テリトリー・ロボットの座標とルールは `Server/infra/matchrun/layouts.json` で定義する
- ステップ数・メモリ・呼び出し深度に上限を設け、実行エラーのロボットは停止する。申込側の解析・型検査に失敗した場合は `400` とし、相手側の場合は該当ロボットを停止する

## ランキングについて

- レートは対戦を申し込んだ側だけ変動させる。計算と反映の詳細は [rating.md](rating.md) を参照
  - 相手側は自分の知らないところで対戦されるため、気づいたときに勝手にレートが上下するのはあまり直感的でない
- ランキング掲載には DisplayName の設定を必須とする
  - なお、 DisplayName は変更不可 (ref: [users.md](users.md))

### エラーの扱い

- DisplayName が設定されていない状態でのランク対戦開始には `403` を返す
  - req がおかしいわけではないが、参加権はないため

## 将来の拡張

- シミュレーションで確定した分をインクリメンタルに送る
- 試合一覧から結果を後で見返す機能
- エキシビションマッチ
  - 自分のプログラム同士の対戦など

## 盤面の識別と配置生成

対戦・チュートリアルの実行記録は、具体的な盤面を識別する `fieldId` を `runs.field_id` に保存する。対戦の配置JSONにも `fieldId` だけを持ち、別の配置IDは設けない。チュートリアルでは従来どおり `stage.variants[].id` を実行時の `fieldId` とする。対戦作成レスポンスも `fieldId` を返し、`selectedLayoutId` は返さない。

`go -C Server run ./cmd/generate-layouts --output ../work/layouts.json` で、14×11マス・荷物10個・岩24個の左右対称な64盤面を生成できる。各ロボットの掘削上限は2回、規定160ターンとする。`fieldId` はシードと候補番号を含み、同じシードでは同じ盤面とIDを再現する。生成結果の対戦への採用は `Server/infra/matchrun/layouts.json` の差し替えで行う。
