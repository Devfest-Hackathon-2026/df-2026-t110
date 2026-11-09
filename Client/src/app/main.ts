import { createApp } from 'vue'
import { createPinia } from 'pinia'
import RootApp from './RootApp.vue'

await import('../styles/site.css')
if (window.location.pathname.startsWith('/rooms') || window.location.pathname.startsWith('/matches')) {
  await import('../features/match/match.css')
} else if (/^\/tutorials\/[^/]+\/?$/.test(window.location.pathname)) {
  await import('../features/tutorial/tutorial.css')
}

createApp(RootApp).use(createPinia()).mount('#app')
