import { useEffect, useRef, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';
import { diagnosticLabel, jobLabel, type Document, type Job } from './libraryTypes';
import { newRequestID } from './requestID';

export function ReindexDocument({ document, updated }: { document: Document; updated: (value: Document) => void }) {
  const [requestKey, setRequestKey] = useState(newRequestID);
  const [jobID, setJobID] = useState('');
  const [status, setStatus] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const onUpdate = useRef(updated);
  useEffect(() => { onUpdate.current = updated; }, [updated]);
  useEffect(() => {
    if (!jobID) return;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const job = await api<Job>(`/jobs/${jobID}`, { signal: controller.signal });
        setStatus(jobLabel[job.status] || job.status);
        if (['succeeded', 'failed', 'cancelled'].includes(job.status)) {
          const value = await api<Document>(`/documents/${document.id}`, { signal: controller.signal });
          onUpdate.current(value);
          setJobID(''); setStatus(''); setRequestKey(newRequestID());
          if (job.status === 'succeeded') setSuccess('Reindexación completada. La ficha y el texto están actualizados.');
          else setError(diagnosticLabel[job.last_error_code] || 'La reindexación no terminó. Consulta el diagnóstico en Procesamiento; el texto anterior se conserva.');
          return;
        }
      } catch (cause) {
        if (controller.signal.aborted) return;
        setError(message(cause));
      }
      if (!controller.signal.aborted) timer = setTimeout(() => void poll(), 1500);
    }
    void poll();
    return () => { controller.abort(); clearTimeout(timer); };
  }, [jobID, document.id]);
  return <section className="document-accordion" aria-label="Reindexación del documento">
    <p>Vuelve a extraer el contenido con la configuración actual. Conserva el original y el historial. El texto anterior sigue disponible mientras se procesa.</p>
    <Notice text={error} /><Notice text={success} kind="success" />
    {status && <p role="status">Reindexación: {status}. Puedes cerrar esta ficha; el trabajo continuará en Procesamiento.</p>}
    <button className="secondary" disabled={busy || !!jobID} onClick={async () => {
      setBusy(true); setError(''); setSuccess('');
      try {
        const job = await api<{ id: string }>(`/documents/${document.id}/reindex`, { method: 'POST', body: {}, idempotencyKey: requestKey });
        setJobID(job.id); setStatus('En cola');
      } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
    }}>{busy ? 'Comprobando original…' : 'Reindexar documento'}</button>
  </section>;
}
