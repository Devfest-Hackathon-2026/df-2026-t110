# ルーム配置用ARマーカー

対角2点の位置合わせを試すための、区別できる2枚のマーカーです。画像は[AR.js公式のマーカー例](https://github.com/AR-js-org/AR.js/tree/master/three.js/examples/marker-training/examples)を印刷しやすいように1024×1024 PNGへ整えたものです。

| 位置 | 印刷用画像 | AR.jsの認識用パターン |
| --- | --- | --- |
| A | [Hiro](room-marker-hiro.png) | [marker-hiro.patt](marker-hiro.patt) |
| B | [Kanji](room-marker-kanji.png) | [marker-kanji.patt](marker-kanji.patt) |

2枚を同じ物理サイズで、白い余白を残して印刷してください。黒い正方形の一辺を同じ長さに揃えると、AR.jsの `size` に同じ実測値（メートル）を設定できます。最初は2枚を同時にカメラへ収めて認識を確認します。

```html
<a-scene embedded arjs="sourceType: webcam; debugUIEnabled: false">
  <a-marker id="room-corner-a" type="pattern" url="/markers/marker-hiro.patt" size="0.10" emitevents="true"></a-marker>
  <a-marker id="room-corner-b" type="pattern" url="/markers/marker-kanji.patt" size="0.10" emitevents="true"></a-marker>
  <a-entity camera></a-entity>
</a-scene>
```

上記の `/markers/` は、実装時に `.patt` を公開するパスの例です。`size="0.10"` は黒い正方形の一辺を10 cmで印刷した場合の値です。

これらは位置と向きを追跡するARマーカーです。ルームIDを格納するQRコードではありません。QRによるルーム参加を追加する場合は、QR読み取りを別に組み合わせます。
