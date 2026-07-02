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
            <h2 class="mt-3 text-3xl font-bold text-gray-950 dark:text-white">把 API 密钥接入统一网关</h2>
            <p class="mt-4 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
              这里整理了最常用的接入方式：创建密钥、配置客户端、发起聊天请求、查看模型列表和排查错误。接口兼容 OpenAI 风格，
              客户端通常只需要把 Base URL 指向当前站点，并在请求头中携带 API Key。
            </p>
          </div>

          <div class="quick-grid">
            <div class="quick-item">
              <span>Base URL</span>
              <code>{{ apiBaseUrl }}</code>
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
          <p>
            先在左侧菜单进入 API 密钥页面，创建一个可用密钥。随后把示例中的
            <code>YOUR_API_KEY</code> 替换为你的真实密钥即可开始调用。
          </p>
          <CodeBlock title="cURL" :code="quickstartCode" />
        </section>

        <section id="client-config" class="docs-section">
          <div class="section-heading-row">
            <div>
              <h3>使用密钥文档</h3>
              <p>
                下面是“使用密钥”中的常用客户端配置说明。文档中统一使用占位密钥
                <code>YOUR_API_KEY</code>，复制后替换成你自己的 API Key。
              </p>
            </div>
          </div>

          <div class="client-tabs" aria-label="客户端类型">
            <button
              v-for="client in clients"
              :key="client.id"
              type="button"
              class="client-tab"
              :class="{ 'client-tab-active': activeClient === client.id }"
              @click="activeClient = client.id"
            >
              <Icon :name="client.icon" size="sm" />
              <span>{{ client.label }}</span>
            </button>
          </div>

          <div class="client-guide">
            <div class="client-summary">
              <Icon :name="activeClientConfig.icon" size="lg" />
              <div>
                <h4>{{ activeClientConfig.label }}</h4>
                <p>{{ activeClientConfig.description }}</p>
              </div>
            </div>

            <div class="space-y-4">
              <CodeBlock
                v-for="file in activeClientFiles"
                :key="file.path"
                :title="file.path"
                :code="file.content"
                :hint="file.hint"
              />
            </div>

            <div class="docs-note">
              <Icon name="infoCircle" size="md" />
              <p>{{ activeClientConfig.note }}</p>
            </div>
          </div>
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
              <span>发送 JSON 请求时使用 application/json</span>
            </div>
          </div>
        </section>

        <section id="chat" class="docs-section">
          <h3>聊天接口</h3>
          <p>用于文本对话、代码问答、内容生成等场景。前端在线聊天窗口也会调用这类接口。</p>
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
          <p>接口错误通常会返回 HTTP 状态码和 JSON 错误信息。客户端应优先展示 <code>error.message</code>。</p>
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
import { computed, defineComponent, h, ref } from 'vue'
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
  | 'terminal'
  | 'sparkles'
  | 'infoCircle'

interface DocSection {
  id: string
  title: string
  icon: IconName
}

interface ClientFile {
  path: string
  content: string
  hint?: string
}

interface ClientGuide {
  id: string
  label: string
  icon: IconName
  description: string
  note: string
  files: () => ClientFile[]
}

const origin = computed(() => window.location.origin.replace(/\/+$/, ''))
const apiBaseUrl = computed(() => `${origin.value}`)
const apiBaseV1 = computed(() => `${origin.value}/v1`)
const placeholderKey = 'YOUR_API_KEY'
const activeClient = ref('codex')

const sections: DocSection[] = [
  { id: 'overview', title: '概览', icon: 'book' },
  { id: 'quickstart', title: '快速开始', icon: 'play' },
  { id: 'client-config', title: '客户端配置', icon: 'terminal' },
  { id: 'auth', title: '认证', icon: 'lock' },
  { id: 'chat', title: '聊天接口', icon: 'chat' },
  { id: 'models', title: '模型列表', icon: 'cpu' },
  { id: 'keys', title: '密钥与用量', icon: 'key' },
  { id: 'errors', title: '错误处理', icon: 'exclamationCircle' }
]

const quickstartCode = computed(() => `curl ${apiBaseV1.value}/chat/completions \\
  -H "Authorization: Bearer ${placeholderKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.5",
    "messages": [
      { "role": "user", "content": "你好，请介绍一下你自己" }
    ]
  }'`)

