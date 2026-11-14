/**
 * The two printed sides.
 *
 * renderSideB draws the envelope face: the address, the return address
 * and the stamp, all authored in the panel frame and printed through the
 * scene's sheet placement, which is where the diagonal comes from.
 *
 * renderSideA draws the letter. Side A is the mirror of the sheet frame,
 * so its placement is mirrored back before the text is drawn; the result
 * reads correctly when the sheet is turned over.
 */

import type { Letter, Scene } from '../../bindings';
import { addressLines, returnLines, textAt, TEXT_COLOUR } from './content';
import {
  baselineAngle,
  beginPage,
  DEFAULT_MM_TO_PX,
  MUTED_COLOUR,
  placePoint,
  PRINT_FONT,
  rectPolygon,
  strokePolygon,
} from './geometry';
import { drawSideBOverlay } from './overlay';

/** Options for a render pass. */
export interface RenderOptions {
  /** Pixels per millimetre. */
  mmToPx?: number;
  /** Draw the fold-plan overlay. */
  showFoldLines?: boolean;
}

/** Draws Side B, the envelope face, onto the canvas. */
export function renderSideB(
  canvas: HTMLCanvasElement,
  scene: Scene,
  letter: Letter,
  options: RenderOptions = {},
): void {
  const mmToPx = options.mmToPx ?? DEFAULT_MM_TO_PX;
  const ctx = beginPage(canvas, scene.sheet.width, scene.sheet.height, mmToPx);
  const placement = scene.sheet_placement;
  const angle = baselineAngle(placement);
  const layout = scene.layout;

  if (options.showFoldLines) {
    drawSideBOverlay(ctx, scene);
  }

  // The recipient block, right-aligned at the block's right edge.
  let y = layout.address_rect.y + layout.address_font_mm;
  for (const line of addressLines(letter)) {
    const size = line.bold ? layout.address_font_mm : layout.address_font_mm - 0.4;
    const at = placePoint(placement, {
      x: layout.address_rect.x + layout.address_rect.width,
      y,
    });
    ctx.save();
    ctx.translate(at.x, at.y);
    ctx.rotate(angle);
    textAt(ctx, line.text, size, TEXT_COLOUR, line.bold, 'right');
    ctx.restore();
    y += layout.line_height_mm;
  }

  // The return address.
  y = layout.return_rect.y + layout.return_font_mm;
  for (const line of returnLines(letter)) {
    const at = placePoint(placement, { x: layout.return_rect.x, y });
    ctx.save();
    ctx.translate(at.x, at.y);
    ctx.rotate(angle);
    textAt(ctx, line, layout.return_font_mm, TEXT_COLOUR, false);
    ctx.restore();
    y += layout.return_font_mm * 1.5;
  }

  drawStamp(ctx, scene);
}

/** Draws the stamp guide. */
function drawStamp(ctx: CanvasRenderingContext2D, scene: Scene): void {
  const placement = scene.sheet_placement;
  const rect = scene.layout.stamp_rect;
  const printed = rectPolygon(rect).map((point) => placePoint(placement, point));
  strokePolygon(ctx, printed, MUTED_COLOUR, 0.18, [1.6, 1.2]);

  const at = placePoint(placement, {
    x: rect.x + rect.width / 2,
    y: rect.y + rect.height / 2 + 1,
  });
  ctx.save();
  ctx.translate(at.x, at.y);
  ctx.rotate(baselineAngle(placement));
  ctx.font = `400 2.2px ${PRINT_FONT}`;
  ctx.fillStyle = MUTED_COLOUR;
  ctx.textBaseline = 'alphabetic';
  const width = ctx.measureText('STAMP').width;
  ctx.fillText('STAMP', -width / 2, 0);
  ctx.restore();
}
