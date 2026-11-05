import { createApp } from 'vue'
import RootApp from './RootApp.vue'

if (window.location.pathname.startsWith('/tutorials/demo')) {
  await import('./style.css')
} else {
  await import('./home.css')
  if (/^\/tutorials\/[^/]+\/?$/.test(window.location.pathname)) await import('./tutorial.css')
}

createApp(RootApp).mount('#app')
