import * as Blockly from 'blockly'

export class PersistentToolbox extends Blockly.Toolbox {
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
