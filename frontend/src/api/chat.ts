import type { ApiKey } from '@/types'

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

export function defaultModelForKey(apiKey: ApiKey | null | undefined): string {
  switch (apiKey?.group?.platform) {
    case 'openai':
      return 'gpt-4o-mini'
    case 'gemini':
      return 'gemini-2.5-flash'
    case 'grok':
      return 'grok-3-mini'
    case 'antigravity':
      return 'claude-sonnet-4-5-20250929'
    case 'anthropic':
    default:
      return 'claude-sonnet-4-5-20250929'
  }
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
  let body: any = null

  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = { error: { message: text } }
    }
  }

  if (!response.ok) {
    const message =
      body?.error?.message ||
      body?.message ||
      body?.detail ||
      `Chat request failed with HTTP ${response.status}`
    throw new Error(message)
  }

  return body as ChatCompletionResponse
}

export const chatAPI = {
  createChatCompletion,
  defaultModelForKey
}

export default chatAPI
