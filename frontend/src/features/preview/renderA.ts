/**
 * Side A: the letter, printed on the inside of the sheet.
 */

import type { Letter, Scene } from '../../bindings';
import { wrapText } from '../letter/textLayout';
import { textAt, TEXT_COLOUR } from './content';
import {
  beginPage,
  DEFAULT_MM_TO_PX,
  placePointOnSideA,
  PRINT_FONT,
  sideABaselineAngle,
} from './geometry';
import { drawSideAOverlay } from './overlay';
import type { RenderOptions } from './renderB';

/** Draws Side A, the letter, onto the canvas. */
export function renderSideA(
  canvas: HTMLCanvasElement,
  scene: Scene,
  letter: Letter,
  options: RenderOptions = {},
): void {
  const mmToPx = options.mmToPx ?? DEFAULT_MM_TO_PX;
  const ctx = beginPage(canvas, scene.sheet.width, scene.sheet.height, mmToPx);
  const placement = scene.letter_placement;
  const angle = sideABaselineAngle(placement);
  const rect = scene.layout.letter_rect;
  const size = scene.layout.body_font_mm;
  const lineHeight = scene.layout.line_height_mm;

  if (options.showFoldLines) {
    drawSideAOverlay(ctx, scene, placement);
  }

  // The wrapping must use the same font the lines are drawn with.
  ctx.font = `400 ${size}px ${PRINT_FONT}`;
  const lines = wrapText(letter.side_a, rect.width, (text) => ctx.measureText(text).width);

  let y = rect.y + size;
  for (const line of lines) {
    if (line.text !== '') {
      const at = placePointOnSideA(placement, scene.sheet.width, {
        x: rect.x,
        y,
      });
      ctx.save();
      ctx.translate(at.x, at.y);
      ctx.rotate(angle);
      textAt(ctx, line.text, size, TEXT_COLOUR, false);
      ctx.restore();
    }
    y += lineHeight;
  }
}
