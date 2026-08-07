import { apiClient } from './client'

export interface UserImageModel {
  name: string
  platform: string
  group_ids: number[]
}

export interface ImageGenerationRequest {
  model: string
  prompt: string
  size: string
  quality: string
  n: number
  referenceImage?: File
  signal?: AbortSignal
}

export type ImageGenerationJobStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export interface ImageGenerationJobResult {
  index: number
  url: string
  mime_type: string
  byte_size?: number
  revised_prompt?: string
}

export interface ImageGenerationJob {
  id: string
  status: ImageGenerationJobStatus
  model: string
  prompt: string
  size: string
  quality: string
  image_count: number
  reference_image_name?: string
  reference_image_mime_type?: string
  results: ImageGenerationJobResult[]
  error_message?: string
  attempt_count: number
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export async function listUserImageModels(signal?: AbortSignal): Promise<UserImageModel[]> {
  const { data } = await apiClient.get<UserImageModel[]>('/user/images/models', { signal })
  return Array.isArray(data) ? data : []
}

export async function createImageJob(request: ImageGenerationRequest): Promise<ImageGenerationJob> {
  const payload = {
    model: request.model,
    prompt: request.prompt,
    size: request.size,
    quality: request.quality || 'auto',
    n: request.n,
    response_format: 'b64_json'
  }
  let body: typeof payload | FormData = payload
  if (request.referenceImage) {
    const form = new FormData()
    Object.entries(payload).forEach(([key, value]) => form.append(key, String(value)))
    form.append('image', request.referenceImage, request.referenceImage.name)
    body = form
  }
  const { data } = await apiClient.post<ImageGenerationJob>('/user/images/jobs', body, {
    signal: request.signal,
    headers: request.referenceImage ? { 'Content-Type': false } : undefined
  })
  return data
}

export async function listImageJobs(limit = 24, signal?: AbortSignal): Promise<ImageGenerationJob[]> {
  const { data } = await apiClient.get<ImageGenerationJob[]>('/user/images/jobs', {
    params: { limit },
    signal
  })
  return Array.isArray(data) ? data : []
}

export async function getImageJob(id: string, signal?: AbortSignal): Promise<ImageGenerationJob> {
  const { data } = await apiClient.get<ImageGenerationJob>(`/user/images/jobs/${encodeURIComponent(id)}`, { signal })
  return data
}

export async function fetchImageJobResult(result: ImageGenerationJobResult, signal?: AbortSignal): Promise<Blob> {
  if (/^https?:\/\//i.test(result.url)) {
    const response = await fetch(result.url, { signal })
    if (!response.ok) throw new Error(`Image download failed with HTTP ${response.status}`)
    return response.blob()
  }
  const { data } = await apiClient.get<Blob>(result.url, {
    responseType: 'blob',
    signal,
    timeout: 120000
  })
  return data
}

export interface ImageJobEventHandlers {
  onSnapshot?: (jobs: ImageGenerationJob[]) => void
  onJob?: (job: ImageGenerationJob) => void
  onError?: (error: unknown) => void
}

export function subscribeImageJobs(handlers: ImageJobEventHandlers): () => void {
  const controller = new AbortController()

  const connect = async () => {
    while (!controller.signal.aborted) {
      try {
        const token = localStorage.getItem('auth_token')
        const response = await fetch('/api/v1/user/images/jobs/events', {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
          signal: controller.signal
        })
        if (!response.ok || !response.body) {
          throw new Error(`Image job event stream failed with HTTP ${response.status}`)
        }
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (!controller.signal.aborted) {
          const { value, done } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
          let boundary = buffer.indexOf('\n\n')
          while (boundary >= 0) {
            const block = buffer.slice(0, boundary)
            buffer = buffer.slice(boundary + 2)
            handleImageJobEventBlock(block, handlers)
            boundary = buffer.indexOf('\n\n')
          }
        }
      } catch (error) {
        if (controller.signal.aborted) return
        handlers.onError?.(error)
      }
      if (!controller.signal.aborted) {
        await new Promise((resolve) => window.setTimeout(resolve, 2000))
      }
    }
  }

  void connect()
  return () => controller.abort()
}

function handleImageJobEventBlock(block: string, handlers: ImageJobEventHandlers) {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }
  if (dataLines.length === 0) return
  try {
    const payload = JSON.parse(dataLines.join('\n'))
    if (event === 'snapshot' && Array.isArray(payload)) {
      handlers.onSnapshot?.(payload as ImageGenerationJob[])
    } else if (event === 'job' && payload && typeof payload === 'object') {
      handlers.onJob?.(payload as ImageGenerationJob)
    }
  } catch {
    // Ignore malformed events and keep the stream alive.
  }
}

export const imagesAPI = {
  listUserImageModels,
  createImageJob,
  listImageJobs,
  getImageJob,
  fetchImageJobResult,
  subscribeImageJobs
}

export default imagesAPI
