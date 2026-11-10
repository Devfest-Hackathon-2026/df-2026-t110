<script setup lang="ts">
import { battleGuide } from './battleGuide.ts'
</script>

<template>
  <div class="match-score-rules furigana-text">
    <p class="match-score-formula">
      <strong>
        得点 ＝ 終了時に自分の領地に置かれた荷物の数 × {{ battleGuide.luggagePoints }}点 −
        自分のロボットが海に捨てた荷物の数 × {{ battleGuide.discardPenalty }}点
      </strong>
    </p>
    <ul>
      <li>最終得点が高いほうの勝ち。同点は引き分けです。</li>
      <li>持っているだけの荷物は得点になりません。自分の領地のマスに「置く」必要があります。</li>
      <li>相手の領地にも入れます。置かれた荷物は、どちらのチームも拾って持ち出せます。</li>
      <li>荷物を置いた回数ではなく、終了時に領地に残っている荷物の数で得点が決まります。</li>
    </ul>
  </div>
</template>

<style scoped>
.match-score-rules {
  color: var(--color-ink-2);
}
.match-score-formula {
  margin: 0;
  padding: var(--space-md);
  border-left: 3px solid var(--color-accent);
  background: var(--color-paper-2);
}
ul {
  margin: var(--space-md) 0 0;
  padding-left: 1.4em;
}
li + li {
  margin-top: var(--space-xs);
}
</style>
