import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import './style.css'
import App from './App.vue'

import BlogList from './pages/BlogList.vue'
import BlogPost from './pages/BlogPost.vue'
import LoginPage from './pages/LoginPage.vue'

const routes = [
  { path: '/', name: 'list', component: BlogList },
  { path: '/post/:id', name: 'post', component: BlogPost },
  { path: '/login', name: 'login', component: LoginPage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

createApp(App).use(router).mount('#app')
