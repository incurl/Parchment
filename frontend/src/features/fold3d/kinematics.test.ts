import { describe, expect, it } from 'vitest';


import type { Hinge } from '../../bindings';
import {
  activePanel,
  angleAt,
  apply,
  identity,
  localMatrix,
  multiply,
  normalOf,
  rotationAbout,
  translation,
  worldMatrices,
  type FoldPanel,
  type TimedPanel,
} from './kinematics';

/** Builds a hinge along +x through the origin. */
function hinge(angleDeg: number): Hinge {
  return {
    a: { x: 0, y: 0, z: 0 },
    b: { x: 1, y: 0, z: 0 },
    axis: { x: 1, y: 0, z: 0 },
    angle_deg: angleDeg,
  };
}

/** A root panel and a flap hinged on its top edge. */
function tree(): FoldPanel[] {
  return [
    { id: 'front', parent: '', hinge: null, t_start: 0, t_end: 0 },
    { id: 'top', parent: 'front', hinge: hinge(180), t_start: 0.5, t_end: 1 },
    { id: 'corner', parent: 'top', hinge: hinge(-180), t_start: 0, t_end: 0.25 },
  ];
}

describe('matrix basics', () => {
  it('identity leaves a point alone', () => {
    const p = apply(identity(), { x: 3, y: -4, z: 5 });
    expect(p).toEqual({ x: 3, y: -4, z: 5 });
  });

  it('translation moves a point', () => {
    const p = apply(translation(1, 2, 3), { x: 0, y: 0, z: 0 });
    expect(p).toEqual({ x: 1, y: 2, z: 3 });
  });

  it('multiplies in application order', () => {
    const m = multiply(translation(1, 0, 0), translation(0, 2, 0));
    expect(apply(m, { x: 0, y: 0, z: 0 })).toEqual({ x: 1, y: 2, z: 0 });
  });

  it('rotates a half turn about x', () => {
    const m = rotationAbout({ x: 0, y: 0, z: 0 }, { x: 1, y: 0, z: 0 }, 180);
    const p = apply(m, { x: 0, y: 1, z: 0 });
    expect(p.x).toBeCloseTo(0, 9);
    expect(p.y).toBeCloseTo(-1, 9);
    expect(p.z).toBeCloseTo(0, 9);
  });

  it('rotates about a shifted axis', () => {
    const m = rotationAbout({ x: 0, y: 5, z: 0 }, { x: 1, y: 0, z: 0 }, 90);
    const p = apply(m, { x: 0, y: 6, z: 0 });
    expect(p.y).toBeCloseTo(5, 9);
    expect(p.z).toBeCloseTo(1, 9);
  });
});

describe('angleAt', () => {
  const top = tree()[1];

  it('is zero before the window', () => {
    expect(angleAt(top, 0.2)).toBe(0);
  });

  it('interpolates inside the window', () => {
    expect(angleAt(top, 0.75)).toBeCloseTo(90, 9);
  });

  it('is the full angle after the window', () => {
    expect(angleAt(top, 1)).toBe(180);
    expect(angleAt(top, 2)).toBe(180);
  });

  it('is zero for a panel without a hinge', () => {
    expect(angleAt(tree()[0], 0.5)).toBe(0);
  });
});

describe('worldMatrices', () => {
  it('leaves the root at the identity for every t', () => {
    const panels = tree();
    for (const t of [0, 0.3, 1]) {
      const front = worldMatrices(panels, t).get('front');
      expect(front).toEqual(identity());
    }
  });

  it('composes the parent before the child', () => {
    const panels = tree();
    const matrices = worldMatrices(panels, 1);
    const expected = multiply(
      localMatrix(panels[0], 1),
      localMatrix(panels[1], 1),
    );
    const expectedChild = multiply(expected, localMatrix(panels[2], 1));
    expect(matrices.get('top')).toEqual(expected);
    expect(matrices.get('corner')).toEqual(expectedChild);
  });

  it('flips a flap normal once and a nested flap twice', () => {
    const panels = tree();
    const flat = worldMatrices(panels, 0);
    const folded = worldMatrices(panels, 1);

    expect(normalOf(flat.get('top')!).z).toBeCloseTo(1, 9);
    expect(normalOf(folded.get('top')!).z).toBeCloseTo(-1, 9);
    // The corner is carried by the flap, so its normal returns.
    expect(normalOf(folded.get('corner')!).z).toBeCloseTo(1, 9);
  });

  it('does not rebuild anything when only t changes', () => {
    const panels = tree();
    const a = worldMatrices(panels, 0.4).get('top')!;
    const b = worldMatrices(panels, 0.4).get('top')!;
    expect(a).toEqual(b);
  });
});

describe('activePanel', () => {
  const panels = tree() as TimedPanel[];
  const order = ['corner', 'top'];

  it('reports the moving panel', () => {
    expect(activePanel(order, panels, 0.1)).toBe('corner');
    expect(activePanel(order, panels, 0.7)).toBe('top');
  });

  it('keeps the last finished panel after the end', () => {
    expect(activePanel(order, panels, 1)).toBe('top');
  });

  it('reports nothing before the first fold', () => {
    expect(activePanel(order, panels, -1)).toBeNull();
  });
});
