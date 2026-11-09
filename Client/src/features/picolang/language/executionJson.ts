export function parsePicolangExecutionJson(json: string): unknown {
  return JSON.parse(
    json,
    function (this: Record<string, unknown>, key: string, value: unknown, context?: { source: string }) {
      // The VM uses i64. Preserve the original decimal digits for the watch panel.
      if (
        key === 'value' &&
        this.type === 'integer' &&
        typeof value === 'number' &&
        !Number.isSafeInteger(value)
      ) {
        if (!context?.source) throw new Error('このブラウザでは大きな整数の実行履歴を正確に表示できません')
        return context.source
      }
      return value
    },
  )
}
