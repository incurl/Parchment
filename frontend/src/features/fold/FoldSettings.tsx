/**
 * The paper settings and the fold metrics.
 *
 * Everything here is a parameter of the fold engine, so changing one
 * number changes the panels, the hinges, the address rotation and the
 * packet size together. The metrics underneath are computed by the
 * engine, not by the UI: the UI only displays what the geometry says.
 */

import type { Letter, Params, Scene } from '../../bindings';

/** Props of the paper panel. */
export interface FoldSettingsProps {
  letter: Letter;
  scene: Scene;
  onChange: (patch: Partial<Letter>) => void;
  onReset: () => void;
}

/** A labelled numeric input. */
function NumberField({
  label,
  value,
  min,
  max,
  step,
  suffix,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  suffix: string;
  onChange: (value: number) => void;
}) {
  return (
    <label className="field">
      <span className="field-label">
        {label}
        <span className="field-value">
          {value.toFixed(step < 1 ? 1 : 0)}
          {suffix}
        </span>
      </span>
      <input
        className="slider"
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </label>
  );
}

/** The paper settings panel. */
export function FoldSettings({
  letter,
  scene,
  onChange,
  onReset,
}: FoldSettingsProps) {
  const params = letter.fold;
  const metrics = scene.metrics;

  const setFold = (patch: Partial<Params>) => {
    onChange({ fold: { ...params, ...patch } as Params });
  };

  return (
    <div className="paper-grid">
      <section className="panel">
        <div className="panel-header">
          <span>Paper and fold</span>
          <button type="button" className="btn btn-sm" onClick={onReset}>
            Reset
          </button>
        </div>
        <div className="panel-body">
          <NumberField
            label="Panel width"
            value={params.panel_width}
            min={80}
            max={190}
            step={1}
            suffix=" mm"
            onChange={(value) => setFold({ panel_width: value })}
          />
          <NumberField
            label="Panel height"
            value={params.panel_height}
            min={60}
            max={170}
            step={1}
            suffix=" mm"
            onChange={(value) => setFold({ panel_height: value })}
          />
          <NumberField
            label="Rotation"
            value={params.rotation_deg}
            min={-45}
            max={45}
            step={0.5}
            suffix="°"
            onChange={(value) => setFold({ rotation_deg: value })}
          />
          <NumberField
            label="Offset X"
            value={params.offset_x}
            min={-60}
            max={60}
            step={1}
            suffix=" mm"
            onChange={(value) => setFold({ offset_x: value })}
          />
          <NumberField
            label="Offset Y"
            value={params.offset_y}
            min={-100}
            max={100}
            step={1}
            suffix=" mm"
            onChange={(value) => setFold({ offset_y: value })}
          />
          <NumberField
            label="Corner tuck"
            value={params.tuck_depth}
            min={1}
            max={12}
            step={0.5}
            suffix=" mm"
            onChange={(value) => setFold({ tuck_depth: value })}
          />
          <NumberField
            label="Required seam overlap"
            value={params.seam_overlap}
            min={2}
            max={20}
            step={0.5}
            suffix=" mm"
            onChange={(value) => setFold({ seam_overlap: value })}
          />

          <p className="text-dim">
            The rotation is the diagonal: it is what puts the address at an angle on
            the flat sheet so it reads upright on the folded packet. The corner tuck is
            how far past the panel's edge the corner creases sit; it is what keeps the
            sheet's corners from folding onto the front.
          </p>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <span>Computed result</span>
          <span className={`badge ${metrics.within_limits ? 'badge-good' : 'badge-warn'}`}>
            {metrics.within_limits ? 'Within limits' : 'Out of limits'}
          </span>
        </div>
        <div className="panel-body">
          <dl className="inspector">
            <div>
              <dt>Packet</dt>
              <dd>
                {metrics.packet_width_mm.toFixed(1)} x{' '}
                {metrics.packet_height_mm.toFixed(1)} mm
              </dd>
            </div>
            <div>
              <dt>In inches</dt>
              <dd>
                {metrics.height_in.toFixed(2)} high x {metrics.length_in.toFixed(2)} long
              </dd>
            </div>
            <div>
              <dt>Address rotation</dt>
              <dd>{scene.front_panel.rotation_deg.toFixed(2)}° (derived)</dd>
            </div>
            <div>
              <dt>Panels</dt>
              <dd>
                {scene.panels.length} rigid, {scene.fold_order.length} hinges
              </dd>
            </div>
            <div>
              <dt>Back coverage</dt>
              <dd>{(metrics.back_coverage * 100).toFixed(1)}%</dd>
            </div>
            <div>
              <dt>Flap protrusion</dt>
              <dd>{metrics.protrusion_mm.toFixed(2)} mm</dd>
            </div>
            <div>
              <dt>Seam overlap</dt>
              <dd>{metrics.seam_overlap_mm.toFixed(1)} mm</dd>
            </div>
            <div>
              <dt>Flap depths</dt>
              <dd>
                L {depth(metrics.flap_depth_mm.left)} · R {depth(metrics.flap_depth_mm.right)} · T{' '}
                {depth(metrics.flap_depth_mm.top)} · B {depth(metrics.flap_depth_mm.bottom)}
              </dd>
            </div>
            <div>
              <dt>Panel area / sheet</dt>
              <dd>
                {metrics.partition_area_mm2.toFixed(0)} /{' '}
                {metrics.sheet_area_mm2.toFixed(0)} mm²
              </dd>
            </div>
          </dl>

          {metrics.warnings.length > 0 ? (
            <ul className="warn-list">
              {metrics.warnings.map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
          ) : (
            <p className="text-dim">
              Every check passes: the flaps cover the back, the packet is inside the
              letter range and the seams clear the corner folds.
            </p>
          )}
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <span>Fold sequence</span>
        </div>
        <div className="panel-body">
          <ol className="sequence">
            {scene.fold_order.map((id) => {
              const panel = scene.panels.find((candidate) => candidate.id === id);
              return (
                <li key={id}>
                  <code>{id}</code>
                  <span className="text-dim">
                    {panel
                      ? `${(panel.t_start * 100).toFixed(0)}–${(panel.t_end * 100).toFixed(0)}% · ${panel.role}`
                      : ''}
                  </span>
                </li>
              );
            })}
          </ol>

          <p className="text-dim">
            The front panel is not in this list: it is the root of the hinge tree and
            never moves, which is why no crease can appear on the front. Every panel
            behind it is a child of its hinge.
          </p>
        </div>
      </section>
    </div>
  );
}

/** Formats one flap depth, or a dash when the panel is absent. */
function depth(value: number | undefined): string {
  return value === undefined ? '—' : `${value.toFixed(0)} mm`;
}

