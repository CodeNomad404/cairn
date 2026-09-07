import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './app/App.vue'
import { router } from './app/router'
import './shared/styles/base.css'
import './app/styles/tokens.css'
import './app/styles/global.css'
import './modules/blog/blog.css'
import './modules/auth/auth.css'
import './modules/tool/tool.css'

createApp(App).use(createPinia()).use(router).mount('#app')
