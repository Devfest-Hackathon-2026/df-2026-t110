<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { Icon } from '@iconify/vue'
import infoIcon from '@iconify-icons/lucide/info'
import bookOpenIcon from '@iconify-icons/lucide/book-open'
import swordsIcon from '@iconify-icons/lucide/swords'
import logInIcon from '@iconify-icons/lucide/log-in'
import circleHelpIcon from '@iconify-icons/lucide/circle-help'
import { LOCAL_LESSONS } from '../features/tutorial/tutorialLessons.ts'
import AppHeader from './AppHeader.vue'

const StageViewer = defineAsyncComponent(() => import('../features/stage/StageViewer.vue'))
const TutorialPage = defineAsyncComponent(() => import('../features/tutorial/TutorialPage.vue'))
const MatchPage = defineAsyncComponent(() => import('../features/match/MatchPage.vue'))
const path = window.location.pathname.replace(/\/$/, '') || '/'
const isStage = path === '/tutorials/demo'
const tutorialMatch = /^\/tutorials\/([^/]+)$/.exec(path)
const tutorialId = tutorialMatch && !isStage ? decodeURIComponent(tutorialMatch[1]) : null
const roomMatch = /^\/rooms\/([^/]+)$/.exec(path)
const roomId = roomMatch ? decodeURIComponent(roomMatch[1]) : null
const matchRoute = /^\/matches\/([^/]+)$/.exec(path)
const matchId = matchRoute ? decodeURIComponent(matchRoute[1]) : null
const page = computed(() => {
  if (path === '/tutorials') return 'tutorials'
  if (tutorialId) return 'tutorial-detail'
  if (path === '/rooms') return 'rooms'
  if (roomId) return 'rooms'
  if (matchId) return 'match-detail'
  if (path === '/account') return 'account'
  if (path === '/') return 'home'
  return 'missing'
})

document.title = isStage
  ? 'preview'
  : tutorialId
    ? '教材 | Robot Field'
    : matchId
      ? '対戦結果 | Robot Field'
      : path === '/rooms'
        ? '対戦 | Robot Field'
        : 'Robot Field'
</script>

<template>
  <div :class="isStage ? 'stage-page' : 'site'">
    <AppHeader :active-page="isStage ? 'tutorials' : page === 'match-detail' ? 'rooms' : page" />

    <StageViewer v-if="isStage" />

    <main v-else-if="page === 'home'">
      <section class="home-hero container" aria-labelledby="home-title">
        <div class="hero-copy">
          <h1 id="home-title" class="heading-with-icon">
            <Icon :icon="infoIcon" aria-hidden="true" /><span>About</span>
          </h1>
          <p class="hero-lead">
            命令を組んで、ロボットを動かそう。岩をこわし、海に道をつくりながら、ゴールまでの道を考える教材です。
          </p>
          <div class="hero-actions">
            <a class="action-primary" href="/tutorials">教材を始める</a>
            <a class="text-link" href="/rooms">対戦について</a>
          </div>
        </div>
        <a class="hero-preview">
          <img
            src="/preview-first-step.png"
            alt="緑のロボットがスタートに立ち、右隣にゴールがある3Dステージ"
            width="768"
            height="1024"
          />
        </a>
      </section>

      <section class="home-explainer container" aria-labelledby="learn-title">
        <h1 id="learn-title" class="heading-with-icon">
          <Icon :icon="bookOpenIcon" aria-hidden="true" /><span>学習チュートリアル機能</span>
        </h1>
        <div class="lesson-list">
          <p><span>01</span>前へ進み、向きを変える</p>
          <p><span>02</span>岩をこわし、海を埋める</p>
          <p><span>03</span>条件と繰り返しで道を見つける</p>
        </div>
      </section>

      <section class="home-next">
        <div class="container next-inner">
          <div>
            <h1 class="heading-with-icon">
              <Icon :icon="swordsIcon" aria-hidden="true" /><span>対戦機能</span>
            </h1>
          </div>
          <div class="next-links">
            <a href="/tutorials"><strong>教材を始める</strong></a>
            <a href="/rooms"><strong>対戦を見る</strong></a>
          </div>
        </div>
      </section>
    </main>

    <main v-else-if="page === 'tutorial-detail'" class="tutorial-main">
      <TutorialPage :tutorial-id="tutorialId ?? ''" />
    </main>

    <MatchPage v-else-if="page === 'rooms' || page === 'match-detail'" :match-id="matchId" />

    <main v-else class="inner-page container">
      <template v-if="page === 'tutorials'">
        <h1 class="heading-with-icon">
          <Icon :icon="bookOpenIcon" aria-hidden="true" /><span>最初の一歩から。</span>
        </h1>
        <p class="inner-lead">短い道から始めて、ロボットにできることを少しずつ増やしていきます。</p>
        <a
          v-for="(lesson, index) in LOCAL_LESSONS"
          :key="lesson.id"
          class="lesson-entry"
          :href="`/tutorials/${lesson.id}`"
          ><span>{{ String(index + 1).padStart(2, '0') }}</span
          ><span
            ><strong>{{ lesson.title }}</strong
            ><small>{{ lesson.goal }}</small></span
          ><span aria-hidden="true">↗</span></a
        >
        <p class="page-note">
          教材とフィールドは現在サンプルデータです。<a href="/tutorials/demo"
            >ステージ表示の開発用プレビュー</a
          >も確認できます。
        </p>
      </template>
      <template v-else-if="page === 'account'">
        <h1 class="heading-with-icon">
          <Icon :icon="logInIcon" aria-hidden="true" /><span>ログインは、準備中です。</span>
        </h1>
        <p class="inner-lead">
          教材を試すためのアカウント連携を準備しています。今はステージのプレビューを見られます。
        </p>
        <a class="action-primary" href="/tutorials">教材を見る</a>
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
