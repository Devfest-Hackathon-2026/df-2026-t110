import { createApp } from 'vue'
import RootApp from './RootApp.vue'

await import('../styles/site.css')
if (window.location.pathname.startsWith('/tutorials/demo')) {
  await import('../features/stage/stage.css')
} else if (/^\/tutorials\/[^/]+\/?$/.test(window.location.pathname)) {
  await import('../features/tutorial/tutorial.css')
}

createApp(RootApp).mount('#app')
