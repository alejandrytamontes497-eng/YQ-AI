<template>
  <AppLayout>
    <div class="docs-page">
      <aside class="docs-nav">
        <div class="px-5 py-5">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600 text-white">
              <Icon name="book" size="md" />
            </div>
            <div>
              <h1 class="text-lg font-semibold text-gray-950 dark:text-white">文档</h1>
              <p class="text-xs text-gray-500 dark:text-gray-400">API 接入指南</p>
            </div>
          </div>
        </div>

        <nav class="space-y-1 px-3 pb-5">
          <a
            v-for="section in sections"
            :key="section.id"
            :href="`#${section.id}`"
            class="docs-nav-link"
          >
            <Icon :name="section.icon" size="sm" />
            <span>{{ section.title }}</span>
          </a>
        </nav>
      </aside>

      <main class="docs-content">
        <section id="overview" class="docs-hero">
          <div>
            <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">Sub2API Docs</p>
            <h2 class="mt-3 text-3xl font-bold text-gray-950 dark:text-white">把你的 API 密钥接入统一网关</h2>
            <p class="mt-4 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
              这里整理了本地服务最常用的接入方式：创建密钥、发起聊天请求、查看用量、排查错误。接口兼容 OpenAI 风格，
              客户端只需要把 Base URL 指向当前站点即可。
            </p>
          </div>

          <div class="quick-grid">
            <div class="quick-item">
              <span>Base URL</span>
              <code>{{ origin }}/v1</code>
            </div>
            <div class="quick-item">
              <span>认证方式</span>
              <code>Authorization: Bearer YOUR_API_KEY</code>
            </div>
            <div class="quick-item">
              <span>计费来源</span>
              <code>请求使用的 API 密钥</code>
            </div>
          </div>
        </section>

        <section id="quickstart" class="docs-section">
          <h3>快速开始</h3>
          <p>先在左侧菜单进入 API 密钥页面创建一个可用密钥，然后用它调用聊天接口。</p>
          <CodeBlock title="cURL" :code="quickstartCode" />
        </section>

        <section id="auth" class="docs-section">
          <h3>认证</h3>
          <p>所有网关请求都通过请求头中的 Bearer Token 认证。不同密钥会分别记录用量、余额消耗和限额。</p>
          <div class="docs-table">
            <div class="docs-table-row docs-table-head">
              <span>字段</span>
              <span>说明</span>
            </div>
            <div class="docs-table-row">
              <code>Authorization</code>
              <span>固定格式：Bearer + 空格 + API 密钥</span>
            </div>
            <div class="docs-table-row">
              <code>Content-Type</code>
              <span>发送 JSON 时使用 application/json</span>
            </div>
          </div>
        </section>

        <section id="chat" class="docs-section">
          <h3>聊天接口</h3>
          <p>用于文本对话、代码问答、内容生成等场景。前端在线聊天窗口也是调用这个接口。</p>
          <Endpoint method="POST" path="/v1/chat/completions" />
          <CodeBlock title="请求体" :code="chatBodyCode" />
          <CodeBlock title="响应示例" :code="chatResponseCode" />
        </section>

        <section id="models" class="docs-section">
          <h3>模型列表</h3>
          <p>如果上游渠道支持模型列表，可以通过兼容接口获取；也可以在后台渠道配置中维护可用模型。</p>
          <Endpoint method="GET" path="/v1/models" />
          <CodeBlock title="cURL" :code="modelsCode" />
        </section>

        <section id="keys" class="docs-section">
          <h3>密钥与用量</h3>
          <p>密钥由当前站点管理。用户可以创建多个密钥分别用于不同应用，每个密钥独立记录用量和限额。</p>
          <div class="feature-list">
            <div>
              <Icon name="key" size="md" />
              <span>按密钥计费</span>
            </div>
            <div>
              <Icon name="chart" size="md" />
              <span>用量记录可追踪</span>
            </div>
            <div>
              <Icon name="shield" size="md" />
              <span>支持状态和限额控制</span>
            </div>
          </div>
        </section>

        <section id="errors" class="docs-section">
          <h3>错误处理</h3>
          <p>接口错误通常会返回 HTTP 状态码和 JSON 错误信息。客户端应优先展示 error.message。</p>
          <div class="docs-table">
            <div class="docs-table-row docs-table-head">
              <span>状态码</span>
              <span>常见原因</span>
            </div>
            <div class="docs-table-row">
              <code>401</code>
              <span>API 密钥缺失、无效或已停用</span>
            </div>
            <div class="docs-table-row">
              <code>429</code>
              <span>达到请求频率或用量限制</span>
            </div>
            <div class="docs-table-row">
              <code>500</code>
              <span>上游服务异常或本地服务处理失败</span>
            </div>
          </div>
        </section>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

type IconName =
  | 'book'
  | 'play'
  | 'lock'
  | 'chat'
  | 'cpu'
  | 'key'
  | 'exclamationCircle'

interface DocSection {
  id: string
  title: string
  icon: IconName
}

const origin = computed(() => window.location.origin)

