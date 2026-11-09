<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import userRoundIcon from '@iconify-icons/lucide/user-round'
import logOutIcon from '@iconify-icons/lucide/log-out'
import { loginUrl, logout, setMyUsername } from './matchApi.ts'
import { useMatchStore } from './useMatchStore.ts'
import './account.css'

const store = useMatchStore()
const username = ref('')
const busy = ref(false)
const error = ref('')
onMounted(() => {
  void store.loadMe()
})

async function saveUsername(): Promise<void> {
  if (busy.value || !username.value.trim()) return
  busy.value = true
  error.value = ''
  try {
    await setMyUsername(username.value.trim())
    await store.loadMe(true)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'ユーザー名を設定できませんでした'
  } finally {
    busy.value = false
  }
}

async function signOut(): Promise<void> {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await logout()
    await store.loadMe(true)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'ログアウトできませんでした'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="account-page container">
    <div class="account-page-heading">
      <p class="account-eyebrow">ROBOT FIELD / ACCOUNT</p>
      <h1>アカウント</h1>
      <p>ロボットと学ぶための、あなたの場所。</p>
    </div>

    <p v-if="error" class="account-error" role="alert">{{ error }}</p>

    <section v-if="store.authStatus === 'checking'" class="account-panel" aria-live="polite">
      <p class="account-loading">ログイン状態を確認しています…</p>
    </section>

    <template v-else-if="store.authStatus === 'signed-in'">
      <section class="account-panel" aria-labelledby="account-profile-title">
        <div class="account-profile">
          <div class="account-avatar" aria-hidden="true">
            {{ (store.user?.name || 'R').slice(0, 1).toUpperCase() }}
          </div>
          <div class="account-identity">
            <p class="account-status"><span aria-hidden="true"></span>ログイン中</p>
            <h2 id="account-profile-title">{{ store.user?.name || '名前を設定してください' }}</h2>
            <p>教材の実行と対戦に、このアカウントを使用します。</p>
          </div>
        </div>

        <form v-if="!store.user?.name" class="account-name-form" @submit.prevent="saveUsername">
          <label for="account-username">表示名を設定する</label>
          <p>半角英数字・_・-を使用できます。設定後は変更できません。</p>
          <div class="account-name-controls">
            <input
              id="account-username"
              v-model="username"
              required
              minlength="1"
              maxlength="32"
              pattern="[A-Za-z0-9_-]{1,32}"
              autocomplete="off"
            />
            <button type="submit" :disabled="busy">名前を保存</button>
          </div>
        </form>

        <div class="account-panel-footer">
          <span>アカウントの操作</span>
          <button type="button" class="account-logout" :disabled="busy" @click="signOut">
            <Icon :icon="logOutIcon" aria-hidden="true" />ログアウト
          </button>
        </div>
      </section>
    </template>

    <section v-else class="account-panel account-panel--guest" aria-labelledby="account-guest-title">
      <div class="account-profile">
        <div class="account-avatar account-avatar--guest" aria-hidden="true">
          <Icon :icon="userRoundIcon" />
        </div>
        <div class="account-identity">
          <p class="account-status account-status--guest">未ログイン</p>
          <h2 id="account-guest-title">学びの続きを始めよう</h2>
          <p>Googleアカウントでログインすると、教材のプログラム保存・実行と対戦ができます。</p>
        </div>
      </div>
      <div class="account-guest-actions">
        <a class="action-primary" :href="loginUrl">Googleでログイン</a>
        <a class="text-link" href="/tutorials">教材を見る</a>
      </div>
    </section>
  </main>
</template>
