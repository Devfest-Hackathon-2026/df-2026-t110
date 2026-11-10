import * as Blockly from 'blockly'

export class PersistentToolbox extends Blockly.Toolbox {
  private touchActivation: {
    item: Blockly.ISelectableToolboxItem
    wasSelected: boolean
    pointerId: number
  } | null = null

  protected override attachEvents_(container: HTMLDivElement, contents: HTMLDivElement): void {
    super.attachEvents_(container, contents)
    this.boundEvents_.push(
      Blockly.browserEvents.bind(container, 'click', this, this.onTouchClick),
      Blockly.browserEvents.bind(container, 'pointercancel', this, (event: PointerEvent) => {
        if (this.touchActivation?.pointerId === event.pointerId) this.touchActivation = null
      }),
    )
  }

  protected override onClick_(event: PointerEvent): void {
    this.touchActivation = null
    if (event.pointerType === 'touch') {
      const item = this.itemForEvent(event)
      if (!event.isPrimary || item) {
        if (event.isPrimary && item) {
          this.touchActivation = {
            item,
            wasSelected: this.getSelectedItem() === item,
            pointerId: event.pointerId,
          }
        }
        // Defer category toggling until Safari has applied native touch focus.
        Blockly.Touch.clearTouchIdentifier()
        return
      }
    }
    super.onClick_(event)
  }

  private itemForEvent(event: Event): Blockly.ISelectableToolboxItem | null {
    let target = event.target
    while (target instanceof Element && target !== this.HtmlDiv) {
      const item = this.getToolboxItemById(target.id)
      if (item?.isSelectable()) return item as Blockly.ISelectableToolboxItem
      target = target.parentElement
    }
    return null
  }

  private onTouchClick(event: Event): void {
    const activation = this.touchActivation
    this.touchActivation = null
    if (!activation || !this.isVisible_ || activation.item !== this.itemForEvent(event)) return

    const { item, wasSelected } = activation
    item.onClick(event)
    // Keep focus on the category so delayed Safari focus cannot select it again.
    Blockly.getFocusManager().focusNode(item)
    if (wasSelected) this.clearSelection()
    else if (this.getSelectedItem() !== item) this.setSelectedItem(item)
    this.workspace_.hideChaff(!wasSelected)
    event.preventDefault()
  }

  protected override onKeyDown_(event: KeyboardEvent): void {
    this.touchActivation = null
    super.onKeyDown_(event)
  }

  protected override createFlyout_(): Blockly.IFlyout {
    const options = this.workspace_.copyOptionsForFlyout()
    options.toolboxPosition = this.workspace_.options.toolboxPosition
    return new PersistentFlyout(options)
  }
}

class PersistentFlyout extends Blockly.VerticalFlyout {
  override autoClose = false

  override init(targetWorkspace: Blockly.WorkspaceSvg): void {
    super.init(targetWorkspace)
    const workspace = this.getWorkspace()
    // Wide blocks can scroll horizontally without covering the main workspace.
    workspace.scrollbar?.dispose()
    workspace.scrollbar = new Blockly.ScrollbarPair(
      workspace,
      true,
      true,
      'blocklyFlyoutScrollbar',
      this.SCROLLBAR_MARGIN,
    )
    workspace.scrollbar.setContainerVisible(this.isVisible())
  }

  override getWidth(): number {
    // Blockly reserves space for non-autoclosing flyouts, even when hidden.
    if (!this.isVisible()) return 0
    const metrics = this.getTargetWorkspace().getMetricsManager()
    const availableWidth = metrics.getSvgMetrics().width - metrics.getToolboxMetrics().width
    // Keep at least half of the available area usable for dragging blocks in.
    return Math.min(super.getWidth(), Math.max(0, availableWidth / 2))
  }

  protected override setMetrics_(ratio: { x: number; y: number }): void {
    super.setMetrics_(ratio)
    if (!this.isVisible() || typeof ratio.x !== 'number') return
    const workspace = this.getWorkspace()
    const metrics = workspace.getMetricsManager()
    const scroll = metrics.getScrollMetrics()
    const view = metrics.getViewMetrics()
    const absolute = metrics.getAbsoluteMetrics()
    workspace.scrollX = -(scroll.left + Math.max(0, scroll.width - view.width) * ratio.x)
    workspace.translate(workspace.scrollX + absolute.left, workspace.scrollY + absolute.top)
  }

  override scrollToStart(): void {
    super.scrollToStart()
    this.getWorkspace().scrollbar?.hScroll?.set(0)
  }

  override hide(): void {
    const wasVisible = this.isVisible()
    super.hide()
    if (wasVisible) this.getTargetWorkspace().resizeContents()
  }
}
