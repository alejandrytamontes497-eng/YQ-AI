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

  const { data } = await apiClient.post<ImageGenerationResponse>('/user/images/generations', payload, {
    signal: request.signal,
    timeout: 300000
  })
  return data
}

export const imagesAPI = {
  listUserImageModels,
  generateImage
}

export default imagesAPI
