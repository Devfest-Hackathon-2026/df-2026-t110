<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import userRoundIcon from '@iconify-icons/lucide/user-round'
import logOutIcon from '@iconify-icons/lucide/log-out'
import { loginUrl, logout, MatchApiError, setMyUsername } from './matchApi.ts'
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
    console.error('表示名の設定に失敗しました', cause)
    const status = cause instanceof MatchApiError ? cause.status : undefined
    error.value =
      status === 401
        ? '表示名を設定するには、ログインし直してください。'
        : status === 400
          ? '表示名を確認してください。32文字以内の半角英数字・_・-を使用でき、「self」「std」は使えません。'
          : status === 409
            ? 'この表示名は使用済みか、このアカウントにはすでに表示名が設定されています。'
            : '表示名を保存できませんでした。しばらくしてから再試行してください。'
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
    console.error('ログアウトに失敗しました', cause)
    error.value = 'ログアウトできませんでした。もう一度お試しください。'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="account-page container">
    <div class="account-page-heading">
      <h1>アカウント</h1>
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
            <h2 id="account-profile-title">{{ store.user?.name || '名前を設定してください' }}</h2>
            <p>チュートリアルの実行と対戦に、このアカウントを使用します。</p>
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
          <h2 id="account-guest-title">ログイン</h2>
          <p>Googleアカウントでログインすると、チュートリアルのプログラム保存・実行と対戦ができます。</p>
        </div>
      </div>
      <div class="account-guest-actions">
        <a class="action-primary" :href="loginUrl">Googleでログイン</a>
        <a class="text-link" href="/tutorials">チュートリアルを見る</a>
      </div>
    </section>
  </main>
</template>
