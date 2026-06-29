export type ChatRole = 'system' | 'user' | 'assistant'

export interface ChatMessage {
  role: ChatRole
  content: string
}

export interface ChatCompletionRequest {
  apiKey: string
  model: string
  messages: ChatMessage[]
  temperature?: number
  max_tokens?: number
  signal?: AbortSignal
}

export interface ChatCompletionUsage {
  prompt_tokens?: number
  completion_tokens?: number
  total_tokens?: number
}

export interface ChatCompletionResponse {
  id?: string
  model?: string
  usage?: ChatCompletionUsage
  choices?: Array<{
    message?: ChatMessage
    finish_reason?: string
  }>
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

export async function createChatCompletion(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`
    },
    body: JSON.stringify({
      model: request.model,
      messages: request.messages,
      temperature: request.temperature,
      max_tokens: request.max_tokens,
      stream: false
    }),
    signal: request.signal
  })

  const text = await response.text()
  const body = parseJsonBody(text)

  if (!response.ok) {
    throw new Error(errorMessageFromBody(body, `Chat request failed with HTTP ${response.status}`))
  }

  return body as ChatCompletionResponse
}

function normalizeModelItem(item: unknown): string {
  if (typeof item === 'string') return item
  if (item && typeof item === 'object') {
    const model = item as Record<string, unknown>
    if (typeof model.id === 'string') return model.id
    if (typeof model.name === 'string') return model.name
    if (typeof model.model === 'string') return model.model
  }
  return ''
}

function normalizeModelsBody(body: any): string[] {
  const list: unknown[] =
    Array.isArray(body?.data) ? body.data :
    Array.isArray(body?.models) ? body.models :
    Array.isArray(body) ? body :
    []

  return Array.from(new Set(list.map(normalizeModelItem).map((name: string) => name.trim()).filter(Boolean)))
}

export async function listModels(apiKey: string, signal?: AbortSignal): Promise<string[]> {
  const response = await fetch('/v1/models', {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${apiKey}`
    },
    signal
  })

  const text = await response.text()
  const body = parseJsonBody(text)

  if (!response.ok) {
    throw new Error(errorMessageFromBody(body, `Load models failed with HTTP ${response.status}`))
  }

  return normalizeModelsBody(body)
}

export const chatAPI = {
  createChatCompletion,
  listModels
}

export default chatAPI
