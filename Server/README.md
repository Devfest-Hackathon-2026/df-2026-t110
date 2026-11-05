# Game module

`game` は共通の型と `Robot` / `Board` インターフェースを定義します。
既存の Go module 内のパッケージとして利用できます。

- `game/hub`: board の指示に従い、robot の呼び出しと応答を仲介します。
- `game/board`: 盤面、tick、実行順、action の同時適用を管理します。
- `game/robot`: stdin / stdout を共有する対話用 robot を提供します。

## 対話デモ

`Server/` で実行します。

```sh
go run ./cmd/game
```

入力対象は `p1>` / `p2>` で表示されます。表示された ID を付けて入力してください。

```text
p1 drill_durability
p1 forward
p2 forward
p1 forward
p2 forward
p1 done
p2 done
```

残量照会は `3` を返し、引き続き p1 の入力を求めます。
最初の移動で p1 は `(1,0)`、p2 は `(3,0)` へ進みます。
次の移動は両者が `(2,0)` を目指すため、どちらも失敗します。
全 robot が終了するとデモも終了します。

| コマンド | 消費 tick | 戻り値 |
| --- | --- | --- |
| `<id> forward` | 1 | 移動したかどうかの bool |
| `<id> back` | 1 | 移動したかどうかの bool |
| `<id> rotate left` / `<id> rotate right` | 1 | unit `()` |
| `<id> drill_durability` | 0 | 残量の integer |
| `<id> done` | 0 | robot の終了 |
| `help` | 0 | 入力方法を表示 |

不正入力は再入力になります。EOF（通常は Ctrl+D）は全 robot の入力を終了し、
受理済みの action を完了させます。Ctrl+C は入力待ちも解除して中断し、
まだ commit していない action は盤面に適用しません。

```sh
go run ./cmd/game -max-ticks 1000 -max-resumes 10000
```

上限は正の値が必要です。tick 上限は更新前に確認します。
上限の tick までは更新可能で、その後 `done` を返すこともできますが、
上限を超える次の更新は拒否します。再開回数には照会と終了確認も含み、
tick を消費しない操作の無限反復も停止できます。

## 差し替え契約

`hub.New(board, participants)` に board と登録順の `hub.Participant` を渡し、
`Run(ctx, hub.Options{...})` で実行します。ID は空文字列不可・重複不可です。
hub と board は1回の実行専用です。具体的な board / robot は相互に参照しません。

1. hub が `Board.Start(ctx, robotIDs)` を呼び、`Board.Next(ctx, nil)` で開始します。
2. board が `Decision.Request` で実行対象と戻り値を指定します。
3. hub が `Robot.Next(ctx, returnValue)` を呼び、action・終了・エラーのいずれかを
   `Reply` として `Board.Next` に渡します。
4. board の `AdvanceTo` は次の tick 更新の提案です。hub が上限と中断を確認し、
   `Board.Next(ctx, nil)` を呼ぶと board が atomic に commit します。
5. board は commit した `TickUpdate` を返し、hub は `Options.OnTick` に通知します。
   次の `Board.Next(ctx, nil)` でスケジューリングを続けます。
6. 全 robot の終了後、board が `Decision.Done` を返します。

各 `Decision` は1つのイベントだけを持ちます。
`AdvanceTo` の直後には、その tick と一致する `TickUpdate` が必要です。
これにより、将来の board でも完了予定 tick へ飛ぶ前に hub が上限を確認できます。

robot に渡す情報は、初回の `nil` と、以後の組み込み関数の戻り値だけです。
tick、盤面 snapshot、スケジューリング情報は渡しません。
robot は内部の制御状態を保持し、board は盤面上の機体の状態を保持します。

robot の実行エラーはその robot のみを停止させます。機体は盤面に残ります。
board エラー、契約違反、tick 通知のエラー、中断、上限到達では全体を停止します。
`Run` はエラーの場合も、最後に commit された tick、再開回数、各 robot の終了状態を返します。
`stopped` は全体停止時にまだ正常終了・実行エラーのどちらにも達していない状態です。

stdio の `Session` は渡された `io.ReadCloser` を所有します。利用後は `Close` してください。
入力の `Close` が進行中の `Read` を解除できる必要があります（`os.File` や `io.PipeReader` など）。
1つの session を使う `Robot.Next` は順番に呼び出してください。
`Robot.Next` / `Board.Start` / `Board.Next` は context の中断に応答する契約です。

## 最小 board の規則

同じ tick の実行可能な robot を登録順に呼び出し、非消費 action では同じ robot を
続けて再開します。全員が消費 action を返すか終了したところで tick を進めます。
最小 board の消費 action は全て1 tick です。将来の複数 tick action では、
最も早い完了予定 tick で、その時刻に完了する action を同時に適用します。

座標は右が `+x`、下が `+y` です。前進・後退は1マス、回転は90度です。
範囲外・岩・海への移動は失敗し、その場合も1 tick 消費します。
同じ移動先への集中、位置交換、循環移動は失敗します。
他機が正常に移動して空くマスへの移動は成功します。
先行機が移動できなければ後続の移動も失敗します。

`board.New` は地形・初期位置・向き・ドリル残量を検証し、設定をコピーします。
`Snapshot` もコピーを返すので、呼び出し側から内部状態は変更できません。

今回の対象は移動・回転・残量照会までです。picolang、wait、ゴール判定、
岩の破壊、荷物、HTTP/SSE、既存 JSON スキーマとの接続は含みません。

## 検証

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
```
