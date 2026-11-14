/**
 * The workspace state: the open draft, the fold scene derived from it,
 * the library and the actions.
 *
 * It is a hook rather than a set of components so the layout stays a
 * layout. Every backend call goes through `bindings`; the scene is
 * recomputed on the backend whenever the fold parameters change, on a
 * debounce so a slider drag does not queue a scene per pixel.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Events } from '@wailsio/runtime';

import * as api from '../bindings';
import type { AppInfo, ConsoleEntry, ExportResult, Letter, Params, Scene } from '../bindings';
import { usePrintSheets, type PrintSheets } from '../features/preview/usePrintSheets';

/** The scene is rebuilt after this many milliseconds of quiet. */
const SCENE_DEBOUNCE_MS = 110;

/** The debug console refreshes on this interval as a fallback. */
const CONSOLE_POLL_MS = 4000;

export interface Workspace {
  draft: Letter | null;
  scene: Scene | null;
  sceneError: string | null;
  dirty: boolean;
  sheets: PrintSheets;
  letters: Letter[];
  entries: ConsoleEntry[];
  consoleErrors: number;
  appInfo: AppInfo | undefined;
  exported: ExportResult | null;
  saving: boolean;
  deleting: boolean;
  exporting: boolean;

  patch: (changes: Partial<Letter>) => void;
  open: (letter: Letter) => void;
  create: () => void;
  save: () => void;
  remove: (letterId: string) => void;
  exportPDF: () => void;
  resetFold: () => void;
  clearConsole: () => void;
}

/** Builds the workspace and keeps it in step with the backend. */
export function useWorkspace(
  notify: (message: string, kind?: 'info' | 'error' | 'success') => void,
  showFoldLines: boolean,
): Workspace {
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<Letter | null>(null);
  const [dirty, setDirty] = useState(false);
  const [exported, setExported] = useState<ExportResult | null>(null);
  const [scene, setScene] = useState<Scene | null>(null);
  const [sceneError, setSceneError] = useState<string | null>(null);

  const lettersQuery = useQuery({
    queryKey: ['letters'],
    queryFn: () => api.listLetters(),
  });

  const consoleQuery = useQuery({
    queryKey: ['console'],
    queryFn: () => api.listConsoleEntries(),
    refetchInterval: CONSOLE_POLL_MS,
  });

  const appInfoQuery = useQuery({
    queryKey: ['appInfo'],
    queryFn: () => api.getAppInfo(),
  });

  // The first draft opens when the library arrives.
  useEffect(() => {
    if (draft !== null) {
      return;
    }
    const first = lettersQuery.data?.[0];
    if (first) {
      setDraft(first);
    }
  }, [lettersQuery.data, draft]);

  // The console pushes entries as events; the interval is the fallback.
  useEffect(() => {
    const unsubscribe = Events.On(api.EVENT_CONSOLE_ENTRY, () => {
      void queryClient.invalidateQueries({ queryKey: ['console'] });
    });
    return () => {
      unsubscribe();
    };
  }, [queryClient]);

  // --- Scene --------------------------------------------------------

  // The scene only depends on the fold parameters. Keeping its identity
  // stable while the user types means the 3D viewer does not rebuild its
  // panel tree on every keystroke.
  const foldRef = useRef(draft?.fold);
  foldRef.current = draft?.fold;
  const foldKey = JSON.stringify(draft?.fold ?? null);

  useEffect(() => {
    const params = foldRef.current;
    if (!params) {
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      api
        .foldScene(params)
        .then((next) => {
          if (!cancelled) {
            setScene(next);
            setSceneError(null);
          }
        })
        .catch((error: unknown) => {
          if (!cancelled) {
            setSceneError(errorText(error));
          }
        });
    }, SCENE_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [foldKey]);

  const sheets = usePrintSheets(scene, draft, showFoldLines);



  // --- Mutations ----------------------------------------------------

  const saveMutation = useMutation({
    mutationFn: (letter: Letter) => api.saveLetter(letter),
    onSuccess: (saved) => {
      setDraft(saved);
      setDirty(false);
      void queryClient.invalidateQueries({ queryKey: ['letters'] });
      notify('Saved to the embedded database.', 'success');
    },
    onError: (error: unknown) => notify(`Could not save: ${errorText(error)}`, 'error'),
  });

  const deleteMutation = useMutation({
    mutationFn: (letterId: string) => api.deleteLetter(letterId),
    onSuccess: async () => {
      setDraft(null);
      setDirty(false);
      await queryClient.invalidateQueries({ queryKey: ['letters'] });
      notify('Draft deleted.', 'info');
    },
    onError: (error: unknown) => notify(`Could not delete: ${errorText(error)}`, 'error'),
  });

  const exportMutation = useMutation({
    mutationFn: (letterId: string) => api.exportLetterPDF(letterId, showFoldLines),
    onSuccess: (result) => {
      setExported(result);
      notify(`Exported ${result.pages} A4 pages to ${result.path}`, 'success');
    },
    onError: (error: unknown) => notify(`Could not export: ${errorText(error)}`, 'error'),
  });

  // --- Actions ------------------------------------------------------

  const patch = useCallback((changes: Partial<Letter>) => {
    setDraft((current) => (current ? { ...current, ...changes } : current));
    setDirty(true);
  }, []);

  const open = useCallback((letter: Letter) => {
    setDraft(letter);
    setDirty(false);
    setExported(null);
  }, []);

  const create = useCallback(() => {
    void api.defaultFoldParams().then((params: Params) => {
      setDraft(blankLetter(params));
      setDirty(true);
      setExported(null);
    });
  }, []);

  const save = useCallback(() => {
    if (draft) {
      saveMutation.mutate(draft);
    }
  }, [draft, saveMutation]);

  const remove = useCallback(
    (letterId: string) => deleteMutation.mutate(letterId),
    [deleteMutation],
  );

  const exportPDF = useCallback(() => {
    if (draft && draft.id !== '') {
      exportMutation.mutate(draft.id);
    }
  }, [draft, exportMutation]);

  const resetFold = useCallback(() => {
    void api.defaultFoldParams().then((params: Params) => patch({ fold: params }));
  }, [patch]);

  const clearConsole = useCallback(() => {
    void api.clearConsoleEntries().then(() => {
      void queryClient.invalidateQueries({ queryKey: ['console'] });
    });
  }, [queryClient]);

  const entries = consoleQuery.data ?? [];

  return {
    draft,
    scene,
    sceneError,
    dirty,
    sheets,
    letters: lettersQuery.data ?? [],
    entries,
    consoleErrors: entries.filter((entry) => entry.level === 'error').length,
    appInfo: appInfoQuery.data,
    exported,
    saving: saveMutation.isPending,
    deleting: deleteMutation.isPending,
    exporting: exportMutation.isPending,
    patch,
    open,
    create,
    save,
    remove,
    exportPDF,
    resetFold,
    clearConsole,
  };
}

/** A new, empty draft with the given fold. */
export function blankLetter(fold: Params): Letter {
  return {
    id: '',
    title: 'Untitled letter',
    side_a: '',
    recipient_name: '',
    recipient_street: '',
    recipient_city: '',
    recipient_state: '',
    recipient_zip: '',
    return_name: '',
    return_street: '',
    return_city: '',
    return_state: '',
    return_zip: '',
    fold,
    created: '',
    updated: '',
  };
}

/** Extracts a readable message from an unknown rejection. */
export function errorText(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
