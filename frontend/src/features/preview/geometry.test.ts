import { describe, expect, it } from 'vitest';

import type { Letter, Placement } from '../../bindings';
import { addressLines, joinCityLine } from './content';
import {
  baselineAngle,
  placePoint,
  placePointOnSideA,
  sideABaselineAngle,
} from './geometry';
import { centroid } from './overlay';

/** A placement rotated by 30 degrees with a translation. */
function rotated(deg: number): Placement {
  const rad = (deg * Math.PI) / 180;
  return {
    origin: { x: 40, y: 25 },
    x_axis: { x: Math.cos(rad), y: Math.sin(rad) },
    y_axis: { x: -Math.sin(rad), y: Math.cos(rad) },
    rotation_deg: deg,
    mirrored: false,
  };
}

describe('placePoint', () => {
  it('applies the origin for the local origin', () => {
    const point = placePoint(rotated(30), { x: 0, y: 0 });
    expect(point.x).toBeCloseTo(40, 10);
    expect(point.y).toBeCloseTo(25, 10);
  });

  it('moves along the rotated axes', () => {
    const point = placePoint(rotated(90), { x: 10, y: 0 });
    expect(point.x).toBeCloseTo(40, 10);
    expect(point.y).toBeCloseTo(35, 10);
  });

  it('moves down the y axis', () => {
    const point = placePoint(rotated(0), { x: 0, y: 10 });
    expect(point.y).toBeCloseTo(35, 10);
  });
});

describe('placePointOnSideA', () => {
  it('mirrors the x coordinate and keeps y', () => {
    const point = placePointOnSideA(rotated(0), 210, { x: 10, y: 5 });
    expect(point.x).toBeCloseTo(210 - 50, 10);
    expect(point.y).toBeCloseTo(30, 10);
  });
});

describe('baseline angles', () => {
  it('reports the sheet angle', () => {
    expect((baselineAngle(rotated(15)) * 180) / Math.PI).toBeCloseTo(15, 10);
  });

  it('mirrors the x direction on Side A', () => {
    const angle = (sideABaselineAngle(rotated(15)) * 180) / Math.PI;
    // The mirrored direction of 15 degrees is 165 degrees.
    expect(angle).toBeCloseTo(165, 10);
  });
});

describe('joinCityLine', () => {
  it('joins city, state and ZIP', () => {
    expect(joinCityLine('Austin', 'TX', '78701')).toBe('Austin, TX  78701');
  });

  it('skips empty parts', () => {
    expect(joinCityLine('Austin', '', '78701')).toBe('Austin  78701');
    expect(joinCityLine('', 'TX', '')).toBe('TX');
    expect(joinCityLine('', '', '')).toBe('');
  });
});

describe('addressLines', () => {
  it('drops empty lines but keeps the order', () => {
    const letter = {
      recipient_name: 'Avery Lindqvist',
      recipient_street: '',
      recipient_city: 'Port Isabel',
      recipient_state: 'TX',
      recipient_zip: '78578',
    } as Letter;

    const lines = addressLines(letter);
    expect(lines.map((line) => line.text)).toEqual([
      'Avery Lindqvist',
      'Port Isabel, TX  78578',
    ]);
    expect(lines[0].bold).toBe(true);
  });
});

describe('centroid', () => {
  it('finds the centre of a rectangle', () => {
    const centre = centroid([
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 4 },
      { x: 0, y: 4 },
    ]);
    expect(centre.x).toBeCloseTo(5, 9);
    expect(centre.y).toBeCloseTo(2, 9);
  });

  it('finds the centre of a triangle', () => {
    const centre = centroid([
      { x: 0, y: 0 },
      { x: 6, y: 0 },
      { x: 0, y: 6 },
    ]);
    expect(centre.x).toBeCloseTo(2, 9);
    expect(centre.y).toBeCloseTo(2, 9);
  });
});
