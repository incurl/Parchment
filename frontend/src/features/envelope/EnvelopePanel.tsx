/**
 * The Side B editor.
 *
 * The fields are plain text; the interesting output is the placement
 * panel underneath: the address block, the return address and the stamp
 * are authored in the packet's frame, and the scene reports the angle
 * they end up printed at on the flat sheet. That angle is the fold's own
 * rotation, so changing the fold changes it.
 */

import type { Letter, Params, Scene } from '../../bindings';

/** Props of the envelope editor. */
export interface EnvelopePanelProps {
  letter: Letter;
  scene: Scene;
  onChange: (patch: Partial<Letter>) => void;
}

/** A labelled text input. */
function TextField({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      <input
        className="input"
        type="text"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

/** The Side B editor. */
export function EnvelopePanel({ letter, scene, onChange }: EnvelopePanelProps) {
  const layout = scene.layout;
  const addressCentre = {
    x: layout.address_rect.x + layout.address_rect.width / 2,
    y: layout.address_rect.y + layout.address_rect.height / 2,
  };

  return (
    <div className="envelope-grid">
      <section className="panel">
        <div className="panel-header">
          <span>Recipient</span>
        </div>
        <div className="panel-body">
          <TextField
            label="Name"
            value={letter.recipient_name}
            onChange={(value) => onChange({ recipient_name: value })}
            placeholder="Avery Lindqvist"
          />
          <TextField
            label="Street"
            value={letter.recipient_street}
            onChange={(value) => onChange({ recipient_street: value })}
            placeholder="1187 Harbour Lane"
          />
          <div className="field-row">
            <TextField
              label="City"
              value={letter.recipient_city}
              onChange={(value) => onChange({ recipient_city: value })}
            />
            <TextField
              label="State"
              value={letter.recipient_state}
              onChange={(value) => onChange({ recipient_state: value })}
            />
            <TextField
              label="ZIP"
              value={letter.recipient_zip}
              onChange={(value) => onChange({ recipient_zip: value })}
            />
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <span>Return address</span>
        </div>
        <div className="panel-body">
          <TextField
            label="Name"
            value={letter.return_name}
            onChange={(value) => onChange({ return_name: value })}
          />
          <TextField
            label="Street"
            value={letter.return_street}
            onChange={(value) => onChange({ return_street: value })}
          />
          <div className="field-row">
            <TextField
              label="City"
              value={letter.return_city}
              onChange={(value) => onChange({ return_city: value })}
            />
            <TextField
              label="State"
              value={letter.return_state}
              onChange={(value) => onChange({ return_state: value })}
            />
            <TextField
              label="ZIP"
              value={letter.return_zip}
              onChange={(value) => onChange({ return_zip: value })}
            />
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <span>Stamp and placement</span>
        </div>
        <div className="panel-body">
          <label className="field">
            <span className="field-label">Stamp corner</span>
            <select
              className="input"
              value={letter.fold.stamp_corner}
              onChange={(event) =>
                onChange({
                  fold: { ...letter.fold, stamp_corner: event.target.value } as Params,
                })
              }
            >
              <option value="top-right">Top right (USPS)</option>
              <option value="top-left">Top left</option>
              <option value="bottom-right">Bottom right</option>
            </select>
          </label>

          <dl className="inspector">
            <div>
              <dt>Panel</dt>
              <dd>
                {scene.front_panel.size.width.toFixed(1)} x{' '}
                {scene.front_panel.size.height.toFixed(1)} mm
              </dd>
            </div>
            <div>
              <dt>Printed at</dt>
              <dd>{scene.front_panel.rotation_deg.toFixed(2)}° on the flat sheet</dd>
            </div>
            <div>
              <dt>Address block</dt>
              <dd>
                {addressCentre.x.toFixed(0)}, {addressCentre.y.toFixed(0)} mm in the packet
              </dd>
            </div>
            <div>
              <dt>Upright on the packet</dt>
              <dd>Yes — the block is laid out in the packet frame</dd>
            </div>
          </dl>

          <p className="text-dim">
            The rotation is not a constant: it is the direction of the front panel's
            first edge, measured on the sheet. Change the panel size or the fold angle
            in the Paper tab and this number follows, keeping the address upright on
            the folded packet.
          </p>
        </div>
      </section>
    </div>
  );
}
