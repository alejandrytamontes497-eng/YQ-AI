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
          :disabled="submitting"
        ></textarea>

        <div class="reference-field">
          <label class="field-label">{{ t('imageGenerate.referenceImage') }}</label>
          <input
            id="reference-image-input"
            ref="referenceImageInput"
            class="sr-only"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            :disabled="submitting"
            @change="selectReferenceImage"
          />
          <div v-if="referenceImage && referenceImagePreview" class="reference-preview">
            <img :src="referenceImagePreview" :alt="t('imageGenerate.referenceImagePreview')" />
            <div class="reference-details">
              <p>{{ referenceImage.name }}</p>
              <span>{{ formatFileSize(referenceImage.size) }}</span>
            </div>
            <label class="reference-action" for="reference-image-input">
              <Icon name="upload" size="xs" :stroke-width="2" />
              <span>{{ t('imageGenerate.replaceReference') }}</span>
            </label>
            <button class="reference-remove" type="button" :title="t('imageGenerate.removeReference')" :disabled="submitting" @click="removeReferenceImage">
              <Icon name="x" size="sm" :stroke-width="2" />
            </button>
          </div>
          <label v-else class="reference-empty" for="reference-image-input">
            <Icon name="upload" size="sm" :stroke-width="2" />
            <span>{{ t('imageGenerate.uploadReference') }}</span>
            <small>{{ t('imageGenerate.referenceHint') }}</small>
          </label>
        </div>

        <div class="settings-grid">
          <div>
            <label class="field-label">{{ t('imageGenerate.model') }}</label>
            <Select v-model="selectedModel" :options="modelOptions" :placeholder="modelPlaceholder" :disabled="loading || submitting" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.size') }}</label>
            <Select v-model="selectedSize" :options="sizeOptions" :disabled="submitting" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.quality') }}</label>
            <Select v-model="selectedQuality" :options="qualityOptions" :disabled="submitting" />
          </div>
          <div>
            <label class="field-label">{{ t('imageGenerate.count') }}</label>
            <Select v-model="selectedCount" :options="countOptions" :disabled="submitting" />
          </div>
        </div>

        <div v-if="errorMessage" class="error-box">
          {{ errorMessage }}
        </div>

        <div class="actions-row">
          <button class="secondary-button" type="button" :disabled="submitting || !prompt.trim()" @click="optimizePrompt">
            <Icon name="edit" size="sm" :stroke-width="2" />
            <span>{{ t('imageGenerate.optimizePrompt') }}</span>
          </button>
          <button class="primary-button" type="button" :disabled="!canGenerate" @click="generate">
            <Icon v-if="!submitting" name="sparkles" size="sm" :stroke-width="2" />
            <span v-else class="spinner"></span>
            <span>{{ submitting ? t('imageGenerate.submitting') : t('imageGenerate.generateNow') }}</span>
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

        <div v-if="submitting" class="async-status" role="status" aria-live="polite">
          <span class="spinner large"></span>
          <div>
            <p>{{ t('imageGenerate.creatingJob') }}</p>
            <span>{{ t('imageGenerate.creatingJobHint') }}</span>
          </div>
        </div>

        <div v-else-if="hydratingCount > 0" class="async-status" role="status" aria-live="polite">
          <span class="spinner large"></span>
          <div>
            <p>{{ t('imageGenerate.loadingResult') }}</p>
            <span>{{ t('imageGenerate.loadingResultHint') }}</span>
          </div>
        </div>

        <div v-if="activeJobs.length" class="job-list">
          <article v-for="job in activeJobs" :key="job.id" class="job-item">
            <span class="spinner large"></span>
            <div class="min-w-0 flex-1">
              <div class="job-title-row">
                <p>{{ job.prompt }}</p>
                <span>{{ job.status === 'pending' ? t('imageGenerate.queued') : t('imageGenerate.running') }}</span>
              </div>
              <p class="image-sub">{{ job.model }} · {{ job.size }} · {{ formatJobTime(job.created_at) }}</p>
            </div>
          </article>
        </div>

        <div v-if="gallery.length === 0 && activeJobs.length === 0 && !submitting && hydratingCount === 0" class="empty-state">
          <Icon name="grid" size="xl" class="text-gray-400" />
          <p>{{ t('imageGenerate.empty') }}</p>
        </div>

        <div v-if="gallery.length" class="gallery-grid">
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
                <button type="button" :title="t('common.delete')" @click="deleteGalleryItem(item)">
                  <Icon name="trash" size="xs" :stroke-width="2" />
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
import { imagesAPI, type ImageGenerationJob, type UserImageModel } from '@/api/images'
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
  originalUrl: string
  prompt: string
  revisedPrompt?: string
  model: string
  size: string
  quality: string
  mimeType: string
  createdAt: string
}

