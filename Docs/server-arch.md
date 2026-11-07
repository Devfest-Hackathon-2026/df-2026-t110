# Server アーキテクチャ

## 現状: 単一アプリ

Server は 1 つの Go バイナリとして Cloud Run 上で stateless に動かす。状態は DB にのみ持つ。

レイヤーは `infra` → `usecase` → `domain` の一方向依存とし、実装の組み立ては `cmd` で行う (詳細は `Server/AGENTS.md`)。

## simulation

simulation (`game/hub` の `Hub.Run`) は、1 回の実行の中だけで完結する計算である。Board は単回利用で、実行は `MaxTicks` / `MaxResumes` で上限が決まっており、実行間で状態を共有しない。そのため同一ユーザー間の同期制御を必要とせず、stateless な Cloud Run にそのまま載せられ、他のコンポーネントから切り出すのも容易である。

simulation の完了は、実行結果が同じ Cloud SQL の所定のテーブルに書き込まれた時点とする。完了は結果の書き込みで判定でき、正常終了かどうかもテーブルで確認できるため、呼び出し元がレスポンスを待つ必要はない。

非同期のランク対戦 API から simulation をどう起動するか (リクエスト内で実行するか、Cloud Tasks や Cloud Run Jobs などに逃がすか) は、実行時間が Cloud Run のリクエストタイムアウトや CPU の割り当てに収まるかで決める。

## 生成 quota

プログラム生成のトークン予算を、Google subject ごとに管理する。

### モデル

- 使用量は更新する値ではなく、追記専用のログ `(subject, tokens, created_at)` として記録する。
- 使用量 `Used` は、窓内のログを `SUM` して毎回計算する。
- `domain.GenerationQuota{GoogleSubject, Limit, Used}` が残量の計算と予約可否の判定を担う。窓の範囲と集計は domain に含めない。
- 予約したトークンは、生成が失敗しても消費済みとして扱う。

### 並行リクエスト

- 同一 subject の「集計 → 判定 → 挿入」だけを直列化する。DB のロック (例: 同一トランザクション内での `SELECT ... FOR UPDATE`) を使い、トランザクションは予約の中だけで開閉する。
- 生成 API の呼び出しはロックの外で並列に実行する。予約済みの枠は確保されているので、並列でも合計は上限を超えない。
- subject が異なればロックを共有しないため、他ユーザー間では競合しない。
- repository は「ロックして集計し、判定関数を呼び、挿入する」を 1 メソッドとして公開する。集計と挿入を別メソッドにすると、usecase 側で競合が起きるため。

## 将来設計: quota の分離

現時点では実装しない。必要になった場合の方針を記す。

- quota を独立した pod 群として切り出す。subject の consistent hash で担当 pod を決め、水平にスケールさせる。
- router は BFF として振る舞い、quota を gRPC で呼び出す。
- 同一 subject の予約が 1 つの pod に集まるため、pod 内の keyed mutex で直列化でき、DB の行ロックが不要になる。
- 分離の継ぎ目は usecase の `GenerationQuota` インターフェース (`Reserve(ctx, subject, maxTokens)`)。DB 実装を gRPC クライアント実装に差し替えるだけでよく、domain と usecase は変更しない。切り替えは `cmd` のエントリポイントで行う。

### 将来設計: simulation の分離

- 同期制御が不要なので、quota と異なり、分離にあたって直列化や担当割り当ての仕組みを必要としない。entrypoint を分けて別の Cloud Run サービスとして切り出し、router (BFF) から呼び出す。
- スケールは Cloud Run の水平オートスケール (最大インスタンス数や同時リクエスト数の調整) で対応する。まず Cloud Run のサービスを機能ごとに複数構成にすることを検討する。
- より細かい制御 (ノードのスペック指定、常駐プロセスなど) が必要になった場合のみ、GKE などのコンテナオーケストレーションを検討する。本プロジェクトは Google Cloud を使う方針のため、マネージド Kubernetes は GKE が候補になる。

### 分離時の課題 (quota)

- pod の増減や障害で担当が切り替わる間は、旧担当と新担当が同じ subject を処理しうる。DB 側の条件付き挿入のような保険を残すか、切り替え時の僅かな超過を許容するかを決める。
- quota が停止したときは、生成を拒否する (課金を伴うため、制限なしで通さない)。
- DB のロックで実際に待ちが問題になってから分離する。stateless な pod を水平に増やすだけなら、DB のロック方式のまま対応できる。
