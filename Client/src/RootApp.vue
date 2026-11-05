<script setup>
import { computed, defineAsyncComponent } from 'vue'

const StageViewer = defineAsyncComponent(() => import('./App.vue'))
const path = window.location.pathname.replace(/\/$/, '') || '/'
const isStage = path === '/tutorials/demo'
const page = computed(() => {
  if (path === '/tutorials') return 'tutorials'
  if (path === '/rooms') return 'rooms'
  if (path === '/account') return 'account'
  if (path === '/') return 'home'
  return 'missing'
})

document.title = isStage ? 'ステージプレビュー | Robot Field' : 'Robot Field | ロボットを動かして学ぼう'
</script>

<template>
  <StageViewer v-if="isStage" />
  <div v-else class="site">
    <header class="site-nav container">
      <a class="site-logo" href="/" aria-label="Robot Field ホーム">Robot <span>Field</span></a>
      <nav aria-label="メインナビゲーション">
        <a href="/tutorials" :aria-current="page === 'tutorials' ? 'page' : undefined">教材</a>
        <a href="/rooms" :aria-current="page === 'rooms' ? 'page' : undefined">対戦</a>
        <a href="/account" :aria-current="page === 'account' ? 'page' : undefined">ログイン</a>
      </nav>
    </header>

    <main v-if="page === 'home'">
      <section class="home-hero container" aria-labelledby="home-title">
        <div class="hero-copy">
          <h1 id="home-title">考えた通りに、<br />動くかな。</h1>
        </div>
        <div class="hero-detail">
          <p class="hero-lead">命令を組んで、ロボットを動かそう。岩をこわし、海に道をつくりながら、ゴールまでの道を考える教材です。</p>
          <div class="hero-actions">
            <a class="action-primary" href="/tutorials">教材を始める</a>
            <a class="text-link" href="/rooms">対戦について</a>
          </div>
        </div>
      </section>

      <section class="home-mission" aria-label="教材の例">
        <div class="container mission-inner">
          <div class="mission-copy">
            <span class="mission-number">01 / 最初のミッション</span>
            <p>まずは、<strong>1マス</strong>進めてみよう。</p>
          </div>
          <div class="mission-board">
            <div class="example-map" aria-label="スタートの右にゴールがある2マスの課題"><span>S</span><span>G</span></div>
            <div class="example-foot"><span>スタート</span><span>ゴール</span></div>
          </div>
        </div>
      </section>

      <section class="home-explainer container" aria-labelledby="learn-title">
        <div><h2 id="learn-title">一歩ずつ、できることを増やす。</h2></div>
        <div class="lesson-list">
          <p><span>01</span>前へ進み、向きを変える</p>
          <p><span>02</span>岩をこわし、海を埋める</p>
          <p><span>03</span>条件と繰り返しで道を見つける</p>
        </div>
      </section>

      <section class="home-next">
        <div class="container next-inner">
          <div><h2>まずはひとりで。<br />いつか、だれかと。</h2></div>
          <div class="next-links">
            <a href="/tutorials"><strong>教材を始める</strong><span>順番に試せる課題へ</span></a>
            <a href="/rooms"><strong>対戦を見る</strong><span>マッチング機能は準備中です</span></a>
          </div>
        </div>
      </section>
    </main>

    <main v-else class="inner-page container">
      <template v-if="page === 'tutorials'">
        <h1>最初の一歩から。</h1>
        <p class="inner-lead">命令を試しながら、ロボットの動かし方を覚えていきます。現在はステージ表示のプレビューを確認できます。</p>
        <a class="lesson-entry" href="/tutorials/demo"><span>01</span><span><strong>結晶の道</strong><small>配置候補とロボットの動作を確認する</small></span><span aria-hidden="true">↗</span></a>
        <p class="page-note">プログラミング教材の画面は準備中です。</p>
      </template>
      <template v-else-if="page === 'rooms'">
        <h1>対戦は、準備中です。</h1>
        <p class="inner-lead">自分で考えた命令でロボットを動かし、ほかの人と同じフィールドに挑む機能を予定しています。</p>
        <a class="action-primary" href="/tutorials">先に教材を試す</a>
      </template>
      <template v-else-if="page === 'account'">
        <h1>ログインは、準備中です。</h1>
        <p class="inner-lead">教材を試すためのアカウント連携を準備しています。今はステージのプレビューを見られます。</p>
        <a class="action-primary" href="/tutorials">教材を見る</a>
      </template>
      <template v-else>
        <h1>ページが見つかりません。</h1>
        <a class="action-primary" href="/">ホームへ戻る</a>
      </template>
    </main>

    <footer class="site-footer container">
      <span>Robot Field</span>
      <p>ロボットを動かして、考えるための場所。</p>
      <a href="/">ホームへ</a>
    </footer>
  </div>
</template>
