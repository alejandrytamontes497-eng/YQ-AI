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

const ERROR_MESSAGE_PREVIEW_LIMIT = 1200

function compactWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

function decodeHtmlEntities(value: string): string {
  const namedEntities: Record<string, string> = {
    amp: '&',
    gt: '>',
    lt: '<',
    nbsp: ' ',
    quot: '"'
  }

  return value.replace(/&#(\d+);|&#x([0-9a-f]+);|&([a-z]+);/gi, (_, decimal, hex, named) => {
    if (decimal) return String.fromCodePoint(Number(decimal))
    if (hex) return String.fromCodePoint(parseInt(hex, 16))
    return namedEntities[String(named).toLowerCase()] ?? `&${named};`
  })
}

function stripHtml(text: string): string {
  return decodeHtmlEntities(
    text
      .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, ' ')
      .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, ' ')
      .replace(/<[^>]+>/g, ' ')
  )
}

function looksLikeHtml(text: string): boolean {
  const preview = text.trim().slice(0, 300).toLowerCase()
  return (
    preview.startsWith('<!doctype') ||
    preview.startsWith('<html') ||
    preview.includes('<html') ||
    preview.includes('<body') ||
    preview.includes('<head')
  )
}

function htmlStatusLabel(text: string, status?: number, statusText?: string): string {
  const visibleText = compactWhitespace(stripHtml(text)).slice(0, 600)
  const match = visibleText.match(/\b([45]\d{2})\s*:?\s*(Bad Gateway|Service Unavailable|Gateway Timeout|Internal Server Error|Too Many Requests|Forbidden|Unauthorized|Not Found|Bad Request)?\b/i)
  if (match) {
    const code = match[1]
    const label = compactWhitespace(match[2] ?? '')
    return label ? `HTTP ${code} ${label}` : `HTTP ${code}`
  }

  if (status && status >= 400) {
    const label = compactWhitespace(statusText ?? '')
    return label ? `HTTP ${status} ${label}` : `HTTP ${status}`
  }

  return ''
}

function htmlErrorMessage(text: string, status?: number, statusText?: string): string {
  const statusLabel = htmlStatusLabel(text, status, statusText)
  return statusLabel ? `上游服务暂时不可用（${statusLabel}）` : '上游服务暂时不可用'
}

function parseJsonBody(text: string): any {
  if (!text) return null

  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

function normalizeErrorMessage(value: unknown, fallback: string, status?: number, statusText?: string): string {
  const raw = typeof value === 'string' ? value : value == null ? '' : String(value)
  const message = raw.trim()
  if (!message) return fallback
  if (looksLikeHtml(message)) return htmlErrorMessage(message, status, statusText)
  if (message.length > ERROR_MESSAGE_PREVIEW_LIMIT) {
    return `${message.slice(0, ERROR_MESSAGE_PREVIEW_LIMIT)}...`
  }
  return message
}

function errorMessageFromBody(body: any, fallback: string, status?: number, statusText?: string): string {
  const message = body?.error?.message || body?.message || body?.detail || (typeof body === 'string' ? body : '')
  return normalizeErrorMessage(message, fallback, status, statusText)
}

function errorMessageFromText(text: string, fallback: string, status?: number, statusText?: string): string {
  const body = parseJsonBody(text)
  if (body) return errorMessageFromBody(body, fallback, status, statusText)
  return normalizeErrorMessage(text, fallback, status, statusText)
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
    throw new Error(errorMessageFromText(text, `Chat request failed with HTTP ${response.status}`, response.status, response.statusText))
  }

  if (!body) {
    throw new Error(errorMessageFromText(text, 'Chat request returned an invalid response', response.status, response.statusText))
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

function readStreamError(body: any): string {
  if (!body?.error && !body?.message && !body?.detail) return ''
  return errorMessageFromBody(body, '')
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
    throw new Error(errorMessageFromText(text, `Chat request failed with HTTP ${response.status}`, response.status, response.statusText))
  }

  if (!response.body) {
    throw new Error('Streaming response is not available')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let responsePreview = ''
  let usage: ChatCompletionUsage | null = null
  let hasStreamContent = false
  let hasDataLine = false
  let streamDone = false

  const handleLine = (line: string) => {
    const trimmed = line.trim()
    if (!trimmed.startsWith('data:')) return
    hasDataLine = true

    const data = trimmed.slice(5).trim()
    if (!data) return
    if (data === '[DONE]') {
      streamDone = true
      return
    }

    const body = parseJsonBody(data)
    if (!body) {
      if (looksLikeHtml(data)) {
        throw new Error(htmlErrorMessage(data, response.status, response.statusText))
      }
      return
    }

    const streamError = readStreamError(body)
    if (streamError) {
      throw new Error(streamError)
    }

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

    const chunk = decoder.decode(value, { stream: true })
    if (responsePreview.length < ERROR_MESSAGE_PREVIEW_LIMIT) {
      responsePreview += chunk.slice(0, ERROR_MESSAGE_PREVIEW_LIMIT - responsePreview.length)
    }
    buffer += chunk
    const parsed = parseSSELines(buffer)
    buffer = parsed.rest
    parsed.lines.forEach(handleLine)
    if (streamDone) {
      try {
        await reader.cancel()
      } catch {
        // The server already sent the terminal event; ignore transport cleanup noise.
      }
      break
    }
  }

  if (!streamDone) {
    const tail = decoder.decode()
    if (responsePreview.length < ERROR_MESSAGE_PREVIEW_LIMIT) {
      responsePreview += tail.slice(0, ERROR_MESSAGE_PREVIEW_LIMIT - responsePreview.length)
    }
    buffer += tail
    if (buffer.trim()) handleLine(buffer)
  }

  if (!hasDataLine && looksLikeHtml(responsePreview)) {
    throw new Error(htmlErrorMessage(responsePreview, response.status, response.statusText))
  }

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
    throw new Error(errorMessageFromText(text, `Load models failed with HTTP ${response.status}`, response.status, response.statusText))
  }

  if (!body) {
    throw new Error(errorMessageFromText(text, 'Load models returned an invalid response', response.status, response.statusText))
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
