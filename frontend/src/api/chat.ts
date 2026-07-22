import { apiClient } from './client'

export type ChatRole = 'system' | 'user' | 'assistant'

export type ChatContentPart =
  | {
      type: 'text'
      text: string
    }
  | {
      type: 'image_url'
      image_url: {
        url: string
        detail?: 'auto' | 'low' | 'high'
      }
    }

export interface ChatMessage {
  role: ChatRole
  content: string | ChatContentPart[]
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

export interface ChatCompletionStreamResult {
  usage: ChatCompletionUsage | null
  finishReason: string
  hasReasoning: boolean
  hasToolCalls: boolean
  receivedData: boolean
  continuationCount?: number
}

export interface ChatCompletionContinuationOptions {
  maxContinuations?: number
  continuationPrompt?: string
  emptyContinuationPrompt?: string
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

function readContentText(content: unknown): string {
  if (typeof content === 'string') return content
  if (Array.isArray(content)) return content.map(readContentText).join('')
  if (!content || typeof content !== 'object') return ''

  const part = content as Record<string, unknown>
  if (typeof part.text === 'string') return part.text
  if (part.text && typeof part.text === 'object') {
    const value = (part.text as Record<string, unknown>).value
    if (typeof value === 'string') return value
  }
  if (typeof part.output_text === 'string') return part.output_text
  if (typeof part.refusal === 'string') return part.refusal
  if (part.content !== undefined) return readContentText(part.content)
  return ''
}

function readStreamDelta(body: any): string {
  const choice = body?.choices?.[0]
  const deltaText = readContentText(choice?.delta?.content)
  if (deltaText) return deltaText

  const refusal = readContentText(choice?.delta?.refusal)
  if (refusal) return refusal

  const messageText = readContentText(choice?.message?.content)
  if (messageText) return messageText

  const messageRefusal = readContentText(choice?.message?.refusal)
  if (messageRefusal) return messageRefusal

  if (body?.type === 'response.output_text.delta' && typeof body?.delta === 'string') {
    return body.delta
  }
  return ''
}

function readResponseText(body: any): string {
  const choiceText = readStreamDelta(body)
  if (choiceText) return choiceText
  if (typeof body?.output_text === 'string') return body.output_text
  if (Array.isArray(body?.output)) {
    return body.output
      .filter((item: any) => item?.type === 'message' || item?.content !== undefined)
      .map((item: any) => readContentText(item?.content))
      .join('')
  }
  return ''
}

function readStreamUsage(body: any): ChatCompletionUsage | null {
  return body?.usage ?? null
}

function readStreamError(body: any): string {
  if (!body?.error && typeof body?.message !== 'string' && typeof body?.detail !== 'string') return ''
  return errorMessageFromBody(body, '')
}

function readFinishReason(body: any): string {
  const choiceReason = body?.choices?.[0]?.finish_reason
  if (typeof choiceReason === 'string' && choiceReason) return choiceReason

  const response = body?.response ?? body
  if (response?.status === 'incomplete' || body?.type === 'response.incomplete') {
    const reason = response?.incomplete_details?.reason ?? body?.incomplete_details?.reason
    return reason === 'max_output_tokens' ? 'length' : 'incomplete'
  }
  if (response?.status === 'failed' || body?.type === 'response.failed') return 'failed'
  if (response?.status === 'completed' || body?.type === 'response.completed' || body?.type === 'response.done') return 'stop'
  return ''
}

function hasReasoningContent(body: any): boolean {
  const delta = body?.choices?.[0]?.delta
  const message = body?.choices?.[0]?.message
  return Boolean(
    readContentText(delta?.reasoning_content) ||
    readContentText(message?.reasoning_content) ||
    (typeof body?.type === 'string' && body.type.includes('reasoning') && readContentText(body?.delta))
  )
}

function hasToolCallContent(body: any): boolean {
  const choice = body?.choices?.[0]
  if (Array.isArray(choice?.delta?.tool_calls) && choice.delta.tool_calls.length > 0) return true
  if (Array.isArray(choice?.message?.tool_calls) && choice.message.tool_calls.length > 0) return true
  return Array.isArray(body?.output) && body.output.some((item: any) =>
    item?.type === 'function_call' || item?.type === 'custom_tool_call'
  )
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
): Promise<ChatCompletionStreamResult> {
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
  let rawResponse = ''
  let responsePreview = ''
  let usage: ChatCompletionUsage | null = null
  let hasVisibleContent = false
  let hasDataLine = false
  let streamDone = false
  let finishReason = ''
  let hasReasoning = false
  let hasToolCalls = false

  const handleBody = (body: any) => {
    const streamError = readStreamError(body)
    if (streamError) {
      throw new Error(streamError)
    }

    const delta = readStreamDelta(body)
    if (delta) {
      hasVisibleContent = true
      callbacks.onDelta?.(delta)
    }

    const nextUsage = readStreamUsage(body)
    if (nextUsage) {
      usage = nextUsage
      callbacks.onUsage?.(nextUsage)
    }

    finishReason = readFinishReason(body) || finishReason
    hasReasoning = hasReasoningContent(body) || hasReasoning
    hasToolCalls = hasToolCallContent(body) || hasToolCalls
  }

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

    handleBody(body)
  }

