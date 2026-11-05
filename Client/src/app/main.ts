import { createApp } from 'vue'
import RootApp from './RootApp.vue'

if (window.location.pathname.startsWith('/tutorials/demo')) {
  await import('../features/stage/stage.css')
} else {
  await import('../styles/site.css')
  if (/^\/tutorials\/[^/]+\/?$/.test(window.location.pathname))
    await import('../features/tutorial/tutorial.css')
}

createApp(RootApp).mount('#app')
