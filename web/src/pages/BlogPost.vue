<template>
  <div class="page">
    <div class="reader">
      <a class="back" @click="goBack">← 返回手账</a>

      <!-- 加载中 -->
      <div v-if="loading" class="empty-state">
        <span class="icon">🗺️</span>
        <p>正在翻阅…</p>
      </div>

      <!-- 加载出错 -->
      <div v-else-if="error" class="empty-state">
        <span class="icon">🌧️</span>
        <p>{{ error }}</p>
      </div>

      <!-- 文章内容 -->
      <template v-else-if="post">
        <div class="fieldlog">
          <svg class="marker" viewBox="0 0 48 48" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M24 6 v36" />
            <path d="M24 12 h14 l4 4 -4 4 h-14 z" />
            <path d="M24 22 h-14 l-4 4 4 4 h14 z" />
            <path d="M17 42 h14" />
          </svg>
          <div class="meta">
            {{ post.weather || '云开' }}，↑{{ post.elevation || '2140' }}m · 用时约 {{ post.readTime || '6' }} 分钟
            <span class="n">{{ formatMeta(post) }}</span>
          </div>
        </div>

        <article class="article">
          <h1>{{ post.title }}</h1>
          <div v-html="renderContent(post.content)"></div>
        </article>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { request } from '../api/request'

const router = useRouter()
const route = useRoute()
const post = ref(null)
const loading = ref(true)
const error = ref('')

function formatMeta(p) {
  const parts = []
  if (p.waypoint) parts.push(p.waypoint)
  else parts.push('WP.' + (p.id || '11').toString().padStart(2, '0'))
  if (p.coords) parts.push(p.coords)
  else parts.push('N 40.09° E 116.35°')
  if (p.created_at) parts.push(new Date(p.created_at).toISOString().slice(0, 10))
  else if (p.date) parts.push(p.date)
  else parts.push('2026-07-18')
  return parts.join(' · ')
}

function renderContent(content) {
  if (!content) return '<p>（暂无内容）</p>'
  // 简单 markdown 渲染：段落、加粗、代码块
  let html = content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // 代码块 ```...```
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
  // 行内代码 `...`
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')
  // 加粗 **...**
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  // 斜体 *...*
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  // 标题 ### ...
  html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
  // 引用 >
  html = html.replace(/^&gt; (.+)$/gm, '<blockquote>$1</blockquote>')
  // 链接 [text](url)
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
  // 段落：双换行分隔
  html = html.replace(/\n\n/g, '</p><p>')
  html = html.replace(/\n/g, '<br>')
  html = '<p>' + html + '</p>'

  return html
}

function goBack() {
  router.push('/')
}

onMounted(async () => {
  try {
    const id = route.params.id
    const res = await request(`/api/post/${id}`)
    post.value = res.data || res
  } catch (e) {
    // 如果 API 不可用，显示模拟数据
    post.value = {
      id: route.params.id,
      title: '在 Windows 上用 Claude Code 接入 DeepSeek',
      content: `很多人第一次看到这个组合会困惑：既然想用的是 DeepSeek，为什么还要安装 Claude Code 呢？

答案是：**Claude Code 是终端里的编程客户端，DeepSeek 才是实际回答和生成代码的模型后端。**

DeepSeek 官方提供了 Anthropic API 兼容接口。只要把 Claude Code 的接口地址、认证令牌和模型名改为 DeepSeek 的配置，Claude Code 发出的请求就会交给 DeepSeek 处理；不需要 Anthropic API Key，也不会调用 Claude 模型。

> 本文最后核对：2026-07-18。模型名称和 API 能力会更新，实际使用时以 DeepSeek API 文档为准。

## 配置环境变量

核心就是三个变量：接口地址、令牌、模型名。在 PowerShell 里这样设置：

\`\`\`
setx ANTHROPIC_BASE_URL "https://api.deepseek.com/anthropic"
setx ANTHROPIC_AUTH_TOKEN "sk-你的-deepseek-key"
setx ANTHROPIC_MODEL "deepseek-chat"
\`\`\`

重开一个终端让变量生效，然后 \`claude\` 启动即可。第一次运行会让你确认工作目录信任，选 yes 就能开始。

## 为什么值得这么折腾

你拿到的是 Claude Code 的终端体验——多文件编辑、工具调用、项目上下文——而账单走 DeepSeek，成本低很多。对个人项目来说，这是目前性价比很高的一条路线。`,
      weather: '云开',
      elevation: '2140',
      readTime: '6',
      waypoint: 'WP.11',
      coords: 'N 40.09° · E 116.35°',
      created_at: '2026-07-18'
    }
  } finally {
    loading.value = false
  }
})
</script>
