import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createChatCompletionStream,
  createChatCompletionStreamWithContinuation,
  type ChatCompletionRequest
} from '@/api/chat'

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

  it('continues a length-limited answer and accumulates usage', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(streamResponse([
        'data: {"choices":[{"delta":{"content":"第一段"}}]}\n\n',
        'data: {"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}\n\n',
        'data: [DONE]\n\n'
      ]))
      .mockResolvedValueOnce(streamResponse([
        'data: {"choices":[{"delta":{"content":"第二段"}}]}\n\n',
        'data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":35,"completion_tokens":5,"total_tokens":40}}\n\n',
        'data: [DONE]\n\n'
      ]))
    const deltas: string[] = []

    const result = await createChatCompletionStreamWithContinuation(
      request,
      { onDelta: (delta) => deltas.push(delta) },
      { maxContinuations: 3 }
    )

    expect(deltas).toEqual(['第一段', '第二段'])
    expect(result.finishReason).toBe('stop')
    expect(result.continuationCount).toBe(1)
    expect(result.usage).toEqual({
      prompt_tokens: 45,
      completion_tokens: 25,
      total_tokens: 70
    })

    const [, secondInit] = vi.mocked(fetch).mock.calls[1]
    const secondBody = JSON.parse(secondInit?.body as string)
    expect(secondBody.messages).toEqual([
      ...request.messages,
      { role: 'assistant', content: '第一段' },
      { role: 'user', content: '请从上一条回答的中断处继续，只输出尚未输出的内容，不要重复已有内容。' }
    ])
  })

  it('asks for a direct answer when reasoning exhausts the first output limit', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(streamResponse([
        'data: {"choices":[{"delta":{"reasoning_content":"分析"}}]}\n\n',
        'data: {"choices":[{"delta":{},"finish_reason":"length"}]}\n\n',
        'data: [DONE]\n\n'
      ]))
      .mockResolvedValueOnce(streamResponse([
        'data: {"choices":[{"delta":{"content":"最终答案"},"finish_reason":"stop"}]}\n\n',
        'data: [DONE]\n\n'
      ]))

    const result = await createChatCompletionStreamWithContinuation(
      request,
      {},
      { maxContinuations: 1 }
    )

    expect(result.finishReason).toBe('stop')
    expect(result.hasReasoning).toBe(true)
    const [, secondInit] = vi.mocked(fetch).mock.calls[1]
    const secondBody = JSON.parse(secondInit?.body as string)
    expect(secondBody.messages).toEqual([
      ...request.messages,
      { role: 'user', content: '请直接回答上一条请求。上一轮未产生可见答案，请控制推理长度并优先给出完整的最终答案。' }
    ])
  })

  it('stops automatically continuing at the configured limit', async () => {
    vi.mocked(fetch).mockImplementation(async () => streamResponse([
      'data: {"choices":[{"delta":{},"finish_reason":"length"}]}\n\n',
      'data: [DONE]\n\n'
    ]))

    const result = await createChatCompletionStreamWithContinuation(
      request,
      {},
      { maxContinuations: 2 }
    )

    expect(fetch).toHaveBeenCalledTimes(3)
    expect(result.finishReason).toBe('length')
    expect(result.continuationCount).toBe(2)
  })
})
