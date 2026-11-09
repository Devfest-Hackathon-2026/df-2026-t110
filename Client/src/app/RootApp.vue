<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { Icon } from '@iconify/vue'
import bookOpenIcon from '@iconify-icons/lucide/book-open'
import swordsIcon from '@iconify-icons/lucide/swords'
import arrowUpRightIcon from '@iconify-icons/lucide/arrow-up-right'
import circleHelpIcon from '@iconify-icons/lucide/circle-help'
import AppHeader from './AppHeader.vue'
import HomeFeatureCarousel from './HomeFeatureCarousel.vue'

const tutorialSlides = [
  {
    image: '/home-tutorial-first.svg',
    alt: '緑のロボットが隣のゴールへ進むイメージ',
    caption: '最初の一歩は、右隣のゴールへ。命令と動きのつながりを確かめます。',
  },
  {
    image: '/home-tutorial-beam.svg',
    alt: 'ロボットが目のビームで岩を壊して道を作るイメージ',
    caption: '道をふさぐ岩にはビーム。地形を観察して、次の行動を選びます。',
  },
  {
    image: '/home-tutorial-route.svg',
    alt: 'ロボットが海を渡り、ゴールまでの道を探すイメージ',
    caption: '海を埋めて進みます。同じ命令でも、盤面に合わせて考え方が変わります。',
  },
]

const matchSlides = [
  {
    image: '/home-match-programs.svg',
    alt: '緑と紫のロボットにプログラムを割り当てる対戦準備のイメージ',
    caption: '作ったプログラムをロボットに割り当て、対戦の準備をします。',
  },
  {
    image: '/home-match-field.svg',
    alt: '緑と紫のロボットが荷物のあるフィールドで競うイメージ',
    caption: 'どの道を通るか、いつ荷物を運ぶか。作戦をプログラムにします。',
  },
  {
    image: '/home-match-replay.svg',
    alt: '対戦結果と行動の再生画面のイメージ',
    caption: '結果を見たら動きを再生。うまくいかなかった理由を探せます。',
  },
]

const TutorialPage = defineAsyncComponent(() => import('../features/tutorial/TutorialPage.vue'))
const TutorialCatalog = defineAsyncComponent(() => import('../features/tutorial/TutorialCatalog.vue'))
const MatchPage = defineAsyncComponent(() => import('../features/match/MatchPage.vue'))
const AccountPage = defineAsyncComponent(() => import('../features/match/AccountPage.vue'))
const RankingPage = defineAsyncComponent(() => import('../features/ranking/RankingPage.vue'))
const path = window.location.pathname.replace(/\/$/, '') || '/'
const tutorialMatch = /^\/tutorials\/([^/]+)$/.exec(path)
const tutorialId = tutorialMatch ? decodeURIComponent(tutorialMatch[1]) : null
const matchRoute = /^\/matches\/([^/]+)$/.exec(path)
const matchId = matchRoute ? decodeURIComponent(matchRoute[1]) : null
const rankingRoute = /^\/ranking(?:\/([1-9]\d*))?$/.exec(path)
const page = computed(() => {
  if (path === '/tutorials') return 'tutorials'
  if (tutorialId) return 'tutorial-detail'
  if (path === '/rooms') return 'rooms'
  if (rankingRoute) return 'ranking'
  if (matchId) return 'match-detail'
  if (path === '/account') return 'account'
  if (path === '/') return 'home'
  return 'missing'
})

document.title = tutorialId
  ? '教材 | Robot Field'
  : matchId
    ? '対戦結果 | Robot Field'
    : path === '/rooms'
      ? '対戦 | Robot Field'
      : rankingRoute
        ? 'ランキング | Robot Field'
        : 'Robot Field'
</script>

<template>
  <div class="site">
    <AppHeader :active-page="page === 'match-detail' ? 'rooms' : page" />

    <main v-if="page === 'home'">
      <section class="home-hero container" aria-labelledby="home-title">
        <h2>命令を組んで、ロボットを動かそう。</h2>
        <p class="hero-lead">
          岩をこわし、海に道をつくる。小さなロボットと一緒に、考えた手順をフィールドで試すプログラミング教材です。
        </p>
        <div class="home-entry-grid">
          <a class="home-entry-card" href="/tutorials">
            <Icon :icon="bookOpenIcon" aria-hidden="true" />
            <span><strong>教材を始める</strong><small>プログラムを組んでロボットを動かす</small></span>
            <Icon :icon="arrowUpRightIcon" class="home-entry-arrow" aria-hidden="true" />
          </a>
          <a class="home-entry-card" href="/rooms">
            <Icon :icon="swordsIcon" aria-hidden="true" />
            <span><strong>対戦を始める</strong><small>ロボットにプログラムを割り当てる</small></span>
            <Icon :icon="arrowUpRightIcon" class="home-entry-arrow" aria-hidden="true" />
          </a>
        </div>
      </section>

      <section class="home-explainer container" aria-labelledby="learn-title">
        <div class="home-feature-copy">
          <p class="home-eyebrow">01 / LEARN</p>
          <h2 id="learn-title" class="heading-with-icon">
            <Icon :icon="bookOpenIcon" aria-hidden="true" /><span>学習チュートリアル</span>
          </h2>
          <p>一つずつ命令を試しながら、条件や繰り返しへ。ロボットの動きを見て、プログラムを直せます。</p>
        </div>
        <HomeFeatureCarousel label="学習チュートリアルの紹介" :slides="tutorialSlides" />
        <a class="home-feature-link" href="/tutorials">教材を見る <span aria-hidden="true">↗</span></a>
      </section>

      <section class="home-next" aria-labelledby="match-title">
        <div class="container next-inner">
          <div class="home-feature-copy">
            <p class="home-eyebrow">02 / MATCH</p>
            <h2 id="match-title" class="heading-with-icon">
              <Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦機能</span>
            </h2>
            <p>
              自分のプログラムをロボットに設定して対戦。結果だけでなく、どんな行動をしたかも振り返れます。
            </p>
          </div>
          <HomeFeatureCarousel label="対戦機能の紹介" :slides="matchSlides" />
          <a class="home-feature-link" href="/rooms">対戦を見る <span aria-hidden="true">↗</span></a>
        </div>
      </section>
    </main>

    <main v-else-if="page === 'tutorial-detail'" class="tutorial-main">
      <TutorialPage :tutorial-id="tutorialId ?? ''" />
    </main>

    <MatchPage v-else-if="page === 'rooms' || page === 'match-detail'" :match-id="matchId" />
    <AccountPage v-else-if="page === 'account'" />
    <RankingPage v-else-if="page === 'ranking'" />

    <main v-else class="inner-page container">
      <template v-if="page === 'tutorials'">
        <h1 class="heading-with-icon">
          <Icon :icon="bookOpenIcon" aria-hidden="true" /><span>最初の一歩から。</span>
        </h1>
        <p class="inner-lead">短い道から始めて、ロボットにできることを少しずつ増やしていきます。</p>
        <TutorialCatalog />
      </template>
      <template v-else>
        <h1 class="heading-with-icon">
          <Icon :icon="circleHelpIcon" aria-hidden="true" /><span>ページが見つかりません。</span>
        </h1>
        <a class="action-primary" href="/">ホームへ戻る</a>
      </template>
    </main>
  </div>
</template>
