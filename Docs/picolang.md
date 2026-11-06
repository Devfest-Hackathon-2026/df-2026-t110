# picolang

## 仕様
### 型

bool、integer、unit、terrain、rotationはprimitive type、vectorはcompound typeとする。

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
- vector\<T>
    - 型Tの要素を持つ可変長配列
    - 同じvectorの要素はすべて同じ型

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
- vector
    - new: () -> vector\<T>
    - push_back: (vector\<T>, T) -> vector\<T>
        - 末尾に要素の値を追加した新しいvectorを返す
    - pop_back: vector\<T> -> vector\<T>
        - 末尾の要素を取り除いた新しいvectorを返す
        - 要素数が0の場合はパニック
    - back: vector\<T> -> T
        - 末尾の要素を独立した値として返す。元のvectorの要素数は変わらない
        - 要素数が0の場合はパニック
    - size: vector\<T> -> integer
- terrain
    - is_land: terrain -> bool
    - is_goal: terrain -> bool
    - is_checkpoint: terrain -> bool
    - is_laggage: terrain -> bool
    - is_sea: terrain -> bool
    - is_rock: terrain -> bool
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
- primitive type、compound typeを問わず、変数定義・代入、引数・戻り値、vectorの要素の受け渡しはすべて値渡しとし、値全体を複製したものとして扱う。
- vectorの値は、入れ子のvectorを含めてそれぞれ独立している。一方の値への更新は、別の変数や取得済みの値に伝播しない。
- 関数内での引数への再代入は、その関数のローカルな値にだけ反映される。
- push_back、pop_backの結果を元の変数に反映するには、`v = push_back(v, x)`のように戻り値を代入する。
- 末尾の要素を取り出して残りのvectorに更新する場合は、`x = back(v)`の後に`v = pop_back(v)`を行う。

### VM
- VMのパニックはgoのerrに変換する
    - 0除算も同様
- VM全体でステップ数に制限を掛ける
- VM全体でメモリ使用量に制限を掛ける
- 実行開始時に乱数のシード値を渡せるようにする。

## 実装

### tokenize/parse
- google/jsonschema-goを使用する。
- プログラム全体で一括実行する。

### 意味解析

関数名と型の情報を持つmapを用意し、あらかじめ関数のシグネチャを入れておく。プログラム内の関数名が重複する、あるいは組み込み関数と同じ名前の関数を定義する場合はエラー。

組み込み関数のbackには2つのシグネチャを登録し、引数なしの呼び出しは後退アクション、vectorを1つ渡す呼び出しは末尾要素の取得として解決する。それ以外の引数の個数・型はエラーにする。

ここから先は関数ごとに実行する(関数ごとに分離してよい)。

- 変数に以下の方法でインデックスを付ける
    - (変数名, スコープの深さ)を持つスタックを用意し、あらかじめ引数を入れておく
    - 変数宣言後にスタックにpushする
        - すでに同じ変数名が存在する場合はエラー
        - 同名の関数が存在する場合はエラー
    - スコープから抜けるときにpopする
    - 変数の読み出し時に補助的なmapを使ってインデックスを取得する
        - 存在しない場合エラー
- 型付けを行う
    - 変数宣言時に型を付ける
    - 変数の読み出し時に型を確認する
    - 変数の型が異なる場合エラー
    - 関数全体のブロックの末尾にreturn文がない場合エラー
        - return文以降に文がある場合やif-elseの両方にreturn文がある場合など、return文が必ず実行されることを保証する条件はほかにもあるが、末尾にreturn文がある条件を満たさない限りエラーとする
    - return文の型が異なる場合エラー

### 実行

内部表現として不変の木構造を用い、値の複製や更新時に変更のない部分木を共有してよい。更新時には変更する経路を再構築し、既存の値を保持する部分木は変更しない。内部表現の共有によって、言語上の値の独立性が変わることはない。値全体を物理的にコピーすることは要求しない。

実行時は、関数呼び出しのスタックを用意し、関数呼び出し時にpush、終了時にpopする

実行前に渡す情報は以下の通り
- 乱数のシード値
- ステップ数のコストの上限値
- メモリコストの上限値
- 関数呼び出しのスタックの最大深さ

以下の操作はステップ数のコストを消費する
- 関数呼び出し
- 変数定義
- 変数の読み出し
- 変数代入
- ブロックの開始

以下の操作はメモリコストを消費する
- 変数定義
- 値の生成・複製

メモリコストは内部表現の共有方法に依存しない、論理的な値のサイズに応じた推定値を使う。vectorのサイズはコンテナと入れ子を含む全要素のサイズを再帰的に合計し、論理的に複製された値もそれぞれ別々に計上する。スコープ終了、再代入、式評価の完了などで不要になった値をdropし、その値が保持していたメモリコストを減算する。GCによる実際の解放時期は考慮しない。

保持中の値の合計が上限を超える場合はVMを停止する。中断・再開時もコストを引き継ぐ。

VMの停止は以下の条件で行う
- リソース制限に達した場合
- main関数が終了した場合
- VMがパニックした場合
- action系の組み込み関数が呼ばれた場合
    - この場合、該当の組み込み関数の返り値とともに実行を再開させることができる。

### Debug情報の出力
JSON形式の段階で、各ノードに対してヒントとしてuuidを付与する。hint idはプログラム全体で一意であることを検査する。ASTの各表現に対して、対応するhint idを付与する。VMの実行時に、各処理のログをhint idとともに記録し、VMの停止・中断時に出力する。