  while (true) {
    let result: ReadableStreamReadResult<Uint8Array>
    try {
      result = await reader.read()
    } catch (error) {
      if (hasVisibleContent) {
        return { usage, finishReason, hasReasoning, hasToolCalls, receivedData: hasDataLine }
      }
      throw error
    }
    const { value, done } = result
    if (done) break

    const chunk = decoder.decode(value, { stream: true })
    if (responsePreview.length < ERROR_MESSAGE_PREVIEW_LIMIT) {
      responsePreview += chunk.slice(0, ERROR_MESSAGE_PREVIEW_LIMIT - responsePreview.length)
    }
    if (!hasDataLine) rawResponse += chunk
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
    if (!hasDataLine) rawResponse += tail
    buffer += tail
    if (buffer.trim()) handleLine(buffer)
  }

  if (!hasDataLine && looksLikeHtml(responsePreview)) {
    throw new Error(htmlErrorMessage(responsePreview, response.status, response.statusText))
  }

  if (!hasDataLine) {
    const body = parseJsonBody(rawResponse)
    if (!body) {
      throw new Error('模型返回了无法识别的响应格式。')
    }
    const responseError = readStreamError(body)
    if (responseError) throw new Error(responseError)

    const text = readResponseText(body)
    if (text) {
      hasVisibleContent = true
      callbacks.onDelta?.(text)
    }
    const nextUsage = readStreamUsage(body)
    if (nextUsage) {
      usage = nextUsage
      callbacks.onUsage?.(nextUsage)
    }
    finishReason = readFinishReason(body) || finishReason
    hasReasoning = hasReasoningContent(body) || hasReasoning
    hasToolCalls = hasToolCallContent(body) || hasToolCalls
  }

  return { usage, finishReason, hasReasoning, hasToolCalls, receivedData: hasDataLine || Boolean(rawResponse.trim()) }
}

const DEFAULT_CONTINUATION_PROMPT = '请从上一条回答的中断处继续，只输出尚未输出的内容，不要重复已有内容。'
const DEFAULT_EMPTY_CONTINUATION_PROMPT = '请直接回答上一条请求。上一轮未产生可见答案，请控制推理长度并优先给出完整的最终答案。'

function sumUsage(
  current: ChatCompletionUsage | null,
  next: ChatCompletionUsage | null
): ChatCompletionUsage | null {
  if (!current) return next ? { ...next } : null
  if (!next) return current

  const sum = (left?: number, right?: number) =>
    left === undefined && right === undefined ? undefined : (left ?? 0) + (right ?? 0)

  return {
    prompt_tokens: sum(current.prompt_tokens, next.prompt_tokens),
    completion_tokens: sum(current.completion_tokens, next.completion_tokens),
    total_tokens: sum(current.total_tokens, next.total_tokens)
  }
}

function continuationMessages(
  messages: ChatMessage[],
  generatedText: string,
  options: ChatCompletionContinuationOptions
): ChatMessage[] {
  const next = [...messages]
  if (generatedText.trim()) {
    next.push({ role: 'assistant', content: generatedText })
    next.push({
      role: 'user',
      content: options.continuationPrompt || DEFAULT_CONTINUATION_PROMPT
    })
  } else {
    next.push({
      role: 'user',
      content: options.emptyContinuationPrompt || DEFAULT_EMPTY_CONTINUATION_PROMPT
    })
  }
  return next
}

export async function createChatCompletionStreamWithContinuation(
  request: ChatCompletionRequest,
  callbacks: ChatCompletionStreamCallbacks = {},
  options: ChatCompletionContinuationOptions = {}
): Promise<ChatCompletionStreamResult> {
  const maxContinuations = Math.max(0, Math.floor(options.maxContinuations ?? 0))
  let continuationCount = 0
  let generatedText = ''
  let totalUsage: ChatCompletionUsage | null = null
  let requestMessages = request.messages
  let hasReasoning = false
  let hasToolCalls = false
  let receivedData = false

  while (true) {
    const result = await createChatCompletionStream(
      { ...request, messages: requestMessages },
      {
        onDelta: (delta) => {
          generatedText += delta
          callbacks.onDelta?.(delta)
        }
      }
    )

    totalUsage = sumUsage(totalUsage, result.usage)
    hasReasoning = hasReasoning || result.hasReasoning
    hasToolCalls = hasToolCalls || result.hasToolCalls
    receivedData = receivedData || result.receivedData
    if (totalUsage) callbacks.onUsage?.(totalUsage)

    if (result.finishReason !== 'length' || continuationCount >= maxContinuations) {
      return {
        ...result,
        usage: totalUsage,
        hasReasoning,
        hasToolCalls,
        receivedData,
        continuationCount
      }
    }

    continuationCount += 1
    requestMessages = continuationMessages(request.messages, generatedText, options)
  }
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
  createChatCompletionStreamWithContinuation,
  listModels,
  listUserChatModels
}

export default chatAPI