interface StoredGalleryItem {
  id: string
  src?: string
  prompt: string
  revisedPrompt?: string
  model: string
  size: string
  quality: string
  mimeType?: string
  createdAt: string
}

interface StoredImageRecord {
  id: string
  original: Blob
  preview: Blob
  mimeType: string
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
const referenceImage = ref<File | null>(null)
const referenceImagePreview = ref('')
const referenceImageInput = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const gallery = ref<GalleryItem[]>([])
const jobs = ref<ImageGenerationJob[]>([])
const hydratingCount = ref(0)
const copiedId = ref('')
let abortController: AbortController | null = null
let copyFeedbackTimer: number | null = null
let imageDBPromise: Promise<IDBDatabase> | null = null
const originalBlobs = new Map<string, Blob>()
const objectUrls = new Set<string>()
const hydratingJobIDs = new Set<string>()
const processedJobResultIDs = new Set<string>()
const notifiedFailedJobIDs = new Set<string>()
const dismissedImageIDs = new Set<string>()
let unsubscribeImageJobs: (() => void) | null = null

const PREVIEW_MAX_EDGE = 960
const PREVIEW_QUALITY = 0.76
const IMAGE_DB_NAME = 'image_generation_gallery'
const IMAGE_DB_VERSION = 1
const IMAGE_STORE_NAME = 'images'
const ERROR_MESSAGE_MAX_LENGTH = 360
const REFERENCE_IMAGE_MAX_BYTES = 20 * 1024 * 1024
const REFERENCE_IMAGE_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])

const sizeOptions: SelectOption[] = [
  { value: '1024x1024', label: '1:1 · 1024x1024 · 方图' },
  { value: '1536x1024', label: '3:2 · 1536x1024 · 横图' },
  { value: '1792x1024', label: '16:9 · 1792x1024 · 横图' },
  { value: '1024x1536', label: '2:3 · 1024x1536 · 竖图' },
  { value: '1024x1792', label: '9:16 · 1024x1792 · 竖图' },
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
  Boolean(prompt.value.trim() && selectedModelOption.value && !loading.value && !submitting.value)
)
const activeJobs = computed(() =>
  jobs.value.filter((job) => job.status === 'pending' || job.status === 'running')
)

onMounted(async () => {
  loadDismissedImageIDs()
  await loadGallery()
  await Promise.all([loadData(), loadImageJobs()])
  unsubscribeImageJobs = imagesAPI.subscribeImageJobs({
    onSnapshot: handleJobSnapshot,
    onJob: handleJobUpdate
  })
})

