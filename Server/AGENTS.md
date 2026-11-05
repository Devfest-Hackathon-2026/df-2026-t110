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

## Goのテスト

- `t.Fatal` / `t.Fatalf` はテスト準備の失敗など、テストを続行できない場合にのみ使うこと。
- 続行可能なアサーションには `t.Error` / `t.Errorf` を使い、1回の実行で複数の失敗を確認できるようにすること。続行時のnil参照などにも注意すること。
- 外部依存や共有状態がなく、独立して実行できるテストには `t.Parallel()` を付けること。サブテストも同様とすること。
