<template>
  <AppLayout>
    <div class="image-page">
      <section class="image-hero">
        <h1>{{ t('imageGenerate.title') }}</h1>
        <p>{{ t('imageGenerate.description') }}</p>
      </section>

      <section class="generator-panel">
        <div class="panel-title">
          <Icon name="sparkles" size="sm" :stroke-width="2" />
          <span>{{ t('imageGenerate.quickGenerate') }}</span>
          <button class="icon-refresh" type="button" :title="t('imageGenerate.refreshModels')" :disabled="loading" @click="loadData">
            <Icon name="refresh" size="sm" :stroke-width="2" />
          </button>
        </div>

        <label class="field-label" for="image-prompt">{{ t('imageGenerate.promptLabel') }}</label>
        <textarea
          id="image-prompt"
          v-model="prompt"
          class="prompt-input"
          :placeholder="t('imageGenerate.promptPlaceholder')"
          :disabled="generating"
        ></textarea>

        <div class="settings-grid">
          <div>
            <label class="field-label">{{ t('imageGenerate.model') }}</label>
            <Select v-model="selectedModel" :options="modelOptions" :placeholder="modelPlaceholder" :disabled="loading || generating" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.size') }}</label>
            <Select v-model="selectedSize" :options="sizeOptions" :disabled="generating" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.quality') }}</label>
            <Select v-model="selectedQuality" :options="qualityOptions" :disabled="generating" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.count') }}</label>
            <Select v-model="selectedCount" :options="countOptions" :disabled="generating" />
          </div>
        </div>

        <div v-if="errorMessage" class="error-box">
          {{ errorMessage }}
        </div>

        <div class="actions-row">
          <button class="secondary-button" type="button" :disabled="generating || !prompt.trim()" @click="optimizePrompt">
            <Icon name="edit" size="sm" :stroke-width="2" />
            <span>{{ t('imageGenerate.optimizePrompt') }}</span>
          </button>
          <button class="primary-button" type="button" :disabled="!canGenerate" @click="generate">
            <Icon v-if="!generating" name="sparkles" size="sm" :stroke-width="2" />
            <span v-else class="spinner"></span>
            <span>{{ generating ? t('imageGenerate.generating') : t('imageGenerate.generateNow') }}</span>
          </button>
        </div>
      </section>

      <section class="result-section">
        <div class="section-heading">
          <h2>{{ t('imageGenerate.results') }}</h2>
          <button v-if="gallery.length" class="text-button" type="button" @click="clearGallery">
            {{ t('imageGenerate.clearGallery') }}
          </button>
        </div>

        <div v-if="generating" class="empty-state">
          <span class="spinner large"></span>
          <p>{{ t('imageGenerate.waiting') }}</p>
        </div>

        <div v-else-if="gallery.length === 0" class="empty-state">
          <Icon name="grid" size="xl" class="text-gray-400" />
          <p>{{ t('imageGenerate.empty') }}</p>
        </div>

        <div v-else class="gallery-grid">
          <article v-for="item in gallery" :key="item.id" class="image-card">
            <img :src="item.src" :alt="item.prompt" loading="lazy" />
            <div class="image-meta">
              <div>
                <p class="image-prompt">{{ item.prompt }}</p>
                <p class="image-sub">{{ item.model }} · {{ item.size }} · {{ item.createdAt }}</p>
              </div>
              <div class="image-actions">
                <button type="button" :title="t('imageGenerate.copyImage')" @click="copyImage(item)">
                  <Icon :name="copiedId === item.id ? 'check' : 'copy'" size="xs" :stroke-width="2" />
                </button>
                <button type="button" :title="t('imageGenerate.download')" @click="downloadImage(item)">
                  <Icon name="download" size="xs" :stroke-width="2" />
                </button>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { imagesAPI, type UserImageModel } from '@/api/images'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'

interface ImageModelOption extends SelectOption {
  value: string
  label: string
  model: string
  platform: string
  groupIds: number[]
}

interface GalleryItem {
  id: string
  src: string
  prompt: string
  revisedPrompt?: string
  model: string
  size: string
  quality: string
  createdAt: string
}

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const imageModels = ref<ImageModelOption[]>([])
const selectedModel = ref('')
const selectedSize = ref('1024x1024')
const selectedQuality = ref('auto')
const selectedCount = ref(1)
const prompt = ref('')
const loading = ref(false)
const generating = ref(false)
const errorMessage = ref('')
const gallery = ref<GalleryItem[]>([])
const copiedId = ref('')
let abortController: AbortController | null = null
let copyFeedbackTimer: number | null = null

const sizeOptions: SelectOption[] = [
  { value: '1024x1024', label: '1:1 · 1024x1024 · 方图' },
  { value: '1536x1024', label: '3:2 · 1536x1024 · 横图' },
  { value: '1024x1536', label: '2:3 · 1024x1536 · 竖图' },
  { value: 'auto', label: 'auto · 自动' }
]

