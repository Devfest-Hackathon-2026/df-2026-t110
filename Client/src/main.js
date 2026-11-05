import { createApp } from 'vue'
import RootApp from './RootApp.vue'

if (window.location.pathname.startsWith('/tutorials/demo')) {
  await import('./style.css')
} else {
  await import('./home.css')
}

createApp(RootApp).mount('#app')
