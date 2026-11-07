export interface RobotExecutionResult {
  interpreterId: string
  hintId: string
  reason: string
  message: string
}

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

// History may also contain field events, older debug events and future event kinds.
export function robotExecutionResults(history: unknown[]): RobotExecutionResult[] {
  const results: RobotExecutionResult[] = []
  for (const entry of history) {
    if (!isObject(entry) || entry.type !== 'picolang' || !isObject(entry.event)) continue
    const event = entry.event
    if (
      event.kind !== 'execution_finished' ||
      typeof entry.interpreterId !== 'string' ||
      !entry.interpreterId ||
      typeof event.hintId !== 'string' ||
      !event.hintId ||
      typeof event.reason !== 'string' ||
      !event.reason ||
      typeof event.message !== 'string' ||
      !event.message
    )
      continue
    results.push({
      interpreterId: entry.interpreterId,
      hintId: event.hintId,
      reason: event.reason,
      message: event.message,
    })
  }
  return results
}

export function isExecutionFailure(result: RobotExecutionResult): boolean {
  return result.reason !== 'completed' && result.reason !== 'interrupted'
}

export function executionReasonText(result: RobotExecutionResult): string {
  switch (result.reason) {
    case 'completed':
      return 'プログラムの実行が正常に終了しました。'
    case 'interrupted':
      return 'プログラムの実行が打ち切られました。'
    case 'division_by_zero':
      return '0で割る計算があったため停止しました。'
    case 'step_limit':
      return '実行ステップ数の上限に達したため停止しました。'
    case 'memory_limit':
      return 'メモリ使用量の上限に達したため停止しました。'
    case 'call_depth_limit':
      return '関数呼び出しの深さの上限に達したため停止しました。'
    default:
      return `実行中にエラーが発生したため停止しました。${result.message}`
  }
}
