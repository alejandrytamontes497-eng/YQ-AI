import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

describe('image jobs API', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: { id: 'job-1' } })
  })

  it('requests URL output so the backend avoids large base64 gateway responses', async () => {
    const { createImageJob } = await import('@/api/images')

    await createImageJob({
      model: 'gpt-image-2-4k',
      prompt: 'draw',
      size: '3840x2160',
      quality: 'high',
      n: 1
    })

    expect(post).toHaveBeenCalledWith(
      '/user/images/jobs',
      expect.objectContaining({ response_format: 'url' }),
      expect.any(Object)
    )
  })
})
