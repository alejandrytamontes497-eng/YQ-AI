import { apiClient } from './client'

export interface UserImageModel {
  name: string
  platform: string
  group_ids: number[]
}

export interface ImageGenerationRequest {
  apiKey: string
  model: string
  prompt: string
  size: string
  quality: string
  n: number
  signal?: AbortSignal
}

export interface GeneratedImageItem {
  url?: string
  download_url?: string
  b64_json?: string
  revised_prompt?: string
}

export interface ImageGenerationResponse {
  created?: number
  data?: GeneratedImageItem[]
}

function parseJsonBody(text: string): any {
  if (!text) return null

  try {
    return JSON.parse(text)
  } catch {
    return { error: { message: text } }
  }
}

function errorMessageFromBody(body: any, fallback: string): string {
  return body?.error?.message || body?.message || body?.detail || fallback
}

export async function listUserImageModels(signal?: AbortSignal): Promise<UserImageModel[]> {
  const { data } = await apiClient.get<UserImageModel[]>('/user/images/models', { signal })
  return Array.isArray(data) ? data : []
}

export async function generateImage(request: ImageGenerationRequest): Promise<ImageGenerationResponse> {
  const payload: Record<string, unknown> = {
    model: request.model,
    prompt: request.prompt,
    size: request.size,
    n: request.n,
    response_format: 'b64_json'
  }

  if (request.quality && request.quality !== 'auto') {
    payload.quality = request.quality
  } else {
    payload.quality = 'auto'
  }

  const response = await fetch('/v1/images/generations', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`
    },
    body: JSON.stringify(payload),
    signal: request.signal
  })

  const text = await response.text()
  const body = parseJsonBody(text)

  if (!response.ok) {
    throw new Error(errorMessageFromBody(body, `Image generation failed with HTTP ${response.status}`))
  }

  return body as ImageGenerationResponse
}

export const imagesAPI = {
  listUserImageModels,
  generateImage
}

export default imagesAPI
