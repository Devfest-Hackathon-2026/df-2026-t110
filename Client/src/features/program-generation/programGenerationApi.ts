import { isPicoProgram, type PicoTypedProgram } from '../picolang/language/picolang.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface ProgramGenerationRequest {
  prompt: string
  fieldId?: string
  isBattle?: boolean
}

export class ProgramGenerationApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

const errorMessages: Record<number, string> = {
  400: '生成依頼の内容を確認してください。',
  401: 'プログラムの生成にはログインが必要です。',
  404: '指定したチュートリアルが見つかりません。',
  413: '依頼文が長すぎます。',
  422: '有効なプログラムを生成できませんでした。依頼文を変えて再試行してください。',
  429: 'プログラム生成の利用上限に達しました。',
  501: 'プログラム生成は現在利用できません。',
  504: 'プログラム生成が時間内に終わりませんでした。もう一度お試しください。',
}

export async function generateProgram(
  request: ProgramGenerationRequest,
  signal: AbortSignal,
): Promise<PicoTypedProgram> {
  const response = await fetch(`${base}/program/create-with-vibe`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
    signal,
  })
  if (!response.ok) {
    throw new ProgramGenerationApiError(
      response.status,
      errorMessages[response.status] ?? 'プログラムを生成できませんでした。もう一度お試しください。',
    )
  }
  if (!response.headers.get('content-type')?.includes('application/json')) {
    throw new ProgramGenerationApiError(
      response.status,
      response.redirected
        ? 'プログラムを生成するには、ログインし直してください。'
        : '生成したプログラムを受け取れませんでした。もう一度お試しください。',
    )
  }
  const result: unknown = await response.json()
  if (!isPicoProgram(result))
    throw new ProgramGenerationApiError(
      response.status,
      '生成したプログラムを使用できませんでした。文章を変えて、もう一度お試しください。',
    )
  return result
}
