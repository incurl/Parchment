/**
 * The 2D print preview.
 *
 * The two canvases are the shared printed sheets: they are attached to
 * this view's DOM, and the 3D viewer uses the same canvases as textures.
 * The preview can show them scaled to fit or at actual size (CSS
 * millimetres, which the webview maps to real millimetres).
 */

import { useEffect, useRef, useState } from 'react';

import type { Letter, Scene } from '../../bindings';
import type { PrintSheets } from './usePrintSheets';

/** Props of the preview. */
export interface PrintPreviewProps {
  scene: Scene;
  letter: Letter;
  sheets: PrintSheets;
  showFoldLines: boolean;
  onToggleFoldLines: (value: boolean) => void;
}

/** Opens a print window with the two sheets at full A4 size. */
export function printSheets(canvasA: HTMLCanvasElement, canvasB: HTMLCanvasElement): boolean {
  const printWindow = window.open('', '_blank', 'width=900,height=1000');
  if (!printWindow) {
    return false;
  }

  const style = `
    @page { size: A4; margin: 0; }
    html, body { margin: 0; padding: 0; background: #fff; }
    img { display: block; width: 210mm; height: 297mm; page-break-after: always; }
    img:last-child { page-break-after: auto; }
  `;

  printWindow.document.write(
    `<!doctype html><html><head><title>Parchment</title><style>${style}</style></head>` +
      `<body>` +
      `<img src="${canvasA.toDataURL('image/png')}" alt="Side A">` +
      `<img src="${canvasB.toDataURL('image/png')}" alt="Side B">` +
      `</body></html>`,
  );
  printWindow.document.close();
  printWindow.focus();
  printWindow.print();
  return true;
}

/** The preview panel. */
export function PrintPreview({
  scene,
  letter,
  sheets,
  showFoldLines,
  onToggleFoldLines,
}: PrintPreviewProps) {
  const sideARef = useRef<HTMLDivElement>(null);
  const sideBRef = useRef<HTMLDivElement>(null);
  const [actualSize, setActualSize] = useState(false);
  const [printFailed, setPrintFailed] = useState(false);

  useEffect(() => {
    const container = sideARef.current;
    if (container && sheets.canvasA.parentElement !== container) {
      container.replaceChildren(sheets.canvasA);
    }
  }, [sheets.canvasA]);

  useEffect(() => {
    const container = sideBRef.current;
    if (container && sheets.canvasB.parentElement !== container) {
      container.replaceChildren(sheets.canvasB);
    }
  }, [sheets.canvasB]);

  return (
    <div className="preview">
      <div className="preview-toolbar">
        <label className="checkbox">
          <input
            type="checkbox"
            checked={showFoldLines}
            onChange={(event) => onToggleFoldLines(event.target.checked)}
          />
          Fold plan overlay
        </label>

        <button
          type="button"
          className={`btn btn-sm${actualSize ? ' btn-active' : ''}`}
          onClick={() => setActualSize((value) => !value)}
          aria-pressed={actualSize}
        >
          Actual size
        </button>

        <button
          type="button"
          className="btn btn-sm"
          onClick={() => setPrintFailed(!printSheets(sheets.canvasA, sheets.canvasB))}
        >
          Print
        </button>

        <span className="text-dim preview-hint">
          Address printed at {scene.front_panel.rotation_deg.toFixed(1)}° on Side B
        </span>
      </div>

      {printFailed && (
        <p className="warn">
          The webview blocked the print window. Use Export PDF instead — it is the
          precise path.
        </p>
      )}

      <div className={`preview-sheets${actualSize ? ' is-actual' : ''}`}>
        <figure className="sheet-frame">
          <div className="sheet-canvas" ref={sideARef} />
          <figcaption>
            Side A — the letter (inside)
            <span className="text-dim">
              {' · '}
              {letter.side_a.split('\n').length} lines
            </span>
          </figcaption>
        </figure>

        <figure className="sheet-frame">
          <div className="sheet-canvas" ref={sideBRef} />
          <figcaption>
            Side B — the envelope face (outside)
            <span className="text-dim">
              {' · '}
              {scene.panels.length} panels
            </span>
          </figcaption>
        </figure>
      </div>
    </div>
  );
}
