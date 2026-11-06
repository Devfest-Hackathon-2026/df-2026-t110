<script setup lang="ts">
defineProps<{ activePage: 'home' | 'tutorials' | 'tutorial-detail' | 'rooms' | 'account' | 'missing' }>()
</script>

<template>
  <header class="site-nav container">
    <a class="site-logo" href="/" aria-label="Robot Field ホーム">Robot <span>Field</span></a>
    <nav aria-label="メインナビゲーション">
      <a
        href="/tutorials"
        :aria-current="activePage === 'tutorials' || activePage === 'tutorial-detail' ? 'page' : undefined"
        >教材</a
      >
      <a href="/rooms" :aria-current="activePage === 'rooms' ? 'page' : undefined">対戦</a>
      <a href="/account" :aria-current="activePage === 'account' ? 'page' : undefined">ログイン</a>
    </nav>
  </header>
</template>
