import { executePicolang } from '../picolang/executePicolang.ts'
import { isPicoProgram, type PicoProgram } from '../picolang/picolang.ts'
import { normalizeStage } from '../stage/stage.ts'
import { LOCAL_LESSONS } from '../tutorial/tutorialLessons.ts'
import { createTutorialWorld } from '../tutorial/tutorialWorld.ts'
import type { ProgramRecord } from './matchApi.ts'

const key = (fieldId: string): string => `match:guest:programs:${fieldId}`

export function guestFieldIds(): string[] {
  return LOCAL_LESSONS.map((lesson) => lesson.fieldId)
}

export function loadGuestPrograms(fieldId: string): ProgramRecord[] {
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(key(fieldId)) ?? '[]')
    if (!Array.isArray(raw)) return []
    return raw.filter(
      (item): item is ProgramRecord =>
        item !== null &&
        typeof item === 'object' &&
        typeof item.id === 'string' &&
        typeof item.name === 'string' &&
        isPicoProgram(item.source),
    )
  } catch {
    return []
  }
}

export function saveGuestProgram(fieldId: string, name: string, source: PicoProgram): ProgramRecord {
  const saved = { id: crypto.randomUUID(), name: name.trim(), source }
  localStorage.setItem(key(fieldId), JSON.stringify([...loadGuestPrograms(fieldId), saved]))
  return saved
}

export async function runGuestProgram(fieldId: string, source: PicoProgram): Promise<boolean> {
  const lesson = LOCAL_LESSONS.find((item) => item.fieldId === fieldId)
  if (!lesson) throw new Error('ローカル練習で使えるフィールドを選んでください。')
  const stage = normalizeStage({ id: lesson.id, title: lesson.title, map: lesson.map })
  const world = createTutorialWorld(stage)
  await executePicolang(
    source,
    {
      call: async (name, args) => world.call(name, args).value,
    },
    new AbortController().signal,
    2000,
  )
  return world.reachedGoal
}
