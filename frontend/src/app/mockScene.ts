/**
 * The scene and letter the render smoke test uses.
 *
 * The panel numbers come from the Go engine's fixture; the fields the
 * fixture does not carry (the layout, the placements, the metrics) are
 * filled in here, so the test exercises a realistic shape rather than a
 * hand-waved one.
 */

import fixture from '../features/fold3d/foldScene.fixture.json';
import type { Letter, Scene } from '../bindings';

/** The draft the mocked backend returns. */
export const demoLetter: Letter = {
  id: 'demo',
  title: 'Demo letter',
  side_a: 'Dear Avery,\n\nThank you for the paper samples.',
  recipient_name: 'Avery Lindqvist',
  recipient_street: '1187 Harbour Lane',
  recipient_city: 'Port Isabel',
  recipient_state: 'TX',
  recipient_zip: '78578',
  return_name: 'Parchment Studio',
  return_street: '4 Fold Street',
  return_city: 'Austin',
  return_state: 'TX',
  return_zip: '78701',
  fold: fixture.params as Letter['fold'],
  created: '',
  updated: '',
};

/** The scene the mocked engine returns. */
export const demoScene: Scene = {
  params: fixture.params as Scene['params'],
  sheet: { width: 210, height: 297 },
  front_panel: {
    id: 'front',
    polygon: [
      { x: 47.4, y: 73.5 },
      { x: 192.3, y: 112.4 },
      { x: 162.6, y: 223.5 },
      { x: 17.7, y: 184.6 },
    ],
    world_polygon: [
      { x: -75, y: 57.5 },
      { x: 75, y: 57.5 },
      { x: 75, y: -57.5 },
      { x: -75, y: -57.5 },
    ],
    transform_from_flat: { rotation_deg: -15, tx: 24.4, ty: 88.8 },
    transform_to_flat: { rotation_deg: 15, tx: 47.4, ty: 73.5 },
    size: { width: 150, height: 115 },
    rotation_deg: 15,
    centre: { x: 105, y: 148.5 },
  },
  panels: fixture.panels.map((panel) => ({
    id: panel.id,
    parent: panel.parent,
    role: panel.id,
    polygon: panel.polygon,
    sheet_polygon: panel.polygon,
    uv: [],
    hinge: panel.hinge,
    t_start: panel.t_start,
    t_end: panel.t_end,
    layer: 0,
    tint: '',
  })),
  fold_order: fixture.panels
    .filter((panel) => panel.parent !== '')
    .map((panel) => panel.id),
  creases: [],
  layout: {
    address_rect: { x: 57, y: 57.5, width: 87, height: 46 },
    return_rect: { x: 12, y: 12, width: 69, height: 13.8 },
    stamp_rect: { x: 116, y: 12, width: 22, height: 26 },
    letter_rect: { x: 8, y: 8, width: 134, height: 99 },
    address_font_mm: 3.6,
    return_font_mm: 2.6,
    body_font_mm: 3.2,
    line_height_mm: 4.6,
    rotation_deg: 15,
  },
  metrics: {
    packet_width_mm: 150,
    packet_height_mm: 115,
    height_in: 4.53,
    length_in: 5.91,
    within_limits: true,
    flap_depth_mm: { left: 48.6, right: 48.6, top: 118.1, bottom: 118.1 },
    back_coverage: 1,
    protrusion_mm: 3,
    seam_overlap_mm: 49.9,
    partition_area_mm2: 62370,
    sheet_area_mm2: 62370,
    warnings: [],
  },
  sheet_placement: {
    origin: { x: 47.4, y: 73.5 },
    x_axis: { x: 0.9659, y: 0.2588 },
    y_axis: { x: -0.2588, y: 0.9659 },
    rotation_deg: 15,
    mirrored: false,
  },
  letter_placement: {
    origin: { x: 192.3, y: 112.4 },
    x_axis: { x: -0.9659, y: -0.2588 },
    y_axis: { x: -0.2588, y: 0.9659 },
    rotation_deg: 195,
    mirrored: true,
  },
};
