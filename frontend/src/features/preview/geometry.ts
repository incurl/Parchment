/**
 * The 2D print renderer's geometry helpers.
 *
 * Both sides are drawn in millimetres at a fixed pixel-per-millimetre
 * ratio, so the same functions serve the preview on screen, the print
 * view at actual size, and the texture the 3D viewer wraps around the
 * panels. There is no second drawing path to keep in sync.
 *
 * Every position comes from the scene the Go engine produced. In
 * particular the address block is placed with the scene's
 * `sheet_placement`, whose rotation is the diagonal derived from the
 * fold - this module never computes a fold angle of its own.
 */

import type { Placement, Vec2 } from '../../bindings';

/** The pixel density used for the textures and the default preview. */
export const DEFAULT_MM_TO_PX = 3;

/** The font stack used for every printed line. */
export const PRINT_FONT = '"Helvetica Neue", Helvetica, Arial, sans-serif';

/** The paper colour. Slightly warm, so white UI chrome reads as chrome. */
export const PAPER_COLOUR = '#fffdf8';

/** The ink colour. */
export const INK_COLOUR = '#201c17';

/** A muted colour for guides. */
export const MUTED_COLOUR = '#8c7f6b';

/** The tint of the front panel in the fold overlay. */
export const PANEL_TINT = 'rgba(214, 172, 92, 0.16)';

/** The front panel's outline in the fold overlay. */
export const PANEL_LINE = 'rgba(150, 108, 38, 0.85)';

/** A corner crease in the fold overlay. */
export const CREASE_LINE = 'rgba(150, 108, 38, 0.55)';

/** The letter's safe area in the fold overlay. */
export const SAFE_LINE = 'rgba(120, 110, 95, 0.35)';

/** Applies a placement to a point in the local frame. */
export function placePoint(placement: Placement, v: Vec2): Vec2 {
  return {
    x: placement.origin.x + placement.x_axis.x * v.x + placement.y_axis.x * v.y,
    y: placement.origin.y + placement.x_axis.y * v.x + placement.y_axis.y * v.y,
  };
}

/** Maps a point in the local frame onto the printed Side A page. */
export function placePointOnSideA(
  placement: Placement,
  sheetWidth: number,
  v: Vec2,
): Vec2 {
  const sheet = placePoint(placement, v);
  return { x: sheetWidth - sheet.x, y: sheet.y };
}

/** The angle of a placement's baseline in the sheet frame, in radians. */
export function baselineAngle(placement: Placement): number {
  return Math.atan2(placement.x_axis.y, placement.x_axis.x);
}

/**
 * The angle of a placement's baseline as printed on Side A, in radians.
 *
 * Side A is the mirror of the sheet frame, so the direction's x flips.
 */
export function sideABaselineAngle(placement: Placement): number {
  return Math.atan2(placement.x_axis.y, -placement.x_axis.x);
}

/** Sizes the canvas and returns a context whose units are millimetres. */
export function beginPage(
  canvas: HTMLCanvasElement,
  sheetWidth: number,
  sheetHeight: number,
  mmToPx: number,
): CanvasRenderingContext2D {
  canvas.width = Math.round(sheetWidth * mmToPx);
  canvas.height = Math.round(sheetHeight * mmToPx);

  const ctx = canvas.getContext('2d');
  if (!ctx) {
    throw new Error('the 2D canvas is not available');
  }

  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.fillStyle = PAPER_COLOUR;
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.scale(canvas.width / sheetWidth, canvas.height / sheetHeight);
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';
  return ctx;
}

/** Builds a closed path from points. */
function buildPath(ctx: CanvasRenderingContext2D, points: Vec2[]): void {
  points.forEach((point, index) => {
    if (index === 0) {
      ctx.moveTo(point.x, point.y);
    } else {
      ctx.lineTo(point.x, point.y);
    }
  });
  ctx.closePath();
}

/** Strokes a closed polygon. */
export function strokePolygon(
  ctx: CanvasRenderingContext2D,
  points: Vec2[],
  colour: string,
  widthMM: number,
  dash: number[] = [],
): void {
  if (points.length < 2) {
    return;
  }
  ctx.save();
  ctx.beginPath();
  buildPath(ctx, points);
  ctx.setLineDash(dash);
  ctx.lineWidth = widthMM;
  ctx.strokeStyle = colour;
  ctx.stroke();
  ctx.restore();
}

/** Fills a closed polygon. */
export function fillPolygon(
  ctx: CanvasRenderingContext2D,
  points: Vec2[],
  colour: string,
): void {
  if (points.length < 3) {
    return;
  }
  ctx.save();
  ctx.beginPath();
  buildPath(ctx, points);
  ctx.fillStyle = colour;
  ctx.fill();
  ctx.restore();
}

/** Converts a rectangle to its four corners. */
export function rectPolygon(rect: {
  x: number;
  y: number;
  width: number;
  height: number;
}): Vec2[] {
  return [
    { x: rect.x, y: rect.y },
    { x: rect.x + rect.width, y: rect.y },
    { x: rect.x + rect.width, y: rect.y + rect.height },
    { x: rect.x, y: rect.y + rect.height },
  ];
}