const chatBodyCode = `{
  "model": "gpt-5.5",
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
  "model": "gpt-5.5",
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

const modelsCode = computed(() => `curl ${apiBaseV1.value}/models \\
  -H "Authorization: Bearer ${placeholderKey}"`)

const clients = computed<ClientGuide[]>(() => [
  {
    id: 'codex',
    label: 'Codex CLI',
    icon: 'terminal',
    description: '适用于 OpenAI / Responses 兼容分组。将 Codex 的 provider 指向当前网关，并在 auth.json 中保存 API Key。',
    note: '保存后重新启动 Codex CLI；如果你的站点使用自定义域名，请确认 Base URL 与页面顶部展示的一致。',
    files: () => {
      const configDir = '~/.codex'
      return [
        {
          path: `${configDir}/config.toml`,
          hint: '请确保以下内容位于 config.toml 文件的开头部分',
          content: `model_provider = "OpenAI"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${apiBaseUrl.value}"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true`
        },
        {
          path: `${configDir}/auth.json`,
          content: `{
  "OPENAI_API_KEY": "${placeholderKey}"
}`
        }
      ]
    }
  },
  {
    id: 'claude',
    label: 'Claude Code',
    icon: 'terminal',
    description: '适用于 Anthropic / Claude 兼容分组。通过环境变量指定网关地址和 API Key。',
    note: '如果使用 Antigravity 的 Claude 分组，请把 ANTHROPIC_BASE_URL 改为当前站点的 /antigravity 地址。',
    files: () => [
      {
        path: 'macOS / Linux',
        content: `export ANTHROPIC_BASE_URL="${apiBaseUrl.value}"
export ANTHROPIC_AUTH_TOKEN="${placeholderKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
export CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      },
      {
        path: 'PowerShell',
        content: `$env:ANTHROPIC_BASE_URL="${apiBaseUrl.value}"
$env:ANTHROPIC_AUTH_TOKEN="${placeholderKey}"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
$env:CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      },
      {
        path: '~/.claude/settings.json',
        content: `{
  "env": {
    "ANTHROPIC_BASE_URL": "${apiBaseUrl.value}",
    "ANTHROPIC_AUTH_TOKEN": "${placeholderKey}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}`
      }
    ]
  },
  {
    id: 'gemini',
    label: 'Gemini CLI',
    icon: 'sparkles',
    description: '适用于 Gemini 兼容分组。将 Gemini CLI 的 Base URL、Key 和默认模型写入环境变量。',
    note: '不同分组可用模型可能不同；如调用失败，请先在后台渠道或可用模型页面确认模型名称。',
    files: () => [
      {
        path: 'macOS / Linux',
        content: `export GOOGLE_GEMINI_BASE_URL="${apiBaseUrl.value}"
export GEMINI_API_KEY="${placeholderKey}"
export GEMINI_MODEL="gemini-2.0-flash"  # 如果你有 Gemini 3 权限可以填：gemini-3-pro-preview`
      },
      {
        path: 'PowerShell',
        content: `$env:GOOGLE_GEMINI_BASE_URL="${apiBaseUrl.value}"
$env:GEMINI_API_KEY="${placeholderKey}"
$env:GEMINI_MODEL="gemini-2.0-flash"  # 如果你有 Gemini 3 权限可以填：gemini-3-pro-preview`
      }
    ]
  },
  {
    id: 'antigravity',
    label: 'Antigravity',
    icon: 'sparkles',
    description: '适用于 Antigravity 分组，可按客户端选择 Claude 或 Gemini 兼容入口。',
    note: 'Claude 与 Gemini 入口都使用 /antigravity 前缀，请按所用客户端选择对应配置。',
    files: () => [
      {
        path: 'Claude Code',
        content: `export ANTHROPIC_BASE_URL="${apiBaseUrl.value}/antigravity"
export ANTHROPIC_AUTH_TOKEN="${placeholderKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
export CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      },
      {
        path: 'Gemini CLI',
        content: `export GOOGLE_GEMINI_BASE_URL="${apiBaseUrl.value}/antigravity"
export GEMINI_API_KEY="${placeholderKey}"
export GEMINI_MODEL="gemini-2.0-flash"  # 如果你有 Gemini 3 权限可以填：gemini-3-pro-preview`
      }
    ]
  },
  {
    id: 'opencode',
    label: 'OpenCode',
    icon: 'terminal',
    description: '适用于需要配置 provider 的 OpenCode 客户端。示例展示 OpenAI 兼容 provider。',
    note: '如果使用 Anthropic、Gemini 或 Antigravity 分组，把 provider 名称、npm 包和模型列表调整为对应平台即可。',
    files: () => {
      const config = {
        provider: {
          openai: {
            options: {
              baseURL: apiBaseV1.value,
              apiKey: placeholderKey
            },
            models: {
              'gpt-5.2': {
                name: 'GPT-5.2',
                limit: { context: 400000, output: 128000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'gpt-5.5': {
                name: 'GPT-5.5',
                limit: { context: 1050000, output: 128000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'gpt-5.4': {
                name: 'GPT-5.4',
                limit: { context: 1050000, output: 128000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'gpt-5.4-mini': {
                name: 'GPT-5.4 Mini',
                limit: { context: 400000, output: 128000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'gpt-5.3-codex-spark': {
                name: 'GPT-5.3 Codex Spark',
                limit: { context: 128000, output: 32000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'gpt-5.3-codex': {
                name: 'GPT-5.3 Codex',
                limit: { context: 400000, output: 128000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {}, xhigh: {} }
              },
              'codex-mini-latest': {
                name: 'Codex Mini',
                limit: { context: 200000, output: 100000 },
                options: { store: false },
                variants: { low: {}, medium: {}, high: {} }
              }
            }
          }
        },
        agent: {
          build: { options: { store: false } },
          plan: { options: { store: false } }
        },
        $schema: 'https://opencode.ai/config.json'
      }
      return [
        {
          path: 'opencode.json',
          hint: '配置文件路径：~/.config/opencode/opencode.json（或 opencode.jsonc），不存在需手动创建。可使用默认 provider（openai/anthropic/google）或自定义 provider_id。API Key 支持直接配置或通过客户端 /connect 命令配置。示例仅供参考，模型与选项可按需调整。',
          content: JSON.stringify(config, null, 2)
        }
      ]
    }
  }
])

const activeClientConfig = computed(() => clients.value.find((client) => client.id === activeClient.value) ?? clients.value[0])
const activeClientFiles = computed(() => activeClientConfig.value.files())

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
    code: { type: String, required: true },
    hint: { type: String, default: '' }
  },
  setup(props) {
    const copy = async () => {
      await navigator.clipboard?.writeText(props.code)
    }

    return () => h('div', { class: 'code-block' }, [
      h('div', { class: 'code-header' }, [
        h('div', { class: 'min-w-0' }, [
          h('span', { class: 'block truncate' }, props.title),
          props.hint ? h('p', { class: 'mt-1 text-[11px] font-normal text-amber-200' }, props.hint) : null
        ]),
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

.docs-section code {
  @apply rounded bg-gray-100 px-1.5 py-0.5 text-primary-700 dark:bg-dark-900 dark:text-primary-300;
}

.section-heading-row {
  @apply flex flex-col gap-4 md:flex-row md:items-start md:justify-between;
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
  @apply mt-2 block break-all rounded-none bg-transparent p-0 text-sm text-gray-900 dark:text-gray-100;
}

.client-tabs {
  @apply mt-5 flex flex-wrap gap-2 border-b border-gray-200 pb-3 dark:border-dark-700;
}

.client-tab {
  @apply inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-gray-600 transition hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-900 dark:hover:text-white;
}

.client-tab-active {
  @apply bg-primary-50 text-primary-700 ring-1 ring-primary-200 dark:bg-primary-900/20 dark:text-primary-200 dark:ring-primary-800;
}

.client-guide {
  @apply mt-5 space-y-5;
}

.client-summary {
  @apply flex items-start gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900;
}

.client-summary h4 {
  @apply text-base font-semibold text-gray-950 dark:text-white;
}

.client-summary p {
  @apply mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300;
}

.docs-note {
  @apply flex items-start gap-3 rounded-lg border border-blue-100 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300;
}

.docs-note p {
  @apply mt-0 text-blue-700 dark:text-blue-300;
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
  @apply rounded-none bg-transparent p-0 text-primary-700 dark:text-primary-300;
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
  @apply flex items-center justify-between gap-4 border-b border-white/10 px-4 py-2 text-xs font-medium text-gray-300;
}

:deep(.code-header button) {
  @apply shrink-0 rounded px-2 py-1 text-gray-300 transition hover:bg-white/10 hover:text-white;
}

:deep(.code-block pre) {
  @apply overflow-x-auto p-4 text-sm leading-6 text-gray-100;
}
</style>