const sections: DocSection[] = [
  { id: 'overview', title: '概览', icon: 'book' },
  { id: 'quickstart', title: '快速开始', icon: 'play' },
  { id: 'auth', title: '认证', icon: 'lock' },
  { id: 'chat', title: '聊天接口', icon: 'chat' },
  { id: 'models', title: '模型列表', icon: 'cpu' },
  { id: 'keys', title: '密钥与用量', icon: 'key' },
  { id: 'errors', title: '错误处理', icon: 'exclamationCircle' }
]

const quickstartCode = computed(() => `curl ${origin.value}/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-5-20250929",
    "messages": [
      { "role": "user", "content": "你好，介绍一下你自己" }
    ]
  }'`)

const chatBodyCode = `{
  "model": "claude-sonnet-4-5-20250929",
  "messages": [
    { "role": "system", "content": "你是一个简洁可靠的助手。" },
    { "role": "user", "content": "帮我写一个接口调用示例" }
  ],
  "temperature": 0.7,
  "max_tokens": 2048,
  "stream": false
}`

const chatResponseCode = `{
  "id": "chatcmpl_xxx",
  "object": "chat.completion",
  "model": "claude-sonnet-4-5-20250929",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "当然，下面是一个示例..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 24,
    "completion_tokens": 42,
    "total_tokens": 66
  }
}`

const modelsCode = computed(() => `curl ${origin.value}/v1/models \\
  -H "Authorization: Bearer YOUR_API_KEY"`)

const Endpoint = defineComponent({
  props: {
    method: { type: String, required: true },
    path: { type: String, required: true }
  },
  setup(props) {
    return () => h('div', { class: 'endpoint-line' }, [
      h('span', { class: 'endpoint-method' }, props.method),
      h('code', props.path)
    ])
  }
})

const CodeBlock = defineComponent({
  props: {
    title: { type: String, required: true },
    code: { type: String, required: true }
  },
  setup(props) {
    const copy = async () => {
      await navigator.clipboard?.writeText(props.code)
    }

    return () => h('div', { class: 'code-block' }, [
      h('div', { class: 'code-header' }, [
        h('span', props.title),
        h('button', { type: 'button', onClick: copy }, '复制')
      ]),
      h('pre', [h('code', props.code)])
    ])
  }
})
</script>

<style scoped>
.docs-page {
  @apply -m-4 flex min-h-[calc(100vh-4rem)] bg-white dark:bg-dark-900 sm:-m-6;
}

.docs-nav {
  @apply sticky top-0 hidden h-screen w-72 shrink-0 border-r border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950 lg:block;
}

.docs-nav-link {
  @apply flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-gray-600 transition hover:bg-white hover:text-primary-700 dark:text-gray-300 dark:hover:bg-dark-900 dark:hover:text-primary-200;
}

.docs-content {
  @apply min-w-0 flex-1 space-y-6 px-5 py-6 lg:px-10;
}

.docs-hero,
.docs-section {
  @apply scroll-mt-6 rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800;
}

.docs-section h3 {
  @apply text-xl font-semibold text-gray-950 dark:text-white;
}

.docs-section p {
  @apply mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300;
}

.quick-grid {
  @apply mt-6 grid gap-3 lg:grid-cols-3;
}

.quick-item {
  @apply rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900;
}

.quick-item span {
  @apply block text-xs font-medium text-gray-500 dark:text-gray-400;
}

.quick-item code {
  @apply mt-2 block break-all text-sm text-gray-900 dark:text-gray-100;
}

.docs-table {
  @apply mt-5 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700;
}

.docs-table-row {
  @apply grid grid-cols-[140px_1fr] gap-4 border-b border-gray-100 px-4 py-3 text-sm last:border-b-0 dark:border-dark-700;
}

.docs-table-head {
  @apply bg-gray-50 font-semibold text-gray-600 dark:bg-dark-900 dark:text-gray-300;
}

.docs-table code {
  @apply text-primary-700 dark:text-primary-300;
}

.feature-list {
  @apply mt-5 grid gap-3 md:grid-cols-3;
}

.feature-list div {
  @apply flex items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm font-medium text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100;
}

:deep(.endpoint-line) {
  @apply mt-5 flex flex-wrap items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900;
}

:deep(.endpoint-method) {
  @apply rounded bg-primary-600 px-2 py-1 text-xs font-bold text-white;
}

:deep(.endpoint-line code) {
  @apply text-sm text-gray-900 dark:text-gray-100;
}

:deep(.code-block) {
  @apply mt-5 overflow-hidden rounded-lg border border-gray-200 bg-gray-950 dark:border-dark-700;
}

:deep(.code-header) {
  @apply flex items-center justify-between border-b border-white/10 px-4 py-2 text-xs font-medium text-gray-300;
}

:deep(.code-header button) {
  @apply rounded px-2 py-1 text-gray-300 transition hover:bg-white/10 hover:text-white;
}

:deep(.code-block pre) {
  @apply overflow-x-auto p-4 text-sm leading-6 text-gray-100;
}
</style>
