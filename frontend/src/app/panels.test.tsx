/**
 * Render tests for the panels.
 *
 * Each panel is rendered with the scene and letter shapes the backend
 * sends, so the test covers the parts that matter for the UI: the
 * address fields, the fold overlay toggle, the letter fit warning and
 * the computed metrics. jsdom has no canvas or WebGL, so the preview's
 * pixels and the 3D view are out of scope here - the fold geometry
 * itself is tested where it lives.
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Letter } from '../bindings';
import { EnvelopePanel } from '../features/envelope/EnvelopePanel';
import { FoldSettings } from '../features/fold/FoldSettings';
import { LetterEditor } from '../features/letter/LetterEditor';
import { PrintPreview } from '../features/preview/PrintPreview';
import { demoLetter, demoScene } from './mockScene';

globalThis.IS_REACT_ACT_ENVIRONMENT = true;

let root: Root | null = null;
let container: HTMLDivElement | null = null;

afterEach(() => {
  if (root) {
    act(() => root?.unmount());
  }
  container?.remove();
  root = null;
  container = null;
});

/** Renders a node and returns the container. */
function render(node: React.ReactNode): HTMLDivElement {
  container = document.createElement('div');
  document.body.appendChild(container);
  root = createRoot(container);
  act(() => {
    root?.render(node);
  });
  return container;
}

describe('the letter editor', () => {
  it('shows the fit statistics and the text', () => {
    const element = render(
      <LetterEditor letter={demoLetter} scene={demoScene} onChange={() => undefined} />,
    );
    expect(element.textContent).toContain('Side A — the letter');
    expect(element.textContent).toContain('Fits inside the panel');
    expect(element.textContent).toContain('lines');
    expect(element.querySelector('textarea')?.value).toContain('Dear Avery');
  });

  it('warns when the letter is longer than the panel', () => {
    const long: Letter = {
      ...demoLetter,
      side_a: Array.from({ length: 200 }, (_, index) => `line ${index}`).join('\n'),
    };
    const element = render(
      <LetterEditor letter={long} scene={demoScene} onChange={() => undefined} />,
    );
    expect(element.textContent).toContain('Overflows the panel');
    expect(element.textContent).toContain('fold margin');
  });
});

describe('the envelope panel', () => {
  it('shows the recipient, the return address and the derived diagonal', () => {
    const element = render(
      <EnvelopePanel letter={demoLetter} scene={demoScene} onChange={() => undefined} />,
    );
    const inputs = Array.from(element.querySelectorAll('input'));
    expect(inputs[0].value).toBe('Avery Lindqvist');
    expect(inputs[1].value).toBe('1187 Harbour Lane');
    expect(element.textContent).toContain('Return address');
    expect(element.textContent).toContain('Stamp corner');
    expect(element.textContent).toContain('15.00° on the flat sheet');
    expect(element.textContent).toContain('Upright on the packet');
  });
});

describe('the print preview', () => {
  it('offers both sides, the overlay and the computed angle', () => {
    const canvasA = document.createElement('canvas');
    const canvasB = document.createElement('canvas');
    const element = render(
      <PrintPreview
        scene={demoScene}
        letter={demoLetter}
        sheets={{ canvasA, canvasB, version: 1 }}
        showFoldLines={false}
        onToggleFoldLines={() => undefined}
      />,
    );
    expect(element.textContent).toContain('Side A — the letter (inside)');
    expect(element.textContent).toContain('Side B — the envelope face (outside)');
    expect(element.textContent).toContain('Address printed at 15.0°');
    expect(element.textContent).toContain('Fold plan overlay');
    expect(element.textContent).toContain('Actual size');
  });
});

describe('the paper settings', () => {
  it('shows the fold metrics and the sequence', () => {
    const element = render(
      <FoldSettings
        letter={demoLetter}
        scene={demoScene}
        onChange={() => undefined}
        onReset={() => undefined}
      />,
    );
    expect(element.textContent).toContain('Computed result');
    expect(element.textContent).toContain('150.0 x 115.0 mm');
    expect(element.textContent).toContain('15.00° (derived)');
    expect(element.textContent).toContain('100.0%');
    expect(element.textContent).toContain('Fold sequence');
    expect(element.textContent).toContain('tuck-tl');
  });

  it('lists the warnings the engine produced', () => {
    const scene = {
      ...demoScene,
      metrics: {
        ...demoScene.metrics,
        within_limits: false,
        warnings: ['Packet is outside the configured letter range.'],
      },
    };
    const element = render(
      <FoldSettings
        letter={demoLetter}
        scene={scene}
        onChange={() => undefined}
        onReset={() => undefined}
      />,
    );
    expect(element.textContent).toContain('Out of limits');
    expect(element.textContent).toContain('outside the configured letter range');
  });
});
