<script setup lang="ts">
import { battleFieldExample, battleGuide } from './battleGuide.ts'
import MatchScoreRules from './MatchScoreRules.vue'

const width = battleFieldExample[0].length
const height = battleFieldExample.length
const cells = battleFieldExample.flatMap((row, y) =>
  [...row].map((symbol, x) => ({ x, y, symbol, robot: /^[1-3]$/.test(symbol) })),
)
const luggageCount = cells.filter((cell) => cell.symbol === 'O').length
</script>

<template>
  <section class="match-guide" aria-labelledby="match-guide-title">
    <h2 id="match-guide-title" class="furigana-heading">荷物を集めて、<wbr />自分のじん地に置こう！</h2>
    <p class="match-guide-lead furigana-text">
      {{ battleGuide.robotCount }}台のロボットをプログラムで動かす対戦です。荷物を自分のじん地に集めて、試合終了時の得点が高いほうが勝ち。
      じん地に置いた荷物も相手に持ち出されるので、最後まで油断できません。
    </p>
    <details class="match-guide-details">
      <summary class="furigana-heading">遊び方とルール</summary>
      <div class="match-guide-body">
        <section aria-labelledby="match-guide-steps-title">
          <h3 id="match-guide-steps-title" class="furigana-heading">対戦の流れ</h3>
          <ol class="match-guide-steps furigana-text">
            <li>
              <strong>プログラムを選ぶ</strong>
              <p>
                {{ battleGuide.robotCount }}台それぞれに対戦用プログラムを選びます。 同じプログラムを複数のロボットに使えます。
              </p>
            </li>
            <li>
              <strong>自動で対戦する</strong>
              <p>
                「対戦を始める」で相手と盤面が選ばれ、プログラムどおりに動きます。試合中の操作はありません。
              </p>
            </li>
            <li>
              <strong>動きを振り返る</strong>
              <p>結果画面の「対戦を再生」で動きを確認できます。プログラムを直して、また挑戦しましょう。</p>
            </li>
          </ol>
        </section>

        <section aria-labelledby="match-guide-score-title">
          <h3 id="match-guide-score-title" class="furigana-heading">得点と勝ち負け</h3>
          <MatchScoreRules />
        </section>

        <section aria-labelledby="match-guide-field-title">
          <h3 id="match-guide-field-title" class="furigana-heading">フィールドとロボット</h3>
          <dl class="match-guide-facts furigana-text">
            <div>
              <dt>フィールド</dt>
              <dd>{{ width }} × {{ height }}マス</dd>
            </div>
            <div>
              <dt>荷物の数</dt>
              <dd>{{ luggageCount }}個</dd>
            </div>
            <div>
              <dt>試合の長さ</dt>
              <dd>最大{{ battleGuide.turns }}ターン</dd>
            </div>
          </dl>
          <figure class="match-guide-field">
            <div class="match-guide-field-labels furigana-text" aria-hidden="true">
              <strong>自分のじん地</strong><strong>相手のじん地</strong>
            </div>
            <svg :viewBox="`-12 -12 ${width * 32 + 24} ${height * 32 + 24}`" role="img"
              aria-labelledby="match-guide-board-title match-guide-board-description">
              <title id="match-guide-board-title">対戦フィールドの配置例</title>
              <desc id="match-guide-board-description">
                横14マス、縦11マス。左端1列が自分のじん地、右端1列が相手のじん地です。
                両チームのロボット1・2・3は、それぞれ上から6・7・8行目に並び、中央を向いてスタートします。
                岩と荷物は左右対称に配置されています。周囲は海です。
              </desc>
              <rect x="-12" y="-12" :width="width * 32 + 24" :height="height * 32 + 24" fill="#b9dce5" />
              <g v-for="cell in cells" :key="`${cell.x},${cell.y}`"
                :transform="`translate(${cell.x * 32} ${cell.y * 32})`">
                <rect width="32" height="32"
                  :fill="cell.x === 0 ? '#c1ded5' : cell.x === width - 1 ? '#e2d4ed' : '#f1f6f2'" stroke="#a9c9c1"
                  stroke-width="0.6" />
                <path v-if="cell.symbol === '#'" d="M4 25 7 11 17 5 27 14 26 26Z" fill="#637c78" />
                <g v-else-if="cell.symbol === 'O'" fill="#edcc79" stroke="#805d21" stroke-width="1.3">
                  <rect x="8" y="8" width="16" height="17" rx="2" />
                  <path d="M8 13H24M16 8V17" fill="none" />
                </g>
                <g v-else-if="cell.robot" :fill="cell.x === 0 ? '#16655f' : '#704396'">
                  <circle :cx="cell.x === 0 ? 13 : 19" cy="16" r="10" />
                  <path :d="cell.x === 0 ? 'M25 10 31 16 25 22Z' : 'M7 10 1 16 7 22Z'" />
                  <text :x="cell.x === 0 ? 13 : 19" y="20" fill="white" text-anchor="middle" font-size="12"
                    font-weight="700">
                    {{ cell.symbol }}
                  </text>
                </g>
              </g>
            </svg>
            <figcaption class="furigana-text">
              <strong>配置例</strong>。数字は設定欄のロボット番号、矢印は最初の向きです。
              ロボットの初期位置・向きとじん地は固定。岩と荷物の配置は、対戦ごとに候補から選ばれます。
            </figcaption>
            <ul class="match-guide-legend furigana-text" aria-label="配置図の凡例">
              <li>
                <span class="match-guide-swatch match-guide-swatch--own" aria-hidden="true" />自分のじん地
              </li>
              <li>
                <span class="match-guide-swatch match-guide-swatch--opponent" aria-hidden="true" />相手のじん地
              </li>
              <li><span class="match-guide-swatch match-guide-swatch--rock" aria-hidden="true" />岩</li>
              <li><span class="match-guide-swatch match-guide-swatch--luggage" aria-hidden="true" />荷物</li>
              <li><span class="match-guide-swatch match-guide-swatch--sea" aria-hidden="true" />海</li>
            </ul>
          </figure>
          <ul class="match-guide-list furigana-text">
            <li>荷物は1台につき1個まで。「拾う」「置く」はロボットの前のマスに対して行います。</li>
            <li>「置く」で前のマスが海だと、荷物を捨てて減点されます。</li>
            <li>
              「岩をこわす」は1台につき{{ battleGuide.drillLimit }}回まで成功できます。
              開けた道は相手も通れます。
            </li>
            <li>
              移動・回転・荷物の操作・岩をこわす行動は、成功しても失敗しても1ターン使います。地形などを調べるだけならターンは進みません。
            </li>
          </ul>
        </section>

        <details class="match-guide-fine-print">
          <summary class="furigana-heading">詳しいルールとレート</summary>
          <div class="furigana-text">
            <h3 class="furigana-heading">同時に動くとき</h3>
            <ul class="match-guide-list">
              <li>
                全ロボットはターン開始時の盤面を見て、同時に行動します。こわした岩のマスや置いた荷物は次のターンから使えます。
              </li>
              <li>
                同じマスへ進もうとしたロボットは全員、移動に失敗します。2台が互いの位置を入れ替える移動も失敗します。
              </li>
              <li>実行できる「拾う」「置く」が同じ対象で競合すると、競合した操作はすべて失敗します。</li>
            </ul>
            <h3 class="furigana-heading">試合が終わるとき</h3>
            <p>
              {{
                battleGuide.turns
              }}ターンに達する、両チームの全ロボットが停止する、または最初の荷物がすべて海に捨てられると終了します。
              最後のターンの行動も得点に反映されます。
            </p>
            <p>
              プログラムの終了や実行エラーで停止したロボットは盤面から消えます。
            </p>
            <h3 class="furigana-heading">対戦相手とレート</h3>
            <p>レートが近い相手ほど選ばれやすくなります。自分で相手を指定することはできません。</p>
            <p>
              自分から始めた対戦の結果に応じて、自分のレートが変わります。ほかの人から対戦されても、自分のレートは変わりません。
            </p>
          </div>
        </details>
      </div>
    </details>
  </section>
