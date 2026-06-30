import { apiClient } from './client'

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

export interface ChatCompletionStreamCallbacks {
  onDelta?: (content: string) => void
  onUsage?: (usage: ChatCompletionUsage) => void
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

export interface UserChatModel {
  name: string
  platform: string
  group_ids: number[]
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

function readStreamDelta(body: any): string {
  const choice = body?.choices?.[0]
  return choice?.delta?.content || choice?.message?.content || ''
}

function readStreamUsage(body: any): ChatCompletionUsage | null {
  return body?.usage ?? null
}

function parseSSELines(buffer: string): { lines: string[]; rest: string } {
  const normalized = buffer.replace(/\r\n/g, '\n')
  const parts = normalized.split('\n')
  return {
    lines: parts.slice(0, -1),
    rest: parts[parts.length - 1] ?? ''
  }
}

export async function createChatCompletionStream(
  request: ChatCompletionRequest,
  callbacks: ChatCompletionStreamCallbacks = {}
): Promise<ChatCompletionUsage | null> {
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
      stream: true,
      stream_options: {
        include_usage: true
      }
    }),
    signal: request.signal
  })

  if (!response.ok) {
    const text = await response.text()
    const body = parseJsonBody(text)
    throw new Error(errorMessageFromBody(body, `Chat request failed with HTTP ${response.status}`))
  }

  if (!response.body) {
    throw new Error('Streaming response is not available')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let usage: ChatCompletionUsage | null = null
  let hasStreamContent = false

  const handleLine = (line: string) => {
    const trimmed = line.trim()
    if (!trimmed.startsWith('data:')) return

    const data = trimmed.slice(5).trim()
    if (!data || data === '[DONE]') return

    const body = parseJsonBody(data)
    const delta = readStreamDelta(body)
    if (delta) {
      hasStreamContent = true
      callbacks.onDelta?.(delta)
    }

    const nextUsage = readStreamUsage(body)
    if (nextUsage) {
      hasStreamContent = true
      usage = nextUsage
      callbacks.onUsage?.(nextUsage)
    }
  }

  while (true) {
    let result: ReadableStreamReadResult<Uint8Array>
    try {
      result = await reader.read()
    } catch (error) {
      if (hasStreamContent) {
        return usage
      }
      throw error
    }
    const { value, done } = result
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const parsed = parseSSELines(buffer)
    buffer = parsed.rest
    parsed.lines.forEach(handleLine)
  }

  buffer += decoder.decode()
  if (buffer.trim()) handleLine(buffer)

  return usage
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

export async function listUserChatModels(signal?: AbortSignal): Promise<UserChatModel[]> {
  const { data } = await apiClient.get<UserChatModel[]>('/user/chat/models', { signal })
  return Array.isArray(data) ? data : []
}

export const chatAPI = {
  createChatCompletion,
  createChatCompletionStream,
  listModels,
  listUserChatModels
}

export default chatAPI
