import { ref, shallowRef } from 'vue'
import type { PicoProgram } from '../../language/picolang.ts'
import type { AnimationEvent } from '../../../stage/robotAnimator.ts'
import type { StageLayout } from '../../../stage/stage.ts'
import type { PicolangTrace } from '../../language/picolangTrace.ts'
import { picolangTraceSteps } from './picolangTimeline.ts'
import { actionAnimation, actionSpeech, applyReplayFrame } from './fieldReplay.ts'
import type { PicolangSpeech } from './picolangSpeech.ts'

export interface PicolangPlaybackTarget {
  stage: StageLayout
  refreshVisuals(): void
  setCarrying(carrying: boolean): void
}
function wait(duration: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException('再生を中断しました', 'AbortError'))
      return
    }
    const abort = () => {
      clearTimeout(timer)
      reject(new DOMException('再生を中断しました', 'AbortError'))
    }
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', abort)
      resolve()
    }, duration)
    signal.addEventListener('abort', abort, { once: true })
  })
}

export function usePicolangPlayback() {
  const executionTrace = shallowRef<PicolangTrace | null>(null)
  const animationEvent = shallowRef<AnimationEvent | null>(null)
  const speech = shallowRef<PicolangSpeech | null>(null)
  const playbackReady = ref(false)
  const playbackPaused = ref(false)
  const nodeInterval = ref(300)
  let credits = 0
  let animationId = 0
  let active: AbortController | null = null

  function reset(): void {
    active?.abort()
    active = null
    executionTrace.value = null
    animationEvent.value = null
    speech.value = null
    playbackReady.value = false
    playbackPaused.value = false
    credits = 0
  }
  function stepPlayback(): void {
    playbackPaused.value = true
    credits++
  }
  function togglePause(): void {
    playbackPaused.value = !playbackPaused.value
  }
  async function permission(signal: AbortSignal): Promise<void> {
    while (playbackPaused.value && credits === 0) await wait(50, signal)
    if (playbackPaused.value) credits--
    else credits = 0
    if (signal.aborted) throw new DOMException('再生を中断しました', 'AbortError')
  }
  async function play(
    result: { history: unknown[]; summary: { cleared: boolean } },
    program: PicoProgram,
    target: PicolangPlaybackTarget,
    signal: AbortSignal,
  ): Promise<void> {
    reset()
    const controller = new AbortController()
    active = controller
    const abort = () => controller.abort()
    signal.addEventListener('abort', abort, { once: true })
    if (signal.aborted) abort()
    const playbackSignal = controller.signal
    playbackReady.value = true
    try {
      for (const step of picolangTraceSteps(result.history, program)) {
        await permission(playbackSignal)
        if (step.kind === 'debug') {
          executionTrace.value = step.trace
          await wait(nodeInterval.value, playbackSignal)
          continue
        }
        const frame = step.frame
        for (const action of frame.actions) {
          const id = ++animationId
          const message = actionSpeech(action, id)
          if (message) {
            speech.value = message
            await wait(1000, playbackSignal)
            speech.value = null
            continue
          }
          const animation = actionAnimation(action, id)
          if (!animation) continue
          animationEvent.value = animation
          await wait(
            animation.type === 'wait' ? Math.max(1, animation.ticks ?? 1) * 1000 + 40 : 1040,
            playbackSignal,
          )
        }
        if (applyReplayFrame(target.stage, frame)) target.refreshVisuals()
        target.setCarrying(Boolean(frame.robots[0]?.carryingLuggage))
      }
      if (playbackSignal.aborted) throw new DOMException('再生を中断しました', 'AbortError')
      if (result.summary.cleared) animationEvent.value = { id: ++animationId, type: 'clear' }
    } finally {
      signal.removeEventListener('abort', abort)
      if (active === controller) {
        active = null
        playbackReady.value = false
        speech.value = null
      }
    }
  }
  return {
    executionTrace,
    animationEvent,
    speech,
    playbackReady,
    playbackPaused,
    nodeInterval,
    play,
    reset,
    stepPlayback,
    togglePause,
  }
}
