/**
 * The Parchment shell.
 *
 * Left: the library of saved letters. Centre: the working area, one tab
 * per concern (the letter, the envelope face, the print preview, the 3D
 * fold, the paper settings). Right: the inspector, which shows what the
 * fold engine computed. Bottom: the debug console.
 *
 * All backend access goes through the Wails bindings; the frontend never
 * talks to PocketBase or to a remote API directly.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import '../styles.css';

import * as api from '../bindings';
import { applyZoom } from '../ui/zoom';
import { isConsoleShortcut } from '../ui/shortcuts';
import { useWorkspace } from './workspace';
import { ConsolePanel } from '../features/console/ConsolePanel';
import { EnvelopePanel } from '../features/envelope/EnvelopePanel';
import { FoldSettings } from '../features/fold/FoldSettings';
import { FoldViewer } from '../features/fold3d/FoldViewer';
import { LetterEditor } from '../features/letter/LetterEditor';
import { PrintPreview } from '../features/preview/PrintPreview';

/** The working tabs. */
type Tab = 'letter' | 'envelope' | 'preview' | 'fold' | 'paper';

const TABS: { id: Tab; label: string; hint: string }[] = [
  { id: 'letter', label: 'Letter', hint: 'Write Side A' },
  { id: 'envelope', label: 'Envelope', hint: 'Lay out Side B' },
  { id: 'preview', label: 'Print preview', hint: 'True A4 scale' },
  { id: 'fold', label: '3D fold', hint: 'Scrub the fold' },
  { id: 'paper', label: 'Paper', hint: 'Fold parameters' },
];

interface Toast {
  message: string;
  kind: 'info' | 'error' | 'success';
}

