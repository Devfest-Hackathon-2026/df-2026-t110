# picolang

## 仕様
### 型

- bool
    - true/false
- integer (i64より大きい値の使用は未定義動作)
    - 整数リテラル
- unit
    -  ()
- terrain
    - リテラルなし
- rotation
    - left/right

### 組み込み関数

- bool
    - not: bool -> bool
    - and: (bool, bool) -> bool
        - 短絡評価なし
    - or: (bool, bool) -> bool
        - 短絡評価なし
    - bool_eq: (bool, bool) -> bool
    - bool_neq: (bool, bool) -> bool
- integer
    - add: (integer, integer) -> integer
    - minus: integer -> integer
    - sub: (integer, integer) -> integer
    - mul: (integer, integer) -> integer
    - div: (integer, integer) -> integer
        - 0除算はパニック
    - integer_eq: (integer, integer) -> bool
    - integer_neq: (integer, integer) -> bool
    - gt: (integer, integer) -> bool
    - lt: (integer, integer) -> bool
    - geq: (integer, integer) -> bool
    - leq: (integer, integer) -> bool
- cast
    - bool_to_integer: bool -> integer
        - 0 or 1
    - integer_to_bool: integer -> bool
        - fst == 1
- random
    - random_bool: () -> bool
    - random_integer: (integer, integer) -> integer
        - 閉区間[fst, snd]上の一様分布
        - snd < fstの場合はパニック
- action
    - forward: () -> bool
        - 移動した場合true
    - back: () -> bool
        - 移動した場合true
    - rotate: rotation -> ()
    - wait: integer -> ()
    - pick_up: () -> bool
        - 拾った場合true
    - release: () -> bool
        - 置いた/落とした場合true
    - drill: () -> bool
        - 壊した場合true
    - drill_durability: () -> integer
    - distance: () -> integer
    - search_front: () -> terrain
    - search_now: () -> terrain
    - is_land: terrain -> bool
    - is_goal: terrain -> bool
    - is_checkpoint: terrain -> bool
    - is_laggage: terrain -> bool
    - is_sea: terrain -> bool
    - is_rock: terrain -> bool
    - print_integer: integer -> ()
    - print_bool: bool -> ()

### 制御構文
- if <expr> <block>
    - <expr>がtrueの場合に<block>を実行する。<expr>はbool型である必要がある。
- if <expr> <block> else <block>
    - <expr>がtrueの場合は前者の<block>を、falseの場合は後者の<block>を実行する。<expr>はbool型である必要がある。
- while <expr> <block>
    - <expr>がtrueの間繰り返す。<expr>はbool型である必要がある。
- for <ident> <expr> <expr> <block>
    - <ident>をfstからsndまでの半開区間で繰り返す。fst, sndは整数型である必要がある。for文の中で<ident>を変更しても、次のループでは無視される。fst > sndの場合はループしない。

### 構文
関数はfirst classではない。関数内部の文法は概念的にはこんな感じ。
```
<expr> ::=
    <literal>
    | <ident>
    | <call>
<args> ::=
    eps
    | <expr> <args>
<call> ::= <func> ( <args> )
<stmt> ::=
    <expr>
    | <type> <ident> := <expr>
    | <ident> = <expr>
    | if <expr> <block>
    | if <expr> <block> else <block>
    | while <expr> <block>
    | for <ident> <expr> <expr> <block>
    | return <expr>
    | break
    | continue
<stmts> ::=
    eps
    | <stmt> <stmts>
<block> ::= { <stmts> }
```

### 関数とスコープ
- 関数のシグネチャは(T1, T2, ...) -> Sとする。引数は無限個設定可能、ただし実装上の制限が入る可能性がある。
- 関数は名前を持つ。
- エントリーポイントはmain: () -> ()。
- 関数は再帰できる。
- 関数は常にグローバル。
- 関数以外の値のスコープは関数全体かブロック全体のスコープを持つ。
- shadowing, 型エラーは実行前に弾く。
- 関数の引数は先頭から評価される。

### VM
- VMのパニックはgoのerrに変換する
    - 0除算も同様
- VM全体でステップ数に制限を掛ける
- 実行開始時に乱数のシード値を渡せるようにする。

## 実装

### tokenize/parse
- google/jsonschema-goを使用する。
- プログラム全体で一括実行する。

### 意味解析

関数名と型の情報を持つmapを用意し、あらかじめ関数のシグネチャを入れておく。関数名が重複する、あるいは組み込み関数と同じ名前の関数を定義する場合はエラー。

ここから先は関数ごとに実行する(関数ごとに分離してよい)。

- 変数に以下の方法でインデックスを付ける
    - (変数名, スコープの深さ)を持つスタックを用意し、あらかじめ引数を入れておく
    - 変数宣言後にスタックにpushする
        - すでに同じ変数名が存在する場合はエラー
        - 同名の関数が存在する場合はエラー
    - スコープから抜けるときにpopする
    - 変数参照時に補助的なmapを使ってインデックスを取得する
        - 存在しない場合エラー
- 型付けを行う
    - 変数宣言時に型を付ける
    - 変数参照時に型を確認する
    - 変数の型が異なる場合エラー
    - 関数全体のブロックの末尾にreturn文がない場合エラー
        - return文以降に文がある場合やif-elseの両方にreturn文がある場合など、return文が必ず実行されることを保証する条件はほかにもあるが、末尾にreturn文がある条件を満たさない限りエラーとする
    - return文の型が異なる場合エラー

### 実行

実行時は、関数呼び出しのスタックを用意し、関数呼び出し時にpush、終了時にpopする

実行前に渡す情報は以下の通り
- 乱数のシード値
- コストの上限値
- 関数呼び出しのスタックの最大深さ

以下の操作はコストを消費する
- 関数呼び出し
- 変数定義
- 変数参照
- 変数代入
- ブロックの開始

VMの停止は以下の条件で行う
- リソース制限に達した場合
- main関数が終了した場合
- VMがパニックした場合
- action系の組み込み関数が呼ばれた場合
    - この場合、該当の組み込み関数の返り値とともに実行を再開させることができる。

### Debug情報の出力
JSON形式の段階で、各ノードに対してヒントとしてuuidを付与する。hint idはプログラム全体で一意であることを検査する。ASTの各表現に対して、対応するhint idを付与する。VMの実行時に、各処理のログをhint idとともに記録し、VMの停止・中断時に出力する。