const qualityOptions: SelectOption[] = [
  { value: 'auto', label: 'auto' },
  { value: 'low', label: 'low' },
  { value: 'medium', label: 'medium' },
  { value: 'high', label: 'high' }
]

const countOptions: SelectOption[] = [
  { value: 1, label: '1x' },
  { value: 2, label: '2x' },
  { value: 3, label: '3x' },
  { value: 4, label: '4x' }
]

const modelOptions = computed<ImageModelOption[]>(() => imageModels.value)
const selectedModelOption = computed(() =>
  modelOptions.value.find((item) => item.value === selectedModel.value) ?? null
)
const modelPlaceholder = computed(() =>
  loading.value ? t('imageGenerate.loadingModels') : t('imageGenerate.noModels')
)
const canGenerate = computed(() =>
  Boolean(prompt.value.trim() && selectedModelOption.value && !loading.value && !generating.value)
)

onMounted(() => {
  loadGallery()
  void loadData()
})

onBeforeUnmount(() => {
  abortController?.abort()
  if (copyFeedbackTimer !== null) {
    window.clearTimeout(copyFeedbackTimer)
  }
})

watch(gallery, persistGallery, { deep: true })

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    imageModels.value = loadModelsFromUserImageModels(await imagesAPI.listUserImageModels())
    if (!imageModels.value.some((item) => item.value === selectedModel.value)) {
      selectedModel.value = imageModels.value[0]?.value ?? ''
    }
    if (imageModels.value.length === 0) {
      errorMessage.value = t('imageGenerate.noAvailableModels')
    }
  } catch (error) {
    errorMessage.value = imageGenerationErrorMessage(error, t('imageGenerate.loadFailed'))
  } finally {
    loading.value = false
  }
}

function loadModelsFromUserImageModels(models: UserImageModel[]): ImageModelOption[] {
  const byKey = new Map<string, ImageModelOption>()

  for (const rawModel of models) {
    const name = rawModel.name.trim()
    const platform = rawModel.platform || 'openai'
    const groupIds = rawModel.group_ids.filter((id) => typeof id === 'number')
    if (!name || groupIds.length === 0) continue

    const optionKey = `${platform}:${name}`
    const existing = byKey.get(optionKey)
    if (existing) {
      existing.groupIds = Array.from(new Set([...existing.groupIds, ...groupIds]))
      continue
    }

    byKey.set(optionKey, {
      value: optionKey,
      label: name,
      model: name,
      platform,
      groupIds
    })
  }

  return Array.from(byKey.values()).sort((a, b) => a.label.localeCompare(b.label))
}

function optimizePrompt() {
  const text = prompt.value.trim()
  if (!text) return
  if (/真实材质|清晰主视觉|高级棚拍光线|high detail/i.test(text)) return
  prompt.value = `${text}，清晰主视觉，真实材质，高级棚拍光线，细节丰富，构图干净，避免文字水印和畸形元素`
}

async function generate() {
  const option = selectedModelOption.value
  const text = prompt.value.trim()
  if (!option || !text) {
    errorMessage.value = t('imageGenerate.missingRequired')
    return
  }

  abortController?.abort()
  abortController = new AbortController()
  generating.value = true
  errorMessage.value = ''

  try {
    const response = await imagesAPI.generateImage({
      model: option.model,
      prompt: text,
      size: String(selectedSize.value),
      quality: String(selectedQuality.value),
      n: Number(selectedCount.value),
      signal: abortController.signal
    })

    const items = (response.data || [])
      .map((item) => ({
        id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
        src: item.b64_json ? `data:image/png;base64,${item.b64_json}` : item.url || '',
        prompt: text,
        revisedPrompt: item.revised_prompt,
        model: option.model,
        size: String(selectedSize.value),
        quality: String(selectedQuality.value),
        createdAt: new Date().toLocaleString()
      }))
      .filter((item) => item.src)

    if (items.length === 0) {
      throw new Error(t('imageGenerate.emptyResponse'))
    }

    gallery.value = [...items, ...gallery.value].slice(0, 24)
    appStore.showSuccess(t('imageGenerate.generateSuccess', { count: items.length }))
  } catch (error) {
    if ((error as { name?: string })?.name === 'AbortError') return
    errorMessage.value = imageGenerationErrorMessage(error, t('imageGenerate.generateFailed'))
    appStore.showError(errorMessage.value)
  } finally {
    generating.value = false
    abortController = null
  }
}

async function copyImage(item: GalleryItem) {
  try {
    const blob = await imageBlob(item.src)
    await navigator.clipboard.write([
      new ClipboardItem({ [blob.type || 'image/png']: blob })
    ])
    showCopied(item.id)
  } catch {
    await navigator.clipboard?.writeText(item.src)
    showCopied(item.id)
  }
}

