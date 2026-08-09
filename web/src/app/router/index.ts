import { createRouter, createWebHistory } from 'vue-router'
import PostListView from '../../modules/blog/views/PostListView.vue'
import PostDetailView from '../../modules/blog/views/PostDetailView.vue'
import PostCreateView from '../../modules/blog/views/PostCreateView.vue'
import LoginView from '../../modules/auth/views/LoginView.vue'
import ToolsView from '../../modules/tool/views/ToolsView.vue'
import JsonToolView from '../../modules/tool/views/JsonToolView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: PostListView },
    { path: '/posts/new', name: 'new-post', component: PostCreateView },
    { path: '/posts/:slug', name: 'post', component: PostDetailView },
    { path: '/login', name: 'login', component: LoginView },
    { path: '/tools', name: 'tools', component: ToolsView },
    { path: '/tools/json', name: 'json-tool', component: JsonToolView },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../../modules/blog/views/NotFoundView.vue') },
  ],
})