/** The application root. */
function App() {
  const [tab, setTab] = useState<Tab>('letter');
  const [showFoldLines, setShowFoldLines] = useState(false);
  const [consoleOpen, setConsoleOpen] = useState(false);
  const [toast, setToast] = useState<Toast | null>(null);

  // The toast timer is kept so a second toast replaces the first instead
  // of being dismissed by the first one's timer.
  const toastTimer = useRef<number | null>(null);

  const notify = useCallback(
    (message: string, kind: Toast['kind'] = 'info') => {
      setToast({ message, kind });
      if (toastTimer.current !== null) {
        window.clearTimeout(toastTimer.current);
      }
      toastTimer.current = window.setTimeout(() => {
        toastTimer.current = null;
        setToast(null);
      }, 5000);
    },
    [],
  );

  useEffect(() => {
    return () => {
      if (toastTimer.current !== null) {
        window.clearTimeout(toastTimer.current);
      }
    };
  }, []);

  const workspace = useWorkspace(notify, showFoldLines);
  const { draft, scene } = workspace;

  // The stored zoom scales the whole interface. It is applied here so
  // the shell is readable even before a letter is opened.
  const settingsQuery = useQuery({
    queryKey: ['settings'],
    queryFn: () => api.getSettings(),
  });
  useEffect(() => {
    const zoom = settingsQuery.data?.ui.zoom;
    if (typeof zoom === 'number' && zoom > 0) {
      applyZoom(zoom);
    }
  }, [settingsQuery.data]);

  // Ctrl+S saves; Ctrl+Shift+I toggles the console.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (isConsoleShortcut(event)) {
        event.preventDefault();
        setConsoleOpen((value) => !value);
        return;
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        workspace.save();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [workspace]);

  return (
    <div className="app">
      <header className="app-header">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            <span />
            <span />
          </span>
          <span className="brand-text">
            <span className="brand-name">Parchment</span>
            <span className="brand-tag">A4 self-mailer studio</span>
          </span>
        </div>

        <label className="title-field">
          <span className="sr-only">Letter title</span>
          <input
            className="input title-input"
            type="text"
            value={draft?.title ?? ''}
            placeholder="Untitled letter"
            onChange={(event) => workspace.patch({ title: event.target.value })}
            disabled={!draft}
          />
        </label>

        <div className="header-actions">
          {workspace.dirty && <span className="dirty-dot" title="Unsaved changes" />}
          <button type="button" className="btn btn-sm" onClick={workspace.create}>
            New
          </button>
          <button
            type="button"
            className="btn btn-sm"
            onClick={workspace.save}
            disabled={!draft || workspace.saving}
          >
            Save
          </button>
          <button
            type="button"
            className="btn btn-sm btn-primary"
            onClick={workspace.exportPDF}
            disabled={!draft || draft.id === '' || workspace.exporting}
            title={
              draft?.id === ''
                ? 'Save the draft before exporting'
                : 'Write the two-page A4 PDF'
            }
          >
            Export PDF
          </button>
          <button
            type="button"
            className={`btn btn-sm${consoleOpen ? ' btn-active' : ''}`}
            onClick={() => setConsoleOpen((value) => !value)}
            aria-pressed={consoleOpen}
            title="Show or hide the debug console (Ctrl+Shift+I)"
          >
            Console
            {workspace.consoleErrors > 0 && (
              <span className="console-error-count"> · {workspace.consoleErrors}</span>
            )}
          </button>
        </div>
      </header>

      <aside className="app-rail">
        <div className="rail-header">
          <span>Library</span>
          <span className="text-dim">{workspace.letters.length}</span>
        </div>
        <ul className="letter-list">
          {workspace.letters.map((letter) => (
            <li key={letter.id}>
              <button
                type="button"
                className={`letter-item${letter.id === draft?.id ? ' is-active' : ''}`}
                onClick={() => workspace.open(letter)}
              >
                <span className="letter-item-title">{letter.title}</span>
                <span className="letter-item-meta">
                  {letter.fold.panel_width.toFixed(0)}×{letter.fold.panel_height.toFixed(0)} mm
                  {' · '}
                  {letter.fold.rotation_deg.toFixed(1)}°
                </span>
              </button>
              <button
                type="button"
                className="icon-btn"
                title="Delete this draft"
                disabled={workspace.deleting}
                onClick={() => {
                  if (window.confirm(`Delete “${letter.title}”?`)) {
                    workspace.remove(letter.id);
                  }
                }}
              >
                ×
              </button>
            </li>
          ))}
          {workspace.letters.length === 0 && (
            <li className="rail-empty text-dim">
              No drafts yet. Press New to start one; the demo is seeded on first launch.
            </li>
          )}
        </ul>

        <div className="rail-footer">
          <div className="rail-stat">
            <span>Data</span>
            <code title={workspace.appInfo?.data_dir}>
              {shortPath(workspace.appInfo?.data_dir)}
            </code>
          </div>
          <div className="rail-stat">
            <span>Exports</span>
            <code title={workspace.appInfo?.export_dir}>
              {shortPath(workspace.appInfo?.export_dir)}
            </code>
          </div>
        </div>
      </aside>

      <main className="app-main">
        <nav className="tabs" role="tablist" aria-label="Working area">
          {TABS.map((item) => (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={tab === item.id}
              className={`tab${tab === item.id ? ' is-active' : ''}`}
              onClick={() => setTab(item.id)}
              title={item.hint}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="tab-body">
          {workspace.sceneError && (
            <p className="warn">The fold could not be built: {workspace.sceneError}</p>
          )}
          {!draft && <p className="text-dim">Loading the library…</p>}

          {draft && scene && tab === 'letter' && (
            <LetterEditor letter={draft} scene={scene} onChange={workspace.patch} />
          )}

          {draft && scene && tab === 'envelope' && (
            <EnvelopePanel letter={draft} scene={scene} onChange={workspace.patch} />
          )}

          {draft && scene && tab === 'preview' && (
            <PrintPreview
              scene={scene}
              letter={draft}
              sheets={workspace.sheets}
              showFoldLines={showFoldLines}
              onToggleFoldLines={setShowFoldLines}
            />
          )}

          {draft && scene && tab === 'fold' && (
            <FoldViewer
              scene={scene}
              canvasA={workspace.sheets.canvasA}
              canvasB={workspace.sheets.canvasB}
              textureVersion={workspace.sheets.version}
            />
          )}

          {draft && scene && tab === 'paper' && (
            <FoldSettings
              letter={draft}
              scene={scene}
              onChange={workspace.patch}
              onReset={workspace.resetFold}
            />
          )}
        </div>
      </main>

      <aside className="app-inspector">
        <section className="card">
          <h2>Packet</h2>
          {scene ? (
            <>
              <p className="metric-big">
                {scene.metrics.packet_width_mm.toFixed(0)} ×{' '}
                {scene.metrics.packet_height_mm.toFixed(0)}
                <span className="metric-unit">mm</span>
              </p>
              <p className="text-dim">
                {scene.metrics.height_in.toFixed(2)} × {scene.metrics.length_in.toFixed(2)} in
                {' · '}
                <span className={scene.metrics.within_limits ? 'ok' : 'bad'}>
                  {scene.metrics.within_limits ? 'inside limits' : 'out of limits'}
                </span>
              </p>
              <dl className="inspector">
                <div>
                  <dt>Back sealed</dt>
                  <dd>{(scene.metrics.back_coverage * 100).toFixed(1)}%</dd>
                </div>
                <div>
                  <dt>Overhang</dt>
                  <dd>{scene.metrics.protrusion_mm.toFixed(1)} mm</dd>
                </div>
                <div>
                  <dt>Panels</dt>
                  <dd>
                    {scene.panels.length} rigid · {scene.fold_order.length} hinges
                  </dd>
                </div>
              </dl>
            </>
          ) : (
            <p className="text-dim">—</p>
          )}
        </section>

        <section className="card">
          <h2>Diagonal</h2>
          {scene ? (
            <>
              <p className="metric-big">
                {scene.front_panel.rotation_deg.toFixed(2)}
                <span className="metric-unit">°</span>
              </p>
              <p className="text-dim">
                The address is printed at this angle on Side B and reads upright on the
                folded packet.
              </p>
              <dl className="inspector">
                <div>
                  <dt>From flat</dt>
                  <dd>
                    {scene.front_panel.transform_from_flat.rotation_deg.toFixed(2)}° · tx{' '}
                    {scene.front_panel.transform_from_flat.tx.toFixed(1)} · ty{' '}
                    {scene.front_panel.transform_from_flat.ty.toFixed(1)}
                  </dd>
                </div>
                <div>
                  <dt>To flat</dt>
                  <dd>
                    {scene.front_panel.transform_to_flat.rotation_deg.toFixed(2)}° · tx{' '}
                    {scene.front_panel.transform_to_flat.tx.toFixed(1)} · ty{' '}
                    {scene.front_panel.transform_to_flat.ty.toFixed(1)}
                  </dd>
                </div>
                <div>
                  <dt>Panel centre</dt>
                  <dd>
                    {scene.front_panel.centre.x.toFixed(1)},{' '}
                    {scene.front_panel.centre.y.toFixed(1)} mm on the sheet
                  </dd>
                </div>
              </dl>
            </>
          ) : (
            <p className="text-dim">—</p>
          )}
        </section>

        <section className="card">
          <h2>Checks</h2>
          {(scene?.metrics.warnings.length ?? 0) === 0 ? (
            <p className="ok">All invariants hold.</p>
          ) : (
            <ul className="warn-list">
              {scene?.metrics.warnings.map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
          )}
          {workspace.exported && (
            <p className="text-dim export-note">
              Last export: {workspace.exported.pages} pages,{' '}
              {Math.round(workspace.exported.bytes / 1024)} KB
              <br />
              <code>{workspace.exported.path}</code>
            </p>
          )}
        </section>
      </aside>

      {consoleOpen && (
        <ConsolePanel entries={workspace.entries} onClear={workspace.clearConsole} />
      )}

      {toast && (
        <div className={`toast toast-${toast.kind}`} role="status">
          {toast.message}
        </div>
      )}
    </div>
  );
}

/** Shortens a path for the rail. */
function shortPath(path: string | undefined): string {
  if (!path) {
    return '—';
  }
  const home = '/home/';
  if (path.startsWith(home)) {
    const rest = path.slice(home.length);
    const cut = rest.indexOf('/');
    return cut === -1 ? path : `~${rest.slice(cut)}`;
  }
  return path;
}

export default App;


