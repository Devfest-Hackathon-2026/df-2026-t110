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
  404: '指定した教材が見つかりません。',
  413: '依頼文が長すぎます。',
  422: '有効なプログラムを生成できませんでした。依頼文を変えて再試行してください。',
  429: '生成の利用枠を超えました。',
  501: 'サーバーの生成機能が設定されていません。',
  504: '生成が時間切れになりました。',
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
      errorMessages[response.status] ?? `プログラムの生成に失敗しました (HTTP ${response.status})`,
    )
  }
  if (!response.headers.get('content-type')?.includes('application/json')) {
    throw new Error(
      response.redirected
        ? 'ログイン画面へ転送されました。ログイン状態を確認してください。'
        : '生成APIからJSONが返されませんでした。',
    )
  }
  const result: unknown = await response.json()
  if (!isPicoProgram(result)) throw new Error('生成APIから有効なPicolangプログラムが返されませんでした。')
  return result
}
