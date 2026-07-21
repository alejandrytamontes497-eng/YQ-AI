import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createChatCompletionStream, type ChatCompletionRequest } from '@/api/chat'

const request: ChatCompletionRequest = {
  apiKey: 'test-key',
  model: 'test-model',
  messages: [{ role: 'user', content: '推荐一道题' }]
}

function streamResponse(chunks: string[]): Response {
  const encoder = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)))
      controller.close()
    }
  })

  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    body
  } as Response
}

describe('chat completion stream parsing', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('reads text content parts and finish metadata from SSE chunks', async () => {
    vi.mocked(fetch).mockResolvedValue(streamResponse([
      'data: {"choices":[{"delta":{"content":[{"type":"text","text":"经典"},',
      '{"type":"output_text","text":"题"}]}}]}\n\n',
      'data: {"choices":[{"delta":{"reasoning_content":"分析"}}]}\n\n',
      'data: {"choices":[{"delta":{},"finish_reason":"length"}]}\n\n',
      'data: {"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}\n\n',
      'data: [DONE]\n\n'
    ]))
    const deltas: string[] = []

    const result = await createChatCompletionStream(request, {
      onDelta: (delta) => deltas.push(delta)
    })

    expect(deltas).toEqual(['经典题'])
    expect(result).toEqual({
      usage: { prompt_tokens: 10, completion_tokens: 20, total_tokens: 30 },
      finishReason: 'length',
      hasReasoning: true,
      hasToolCalls: false,
      receivedData: true
    })
  })

  it('forwards the configured output token limit', async () => {
    vi.mocked(fetch).mockResolvedValue(streamResponse(['data: [DONE]\n\n']))

    await createChatCompletionStream({ ...request, max_tokens: 32768 })

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(JSON.parse(init?.body as string)).toMatchObject({
      max_tokens: 32768,
      stream: true
    })
  })

  it('uses a JSON completion when an upstream ignores stream mode', async () => {
    const body = JSON.stringify({
      choices: [{
        message: {
          role: 'assistant',
          content: [
            { type: 'text', text: '第一段' },
            { type: 'output_text', text: '第二段' }
          ]
        },
        finish_reason: 'stop'
      }],
      usage: { total_tokens: 8 }
    }, null, 2)
    vi.mocked(fetch).mockResolvedValue(streamResponse([body.slice(0, 40), body.slice(40)]))
    const deltas: string[] = []

    const result = await createChatCompletionStream(request, {
      onDelta: (delta) => deltas.push(delta)
    })

    expect(deltas).toEqual(['第一段第二段'])
    expect(result.finishReason).toBe('stop')
    expect(result.usage).toEqual({ total_tokens: 8 })
    expect(result.receivedData).toBe(true)
  })

  it('recognizes refusal text as displayable output', async () => {
    vi.mocked(fetch).mockResolvedValue(streamResponse([
      'data: {"choices":[{"delta":{"refusal":"无法回答该请求"}}]}\n\n',
      'data: [DONE]\n\n'
    ]))
    const onDelta = vi.fn()

    const result = await createChatCompletionStream(request, { onDelta })

    expect(onDelta).toHaveBeenCalledWith('无法回答该请求')
    expect(result.receivedData).toBe(true)
  })

  it('rejects a successful HTTP response with an unknown body format', async () => {
    vi.mocked(fetch).mockResolvedValue(streamResponse(['not-json']))

    await expect(createChatCompletionStream(request)).rejects.toThrow('模型返回了无法识别的响应格式')
  })
})
