/**
 * The fold controls: the scrubber, the play button and the step list.
 *
 * The slider is the same control that folds and unfolds: t increasing is
 * one direction, t decreasing the other. Play ping-pongs so both
 * directions are visible without touching the slider.
 */

import type { Scene } from '../../bindings';
import { activePanel } from './kinematics';

/** Human labels for the fold steps. */
export const STEP_LABELS: Record<string, string> = {
  'tuck-tl': 'Diagonal corner fold (top left)',
  'tuck-br': 'Diagonal corner fold (bottom right)',
  'tuck-tr': 'Diagonal corner fold (top right)',
  'tuck-bl': 'Diagonal corner fold (bottom left)',
  left: 'Left side flap',
  right: 'Right side flap',
  bottom: 'Bottom flap',
  top: 'Final flap, sealing the seam',
};

/** The props of the fold controls. */
export interface FoldControlsProps {
  scene: Scene;
  progress: number;
  playing: boolean;
  onProgress: (value: number) => void;
  onTogglePlay: () => void;
}

/** The scrubber, the playback button and the step list. */
export function FoldControls({
  scene,
  progress,
  playing,
  onProgress,
  onTogglePlay,
}: FoldControlsProps) {
  const active = activePanel(scene.fold_order, scene.panels, progress);

  return (
    <>
      <div className="fold-controls">
        <button
          type="button"
          className="btn btn-sm"
          onClick={onTogglePlay}
          aria-pressed={playing}
        >
          {playing ? 'Pause' : 'Play'}
        </button>

        <input
          className="fold-slider"
          type="range"
          min={0}
          max={1000}
          value={Math.round(progress * 1000)}
          onChange={(event) => onProgress(Number(event.target.value) / 1000)}
          aria-label="Fold progress"
        />

        <span className="fold-percent">{Math.round(progress * 100)}%</span>

        <button type="button" className="btn btn-sm" onClick={() => onProgress(0)}>
          Flat
        </button>
        <button type="button" className="btn btn-sm" onClick={() => onProgress(1)}>
          Folded
        </button>
      </div>

      <ol className="fold-steps">
        {scene.fold_order.map((id) => {
          const panel = scene.panels.find((candidate) => candidate.id === id);
          const done = panel ? progress >= panel.t_end : false;
          const moving = id === active;
          return (
            <li
              key={id}
              className={`fold-step-item${done ? ' is-done' : ''}${moving ? ' is-active' : ''}`}
            >
              <span className="fold-step-dot" aria-hidden="true" />
              {STEP_LABELS[id] ?? id}
            </li>
          );
        })}
      </ol>
    </>
  );
}

/** The name of the step that is moving at this progress. */
export function currentStepLabel(scene: Scene, progress: number): string {
  const active = activePanel(scene.fold_order, scene.panels, progress);
  if (!active) {
    return progress >= 1 ? 'Folded packet' : 'Flat sheet';
  }
  return STEP_LABELS[active] ?? active;
}
