# AGENTS.md

## Layers

- `domain`
  - entities & their invariants
  - MUST NOT depend on other layers
    - only the std & the generated types in `infra/router/gen/schema` are allowed
- `usecase`
  - use cases
  - Depends only on `domain`, NEVER on `infra`
- `infra`
  - implementations such as databases, external services, and HTTP
  - May depend on `usecase` & `domain`.
- Dependencies point one way: `infra` → `usecase` → `domain`.
  - NEVER import in the opposite direction.
- Define interfaces on the consumer side, not the implementation side.
  - ex: the repository interfaces a use case needs live in `usecase`, & `infra` implements them.
- Wire implementations together (construction & injection) in `cmd`.

## Goのmap

- 集合として使い、値自体に意味を持たせないmapには `map[T]struct{}` を使うこと（例: `map[game.RobotID]struct{}`）。

## Goのテスト

- `t.Fatal` / `t.Fatalf` はテスト準備の失敗など、テストを続行できない場合にのみ使うこと。
- 続行可能なアサーションには `t.Error` / `t.Errorf` を使い、1回の実行で複数の失敗を確認できるようにすること。続行時のnil参照などにも注意すること。
- 外部依存や共有状態がなく、独立して実行できるテストには `t.Parallel()` を付けること。サブテストも同様とすること。

## JSON

Go 1.27 で導入された `encoding/json/v2` パッケージを用いる。

## 識別するためだけの空のメソッドを要求する interface について

ある型 A が型 B の部分集合であることを示すために、空の private method を要求する interface を使うことがある。

```go
type B interface {
  b() // b を実装している型は B である。実際には呼び出しは無く、空の実装になる。
}
```

型 A が構造体の場合には、実際にはメソッドを記述せず、構造体に interface を埋め込むようにする。

```go
type A struct {
  a int
  a2 string

  B // 埋め込み
}
```
