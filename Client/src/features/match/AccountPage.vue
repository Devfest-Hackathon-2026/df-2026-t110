<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Icon } from '@iconify/vue'
import logInIcon from '@iconify-icons/lucide/log-in'
import { loginUrl, logout, setMyUsername } from './matchApi.ts'
import { useMatchStore } from './useMatchStore.ts'

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
    await store.loadMe()
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
    await store.loadMe()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'ログアウトできませんでした'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="inner-page container">
    <h1 class="heading-with-icon"><Icon :icon="logInIcon" aria-hidden="true" /><span>アカウント</span></h1>
    <p v-if="store.authStatus === 'checking'" class="inner-lead">ログイン状態を確認しています…</p>
    <p v-if="error" role="alert">{{ error }}</p>
    <template v-if="store.authStatus === 'signed-in'">
      <p class="inner-lead">{{ store.user?.name || 'ログイン中' }}として利用しています。</p>
      <form v-if="!store.user?.name" @submit.prevent="saveUsername">
        <label for="account-username">ユーザー名</label>
        <input
          id="account-username"
          v-model="username"
          required
          minlength="1"
          maxlength="32"
          pattern="[A-Za-z0-9_-]{1,32}"
          autocomplete="off"
        />
        <p>半角英数字・_・-を使用できます。設定後は変更できません。</p>
        <button type="submit" :disabled="busy">ユーザー名を設定</button>
      </form>
      <a class="action-primary" href="/rooms">対戦の準備へ</a>
      <button type="button" :disabled="busy" @click="signOut">ログアウト</button>
    </template>
    <template v-else-if="store.authStatus === 'guest'">
      <p class="inner-lead">対戦にはGoogleアカウントでログインしてください。</p>
      <a class="action-primary" :href="loginUrl">Googleでログイン</a>
    </template>
  </main>
</template>
