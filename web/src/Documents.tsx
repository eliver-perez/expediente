import { useEffect, useRef, useState } from 'react';
import { api, formatDate, message } from './api';
import { Notice } from './components';
import { availabilityLabel, type Document } from './libraryTypes';

export function DocumentList({ documents, open }: { documents: Document[]; open: (document: Document, page?: number) => void }) {
  return <div className="document-list">{documents.map(document => <article className="document-row" key={document.id}>
    <div className="pdf-symbol" aria-hidden="true">PDF</div><div className="document-summary">
      <button className="text-button document-title" onClick={() => open(document)}>{document.title || document.original_filename}</button>
      <small className="path-text">{document.relative_path}</small>
      <div className="document-badges"><span className={`badge ${document.availability === 'available' ? 'positive' : ''}`}>{availabilityLabel[document.availability]}</span><span className="muted">{document.page_count} {document.page_count === 1 ? 'página' : 'páginas'}</span>{document.extraction_freshness === 'stale' && <span className="badge">Texto anterior conservado</span>}{document.extraction_freshness === 'none' && <span className="badge">Pendiente de extracción</span>}</div>
      {document.matches.map(match => <button className="snippet" key={match.page_number} onClick={() => open(document, match.page_number)}><span className="page-number">Pág. {match.page_number}</span>{match.segments.map((segment, index) => segment.highlighted ? <mark key={index}>{segment.text}</mark> : <span key={index}>{segment.text}</span>)}</button>)}
    </div><button className="secondary compact" onClick={() => open(document)}>Ver ficha</button>
  </article>)}</div>;
}

export function DocumentViewer({ document: initial, initialPage = 1, canRemove, close, changed }: { document: Document; initialPage?: number; canRemove: boolean; close: () => void; changed: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [document, setDocument] = useState(initial);
  const [page, setPage] = useState(initialPage);
  const [text, setText] = useState('');
  const [method, setMethod] = useState('');
  const [error, setError] = useState('');
  const [history, setHistory] = useState<{ id: string; event_type: string; occurred_at: string }[] | null>(null);
  useEffect(() => { const element = dialog.current; element?.showModal(); return () => element?.close(); }, []);
  useEffect(() => { const controller = new AbortController(); api<Document>(`/documents/${initial.id}`, { signal: controller.signal }).then(setDocument).catch(error => { if (!controller.signal.aborted) setError(message(error)); }); return () => controller.abort(); }, [initial.id]);
  useEffect(() => {
    setText(''); setMethod(''); if (!document.page_count) return;
    const controller = new AbortController();
    api<{ text: string; extraction_method: string }>(`/documents/${document.id}/pages/${page}/text`, { signal: controller.signal })
      .then(value => { setText(value.text); setMethod(value.extraction_method); }).catch(error => { if (!controller.signal.aborted) setError(message(error)); });
    return () => controller.abort();
  }, [document.id, document.page_count, page]);
  return <dialog ref={dialog} className="document-dialog" onCancel={close} aria-labelledby="document-heading">
    <div className="section-heading"><div><p className="eyebrow">DOCUMENTO VINCULADO</p><h2 id="document-heading">{document.title || document.original_filename}</h2></div><button className="secondary" onClick={close}>Cerrar ficha</button></div>
    <Notice text={error} /><div className="document-badges"><span className="badge">{availabilityLabel[document.availability]}</span>{document.extraction_freshness === 'stale' && <span className="badge">Texto anterior conservado</span>}{document.can_download && <a className="button-link" href={`/api/v1/documents/${document.id}/download`}>Descargar PDF</a>}</div>
    {document.original_path && <p className="path-text muted">Ruta original: {document.original_path}</p>}
    <div className="document-preview-grid">
      {document.can_preview_original ? <iframe title="Vista previa del PDF" src={`/api/v1/documents/${document.id}/content#page=${page}`} /> : <div className="unavailable-preview"><h3>Original no disponible</h3><p>El texto extraído permanece disponible para consulta.</p></div>}
      <section className="retained-text"><div className="section-heading"><h3>Texto extraído</h3><label>Página<input type="number" min={1} max={Math.max(document.page_count, 1)} value={page} onChange={event => { const number = Number(event.target.value); if (number >= 1 && number <= document.page_count) setPage(number); }} /></label></div><small>{method === 'ocr' ? 'Reconocimiento OCR' : method === 'native' ? 'Texto nativo' : ''}</small><pre>{text || (document.page_count ? 'Esta página no contiene texto.' : 'La extracción está pendiente.')}</pre></section>
    </div>
    <details><summary>Huella y trazabilidad</summary><p className="path-text">SHA-256: {document.sha256}</p><button className="secondary" onClick={() => void api<{ items: { id: string; event_type: string; occurred_at: string }[] }>(`/documents/${document.id}/history`).then(value => setHistory(value.items)).catch(error => setError(message(error)))}>Consultar historial</button>{history && <ul>{history.map(event => <li key={event.id}>{formatDate(event.occurred_at)} · {event.event_type}</li>)}</ul>}</details>
    {canRemove && <details className="remove-index"><summary>Retirar del índice</summary><p>Conserva el archivo original y la evidencia histórica. Dejará de aparecer en búsquedas.</p><form className="filters" onSubmit={async event => { event.preventDefault(); const data = new FormData(event.currentTarget); try { await api(`/documents/${document.id}/remove-index`, { method: 'POST', revision: document.revision, body: { reason: data.get('reason') } }); changed(); close(); } catch (error) { setError(message(error)); } }}><label>Motivo<input name="reason" required maxLength={1000} /></label><button className="secondary">Confirmar retiro del índice</button></form></details>}
  </dialog>;
}