onBeforeUnmount(() => {
  abortController?.abort()
  unsubscribeImageJobs?.()
  unsubscribeImageJobs = null
  if (copyFeedbackTimer !== null) {
    window.clearTimeout(copyFeedbackTimer)
  }
  revokeReferenceImagePreview()
  revokeAllObjectUrls()
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

function selectReferenceImage(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!REFERENCE_IMAGE_TYPES.has(file.type)) {
    errorMessage.value = t('imageGenerate.referenceTypeError')
    input.value = ''
    return
  }
  if (file.size > REFERENCE_IMAGE_MAX_BYTES) {
    errorMessage.value = t('imageGenerate.referenceSizeError')
    input.value = ''
    return
  }
  revokeReferenceImagePreview()
  referenceImage.value = file
  referenceImagePreview.value = URL.createObjectURL(file)
  errorMessage.value = ''
}

function removeReferenceImage() {
  revokeReferenceImagePreview()
  referenceImage.value = null
  if (referenceImageInput.value) referenceImageInput.value.value = ''
}

function revokeReferenceImagePreview() {
  if (!referenceImagePreview.value) return
  URL.revokeObjectURL(referenceImagePreview.value)
  referenceImagePreview.value = ''
}

function formatFileSize(bytes: number): string {
  return bytes >= 1024 * 1024
    ? `${(bytes / 1024 / 1024).toFixed(1)} MB`
    : `${Math.max(1, Math.round(bytes / 1024))} KB`
}

async function generate() {
  if (submitting.value) return

  const option = selectedModelOption.value
  const text = prompt.value.trim()
  if (!option || !text) {
    errorMessage.value = t('imageGenerate.missingRequired')
    return
  }

  abortController?.abort()
  abortController = new AbortController()
  submitting.value = true
  errorMessage.value = ''

  try {
    const job = await imagesAPI.createImageJob({
      model: option.model,
      prompt: text,
      size: String(selectedSize.value),
      quality: String(selectedQuality.value),
      n: Number(selectedCount.value),
      referenceImage: referenceImage.value ?? undefined,
      signal: abortController.signal
    })
    upsertJob(job)
    appStore.showSuccess(t('imageGenerate.jobAccepted'))
  } catch (error) {
    if ((error as { name?: string })?.name === 'AbortError') return
    errorMessage.value = imageGenerationErrorMessage(error, t('imageGenerate.generateFailed'))
    appStore.showError(errorMessage.value)
  } finally {
    submitting.value = false
    abortController = null
  }
}

async function loadImageJobs() {
  try {
    handleJobSnapshot(await imagesAPI.listImageJobs())
  } catch (error) {
    errorMessage.value = imageGenerationErrorMessage(error, t('imageGenerate.loadJobsFailed'))
  }
}

function handleJobSnapshot(nextJobs: ImageGenerationJob[]) {
  jobs.value = nextJobs.slice().sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at))
  for (const job of jobs.value) {
    if (job.status === 'succeeded') void hydrateCompletedJob(job)
  }
}

function handleJobUpdate(job: ImageGenerationJob) {
  upsertJob(job)
  void handleTerminalJob(job)
}

function upsertJob(job: ImageGenerationJob) {
  const index = jobs.value.findIndex((item) => item.id === job.id)
  if (index >= 0) {
    jobs.value.splice(index, 1, job)
  } else {
    jobs.value.unshift(job)
  }
  jobs.value.sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at))
  if (jobs.value.length > 24) jobs.value.splice(24)
}

async function handleTerminalJob(job: ImageGenerationJob) {
  if (job.status === 'failed') {
    if (notifiedFailedJobIDs.has(job.id)) return
    notifiedFailedJobIDs.add(job.id)
    const message = cleanImageGenerationErrorMessage(job.error_message || '', t('imageGenerate.generateFailed'))
    errorMessage.value = message
    appStore.showError(message)
    return
  }
  if (job.status !== 'succeeded' || hydratingJobIDs.has(job.id)) return
  await hydrateCompletedJob(job)
}

