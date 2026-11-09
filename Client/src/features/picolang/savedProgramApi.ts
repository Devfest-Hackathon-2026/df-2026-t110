import { isPicoProgram, type PicoProgram } from './language/picolang.ts'
import { parsePicolangExecutionJson } from './language/executionJson.ts'

const base = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export interface SavedProgram {
  id: string
  name: string
  source: PicoProgram
  isBattle: boolean
  tutorialID?: string
}

export class SavedProgramApiError extends Error {
  readonly status: number

  constructor(status: number) {
    super(
      status === 401
        ? '保存済みプログラムの利用にはログインが必要です。'
        : status === 403
          ? 'このプログラムを編集する権限がありません。'
          : status === 404
            ? '保存済みプログラムが見つかりません。'
            : '保存済みプログラムを取得・更新できませんでした。',
    )
    this.status = status
  }
}

function savedProgram(value: unknown): SavedProgram {
  if (
    !value ||
    typeof value !== 'object' ||
    Array.isArray(value) ||
    typeof (value as SavedProgram).id !== 'string' ||
    typeof (value as SavedProgram).name !== 'string' ||
    typeof (value as SavedProgram).isBattle !== 'boolean' ||
    !isPicoProgram((value as SavedProgram).source)
  )
    throw new Error('保存済みプログラムの形式が不正です。')
  const program = value as SavedProgram
  if (program.tutorialID !== undefined && typeof program.tutorialID !== 'string')
    throw new Error('保存済みプログラムの教材IDが不正です。')
  return program
}

async function readJson(response: Response): Promise<unknown> {
  if (!response.ok) throw new SavedProgramApiError(response.status)
  if (!response.headers.get('content-type')?.includes('json'))
    throw new Error('プログラムAPIからJSONが返されませんでした。')
  // Keep large integer literals from silently changing when parsed by JavaScript.
  return parsePicolangExecutionJson(await response.text())
}

export async function getSavedProgram(id: string, signal?: AbortSignal): Promise<SavedProgram> {
  const response = await fetch(`${base}/program/${encodeURIComponent(id)}`, {
    credentials: 'include',
    cache: 'no-store',
    signal,
  })
  return savedProgram(await readJson(response))
}

export async function updateSavedProgram(
  id: string,
  name: string,
  source: PicoProgram,
  signal?: AbortSignal,
): Promise<SavedProgram> {
  const response = await fetch(`${base}/program/${encodeURIComponent(id)}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, source }),
    signal,
  })
  return savedProgram(await readJson(response))
}

export async function listTutorialPrograms(
  tutorialId: string,
  signal?: AbortSignal,
): Promise<SavedProgram[]> {
  const response = await fetch(`${base}/tutorial/${encodeURIComponent(tutorialId)}/program`, {
    credentials: 'include',
    cache: 'no-store',
    signal,
  })
  const value = await readJson(response)
  if (!Array.isArray(value)) throw new Error('保存済みプログラム一覧の形式が不正です。')
  return value.map(savedProgram)
}
