import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../../modules/auth/views/LoginView.vue'
import PostListView from '../../modules/blog/views/PostListView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: PostListView },
    { path: '/login', name: 'login', component: LoginView },
  ],
})