async function hydrateCompletedJob(job: ImageGenerationJob) {
  const missingResults = job.results.filter((result) => {
    const id = jobImageID(job.id, result.index)
    if (
      processedJobResultIDs.has(id) ||
      dismissedImageIDs.has(id) ||
      gallery.value.some((item) => item.id === id)
    ) {
      return false
    }
    processedJobResultIDs.add(id)
    return true
  })
  if (missingResults.length === 0) return

  hydratingJobIDs.add(job.id)
  hydratingCount.value += 1
  try {
    const items: GalleryItem[] = []
    for (const result of missingResults) {
      const id = jobImageID(job.id, result.index)
      const originalBlob = await imagesAPI.fetchImageJobResult(result)
      const mimeType = originalBlob.type || result.mime_type || 'image/png'
      const previewBlob = await createPreviewBlob(originalBlob)
      originalBlobs.set(id, originalBlob)
      await saveImageRecord({ id, original: originalBlob, preview: previewBlob, mimeType })
      items.push({
        id,
        src: objectUrlForBlob(previewBlob),
        originalUrl: objectUrlForBlob(originalBlob),
        prompt: job.prompt,
        revisedPrompt: result.revised_prompt,
        model: job.model,
        size: job.size,
        quality: job.quality,
        mimeType,
        createdAt: formatJobTime(job.finished_at || job.created_at)
      })
    }
    if (items.length > 0) {
      const nextGallery = uniqueGalleryItems([...items, ...gallery.value])
      for (const droppedItem of nextGallery.slice(24)) {
        revokeGalleryItemUrls(droppedItem)
        originalBlobs.delete(droppedItem.id)
      }
      gallery.value = nextGallery.slice(0, 24)
      appStore.showSuccess(t('imageGenerate.generateSuccess', { count: items.length }))
    }
  } catch (error) {
    for (const result of missingResults) {
      const id = jobImageID(job.id, result.index)
      if (!gallery.value.some((item) => item.id === id)) processedJobResultIDs.delete(id)
    }
    errorMessage.value = imageGenerationErrorMessage(error, t('imageGenerate.resultLoadFailed'))
  } finally {
    hydratingCount.value = Math.max(0, hydratingCount.value - 1)
    hydratingJobIDs.delete(job.id)
  }
}

function jobImageID(jobID: string, index: number): string {
  return `${jobID}:${index}`
}

function uniqueGalleryItems(items: GalleryItem[]): GalleryItem[] {
  const byID = new Map<string, GalleryItem>()
  for (const item of items) {
    if (!byID.has(item.id)) byID.set(item.id, item)
  }
  return Array.from(byID.values())
}

function formatJobTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

