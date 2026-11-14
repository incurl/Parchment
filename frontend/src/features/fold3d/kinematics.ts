/**
 * The fold kinematics.
 *
 * This module is the frontend's half of the contract: the Go engine
 * produces the panels, the hinges and the fold windows, and this computes
 * what the viewer needs for a given progress t. The work is pure matrix
 * composition - one local hinge matrix per panel, multiplied by the
 * parent's world matrix. No geometry is rebuilt while scrubbing.
 *
 * Matrices are 4x4 row-major, the same layout the Go engine uses, so a
 * test can compare the two implementations term by term.
 */

import type { Hinge, Vec3 } from '../../bindings';

/** The subset of a panel the kinematics needs, so tests can build one. */
export interface FoldPanel {
  id: string;
  parent: string;
  hinge?: Hinge | null;
  t_start: number;
  t_end: number;
}

/** A 4x4 row-major matrix. */
export type Mat4 = number[];

/** The identity matrix. */
export function identity(): Mat4 {
  return [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1];
}

/** Returns t * u, so that multiply(t, u) applied to p is t(u(p)). */
export function multiply(t: Mat4, u: Mat4): Mat4 {
  const out = new Array<number>(16).fill(0);
  for (let row = 0; row < 4; row += 1) {
    for (let col = 0; col < 4; col += 1) {
      let sum = 0;
      for (let k = 0; k < 4; k += 1) {
        sum += t[row * 4 + k] * u[k * 4 + col];
      }
      out[row * 4 + col] = sum;
    }
  }
  return out;
}

/** Applies a matrix to a point. */
export function apply(m: Mat4, p: Vec3): Vec3 {
  return {
    x: m[0] * p.x + m[1] * p.y + m[2] * p.z + m[3],
    y: m[4] * p.x + m[5] * p.y + m[6] * p.z + m[7],
    z: m[8] * p.x + m[9] * p.y + m[10] * p.z + m[11],
  };
}

/** Returns a translation matrix. */
export function translation(x: number, y: number, z: number): Mat4 {
  const m = identity();
  m[3] = x;
  m[7] = y;
  m[11] = z;
  return m;
}

/**
 * Returns the rotation of deg degrees about the line through origin with
 * the given direction, using Rodrigues' formula.
 */
export function rotationAbout(origin: Vec3, axis: Vec3, deg: number): Mat4 {
  const rad = (deg * Math.PI) / 180;
  const c = Math.cos(rad);
  const s = Math.sin(rad);

  const length = Math.hypot(axis.x, axis.y, axis.z);
  const ux = length > 0 ? axis.x / length : 0;
  const uy = length > 0 ? axis.y / length : 0;
  const uz = length > 0 ? axis.z / length : 0;

  const r = [
    c + ux * ux * (1 - c),
    ux * uy * (1 - c) - uz * s,
    ux * uz * (1 - c) + uy * s,
    0,
    uy * ux * (1 - c) + uz * s,
    c + uy * uy * (1 - c),
    uy * uz * (1 - c) - ux * s,
    0,
    uz * ux * (1 - c) - uy * s,
    uz * uy * (1 - c) + ux * s,
    c + uz * uz * (1 - c),
    0,
    0,
    0,
    0,
    1,
  ];

  return multiply(
    multiply(translation(origin.x, origin.y, origin.z), r),
    translation(-origin.x, -origin.y, -origin.z),
  );
}

/** The hinge's angle at progress t, in degrees. */
export function angleAt(panel: FoldPanel, t: number): number {
  if (!panel.hinge || panel.t_end <= panel.t_start) {
    return 0;
  }
  const progress = Math.min(1, Math.max(0, (t - panel.t_start) / (panel.t_end - panel.t_start)));
  return panel.hinge.angle_deg * progress;
}

/** The panel's hinge transform in the flat frame, at progress t. */
export function localMatrix(panel: FoldPanel, t: number): Mat4 {
  if (!panel.hinge) {
    return identity();
  }
  const angle = angleAt(panel, t);
  if (Math.abs(angle) < 1e-9) {
    return identity();
  }
  return rotationAbout(panel.hinge.a, panel.hinge.axis, angle);
}

/**
 * The world matrix of every panel at progress t.
 *
 * Panels are resolved parent-first, exactly as the Go engine resolves
 * them, so a child's matrix already contains every ancestor's rotation.
 */
export function worldMatrices(panels: FoldPanel[], t: number): Map<string, Mat4> {
  const byId = new Map<string, FoldPanel>();
  panels.forEach((panel) => byId.set(panel.id, panel));

  const out = new Map<string, Mat4>();
  const resolve = (panel: FoldPanel): Mat4 => {
    const cached = out.get(panel.id);
    if (cached) {
      return cached;
    }

    const parent = panel.parent ? byId.get(panel.parent) : undefined;
    const parentMatrix = parent ? resolve(parent) : identity();
    const matrix = multiply(parentMatrix, localMatrix(panel, t));
    out.set(panel.id, matrix);
    return matrix;
  };

  panels.forEach(resolve);
  return out;
}

/** The image of the panel's +Z normal through its world matrix. */
export function normalOf(matrix: Mat4): Vec3 {
  const x = matrix[2];
  const y = matrix[6];
  const z = matrix[10];
  const length = Math.hypot(x, y, z);
  if (length < 1e-9) {
    return { x: 0, y: 0, z: 0 };
  }
  return { x: x / length, y: y / length, z: z / length };
}

/** The subset of a panel the fold schedule needs. */
export interface TimedPanel {
  id: string;
  t_start: number;
  t_end: number;
}

/**
 * The panel whose hinge is moving at progress t, or the last one that
 * finished. The viewer uses it for the step label.
 */
export function activePanel(
  order: string[],
  panels: TimedPanel[],
  t: number,
): string | null {
  const byId = new Map<string, TimedPanel>();
  panels.forEach((panel) => byId.set(panel.id, panel));

  let last: string | null = null;
  for (const id of order) {
    const panel = byId.get(id);
    if (!panel) {
      continue;
    }
    if (t >= panel.t_start && t <= panel.t_end) {
      return id;
    }
    if (t > panel.t_end) {
      last = id;
    }
  }
  return last;
}
