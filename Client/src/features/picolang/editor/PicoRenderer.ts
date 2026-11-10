import * as Blockly from 'blockly'

export const PICO_RENDERER = 'pico'

class PicoDrawer extends Blockly.zelos.Drawer {
  protected override positionOutputConnection_(): void {
    super.positionOutputConnection_()
    const connection = this.block_.outputConnection
    if (!connection) return
    const { y } = connection.getOffsetInBlock()
    const x = this.info_.width / 2
    connection.setOffsetInBlock(this.info_.RTL ? -x : x, y)
  }

  // Zelos uses inline value inputs even when a block has multiple rows.
  protected override positionInlineInputConnection_(input: Blockly.blockRendering.InlineInput): void {
    super.positionInlineInputConnection_(input)
    const connection = input.connectionModel
    const { y } = connection.getOffsetInBlock()
    // The measured width covers either the empty socket or its nested block.
    const x = input.xPos + input.width / 2
    connection.setOffsetInBlock(this.info_.RTL ? -x : x, y)
  }

  override drawConnectionHighlightPath(
    measurable: Blockly.blockRendering.Connection,
  ): SVGElement | undefined {
    const highlight = super.drawConnectionHighlightPath(measurable)
    if (!highlight) return highlight

    // Logical value anchors are centered; Zelos highlight paths still start
    // at the left edge of the socket or output cap.
    let x: number
    if (measurable instanceof Blockly.blockRendering.InlineInput) {
      x = measurable.xPos + measurable.connectionWidth + measurable.connectionOffsetX
    } else if (measurable instanceof Blockly.blockRendering.OutputConnection) {
      x = this.info_.startX + measurable.connectionOffsetX
    } else return highlight

    const { y } = measurable.connectionModel.getOffsetInBlock()
    highlight.setAttribute(
      'transform',
      `translate(${this.info_.RTL ? -x : x}, ${y})${this.info_.RTL ? ' scale(-1 1)' : ''}`,
    )
    return highlight
  }
}

class PicoRenderer extends Blockly.zelos.Renderer {
  protected override makeDrawer_(
    block: Blockly.BlockSvg,
    info: Blockly.blockRendering.RenderInfo,
  ): Blockly.zelos.Drawer {
    return new PicoDrawer(block, info as Blockly.zelos.RenderInfo)
  }
}

export function registerPicoRenderer(): void {
  if (!Blockly.registry.hasItem(Blockly.registry.Type.RENDERER, PICO_RENDERER)) {
    Blockly.blockRendering.register(PICO_RENDERER, PicoRenderer)
  }
}
