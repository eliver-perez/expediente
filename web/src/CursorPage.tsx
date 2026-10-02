import { useEffect, useState } from 'react';
import { api, message, type Page } from './api';

// One bounded page per section. Changing scope aborts the old request and resets
// its cursor, so a response from a previous folder cannot replace the new one.
export function useCursorPage<T>(endpoint: string) {
  const [position, setPosition] = useState({ endpoint, cursor: '', previous: [] as string[] });
  const [result, setResult] = useState<{ endpoint: string; cursor: string; page: Page<T> } | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const current = position.endpoint === endpoint ? position : { endpoint, cursor: '', previous: [] as string[] };
  const cursor = current.cursor;
  useEffect(() => {
    const controller = new AbortController(); setBusy(true); setError('');
    api<Page<T>>(`${endpoint}${endpoint.includes('?') ? '&' : '?'}cursor=${encodeURIComponent(cursor)}`, { signal: controller.signal })
      .then(page => setResult({ endpoint, cursor, page }))
      .catch(cause => { if (!controller.signal.aborted) setError(message(cause)); })
      .finally(() => { if (!controller.signal.aborted) setBusy(false); });
    return () => controller.abort();
  }, [endpoint, cursor, revision]);
  const page = result?.endpoint === endpoint && result.cursor === cursor ? result.page : null;
  return { items: page?.items || [], error, busy: busy || (!page && !error), pageNumber: current.previous.length + 1,
    canBack: current.previous.length > 0, canNext: !!page?.next_cursor,
    back: () => setPosition({ endpoint, cursor: current.previous.at(-1) || '', previous: current.previous.slice(0, -1) }),
    next: () => { if (page?.next_cursor) setPosition({ endpoint, cursor: page.next_cursor, previous: [...current.previous, cursor] }); },
    reload: () => setRevision(value => value + 1) };
}

export function CursorControls({ page, label }: { page: ReturnType<typeof useCursorPage>; label: string }) {
  return <nav className="cursor-controls" aria-label={label}>
    <button className="secondary compact" disabled={page.busy || !page.canBack} onClick={page.back}>Anterior</button>
    <span>{label} · Página {page.pageNumber}</span>
    <button className="secondary compact" disabled={page.busy || !page.canNext} onClick={page.next}>Siguiente</button>
  </nav>;
}
