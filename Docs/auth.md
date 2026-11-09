# アカウント認証

## 方針

- 認証は外部 IdP への OpenID Connect で行う
  - アプリ自身はパスワードなどの資格情報を持たない
  - フローは認可コードフロー (PKCE・state・nonce 付き)
- IdP は以下を使う
  - prod: Google
  - dev: Dex
- ログイン後はアプリ独自のセッションを発行し、PostgreSQL に保存する
  - Client は `session` Cookie だけを扱い、トークンは保持しない
- ユーザーは IdP の `sub` で識別する
  - 詳細は後述の「ユーザー ID の扱い」と `Docs/users.md`

## 異なる IdP の扱い

- 本番環境 / 開発環境 で異なる IdP を利用するが、両方を同時に使うことはない
  - サービス内固有のユーザー ID による異なる IdP の紐づけは必要ない
