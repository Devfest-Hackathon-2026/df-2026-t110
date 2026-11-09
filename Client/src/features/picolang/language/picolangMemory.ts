import type { components } from '../../../api/schema.d.ts'
import { formatPicoValue } from './formatPicoValue.ts'

type DebugEvent = components['schemas']['event.schema']
type MemoryValue =
  | Exclude<components['schemas']['memory-value.schema'], { type: 'integer' }>
  | { type: 'integer'; value: number | string }

// Replay memory operations, never source code. IDs remain strings (uint64).
export class TraceMemory {
  private entries = new Map<string, { value: MemoryValue | null; refs: number }>([
    ['0', { value: null, refs: 1 }],
  ])
  private entry(id: string) {
    const entry = this.entries.get(id)
    if (!entry) throw new Error(`実行履歴の値 ${id} が見つかりません`)
    return entry
  }
  apply(event: DebugEvent): void {
    if (event.kind === 'memory_allocate') {
      if (this.entries.has(event.id)) throw new Error('実行履歴の値IDが重複しています')
      if (event.value.type === 'list_node') {
        this.entry(event.value.prev).refs++
        this.entry(event.value.value).refs++
      }
      if (event.value.type === 'some') this.entry(event.value.value).refs++
      this.entries.set(event.id, { value: event.value, refs: 1 })
    } else if (event.kind === 'memory_retain') this.entry(event.id).refs++
    else if (event.kind === 'memory_release') {
      const pending = [event.id]
      while (pending.length) {
        const id = pending.pop()!
        const entry = this.entry(id)
        if (id === '0' && entry.refs === 1) throw new Error('実行履歴の空スタックの参照が不正です')
        if (--entry.refs) continue
        this.entries.delete(id)
        if (entry.value?.type === 'list_node') pending.push(entry.value.prev, entry.value.value)
        if (entry.value?.type === 'some') pending.push(entry.value.value)
      }
    }
  }
  format(id: string): string {
    let remaining = 128
    const render = (id: string, depth: number): string => {
      if (--remaining < 0 || depth > 16) return '…'
      const value = this.entry(id).value
      if (value === null) return '[]'
      if (value.type === 'none') return 'none'
      if (value.type === 'some') return `some(${render(value.value, depth + 1)})`
      if (value.type !== 'list_node') return formatPicoValue(value.value)
      const items: string[] = []
      let next = id
      while (next !== '0') {
        if (remaining <= 0) {
          items.push('…')
          break
        }
        const node = this.entry(next).value
        if (node?.type !== 'list_node') throw new Error('実行履歴のスタックが不正です')
        items.push(render(node.value, depth + 1))
        next = node.prev
      }
      return `[${items.reverse().join(', ')}]`
    }
    return render(id, 0)
  }
}
