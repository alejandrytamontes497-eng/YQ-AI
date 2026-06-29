<template>
  <AppLayout>
    <div class="chat-shell">
      <aside class="chat-sidebar">
        <div class="flex items-center justify-between px-5 pt-5">
          <h1 class="text-xl font-semibold text-gray-950 dark:text-white">在线聊天</h1>
          <button class="icon-button" type="button" title="刷新密钥" :disabled="loadingKeys" @click="loadApiKeys">
            <Icon name="refresh" size="sm" />
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
          <button
            v-for="conversation in filteredConversations"
            :key="conversation.id"
            type="button"
            class="conversation-item"
            :class="{ 'conversation-item-active': conversation.id === activeConversationId }"
            @click="selectConversation(conversation.id)"
          >
            <span class="conversation-icon">
              <Icon name="chat" size="sm" />
            </span>
            <span class="min-w-0 flex-1 text-left">
              <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">
                {{ conversation.title }}
              </span>
              <span class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400">
                {{ conversation.preview || '暂无消息' }}
              </span>
              <span class="mt-1 block truncate text-xs text-primary-600 dark:text-primary-300">
                {{ conversation.model || selectedModel || '未选择模型' }} · {{ conversation.updatedAt }}
              </span>
            </span>
          </button>
        </div>
      </aside>

      <section class="chat-main">
        <header class="chat-header">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <h2 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                {{ activeConversation?.title || '新对话' }}
              </h2>
              <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
                {{ platformLabel(selectedKey?.group?.platform) }}
              </span>
            </div>
            <p class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">
              {{ selectedModel || '请选择模型' }}
            </p>
          </div>

          <div class="grid min-w-0 flex-1 gap-3 md:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto]">
            <Select
              v-model="selectedKeyId"
              :options="apiKeyOptions"
              :disabled="loadingKeys || sending"
              searchable
              placeholder="选择计费 API 密钥"
              empty-text="暂无可用密钥"
              @change="onKeyChange"
            >
              <template #selected="{ option }">
                <span v-if="option" class="truncate">{{ option.label }}</span>
                <span v-else class="text-gray-400">选择计费 API 密钥</span>
              </template>
              <template #option="{ option, selected }">
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-medium">{{ option.label }}</div>
                  <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ option.platformLabel }} · {{ option.quotaLabel }}
                  </div>
                </div>
                <Icon v-if="selected" name="check" size="sm" class="text-primary-500" :stroke-width="2" />
              </template>
            </Select>

            <Select
              v-if="modelOptions.length > 0"
              v-model="selectedModel"
              :options="modelOptions"
              disabled
              placeholder="选择或输入模型"
              @change="onModelSelect"
            />
            <Input
              v-else
              v-model="selectedModel"
              disabled
              placeholder="输入模型名称"
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
              把问题、任务、代码片段，或者一段图片描述发过来。
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
              <div class="whitespace-pre-wrap break-words">{{ message.content }}</div>
            </div>
          </div>

          <div v-if="sending" class="message-row message-row-assistant">
            <div class="message-avatar message-avatar-assistant">
              <Icon name="sparkles" size="sm" />
            </div>
            <div class="message-bubble message-bubble-assistant">
              正在生成回复...
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
            <span>{{ selectedKey ? `使用 ${selectedKey.name} 计费` : '请选择 API 密钥' }}</span>
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
import { userChannelsAPI, type UserSupportedModel } from '@/api/channels'
import { chatAPI, type ChatCompletionUsage, type ChatMessage } from '@/api/chat'
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

interface ApiKeyOption extends SelectOption {
  key: ApiKey
  platformLabel: string
  quotaLabel: string
}

const starterPrompts = [
  '帮我把这件事拆成三步执行计划',
  '把这段内容改得更清楚',
  '写一段可以直接发给客户的说明'
]

const apiKeys = ref<ApiKey[]>([])
const supportedModels = ref<UserSupportedModel[]>([])
const selectedKeyId = ref<number | null>(null)
const selectedModel = ref('')
const conversations = ref<Conversation[]>([createConversation('新对话')])
const activeConversationId = ref(conversations.value[0].id)
const conversationSearch = ref('')
const draft = ref('')
const loadingKeys = ref(false)
const sending = ref(false)
const errorMessage = ref('')
const lastUsage = ref<ChatCompletionUsage | null>(null)
const messagesRef = ref<HTMLElement | null>(null)
let abortController: AbortController | null = null

const activeConversation = computed(() =>
  conversations.value.find((item) => item.id === activeConversationId.value) ?? null
)

const messages = computed(() => activeConversation.value?.messages ?? [])

const selectedKey = computed(() => apiKeys.value.find((item) => item.id === selectedKeyId.value) ?? null)
const activeKeys = computed(() => apiKeys.value.filter((item) => item.status === 'active'))

const filteredConversations = computed(() => {
  const query = conversationSearch.value.trim().toLowerCase()
  if (!query) return conversations.value
  return conversations.value.filter((item) =>
    [item.title, item.preview, item.model].some((value) => value.toLowerCase().includes(query))
  )
})

const apiKeyOptions = computed<ApiKeyOption[]>(() =>
  activeKeys.value.map((item) => ({
    value: item.id,
    label: item.name,
    key: item,
    platformLabel: platformLabel(item.group?.platform),
    quotaLabel: item.quota > 0
      ? `$${item.quota_used.toFixed(2)} / $${item.quota.toFixed(2)}`
      : '不限额'
  }))
)

