/**
 * The Side A editor.
 *
 * The warning is the interesting part: the letter is printed inside the
 * front panel, and text that runs past the panel's safe area ends up in
 * the fold margin, where the flaps crease it and partly hide it. The
 * check uses the same wrap the printer uses, measured with the real
 * font, so it does not guess.
 */

import { useMemo } from 'react';

import type { Letter, Scene } from '../../bindings';
import { letterFit } from './textLayout';
import { PRINT_FONT } from '../preview/geometry';

/** Props of the editor. */
export interface LetterEditorProps {
  letter: Letter;
  scene: Scene;
  onChange: (patch: Partial<Letter>) => void;
}

/**
 * measureWithCanvas measures text with the print font at the body size.
 *
 * A canvas is used rather than an estimate so the warning matches what
 * the renderer will do. When the canvas is unavailable (a test
 * environment, a broken webview) the estimate in textLayout takes over.
 */
export function measureWithCanvas(sizeMM: number): (text: string) => number {
  const canvas = document.createElement('canvas');

  let ctx: CanvasRenderingContext2D | null = null;
  try {
    ctx = canvas.getContext('2d');
  } catch {
    ctx = null;
  }
  if (!ctx) {
    return (text) => text.length * sizeMM * 0.52;
  }
  ctx.font = `400 ${sizeMM}px ${PRINT_FONT}`;
  return (text) => ctx.measureText(text).width;
}

/** The letter body editor with its fit warning. */
export function LetterEditor({ letter, scene, onChange }: LetterEditorProps) {
  const rect = scene.layout.letter_rect;

  const fit = useMemo(() => {
    const measure = measureWithCanvas(scene.layout.body_font_mm);
    return letterFit(
      letter.side_a,
      rect.width,
      rect.height,
      scene.layout.line_height_mm,
      measure,
    );
  }, [letter.side_a, rect.width, rect.height, scene.layout.body_font_mm, scene.layout.line_height_mm]);

  const remaining = Math.max(0, rect.height - fit.height);

  return (
    <section className="panel">
      <div className="panel-header">
        <span>Side A — the letter</span>
        <span className={`badge ${fit.fits ? 'badge-good' : 'badge-warn'}`}>
          {fit.fits ? 'Fits inside the panel' : 'Overflows the panel'}
        </span>
      </div>

      <div className="panel-body">
        <label className="field">
          <span className="field-label">Letter body</span>
          <textarea
            className="editor"
            value={letter.side_a}
            onChange={(event) => onChange({ side_a: event.target.value })}
            spellCheck
            rows={22}
          />
        </label>

        <div className="letter-stats">
          <span>{fit.lines.length} lines</span>
          <span>
            {fit.height.toFixed(1)} / {rect.height.toFixed(1)} mm used
          </span>
          <span>{remaining.toFixed(1)} mm left</span>
        </div>

        {!fit.fits && (
          <p className="warn">
            {fit.overflowLines > 0
              ? `${fit.overflowLines} line${fit.overflowLines === 1 ? '' : 's'} run past the panel's width. `
              : ''}
            The text runs past the fold margin: the flaps crease it and it is partly
            hidden when the packet is sealed. Shorten the letter or reduce the body
            size.
          </p>
        )}

        <p className="text-dim">
          The letter is printed on the inside, in the frame of the front panel. It is
          rotated on the page by the same geometry that turns the address upright on
          the outside, so it stays clear of every crease.
        </p>
      </div>
    </section>
  );
}
