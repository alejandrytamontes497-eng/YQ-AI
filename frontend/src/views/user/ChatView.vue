<template>
  <AppLayout>
    <div class="chat-shell">
      <aside v-show="!sidebarCollapsed" class="chat-sidebar">
        <div class="flex items-center gap-2 px-5 pt-5">
          <h1 class="text-xl font-semibold text-gray-950 dark:text-white">在线聊天</h1>
          <button class="icon-button" type="button" title="刷新模型" :disabled="loading" @click="loadChatData">
            <Icon name="refresh" size="sm" />
          </button>
          <button class="icon-button" type="button" title="收起侧边栏" @click="setSidebarCollapsed(true)">
            <Icon name="chevronLeft" size="sm" />
          </button>
        </div>

        <div class="px-4 pt-4">
          <button class="btn btn-primary w-full justify-center" type="button" @click="startNewChat">
            <Icon name="plus" size="sm" class="mr-2" />
            新对话
          </button>
        </div>

        <div class="px-4 pt-3">
          <div class="relative">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="conversationSearch"
              class="h-10 w-full rounded-lg border border-gray-200 bg-white pl-9 pr-3 text-sm outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 dark:border-dark-700 dark:bg-dark-900 dark:text-white"
              placeholder="搜索聊天记录"
            />
          </div>
        </div>

        <div class="min-h-0 flex-1 space-y-2 overflow-y-auto px-3 py-4">
          <div
            v-for="conversation in filteredConversations"
            :key="conversation.id"
            class="conversation-item"
            :class="{ 'conversation-item-active': conversation.id === activeConversationId }"
          >
            <button type="button" class="min-w-0 flex flex-1 gap-3 text-left" @click="selectConversation(conversation.id)">
              <span class="conversation-icon">
                <Icon name="chat" size="sm" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">
                  {{ conversation.title }}
                </span>
                <span class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ conversation.preview || '暂无消息' }}
                </span>
                <span class="mt-1 block truncate text-xs text-primary-600 dark:text-primary-300">
                  {{ conversation.model || selectedModelName || '未选择模型' }} · {{ conversation.updatedAt }}
                </span>
              </span>
            </button>
            <button class="conversation-delete" type="button" title="删除对话" :disabled="sending" @click.stop="deleteConversation(conversation.id)">
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>
      </aside>

      <section class="chat-main" :class="{ 'chat-main-sidebar-collapsed': sidebarCollapsed }">
        <button
          v-if="sidebarCollapsed"
          class="sidebar-expand-button"
          type="button"
          title="展开侧边栏"
          @click="setSidebarCollapsed(false)"
        >
          <Icon name="menu" size="sm" />
        </button>
        <header class="chat-header">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <h2 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                {{ activeConversation?.title || '新对话' }}
              </h2>
              <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
                {{ platformLabel(selectedModelOption?.platform) }}
              </span>
            </div>
            <p class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">
              {{ selectedModelName || '请选择模型' }}
            </p>
          </div>

          <div class="grid min-w-0 flex-1 gap-3 md:grid-cols-[minmax(220px,1fr)_auto]">
            <Select
              v-if="modelOptions.length > 0"
              v-model="selectedModel"
              :options="modelOptions"
              :disabled="sending || loading || modelOptions.length <= 1"
              searchable
              placeholder="选择号池可用模型"
              @change="onModelSelect"
            />
            <Input
              v-else
              v-model="selectedModel"
              disabled
              :placeholder="loading ? '正在加载模型' : '暂无可用模型'"
            />

            <button class="icon-button h-10 w-10" type="button" title="重置当前对话" :disabled="sending" @click="resetActiveConversation">
              <Icon name="refresh" size="sm" />
            </button>
          </div>
        </header>

        <div ref="messagesRef" class="chat-messages">
          <div v-if="messages.length === 0" class="welcome-panel">
            <div class="mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-500 text-white shadow-lg shadow-primary-500/25">
              <Icon name="sparkles" size="lg" />
            </div>
            <h3 class="text-2xl font-bold text-gray-950 dark:text-white">今天想聊点什么？</h3>
            <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
              选择号池可用模型后直接发送消息，费用从账户余额中扣除。
            </p>
            <div class="mt-8 grid gap-3 md:grid-cols-3">
              <button
                v-for="prompt in starterPrompts"
                :key="prompt"
                class="starter-prompt"
                type="button"
                @click="useStarterPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>
          </div>

          <div
            v-for="message in messages"
            :key="message.id"
            class="message-row"
            :class="message.role === 'user' ? 'message-row-user' : 'message-row-assistant'"
          >
            <div class="message-avatar" :class="message.role === 'user' ? 'message-avatar-user' : 'message-avatar-assistant'">
              <Icon :name="message.role === 'user' ? 'user' : 'sparkles'" size="sm" />
            </div>
            <div class="message-bubble" :class="message.role === 'user' ? 'message-bubble-user' : 'message-bubble-assistant'">
              <button
                v-if="message.content"
                class="message-copy-button"
                type="button"
                :title="copiedTarget === `${message.id}:full` ? '已复制' : '复制全文'"
                @click="copyMessageContent(message)"
              >
                <Icon :name="copiedTarget === `${message.id}:full` ? 'check' : 'copy'" size="xs" :stroke-width="2" />
              </button>
              <div class="message-content whitespace-pre-wrap break-words">{{ message.content || '正在生成回复...' }}</div>
              <div v-if="copySegmentsForMessage(message).length > 0" class="message-segment-actions">
                <button
                  v-for="segment in copySegmentsForMessage(message)"
                  :key="segment.id"
                  class="message-segment-copy"
                  type="button"
                  :title="copiedTarget === segment.id ? '已复制' : segment.title"
                  @click="copySegment(segment)"
                >
                  <Icon :name="copiedTarget === segment.id ? 'check' : 'copy'" size="xs" :stroke-width="2" />
                  <span>{{ copiedTarget === segment.id ? '已复制' : segment.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="errorMessage" class="mx-auto w-full max-w-5xl px-5 pb-3">
          <div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
            {{ errorMessage }}
          </div>
        </div>

        <form class="chat-composer" @submit.prevent="sendMessage">
          <div class="composer-inner">
            <textarea
              v-model="draft"
              :disabled="sending"
              rows="1"
              placeholder="发送消息"
              @keydown.enter.exact.prevent="sendMessage"
            />
            <button class="send-button" type="submit" title="发送" :disabled="!canSend">
              <Icon name="arrowUp" size="md" :stroke-width="2" />
            </button>
          </div>
          <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-gray-400">
            <span>{{ selectedModelName ? `账户余额计费 · ${selectedModelName}` : '请选择号池可用模型' }}</span>
            <span v-if="lastUsage">本次 {{ lastUsage.total_tokens ?? 0 }} tokens</span>
          </div>
        </form>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Icon from '@/components/icons/Icon.vue'
import { keysAPI } from '@/api/keys'
import { chatAPI, type ChatCompletionUsage, type ChatMessage, type UserChatModel } from '@/api/chat'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'

interface UiMessage extends ChatMessage {
  id: string
}

interface Conversation {
  id: string
  title: string
  preview: string
  model: string
  updatedAt: string
  messages: UiMessage[]
}

interface ChatModelOption extends SelectOption {
  value: string
  label: string
  model: string
  platform: string
  groupIds: number[]
  keyIds: number[]
}

interface CopySegment {
  id: string
  label: string
  title: string
  content: string
  type: 'code' | 'xml'
}

interface PersistedChatState {
  activeConversationId: string
  selectedModel: string
  conversations: Conversation[]
}

const MAX_CONVERSATIONS = 100
const MAX_LOCAL_MESSAGES = 100
const MAX_CONTEXT_MESSAGES = 10
const CHAT_HISTORY_STORAGE_PREFIX = 'chat_history_v1'
const CHAT_SIDEBAR_COLLAPSED_KEY = 'chat_sidebar_collapsed'
const STREAM_RENDER_MAX_CHARS_PER_FRAME = 160

const starterPrompts = [
  '帮我把这件事拆成三步执行计划',
  '把这段内容改得更清晰',
  '写一段可以直接发给客户的说明'
]

const apiKeys = ref<ApiKey[]>([])
const channelModels = ref<ChatModelOption[]>([])
const availableModelNamesByKeyId = ref<Map<number, Set<string>>>(new Map())
const selectedModel = ref('')
const conversations = ref<Conversation[]>([createConversation('新对话')])
const activeConversationId = ref(conversations.value[0].id)
const conversationSearch = ref('')
const draft = ref('')
const loading = ref(false)
const sending = ref(false)
const errorMessage = ref('')
const lastUsage = ref<ChatCompletionUsage | null>(null)
const messagesRef = ref<HTMLElement | null>(null)
const copiedTarget = ref('')
const sidebarCollapsed = ref(loadSidebarCollapsed())
let abortController: AbortController | null = null
let streamRenderFrame: number | null = null
let streamScrollFrame: number | null = null
let streamAssistantMessage: UiMessage | null = null
let streamDeltaBuffer = ''
let streamDrainResolvers: Array<() => void> = []
let copyFeedbackTimer: number | null = null
const authStore = useAuthStore()
let restoringHistory = true

const activeConversation = computed(() =>
  conversations.value.find((item) => item.id === activeConversationId.value) ?? null
)

const messages = computed(() => activeConversation.value?.messages ?? [])
const activeKeys = computed(() => apiKeys.value.filter((item) => item.status === 'active'))

const filteredConversations = computed(() => {
  const query = conversationSearch.value.trim().toLowerCase()
  if (!query) return conversations.value
  return conversations.value.filter((item) =>
    [item.title, item.preview, item.model].some((value) => value.toLowerCase().includes(query))
  )
})

const modelOptions = computed<ChatModelOption[]>(() => channelModels.value)
const selectedModelOption = computed(() =>
  modelOptions.value.find((item) => item.value === selectedModel.value) ?? null
)
const selectedModelName = computed(() => selectedModelOption.value?.model ?? '')

const selectedKey = computed(() => selectKeyForModel(selectedModelOption.value))

const canSend = computed(() =>
  Boolean(selectedKey.value?.key && hasExactKeyForModel(selectedModelOption.value) && draft.value.trim() && !sending.value)
)

function createConversation(title: string): Conversation {
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    title,
    preview: '',
    model: '',
    updatedAt: formatTime(new Date()),
    messages: []
  }
}

function storageKey(): string {
  const userID = authStore.user?.id ?? storedUserID() ?? 'anonymous'
  return `${CHAT_HISTORY_STORAGE_PREFIX}:${userID}`
}

function storedUserID(): number | string | null {
  try {
    const raw = localStorage.getItem('auth_user')
    if (!raw) return null
    const parsed = JSON.parse(raw) as { id?: number | string } | null
    return parsed?.id ?? null
  } catch {
    return null
  }
}

function loadSidebarCollapsed(): boolean {
  try {
    return localStorage.getItem(CHAT_SIDEBAR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

function setSidebarCollapsed(collapsed: boolean) {
  sidebarCollapsed.value = collapsed
  try {
    localStorage.setItem(CHAT_SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // Ignore storage failures; the current UI state still updates.
  }
}

function sanitizeConversation(conversation: Partial<Conversation> | null | undefined): Conversation | null {
  if (!conversation || typeof conversation !== 'object') return null

  const messages = Array.isArray(conversation.messages)
    ? conversation.messages
        .filter((message): message is UiMessage =>
          message &&
          (message.role === 'system' || message.role === 'user' || message.role === 'assistant') &&
          typeof message.content === 'string'
        )
        .slice(-MAX_LOCAL_MESSAGES)
        .map((message) => ({
          id: typeof message.id === 'string' && message.id ? message.id : `${Date.now()}-${Math.random().toString(36).slice(2)}`,
          role: message.role,
          content: message.content
        }))
    : []

  return {
    id: typeof conversation.id === 'string' && conversation.id ? conversation.id : `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    title: typeof conversation.title === 'string' && conversation.title ? conversation.title : '新对话',
    preview: typeof conversation.preview === 'string' ? conversation.preview : '',
    model: typeof conversation.model === 'string' ? conversation.model : '',
    updatedAt: typeof conversation.updatedAt === 'string' && conversation.updatedAt ? conversation.updatedAt : formatTime(new Date()),
    messages
  }
}

function normalizeConversations(items: unknown): Conversation[] {
  if (!Array.isArray(items)) return [createConversation('新对话')]

  const normalized = items
    .map((item) => sanitizeConversation(item as Partial<Conversation>))
    .filter((item): item is Conversation => item !== null)
    .slice(0, MAX_CONVERSATIONS)

  return normalized.length > 0 ? normalized : [createConversation('新对话')]
}

function loadStoredChatHistory() {
  restoringHistory = true
  try {
    const raw = localStorage.getItem(storageKey())
    if (!raw) return

    const parsed = JSON.parse(raw) as Partial<PersistedChatState>
    const restoredConversations = normalizeConversations(parsed.conversations)
    conversations.value = restoredConversations
    activeConversationId.value = restoredConversations.some((item) => item.id === parsed.activeConversationId)
      ? String(parsed.activeConversationId)
      : restoredConversations[0].id
    selectedModel.value = typeof parsed.selectedModel === 'string' ? parsed.selectedModel : ''
  } catch {
    localStorage.removeItem(storageKey())
  } finally {
    restoringHistory = false
  }
}

function persistChatHistory() {
  if (restoringHistory) return

  const normalized = normalizeConversations(conversations.value)
  const state: PersistedChatState = {
    activeConversationId: normalized.some((item) => item.id === activeConversationId.value)
      ? activeConversationId.value
      : normalized[0].id,
    selectedModel: selectedModel.value,
    conversations: normalized
  }

  try {
    localStorage.setItem(storageKey(), JSON.stringify(state))
  } catch {
    // Ignore storage quota/private-mode failures; the live chat should keep working.
  }
}

function platformLabel(platform?: string): string {
  const labels: Record<string, string> = {
    anthropic: 'Claude',
    openai: 'OpenAI',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    grok: 'Grok'
  }
  return platform ? labels[platform] || platform : '通用'
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function onModelSelect(value: string | number | boolean | null) {
  selectedModel.value = String(value ?? '')
}

function startNewChat() {
  const conversation = createConversation('新对话')
  conversation.model = selectedModelName.value
  conversations.value.unshift(conversation)
  trimConversations()
  activeConversationId.value = conversation.id
  draft.value = ''
  errorMessage.value = ''
  lastUsage.value = null
  persistChatHistory()
}

function selectConversation(id: string) {
  activeConversationId.value = id
  errorMessage.value = ''
  persistChatHistory()
  void nextTick(scrollToBottom)
}

function deleteConversation(id: string) {
  const index = conversations.value.findIndex((item) => item.id === id)
  if (index < 0) return

  conversations.value.splice(index, 1)
  if (conversations.value.length === 0) {
    conversations.value.push(createConversation('新对话'))
  }
  if (activeConversationId.value === id) {
    activeConversationId.value = conversations.value[Math.min(index, conversations.value.length - 1)].id
  }
  errorMessage.value = ''
  lastUsage.value = null
  persistChatHistory()
  void nextTick(scrollToBottom)
}

function trimConversations() {
  if (conversations.value.length <= MAX_CONVERSATIONS) return
  conversations.value.splice(MAX_CONVERSATIONS)
  if (!conversations.value.some((item) => item.id === activeConversationId.value)) {
    activeConversationId.value = conversations.value[0].id
  }
}

function resetActiveConversation() {
  if (!activeConversation.value) return
  activeConversation.value.messages = []
  activeConversation.value.title = '新对话'
  activeConversation.value.preview = ''
  activeConversation.value.updatedAt = formatTime(new Date())
  errorMessage.value = ''
  lastUsage.value = null
  persistChatHistory()
}

function useStarterPrompt(prompt: string) {
  draft.value = prompt
}

async function copyMessageContent(message: UiMessage) {
  if (!message.content) return
  await copyText(message.content, `${message.id}:full`)
}

async function copySegment(segment: CopySegment) {
  await copyText(segment.content, segment.id)
}

async function copyText(text: string, target: string) {
  const value = text.trim()
  if (!value) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
    } else {
      fallbackCopyText(value)
    }
    showCopied(target)
  } catch {
    fallbackCopyText(value)
    showCopied(target)
  }
}

function fallbackCopyText(text: string) {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

function showCopied(target: string) {
  copiedTarget.value = target
  if (copyFeedbackTimer !== null) {
    window.clearTimeout(copyFeedbackTimer)
  }
  copyFeedbackTimer = window.setTimeout(() => {
    copiedTarget.value = ''
    copyFeedbackTimer = null
  }, 1400)
}

function copySegmentsForMessage(message: UiMessage): CopySegment[] {
  return extractCopySegments(message.content).map((segment, index) => ({
    ...segment,
    id: `${message.id}:segment:${index}`
  }))
}

function extractCopySegments(content: string): Omit<CopySegment, 'id'>[] {
  const text = content.trim()
  if (!text) return []

  const segments: Omit<CopySegment, 'id'>[] = []
  const ranges: Array<[number, number]> = []
  const fencedPattern = /```([^\n`]*)\n?([\s\S]*?)```/g
  let fencedMatch: RegExpExecArray | null
  while ((fencedMatch = fencedPattern.exec(content)) !== null) {
    const language = fencedMatch[1]?.trim()
    const body = fencedMatch[2]?.replace(/^\n|\n$/g, '').trim()
    if (!body) continue
    const type = isXmlText(body) || language?.toLowerCase().includes('xml') ? 'xml' : 'code'
    segments.push({
      type,
      label: type === 'xml' ? `复制 XML ${segments.length + 1}` : `复制代码 ${segments.length + 1}`,
      title: type === 'xml' ? '复制该 XML 片段' : '复制该代码片段',
      content: body
    })
    ranges.push([fencedMatch.index, fencedMatch.index + fencedMatch[0].length])
  }

  const xmlPattern = /(?:<\?xml[\s\S]*?\?>\s*)?<([A-Za-z_][\w:.-]*)(?:\s[^<>]*)?>[\s\S]*?<\/\1>/g
  let xmlMatch: RegExpExecArray | null
  while ((xmlMatch = xmlPattern.exec(content)) !== null) {
    const start = xmlMatch.index
    const end = start + xmlMatch[0].length
    if (ranges.some(([from, to]) => start >= from && end <= to)) continue
    const xml = xmlMatch[0].trim()
    if (!isXmlText(xml)) continue
    segments.push({
      type: 'xml',
      label: `复制 XML ${segments.length + 1}`,
      title: '复制该 XML 片段',
      content: xml
    })
  }

  if (segments.length === 0 && looksLikeStandaloneCode(text)) {
    segments.push({
      type: isXmlText(text) ? 'xml' : 'code',
      label: isXmlText(text) ? '复制 XML 1' : '复制代码 1',
      title: isXmlText(text) ? '复制该 XML 片段' : '复制该代码片段',
      content: text
    })
  }

  return segments
}

function isXmlText(text: string): boolean {
  const value = text.trim()
  return /^<\?xml[\s\S]*\?>/.test(value) || /^<([A-Za-z_][\w:.-]*)(?:\s[^<>]*)?>[\s\S]*<\/\1>$/.test(value)
}

function looksLikeStandaloneCode(text: string): boolean {
  const lines = text.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
  if (lines.length < 2) return false
  if (isXmlText(text)) return true
  const codeLineCount = lines.filter((line) =>
    /[{};=<>]/.test(line) ||
    /^(import|export|const|let|var|function|class|interface|type|def|async|await|return|if|for|while|try|catch|package|func)\b/.test(line)
  ).length
  return codeLineCount >= Math.min(3, lines.length)
}

function toGatewayMessages(): ChatMessage[] {
  return messages.value
    .slice(-MAX_CONTEXT_MESSAGES)
    .map((message) => ({
      role: message.role,
      content: message.content
    }))
}

function addMessage(role: ChatMessage['role'], content: string) {
  if (!activeConversation.value) return null

  const message: UiMessage = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    role,
    content
  }
  activeConversation.value.messages.push(message)
  trimActiveConversationMessages()
  touchConversation(content, role)
  return message
}

function trimActiveConversationMessages() {
  if (!activeConversation.value) return
  if (activeConversation.value.messages.length <= MAX_LOCAL_MESSAGES) return
  activeConversation.value.messages.splice(0, activeConversation.value.messages.length - MAX_LOCAL_MESSAGES)
}

function touchConversation(content: string, role: ChatMessage['role'], options: { persist?: boolean } = {}) {
  if (!activeConversation.value) return
  activeConversation.value.preview = content || activeConversation.value.preview
  activeConversation.value.model = selectedModelName.value
  activeConversation.value.updatedAt = formatTime(new Date())
  if (role === 'user' && activeConversation.value.title === '新对话') {
    activeConversation.value.title = content.slice(0, 18) || '新对话'
  }
  if (options.persist !== false) {
    persistChatHistory()
  }
}

function appendAssistantMessage(message: UiMessage, delta: string) {
  if (!delta) return
  streamAssistantMessage = message
  streamDeltaBuffer += delta
  scheduleStreamRender()
}

function scheduleStreamRender() {
  if (streamRenderFrame !== null) return
  streamRenderFrame = window.requestAnimationFrame(() => {
    streamRenderFrame = null
    flushStreamDelta()
  })
}

function streamFrameSize(bufferLength: number): number {
  if (bufferLength > 1200) return STREAM_RENDER_MAX_CHARS_PER_FRAME
  if (bufferLength > 500) return 96
  if (bufferLength > 180) return 48
  if (bufferLength > 60) return 24
  if (bufferLength > 20) return 12
  return Math.max(1, Math.ceil(bufferLength / 2))
}

function flushStreamDelta(options: { all?: boolean } = {}) {
  if (!streamAssistantMessage || !streamDeltaBuffer) return

  const take = options.all
    ? streamDeltaBuffer.length
    : Math.min(streamDeltaBuffer.length, streamFrameSize(streamDeltaBuffer.length))
  streamAssistantMessage.content += streamDeltaBuffer.slice(0, take)
  streamDeltaBuffer = streamDeltaBuffer.slice(take)
  touchConversation(streamAssistantMessage.content, 'assistant', { persist: false })
  queueScrollToBottom()

  if (streamDeltaBuffer) {
    scheduleStreamRender()
  } else {
    resolveStreamDrain()
  }
}

function flushStreamDeltaNow() {
  if (streamRenderFrame !== null) {
    window.cancelAnimationFrame(streamRenderFrame)
    streamRenderFrame = null
  }
  flushStreamDelta({ all: true })
}

function waitForStreamDrain(): Promise<void> {
  if (!streamDeltaBuffer) return Promise.resolve()
  scheduleStreamRender()
  return new Promise((resolve) => {
    streamDrainResolvers.push(resolve)
  })
}

function resolveStreamDrain() {
  if (streamDeltaBuffer || streamDrainResolvers.length === 0) return
  const resolvers = streamDrainResolvers
  streamDrainResolvers = []
  resolvers.forEach((resolve) => resolve())
}

function resetStreamRenderer() {
  if (streamRenderFrame !== null) {
    window.cancelAnimationFrame(streamRenderFrame)
    streamRenderFrame = null
  }
  streamAssistantMessage = null
  streamDeltaBuffer = ''
  resolveStreamDrain()
}

function parsePositiveNumber(value: string, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

async function sendMessage() {
  const requestOption = selectedModelOption.value
  const requestKey = selectedKey.value

  if (!canSend.value || !requestKey) {
    if (!requestKey) {
      errorMessage.value = '当前模型没有匹配到可用 API Key，请先在密钥页创建一个可访问该分组的 Key。'
    }
    return
  }

  const requestModel = requestOption?.model.trim()
  if (!requestModel || !keyHasExactModel(requestKey, requestModel, requestOption)) {
    errorMessage.value = '当前模型不在号池返回的可用模型列表中，请刷新后重新选择。'
    return
  }

  const content = draft.value.trim()
  draft.value = ''
  errorMessage.value = ''
  lastUsage.value = null
  addMessage('user', content)
  const requestMessages = toGatewayMessages()
  const assistantMessage = addMessage('assistant', '')
  await scrollToBottom()

  if (!assistantMessage) return

  resetStreamRenderer()
  streamAssistantMessage = assistantMessage
  abortController = new AbortController()
  sending.value = true

  try {
    const usage = await chatAPI.createChatCompletionStream({
      apiKey: requestKey.key,
      model: requestModel,
      messages: requestMessages,
      temperature: 0.7,
      max_tokens: Math.round(parsePositiveNumber('2048', 2048)),
      signal: abortController.signal
    }, {
      onDelta: (delta) => appendAssistantMessage(assistantMessage, delta),
      onUsage: (nextUsage) => {
        lastUsage.value = nextUsage
      }
    })

    await waitForStreamDrain()
    if (!assistantMessage.content.trim()) {
      assistantMessage.content = '模型没有返回可显示的内容。'
    }
    touchConversation(assistantMessage.content, 'assistant')
    lastUsage.value = usage ?? lastUsage.value
    await refreshSelectedKey()
  } catch (error) {
    flushStreamDeltaNow()
    const message = error instanceof Error ? error.message : '发送失败，请稍后重试。'
    if (assistantMessage.content.trim()) {
      touchConversation(assistantMessage.content, 'assistant')
      return
    }
    errorMessage.value = message
    assistantMessage.content = `请求失败：${message}`
    touchConversation(assistantMessage.content, 'assistant')
  } finally {
    sending.value = false
    abortController = null
    resetStreamRenderer()
    await scrollToBottom()
  }
}

async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

function queueScrollToBottom() {
  if (streamScrollFrame !== null) return
  streamScrollFrame = window.requestAnimationFrame(() => {
    streamScrollFrame = null
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
}

function loadModelsFromUserChatModels(models: UserChatModel[], keys: ApiKey[]): ChatModelOption[] {
  const byKey = new Map<string, ChatModelOption>()
  const namesByKeyId = new Map<number, Set<string>>()

  for (const rawModel of models) {
    const name = rawModel.name.trim()
    const platform = rawModel.platform || 'unknown'
    const groupIds = rawModel.group_ids.filter((id) => typeof id === 'number')
    if (!name || groupIds.length === 0) continue

    const matchingKeyIds = keys
      .filter((key) => typeof key.group_id === 'number' && groupIds.includes(key.group_id))
      .map((key) => {
        const keyModelNames = namesByKeyId.get(key.id) ?? new Set<string>()
        keyModelNames.add(name)
        namesByKeyId.set(key.id, keyModelNames)
        return key.id
      })

    const optionKey = `${platform}:${name}`
    const existing = byKey.get(optionKey)
    if (existing) {
      existing.groupIds = Array.from(new Set([...existing.groupIds, ...groupIds]))
      existing.keyIds = Array.from(new Set([...existing.keyIds, ...matchingKeyIds]))
      continue
    }

    byKey.set(optionKey, {
      value: optionKey,
      label: `${name} 路 ${platformLabel(platform)}`,
      model: name,
      platform,
      groupIds,
      keyIds: matchingKeyIds
    })
  }

  availableModelNamesByKeyId.value = namesByKeyId

  return Array.from(byKey.values())
}

async function loadModelsFromKeys(keys: ApiKey[]): Promise<ChatModelOption[]> {
  const models = await chatAPI.listUserChatModels()
  const options = loadModelsFromUserChatModels(models, keys)

  if (options.length > 0) {
    return options
  }

  const byKey = new Map<string, ChatModelOption>()
  const namesByKeyId = new Map<number, Set<string>>()

  const results = await Promise.allSettled(
    keys.map(async (key) => ({
      key,
      models: await chatAPI.listModels(key.key)
    }))
  )

  for (const result of results) {
    if (result.status !== 'fulfilled') continue

    const apiKey = result.value.key
    const platform = apiKey.group?.platform || 'unknown'
    const groupIds = typeof apiKey.group_id === 'number' ? [apiKey.group_id] : []

    for (const rawName of result.value.models) {
      const name = rawName.trim()
      if (!name) continue

      const keyModelNames = namesByKeyId.get(apiKey.id) ?? new Set<string>()
      keyModelNames.add(name)
      namesByKeyId.set(apiKey.id, keyModelNames)

      const optionKey = `${platform}:${name}`
      const existing = byKey.get(optionKey)
      if (existing) {
        existing.groupIds = Array.from(new Set([...existing.groupIds, ...groupIds]))
        existing.keyIds = Array.from(new Set([...existing.keyIds, apiKey.id]))
        continue
      }

      byKey.set(optionKey, {
        value: optionKey,
        label: `${name} · ${platformLabel(platform)}`,
        model: name,
        platform,
        groupIds,
        keyIds: [apiKey.id]
      })
    }
  }

  availableModelNamesByKeyId.value = namesByKeyId

  return Array.from(byKey.values())
}

function mergeModelOptions(...sources: ChatModelOption[][]): ChatModelOption[] {
  const byKey = new Map<string, ChatModelOption>()

  for (const options of sources) {
    for (const option of options) {
      const existing = byKey.get(option.value)
      if (existing) {
        existing.groupIds = Array.from(new Set([...existing.groupIds, ...option.groupIds]))
        existing.keyIds = Array.from(new Set([...existing.keyIds, ...option.keyIds]))
        continue
      }
      byKey.set(option.value, { ...option })
    }
  }

  return Array.from(byKey.values()).sort((a, b) => a.label.localeCompare(b.label))
}

function selectKeyForModel(option: ChatModelOption | null): ApiKey | null {
  const exactMatches = exactKeysForModel(option)
  if (exactMatches.length === 0) return null

  const pickBestKey = (keys: ApiKey[]) =>
    keys.slice().sort((a, b) => keyRemainingQuota(b) - keyRemainingQuota(a))[0] ?? null

  return pickBestKey(exactMatches)
}

function exactKeysForModel(option: ChatModelOption | null): ApiKey[] {
  const modelName = option?.model.trim()
  if (!option || !modelName) return []

  return activeKeys.value.filter((key) => keyHasExactModel(key, modelName, option))
}

function keyHasExactModel(key: ApiKey | null, modelName: string, option?: ChatModelOption | null): boolean {
  if (!key || !modelName.trim()) return false
  if (option && !option.keyIds.includes(key.id)) return false
  return availableModelNamesByKeyId.value.get(key.id)?.has(modelName) === true
}

function hasExactKeyForModel(option: ChatModelOption | null): option is ChatModelOption {
  return exactKeysForModel(option).length > 0
}

function keyRemainingQuota(key: ApiKey): number {
  if (key.quota <= 0) return Number.POSITIVE_INFINITY
  return Math.max(0, key.quota - key.quota_used)
}

async function loadChatData() {
  loading.value = true
  try {
    const keysResult = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })

    apiKeys.value = keysResult.items
    channelModels.value = mergeModelOptions(await loadModelsFromKeys(activeKeys.value))
    if (!channelModels.value.some((item) => item.value === selectedModel.value)) {
      selectedModel.value = channelModels.value[0]?.value ?? ''
    }
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载模型失败。'
  } finally {
    loading.value = false
  }
}

async function refreshSelectedKey() {
  if (!selectedKey.value) return
  try {
    const updated = await keysAPI.getById(selectedKey.value.id)
    const index = apiKeys.value.findIndex((item) => item.id === updated.id)
    if (index >= 0) {
      apiKeys.value.splice(index, 1, updated)
    }
  } catch {
    // Keep the streamed response visible even if quota refresh fails.
  }
}

watch(selectedModel, () => {
  persistChatHistory()
})

onMounted(async () => {
  loadStoredChatHistory()
  await loadChatData()
})

onBeforeUnmount(() => {
  abortController?.abort()
  resetStreamRenderer()
  if (streamScrollFrame !== null) {
    window.cancelAnimationFrame(streamScrollFrame)
    streamScrollFrame = null
  }
  if (copyFeedbackTimer !== null) {
    window.clearTimeout(copyFeedbackTimer)
    copyFeedbackTimer = null
  }
})
</script>

<style scoped>
.chat-shell {
  @apply -m-4 flex h-[calc(100vh-4rem)] overflow-hidden bg-white dark:bg-dark-900 sm:-m-6;
}

.chat-sidebar {
  @apply hidden w-[286px] shrink-0 flex-col border-r border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950 lg:flex;
}

.chat-sidebar h1 {
  @apply mr-auto;
}

.chat-main {
  @apply relative flex min-w-0 flex-1 flex-col bg-white dark:bg-dark-900;
}

.chat-header {
  @apply grid gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-700 xl:grid-cols-[minmax(180px,360px)_1fr] xl:items-center;
}

.chat-main-sidebar-collapsed .chat-header {
  @apply lg:pl-16;
}

.sidebar-expand-button {
  @apply absolute left-4 top-4 z-20 hidden h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-600 shadow-sm transition hover:bg-gray-50 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700 lg:flex;
}

.chat-messages {
  @apply min-h-0 flex-1 overflow-y-auto px-5 py-6;
}

.welcome-panel {
  @apply mx-auto flex min-h-[54vh] max-w-3xl flex-col items-center justify-center text-center;
}

.starter-prompt {
  @apply rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm text-gray-700 shadow-sm transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-700;
}

.conversation-item {
  @apply flex w-full items-start gap-2 rounded-lg border border-transparent px-3 py-3 transition hover:border-gray-200 hover:bg-white dark:hover:border-dark-700 dark:hover:bg-dark-900;
}

.conversation-item-active {
  @apply border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900;
}

.conversation-icon {
  @apply mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-200;
}

.conversation-delete {
  @apply mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-gray-400 opacity-0 transition hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed dark:hover:bg-red-900/20;
}

.conversation-item:hover .conversation-delete,
.conversation-item-active .conversation-delete {
  @apply opacity-100;
}

.message-row {
  @apply mx-auto mb-5 flex w-full max-w-5xl gap-3;
}

.message-row-user {
  @apply flex-row-reverse;
}

.message-row-assistant {
  @apply flex-row;
}

.message-avatar {
  @apply flex h-9 w-9 shrink-0 items-center justify-center rounded-full;
}

.message-avatar-user {
  @apply bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-950;
}

.message-avatar-assistant {
  @apply bg-primary-500 text-white;
}

.message-bubble {
  @apply relative max-w-[min(760px,calc(100%-3rem))] rounded-2xl px-4 py-3 text-sm leading-6 shadow-sm;
}

.message-copy-button {
  @apply absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition hover:bg-black/5 focus:opacity-100 dark:hover:bg-white/10;
}

.message-bubble:hover .message-copy-button {
  @apply opacity-100;
}

.message-content {
  @apply pr-8;
}

.message-segment-actions {
  @apply mt-3 flex flex-wrap gap-2 border-t border-black/10 pt-3 dark:border-white/10;
}

.message-segment-copy {
  @apply inline-flex items-center gap-1.5 rounded-md border border-black/10 bg-white/70 px-2.5 py-1 text-xs font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-white/10 dark:bg-dark-900/70 dark:text-gray-200 dark:hover:border-primary-700 dark:hover:text-primary-300;
}

.message-bubble-user {
  @apply rounded-tr-sm bg-primary-600 text-white;
}

.message-bubble-assistant {
  @apply rounded-tl-sm border border-gray-200 bg-white text-gray-900 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100;
}

.chat-composer {
  @apply border-t border-gray-200 px-5 py-4 dark:border-dark-700;
}

.composer-inner {
  @apply mx-auto flex max-w-5xl items-end gap-3 rounded-2xl border border-gray-200 bg-white p-3 shadow-lg shadow-gray-900/5 dark:border-dark-700 dark:bg-dark-800;
}

.composer-inner textarea {
  @apply max-h-40 min-h-[42px] flex-1 resize-none bg-transparent px-2 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-white;
}

.send-button {
  @apply flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-600 text-white transition hover:bg-primary-700 disabled:cursor-not-allowed disabled:bg-gray-300 dark:disabled:bg-dark-600;
}

.icon-button {
  @apply inline-flex items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-600 transition hover:border-primary-300 hover:text-primary-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300;
}
</style>