const modelOptions = computed<SelectOption[]>(() => {
  const platform = selectedKey.value?.group?.platform
  const names = supportedModels.value
    .filter((model) => !platform || model.platform === platform)
    .map((model) => model.name)
  return Array.from(new Set(names)).map((name) => ({ value: name, label: name }))
})

const canSend = computed(() =>
  Boolean(selectedKey.value?.key && selectedModel.value.trim() && draft.value.trim() && !sending.value)
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

function onKeyChange(_value: string | number | boolean | null, option: SelectOption | null) {
  const key = (option as ApiKeyOption | null)?.key ?? selectedKey.value
  selectedModel.value = firstModelForKey(key)
}

function onModelSelect(value: string | number | boolean | null) {
  selectedModel.value = String(value ?? '')
}

function firstModelForKey(key: ApiKey | null): string {
  const platform = key?.group?.platform
  const model = supportedModels.value.find((item) => !platform || item.platform === platform)
  return model?.name || chatAPI.defaultModelForKey(key)
}

function startNewChat() {
  const conversation = createConversation('新对话')
  conversation.model = selectedModel.value
  conversations.value.unshift(conversation)
  activeConversationId.value = conversation.id
  draft.value = ''
  errorMessage.value = ''
  lastUsage.value = null
}

function selectConversation(id: string) {
  activeConversationId.value = id
  errorMessage.value = ''
  nextTick(scrollToBottom)
}

function resetActiveConversation() {
  if (!activeConversation.value) return
  activeConversation.value.messages = []
  activeConversation.value.title = '新对话'
  activeConversation.value.preview = ''
  activeConversation.value.updatedAt = formatTime(new Date())
  errorMessage.value = ''
  lastUsage.value = null
}

function useStarterPrompt(prompt: string) {
  draft.value = prompt
}

function toGatewayMessages(): ChatMessage[] {
  return messages.value.map((message) => ({
    role: message.role,
    content: message.content
  }))
}

function addMessage(role: ChatMessage['role'], content: string) {
  if (!activeConversation.value) return
  activeConversation.value.messages.push({
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    role,
    content
  })
  activeConversation.value.preview = content
  activeConversation.value.model = selectedModel.value
  activeConversation.value.updatedAt = formatTime(new Date())
  if (role === 'user' && activeConversation.value.title === '新对话') {
    activeConversation.value.title = content.slice(0, 18) || '新对话'
  }
}

function parsePositiveNumber(value: string, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

async function sendMessage() {
  if (!canSend.value || !selectedKey.value) return

  const content = draft.value.trim()
  draft.value = ''
  errorMessage.value = ''
  lastUsage.value = null
  addMessage('user', content)
  await scrollToBottom()

  abortController = new AbortController()
  sending.value = true

  try {
    const response = await chatAPI.createChatCompletion({
      apiKey: selectedKey.value.key,
      model: selectedModel.value.trim(),
      messages: toGatewayMessages(),
      temperature: 0.7,
      max_tokens: Math.round(parsePositiveNumber('2048', 2048)),
      signal: abortController.signal
    })

    const reply = response.choices?.[0]?.message?.content?.trim()
    addMessage('assistant', reply || '模型没有返回可显示的内容。')
    lastUsage.value = response.usage ?? null
    await refreshSelectedKey()
  } catch (error) {
    const message = error instanceof Error ? error.message : '发送失败，请稍后重试。'
    errorMessage.value = message
    addMessage('assistant', `请求失败：${message}`)
  } finally {
    sending.value = false
    abortController = null
    await scrollToBottom()
  }
}

async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

async function loadApiKeys() {
  loadingKeys.value = true
  try {
    const result = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
    apiKeys.value = result.items
    if (!selectedKeyId.value && activeKeys.value.length > 0) {
      selectedKeyId.value = activeKeys.value[0].id
      selectedModel.value = firstModelForKey(activeKeys.value[0])
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载密钥失败。'
  } finally {
    loadingKeys.value = false
  }
}

async function loadSupportedModels() {
  try {
    const channels = await userChannelsAPI.getAvailable()
    supportedModels.value = channels.flatMap((channel) =>
      channel.platforms.flatMap((platform) => platform.supported_models)
    )
  } catch {
    supportedModels.value = []
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
    // Keep the chat response visible even if quota refresh fails.
  }
}

watch(selectedKey, (key) => {
  if (key && !selectedModel.value) {
    selectedModel.value = firstModelForKey(key)
  }
})

onMounted(async () => {
  await Promise.all([loadApiKeys(), loadSupportedModels()])
  if (selectedKey.value) {
    selectedModel.value = firstModelForKey(selectedKey.value)
  }
})

onBeforeUnmount(() => {
  abortController?.abort()
})
</script>

<style scoped>
.chat-shell {
  @apply -m-4 flex h-[calc(100vh-4rem)] overflow-hidden bg-white dark:bg-dark-900 sm:-m-6;
}

.chat-sidebar {
  @apply hidden w-[286px] shrink-0 flex-col border-r border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950 lg:flex;
}

.chat-main {
  @apply flex min-w-0 flex-1 flex-col bg-white dark:bg-dark-900;
}

.chat-header {
  @apply grid gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-700 xl:grid-cols-[minmax(180px,360px)_1fr] xl:items-center;
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
  @apply flex w-full gap-3 rounded-lg border border-transparent px-3 py-3 transition hover:border-gray-200 hover:bg-white dark:hover:border-dark-700 dark:hover:bg-dark-900;
}

.conversation-item-active {
  @apply border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900;
}

.conversation-icon {
  @apply mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-200;
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
  @apply max-w-[min(760px,calc(100%-3rem))] rounded-2xl px-4 py-3 text-sm leading-6 shadow-sm;
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