async function downloadImage(item: GalleryItem) {
  const blob = await imageBlob(item.src)
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${item.model}-${Date.now()}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

async function imageBlob(src: string): Promise<Blob> {
  const response = await fetch(src)
  return response.blob()
}

function showCopied(id: string) {
  copiedId.value = id
  if (copyFeedbackTimer !== null) {
    window.clearTimeout(copyFeedbackTimer)
  }
  copyFeedbackTimer = window.setTimeout(() => {
    copiedId.value = ''
    copyFeedbackTimer = null
  }, 1400)
}

function galleryStorageKey(): string {
  const userID = authStore.user?.id ?? 'anonymous'
  return `image_generation_gallery_v1:${userID}`
}

function loadGallery() {
  try {
    const raw = localStorage.getItem(galleryStorageKey())
    if (!raw) return
    const parsed = JSON.parse(raw) as GalleryItem[]
    gallery.value = Array.isArray(parsed) ? parsed.filter((item) => item?.src).slice(0, 24) : []
  } catch {
    gallery.value = []
  }
}

function persistGallery() {
  try {
    localStorage.setItem(galleryStorageKey(), JSON.stringify(gallery.value.slice(0, 24)))
  } catch {
    // Local gallery is optional.
  }
}

function clearGallery() {
  gallery.value = []
}

function imageGenerationErrorMessage(error: unknown, fallback: string): string {
  const err = error as { status?: number; code?: string; message?: string } | undefined
  const message = String(err?.message || '').trim()
  if (
    err?.status === 0 ||
    err?.code === 'ERR_NETWORK' ||
    /failed to fetch|network error/i.test(message)
  ) {
    return t('imageGenerate.networkFailed')
  }
  if (err?.code === 'ECONNABORTED' || /timeout/i.test(message)) {
    return t('imageGenerate.timeout')
  }
  return message || fallback
}

</script>

<style scoped>
.image-page {
  @apply min-h-screen bg-slate-50 px-5 py-6 text-slate-900;
}

.image-hero {
  @apply mx-auto max-w-6xl rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm;
}

.image-hero h1 {
  @apply text-3xl font-bold tracking-normal text-slate-950;
}

.image-hero p {
  @apply mt-1 text-sm text-slate-500;
}

.generator-panel {
  @apply mx-auto mt-6 max-w-6xl rounded-lg border border-slate-200 bg-white p-4 shadow-sm;
}

.panel-title {
  @apply mb-4 flex items-center gap-2 text-sm font-semibold text-slate-950;
}

.icon-refresh {
  @apply ml-auto rounded-md p-1.5 text-slate-500 transition hover:bg-slate-100 hover:text-slate-950 disabled:cursor-not-allowed disabled:opacity-50;
}

.field-label {
  @apply mb-2 block text-xs font-semibold text-slate-700;
}

.prompt-input {
  @apply min-h-[300px] w-full resize-y rounded-lg border border-slate-300 bg-white px-4 py-3 text-sm leading-6 text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20;
}

.settings-grid {
  @apply mt-4 grid grid-cols-1 gap-3 md:grid-cols-4;
}

.actions-row {
  @apply mt-4 flex justify-end gap-3;
}

.primary-button,
.secondary-button {
  @apply inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-50;
}

.primary-button {
  @apply bg-primary-600 text-white hover:bg-primary-500;
}

.secondary-button {
  @apply border border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-950;
}

.text-button {
  @apply text-sm font-medium text-slate-500 transition hover:text-slate-950;
}

.error-box {
  @apply mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700;
}

.result-section {
  @apply mx-auto mt-6 max-w-6xl;
}

.section-heading {
  @apply mb-3 flex items-center justify-between;
}

.section-heading h2 {
  @apply text-lg font-semibold text-slate-950;
}

.empty-state {
  @apply flex min-h-[220px] flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-slate-300 bg-white text-sm text-slate-500;
}

.gallery-grid {
  @apply grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3;
}

.image-card {
  @apply overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm;
}

.image-card img {
  @apply aspect-square w-full bg-slate-100 object-contain;
}

.image-meta {
  @apply flex gap-3 border-t border-slate-100 p-3;
}

.image-prompt {
  @apply text-sm text-slate-900;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.image-sub {
  @apply mt-1 text-xs text-slate-500;
}

.image-actions {
  @apply ml-auto flex shrink-0 gap-2;
}

.image-actions button {
  @apply flex h-8 w-8 items-center justify-center rounded-md border border-slate-200 text-slate-500 transition hover:border-primary-500 hover:bg-primary-50 hover:text-primary-600;
}

.spinner {
  @apply h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white;
}

.spinner.large {
  @apply h-7 w-7 border-slate-300 border-t-primary-500;
}
</style>
