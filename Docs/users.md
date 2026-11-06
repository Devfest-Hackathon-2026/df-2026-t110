# ユーザーアカウントの規約

## 前提

アカウントは、 Google OIDC 認証により作成される

## 必須事項

- ライブラリをレジストリにアップロードするとき、 DisplayName の設定が必要
  - S3 の Object 名に利用するため
  - 初回ログインで DisplayName 設定を強制させるよりも、設定タイミングが明確で実装が行いやすい
    - DisplayName は NULL 許容であり、 foreign key にはすべて ID token の sub を用いる

## 禁止事項

- DisplayName の変更
  - S3 Object 名に変更が生じないようにする
  - この部分の proxy や互換性は考えなくてよい

## 現状実装しないこと

- サービス内共通 ID の発行
  - Google OIDC を前提にしているため、他サービスからの連携も考えた共通 ID は user domain に持たせない
  - 将来的に他サービスの認証を受け付けるようになっても、 migration は拡張であり容易であるため、今のところは考えない
