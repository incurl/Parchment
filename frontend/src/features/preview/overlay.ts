/**
 * The fold-plan overlay.
 *
 * On Side B it shows the front panel's region and the corner creases, so
 * the user can see where the address must be printed and where the seams
 * will fall. On Side A it shows the same panel, mirrored, so the letter
 * area is visible from the side it is written on.
 */

import type { Placement, Scene, Vec2 } from '../../bindings';
import {
  CREASE_LINE,
  fillPolygon,
  PANEL_LINE,
  PANEL_TINT,
  placePoint,
  PRINT_FONT,
  rectPolygon,
  SAFE_LINE,
  strokePolygon,
} from './geometry';

/** Returns the area centroid of a polygon. */
export function centroid(points: Vec2[]): Vec2 {
  let cross = 0;
  let x = 0;
  let y = 0;
  for (let i = 0; i < points.length; i += 1) {
    const a = points[i];
    const b = points[(i + 1) % points.length];
    const k = a.x * b.y - b.x * a.y;
    cross += k;
    x += (a.x + b.x) * k;
    y += (a.y + b.y) * k;
  }
  if (Math.abs(cross) < 1e-9) {
    return points.reduce(
      (sum, point) => ({ x: sum.x + point.x / points.length, y: sum.y + point.y / points.length }),
      { x: 0, y: 0 },
    );
  }
  return { x: x / (3 * cross), y: y / (3 * cross) };
}

/** Draws one line segment. */
export function strokeLine(
  ctx: CanvasRenderingContext2D,
  a: Vec2,
  b: Vec2,
  colour: string,
  widthMM: number,
  dash: number[] = [],
): void {
  ctx.save();
  ctx.beginPath();
  ctx.moveTo(a.x, a.y);
  ctx.lineTo(b.x, b.y);
  ctx.setLineDash(dash);
  ctx.lineWidth = widthMM;
  ctx.strokeStyle = colour;
  ctx.stroke();
  ctx.restore();
}

/** Mirrors a polygon from the sheet frame onto the printed Side A page. */
export function mirrorPolygon(points: Vec2[], sheetWidth: number): Vec2[] {
  return points.map((point) => ({ x: sheetWidth - point.x, y: point.y }));
}

/** Draws the fold plan behind the envelope face. */
export function drawSideBOverlay(
  ctx: CanvasRenderingContext2D,
  scene: Scene,
): void {
  fillPolygon(ctx, scene.front_panel.polygon, PANEL_TINT);
  strokePolygon(ctx, scene.front_panel.polygon, PANEL_LINE, 0.25);

  for (const crease of scene.creases) {
    if (crease.kind === 'tuck') {
      strokeLine(ctx, crease.a, crease.b, CREASE_LINE, 0.18, [2, 1.4]);
    }
    if (crease.kind === 'safe') {
      strokeLine(ctx, crease.a, crease.b, SAFE_LINE, 0.12, [1.2, 1.2]);
    }
  }

  // A label so the overlay explains itself when it is printed.
  const label = centroid(scene.front_panel.polygon);
  const angle = (scene.front_panel.rotation_deg * Math.PI) / 180;
  ctx.save();
  ctx.translate(label.x, label.y);
  ctx.rotate(angle);
  ctx.font = `500 3.2px ${PRINT_FONT}`;
  ctx.fillStyle = PANEL_LINE;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText('FRONT PANEL', 0, 0);
  ctx.restore();
}

/** Draws the panel and safe-area outlines on Side A. */
export function drawSideAOverlay(
  ctx: CanvasRenderingContext2D,
  scene: Scene,
  placement: Placement,
): void {
  const width = scene.sheet.width;

  if (scene.front_panel.polygon.length >= 3) {
    strokePolygon(
      ctx,
      mirrorPolygon(scene.front_panel.polygon, width),
      SAFE_LINE,
      0.18,
      [2, 1.6],
    );
  }

  if (scene.layout.letter_rect.width > 0) {
    const safe = rectPolygon(scene.layout.letter_rect).map((point) =>
      placePoint(placement, point),
    );
    strokePolygon(ctx, mirrorPolygon(safe, width), SAFE_LINE, 0.12, [1.2, 1.2]);
  }
}