async function copyImage(item: GalleryItem) {
  try {
    const blob = await originalBlobForItem(item)
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
  let url = item.originalUrl
  let revokeAfterClick = false
  let mimeType = item.mimeType

  if (!url) {
    const blob = await originalBlobForItem(item)
    url = URL.createObjectURL(blob)
    mimeType = blob.type || item.mimeType
    revokeAfterClick = true
  } else if (url.startsWith('data:')) {
    const blob = dataURLToBlob(url)
    url = URL.createObjectURL(blob)
    mimeType = blob.type || item.mimeType
    revokeAfterClick = true
  }

  const link = document.createElement('a')
  link.href = url
  link.download = `${safeFileName(item.model)}-${Date.now()}.${fileExtension(mimeType)}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  if (revokeAfterClick) {
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
}

async function deleteGalleryItem(item: GalleryItem) {
  dismissedImageIDs.add(item.id)
  persistDismissedImageIDs()
  gallery.value = gallery.value.filter((entry) => entry.id !== item.id)
  revokeGalleryItemUrls(item)
  originalBlobs.delete(item.id)
  await deleteImageRecord(item.id)
}

async function imageBlob(src: string): Promise<Blob> {
  if (src.startsWith('data:')) {
    return dataURLToBlob(src)
  }
  const response = await fetch(src)
  return response.blob()
}

function base64ToBlob(base64: string, mimeType: string): Blob {
  const binary = window.atob(base64)
  const chunks: Uint8Array[] = []
  const chunkSize = 8192
  for (let offset = 0; offset < binary.length; offset += chunkSize) {
    const slice = binary.slice(offset, offset + chunkSize)
    const bytes = new Uint8Array(slice.length)
    for (let index = 0; index < slice.length; index += 1) {
      bytes[index] = slice.charCodeAt(index)
    }
    chunks.push(bytes)
  }
  return new Blob(chunks, { type: mimeType })
}

function dataURLToBlob(dataURL: string): Blob {
  const [header, payload = ''] = dataURL.split(',', 2)
  const mimeType = /data:([^;]+)/.exec(header)?.[1] || 'image/png'
  if (header.includes(';base64')) {
    return base64ToBlob(payload, mimeType)
  }
  return new Blob([decodeURIComponent(payload)], { type: mimeType })
}

async function createPreviewBlob(originalBlob: Blob): Promise<Blob> {
  try {
    const bitmap = await createImageBitmap(originalBlob)
    const scale = Math.min(1, PREVIEW_MAX_EDGE / Math.max(bitmap.width, bitmap.height))
    const width = Math.max(1, Math.round(bitmap.width * scale))
    const height = Math.max(1, Math.round(bitmap.height * scale))
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      bitmap.close()
      return originalBlob
    }
    ctx.drawImage(bitmap, 0, 0, width, height)
    bitmap.close()
    const previewBlob = await new Promise<Blob | null>((resolve) => {
      canvas.toBlob(resolve, 'image/webp', PREVIEW_QUALITY)
    })
    return previewBlob || originalBlob
  } catch {
    return originalBlob
  }
}

async function originalBlobForItem(item: GalleryItem): Promise<Blob> {
  const cached = originalBlobs.get(item.id)
  if (cached) return cached
  const record = await getImageRecord(item.id)
  if (record?.original) {
    originalBlobs.set(item.id, record.original)
    return record.original
  }
  return imageBlob(item.originalUrl || item.src)
}

function objectUrlForBlob(blob: Blob): string {
  const url = URL.createObjectURL(blob)
  objectUrls.add(url)
  return url
}

function revokeAllObjectUrls() {
  for (const url of objectUrls) {
    URL.revokeObjectURL(url)
  }
  objectUrls.clear()
}

function revokeGalleryItemUrls(item: GalleryItem) {
  for (const url of [item.src, item.originalUrl]) {
    if (objectUrls.has(url)) {
      URL.revokeObjectURL(url)
      objectUrls.delete(url)
    }
  }
}

function fileExtension(mimeType: string): string {
  switch (mimeType.toLowerCase()) {
    case 'image/jpeg':
      return 'jpg'
    case 'image/webp':
      return 'webp'
    case 'image/png':
    default:
      return 'png'
  }
}

function safeFileName(value: string): string {
  const sanitized = Array.from(String(value || 'image'), (character) =>
    character.charCodeAt(0) <= 31 || '<>:"/\\|?*'.includes(character) ? '-' : character
  ).join('')
  return sanitized.slice(0, 80) || 'image'
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

function dismissedImagesStorageKey(): string {
  const userID = authStore.user?.id ?? 'anonymous'
  return `image_generation_dismissed_v1:${userID}`
}

function loadDismissedImageIDs() {
  try {
    const parsed = JSON.parse(localStorage.getItem(dismissedImagesStorageKey()) || '[]')
    if (!Array.isArray(parsed)) return
    for (const id of parsed) {
      if (typeof id === 'string' && id) dismissedImageIDs.add(id)
    }
  } catch {
    // Dismissal history is optional.
  }
}

function persistDismissedImageIDs() {
  try {
    localStorage.setItem(dismissedImagesStorageKey(), JSON.stringify(Array.from(dismissedImageIDs).slice(-200)))
  } catch {
    // Dismissal history is optional.
  }
}

async function loadGallery() {
  try {
    const raw = localStorage.getItem(galleryStorageKey())
    if (!raw) return
    const parsed = JSON.parse(raw) as StoredGalleryItem[]
    if (!Array.isArray(parsed)) return

    const hydrated: GalleryItem[] = []
    for (const item of parsed.slice(0, 24)) {
      const restored = await restoreGalleryItem(item)
      if (restored) {
        processedJobResultIDs.add(restored.id)
        hydrated.push(restored)
      }
    }
    gallery.value = uniqueGalleryItems(hydrated)
  } catch {
    gallery.value = []
  }
}

function persistGallery() {
  try {
    const stored = gallery.value.slice(0, 24).map((item) => ({
      id: item.id,
      prompt: item.prompt,
      revisedPrompt: item.revisedPrompt,
      model: item.model,
      size: item.size,
      quality: item.quality,
      mimeType: item.mimeType,
      createdAt: item.createdAt
    }))
    localStorage.setItem(galleryStorageKey(), JSON.stringify(stored))
  } catch {
    // Local gallery is optional.
  }
}

async function restoreGalleryItem(item: StoredGalleryItem): Promise<GalleryItem | null> {
  const record = await getImageRecord(item.id)
  if (record) {
    originalBlobs.set(item.id, record.original)
    return {
      id: item.id,
      src: objectUrlForBlob(record.preview),
      originalUrl: objectUrlForBlob(record.original),
      prompt: item.prompt,
      revisedPrompt: item.revisedPrompt,
      model: item.model,
      size: item.size,
      quality: item.quality,
      mimeType: record.mimeType || item.mimeType || record.original.type || 'image/png',
      createdAt: item.createdAt
    }
  }

  if (!item.src) return null
  try {
    const original = await imageBlob(item.src)
    const preview = await createPreviewBlob(original)
    const mimeType = original.type || item.mimeType || 'image/png'
    originalBlobs.set(item.id, original)
    await saveImageRecord({ id: item.id, original, preview, mimeType })
    return {
      id: item.id,
      src: objectUrlForBlob(preview),
      originalUrl: objectUrlForBlob(original),
      prompt: item.prompt,
      revisedPrompt: item.revisedPrompt,
      model: item.model,
      size: item.size,
      quality: item.quality,
      mimeType,
      createdAt: item.createdAt
    }
  } catch {
    return null
  }
}

async function clearGallery() {
  for (const item of gallery.value) dismissedImageIDs.add(item.id)
  persistDismissedImageIDs()
  gallery.value = []
  originalBlobs.clear()
  revokeAllObjectUrls()
  await clearImageRecords()
}

function openImageDB(): Promise<IDBDatabase> {
  if (imageDBPromise) return imageDBPromise
  imageDBPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(IMAGE_DB_NAME, IMAGE_DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(IMAGE_STORE_NAME)) {
        db.createObjectStore(IMAGE_STORE_NAME, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
  return imageDBPromise
}

async function saveImageRecord(record: StoredImageRecord): Promise<void> {
  try {
    const db = await openImageDB()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(IMAGE_STORE_NAME, 'readwrite')
      tx.objectStore(IMAGE_STORE_NAME).put(record)
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch {
    // IndexedDB is an optimization; download still works in the current session.
  }
}

async function getImageRecord(id: string): Promise<StoredImageRecord | null> {
  try {
    const db = await openImageDB()
    return await new Promise<StoredImageRecord | null>((resolve, reject) => {
      const tx = db.transaction(IMAGE_STORE_NAME, 'readonly')
      const request = tx.objectStore(IMAGE_STORE_NAME).get(id)
      request.onsuccess = () => resolve((request.result as StoredImageRecord | undefined) ?? null)
      request.onerror = () => reject(request.error)
    })
  } catch {
    return null
  }
}

async function deleteImageRecord(id: string): Promise<void> {
  try {
    const db = await openImageDB()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(IMAGE_STORE_NAME, 'readwrite')
      tx.objectStore(IMAGE_STORE_NAME).delete(id)
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch {
    // Ignore unavailable storage.
  }
}

async function clearImageRecords(): Promise<void> {
  try {
    const db = await openImageDB()
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(IMAGE_STORE_NAME, 'readwrite')
      tx.objectStore(IMAGE_STORE_NAME).clear()
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch {
    // Ignore unavailable storage.
  }
}

function imageGenerationErrorMessage(error: unknown, fallback: string): string {
  const err = error as {
    status?: number
    code?: string
    message?: string
    error?: { message?: string }
  } | undefined
  const message = String(err?.error?.message || err?.message || '').trim()
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
  return cleanImageGenerationErrorMessage(message, fallback)
}

function cleanImageGenerationErrorMessage(message: string, fallback: string): string {
  const trimmed = message.trim()
  if (
    !trimmed ||
    isMarkupLikeErrorMessage(trimmed) ||
    isGenericUpstreamErrorMessage(trimmed) ||
    isImageToolMetaErrorMessage(trimmed)
  ) {
    return fallback
  }
  if (trimmed.length > ERROR_MESSAGE_MAX_LENGTH) {
    return `${trimmed.slice(0, ERROR_MESSAGE_MAX_LENGTH)}...`
  }
  return trimmed
}

function isMarkupLikeErrorMessage(message: string): boolean {
  const lower = message.trim().toLowerCase()
  return (
    /^<\s*(?:!doctype|html|head|body|svg|path|rect|circle|xml)\b/.test(lower) ||
    lower.includes('<svg') ||
    lower.includes('&lt;svg')
  )
}

function isGenericUpstreamErrorMessage(message: string): boolean {
  return /^(?:upstream request failed(?: \(status \d+\))?|upstream gateway error|image generation request failed)$/i.test(
    message.trim()
  )
}

function isImageToolMetaErrorMessage(message: string): boolean {
  const lower = message.trim().toLowerCase()
  return (
    (lower.startsWith('{') && lower.includes('prompt')) ||
    lower.includes('image_generation tool') ||
    lower.includes('do not answer with text') ||
    lower.includes('return generated image output') ||
    lower.includes("that's not valid") ||
    lower.includes('another possibility')
  )
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

.reference-field {
  @apply mt-4;
}

.reference-empty {
  @apply flex min-h-[92px] cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-slate-300 bg-slate-50 px-4 text-sm font-medium text-slate-700 transition hover:border-primary-400 hover:bg-primary-50 hover:text-primary-700;
}

.reference-empty small {
  @apply text-xs font-normal text-slate-500;
}

.reference-preview {
  @apply relative flex min-h-[92px] items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 pr-11;
}

.reference-preview img {
  @apply h-16 w-16 shrink-0 rounded-md border border-slate-200 bg-white object-cover;
}

.reference-details {
  @apply min-w-0 flex-1;
}

.reference-details p {
  @apply truncate text-sm font-medium text-slate-900;
}

.reference-details span {
  @apply mt-1 block text-xs text-slate-500;
}

.reference-action {
  @apply inline-flex h-9 shrink-0 cursor-pointer items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition hover:border-primary-400 hover:text-primary-700;
}

.reference-remove {
  @apply absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-md text-slate-500 transition hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-50;
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
  @apply mt-4 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700;
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

.async-status {
  @apply mb-4 flex min-h-[76px] items-center gap-4 rounded-lg border border-primary-200 bg-primary-50 px-4 py-3 text-primary-900;
}

.async-status .spinner.large {
  @apply h-6 w-6 shrink-0;
}

.async-status p {
  @apply text-sm font-semibold;
}

.async-status span:not(.spinner) {
  @apply mt-1 block text-xs text-primary-700;
}

.job-list {
  @apply mb-4 space-y-2;
}

.job-item {
  @apply flex min-h-[76px] items-center gap-4 rounded-lg border border-primary-200 bg-white px-4 py-3 shadow-sm;
}

.job-item .spinner.large {
  @apply h-6 w-6 shrink-0;
}

.job-title-row {
  @apply flex items-start justify-between gap-3;
}

.job-title-row p {
  @apply truncate text-sm font-medium text-slate-900;
}

.job-title-row span {
  @apply shrink-0 text-xs font-semibold text-primary-600;
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

@media (max-width: 640px) {
  .reference-preview {
    @apply flex-wrap;
  }

  .reference-details {
    @apply min-w-0;
    flex-basis: calc(100% - 5rem);
  }

  .reference-action {
    @apply ml-[4.75rem];
  }
}
</style>
