import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../../modules/auth'
import DefaultLayout from '../layouts/DefaultLayout.vue'

declare module 'vue-router' {
  interface RouteMeta {
    /** Require authenticated user. Redirects to /login if not authenticated. */
    requiresAuth?: boolean
    /** Require admin role. Implies requiresAuth. */
    requiresAdmin?: boolean
    /** Redirect away if already authenticated (login/register pages). */
    guestOnly?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: DefaultLayout,
    children: [
      {
        path: '',
        name: 'home',
        component: () => import('../../modules/blog/views/PostListView.vue'),
      },
      {
        path: 'posts/new',
        name: 'new-post',
        component: () => import('../../modules/blog/views/PostCreateView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'posts/:slug',
        name: 'post',
        component: () => import('../../modules/blog/views/PostDetailView.vue'),
      },
      {
        path: 'tools',
        name: 'tools',
        component: () => import('../../modules/tool/views/ToolsView.vue'),
      },
      {
        path: 'tools/json',
        name: 'json-tool',
        component: () => import('../../modules/tool/views/JsonToolView.vue'),
      },
      {
        path: 'tools/diff',
        name: 'diff-tool',
        component: () => import('../../modules/tool/views/DiffToolView.vue'),
      },
      {
        path: ':pathMatch(.*)*',
        name: 'not-found',
        component: () => import('../../modules/blog/views/NotFoundView.vue'),
      },
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../../modules/auth/views/LoginView.vue'),
    meta: { guestOnly: true },
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()

  const needsAuthResult =
    to.meta.guestOnly ||
    to.meta.requiresAuth ||
    to.meta.requiresAdmin

  if (!auth.initialized) {
    if (needsAuthResult) {
      try {
        await auth.ensureCurrentUser()
      } catch {
        // Auth server unavailable.
        // guestOnly: allow navigation (don't redirect a potentially-logged-in user).
        // requiresAuth / requiresAdmin: cancel — can't verify identity.
        if (to.meta.guestOnly) return next()
        return next(false)
      }
    } else {
      // Public route: fire in background, don't block navigation
      void auth.ensureCurrentUser().catch(() => {})
    }
  }

  // Guest-only pages redirect to home if already logged in
  if (to.meta.guestOnly && auth.user) {
    return next({ name: 'home' })
  }

  const needsAuth = to.meta.requiresAuth || to.meta.requiresAdmin

  if (needsAuth && auth.initialized && !auth.user) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }

  if (to.meta.requiresAdmin && auth.user?.role !== 'admin') {
    return next({ name: 'home' })
  }

  next()
})
