/**
 * The printed sheets, drawn once and shared.
 *
 * One pair of canvases serves the 2D preview and the 3D textures: the
 * same pixels the user sees in the print preview are the ones wrapped
 * around the paper in the viewer, so the 3D view can never drift from
 * what will print.
 */

import { useEffect, useRef, useState } from 'react';

import type { Letter, Scene } from '../../bindings';
import { renderSideA } from './renderA';
import { renderSideB } from './renderB';

/** The shared canvases and the version of their contents. */
export interface PrintSheets {
  canvasA: HTMLCanvasElement;
  canvasB: HTMLCanvasElement;
  /** Increments on every redraw, so the textures can be refreshed. */
  version: number;
}

/** Pixels per millimetre for the printed sheets. */
const MM_TO_PX = 4;

/**
 * usePrintSheets keeps the two A4 pages up to date.
 *
 * The canvases are created once and never replaced, so a canvas can be
 * attached to the preview's DOM and used as a texture at the same time.
 * The redraw is deferred a little so that typing does not re-render the
 * sheets on every keystroke.
 */
export function usePrintSheets(
  scene: Scene | null,
  letter: Letter | null,
  showFoldLines: boolean,
): PrintSheets {
  const canvasARef = useRef<HTMLCanvasElement | null>(null);
  const canvasBRef = useRef<HTMLCanvasElement | null>(null);
  const [version, setVersion] = useState(0);

  if (canvasARef.current === null) {
    canvasARef.current = document.createElement('canvas');
  }
  if (canvasBRef.current === null) {
    canvasBRef.current = document.createElement('canvas');
  }

  useEffect(() => {
    if (!scene || !letter) {
      return;
    }
    const canvasA = canvasARef.current;
    const canvasB = canvasBRef.current;
    if (!canvasA || !canvasB) {
      return;
    }

    const timer = window.setTimeout(() => {
      try {
        renderSideA(canvasA, scene, letter, { mmToPx: MM_TO_PX, showFoldLines });
        renderSideB(canvasB, scene, letter, { mmToPx: MM_TO_PX, showFoldLines });
        setVersion((current) => current + 1);
      } catch (error) {
        // A canvas that cannot be drawn is a rendering bug, not a
        // reason to lose the letter: report and keep the old pixels.
        console.error('could not render the printed sheets', error);
      }
    }, 60);

    return () => window.clearTimeout(timer);
  }, [scene, letter, showFoldLines]);

  return {
    canvasA: canvasARef.current,
    canvasB: canvasBRef.current,
    version,
  };
}