</template>

<style scoped>
.match-guide {
  padding-block: var(--space-lg);
  border-bottom: 1px solid var(--color-rule);
}

.match-guide h2 {
  word-break: keep-all;
}

.match-guide-lead {
  margin: 0 0 var(--space-sm);
  color: var(--color-ink-2);
}

summary {
  min-height: 44px;
  padding-block: var(--space-xs);
  color: var(--color-accent-deep);
  font-weight: 650;
  cursor: pointer;
}

.match-guide-body {
  display: grid;
  gap: var(--space-lg);
  padding-top: var(--space-md);
}

h3 {
  margin: 0 0 var(--space-sm);
  font-size: 1.125rem;
}

.match-guide-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-xl);
  margin: 0;
  padding-left: 1.5em;
}

.match-guide-steps li::marker {
  color: var(--color-accent-deep);
  font-weight: 700;
}

.match-guide-steps p {
  margin: var(--space-xs) 0 0;
  color: var(--color-ink-2);
}

.match-guide-facts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-md) var(--space-xl);
  margin: 0 0 var(--space-md);
}

.match-guide-facts dt {
  color: var(--color-muted);
  font-size: var(--type-small);
}

.match-guide-facts dd {
  margin: 0;
  font-weight: 650;
}

.match-guide-field {
  max-width: 600px;
  margin: 0 auto;
}

.match-guide-field-labels {
  display: flex;
  justify-content: space-between;
  gap: var(--space-md);
  margin-bottom: var(--space-xs);
}

.match-guide-field-labels strong:first-child {
  color: #16655f;
}

.match-guide-field-labels strong:last-child {
  color: #704396;
}

.match-guide-field svg {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 4px;
}

figcaption {
  margin-top: var(--space-sm);
  color: var(--color-muted);
  font-size: var(--type-small);
}

.match-guide-legend {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-xs) var(--space-md);
  margin: var(--space-sm) 0 0;
  padding: 0;
  list-style: none;
  font-size: var(--type-small);
}

.match-guide-legend li {
  display: flex;
  align-items: center;
  gap: var(--space-2xs);
}

.match-guide-swatch {
  width: 14px;
  height: 14px;
  border: 1px solid var(--color-muted);
}

.match-guide-swatch--own {
  background: #c1ded5;
}

.match-guide-swatch--opponent {
  background: #e2d4ed;
}

.match-guide-swatch--rock {
  background: #637c78;
}

.match-guide-swatch--luggage {
  background: #edcc79;
}

.match-guide-swatch--sea {
  background: #b9dce5;
}

.match-guide-list {
  margin: var(--space-md) 0 0;
  padding-left: 1.4em;
  color: var(--color-ink-2);
}

.match-guide-list li+li {
  margin-top: var(--space-xs);
}

.match-guide-fine-print {
  padding-top: var(--space-sm);
  border-top: 1px solid var(--color-rule);
}

.match-guide-fine-print h3 {
  margin-top: var(--space-md);
}

.match-guide-fine-print p {
  margin: var(--space-xs) 0 0;
  color: var(--color-ink-2);
}

@media (max-width: 600px) {
  .match-guide-steps {
    grid-template-columns: 1fr;
    gap: var(--space-md);
  }
}
</style>
