import { useEffect, useState } from 'react';
import { api, message, formatDate } from './api';
import { Notice, PageEnd, usePage } from './components';
import { diagnosticLabel, type Document, type Library } from './libraryTypes';
import { DocumentViewer } from './Documents';

interface Review { id: string; document_id: string; document_revision: number; title: string; author_name: string; requested_at: string; status: string; reason: string }
interface Materialization { id: string; state: string; error_code: string }
interface Workflow { processing: boolean; requires_review: boolean; can_submit: boolean; can_finalize: boolean; can_approve: boolean; can_reject: boolean; can_retry: boolean; review: Review | null; materialization: Materialization | null }
const operationLabel: Record<string, string> = { planned: 'Esperando turno', copying: 'Copiando al destino', verified: 'Copia verificada', published: 'Confirmando guardado', committed: 'Documento aprobado; retirando temporal', cleaned: 'Guardado definitivo completado', failed: 'El guardado requiere atención' };
const operationStep: Record<string, number> = { planned: 0, copying: 1, verified: 2, published: 3, committed: 4, cleaned: 5 };
export function WorkflowPanel({ document, saved }: { document: Document; saved: (document: Document) => void }) {
  const [workflow, setWorkflow] = useState<Workflow | null>(null); const [error, setError] = useState(''); const [success, setSuccess] = useState(''); const [busy, setBusy] = useState(false); const [reload, setReload] = useState(0);
  const [reviewers, setReviewers] = useState<{ id: string; display_name: string }[]>([]);
  useEffect(() => { const controller = new AbortController(); api<Workflow>(`/documents/${document.id}/workflow`, { signal: controller.signal }).then(setWorkflow).catch(error => { if (!controller.signal.aborted) setError(message(error)); }); return () => controller.abort(); }, [document.id, document.revision, reload]);
  useEffect(() => { if (!workflow?.can_submit) return; const controller = new AbortController(); api<{ items: { id: string; display_name: string }[] }>(`/libraries/${document.library_id}/reviewers`, { signal: controller.signal }).then(result => setReviewers(result.items.filter(person => person.id !== document.created_by))).catch(error => { if (!controller.signal.aborted) setError(message(error)); }); return () => controller.abort(); }, [workflow?.can_submit, document.library_id, document.created_by]);
  useEffect(() => {
    const state = workflow?.materialization?.state;
    if (!state || state === 'cleaned' || state === 'failed' && !workflow?.processing) return;
    const controller = new AbortController();
    const timer = window.setTimeout(async () => { try { const [updated, next] = await Promise.all([api<Document>(`/documents/${document.id}`, { signal: controller.signal }), api<Workflow>(`/documents/${document.id}/workflow`, { signal: controller.signal })]); setWorkflow(next); if (updated.revision !== document.revision) saved(updated); setReload(value => value + 1); } catch (error) { if (!controller.signal.aborted) setError(message(error)); } }, 1500);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [document.id, document.revision, workflow, saved]);
  async function act(endpoint: string, body: unknown, text: string, revision?: number) {
    setBusy(true); setError(''); setSuccess('');
    try { await api(endpoint, { method: 'POST', body, revision }); saved(await api<Document>(`/documents/${document.id}`)); setReload(value => value + 1); setSuccess(text); }
    catch (error) { setError(message(error)); setReload(value => value + 1); }
    finally { setBusy(false); }
  }
  if (!workflow) return <Notice text={error} />;
  const operation = workflow.materialization;
  if (document.storage_source === 'linked' && !document.category_id && !workflow.review && !operation) return null;
  return <section className="workflow-panel" aria-label="Revisión y guardado"><h3>Revisión y guardado</h3><Notice text={error} /><Notice text={success} kind="success" />
    {document.integrity_status === 'changed' && <p className="notice error">El archivo administrado cambió fuera de AIBID. Revisa la versión actual antes de aprobarla. Por ahora no cuenta como aprobado válido en el expediente.</p>}
    {document.approval_status === 'pending_review' && <p>{workflow.review?.document_revision === document.revision ? 'Pendiente de decisión de un revisor autorizado.' : 'El documento cambió después del envío. Debe enviarse nuevamente a revisión.'}</p>}
    {workflow.review?.status === 'rejected' && <p><strong>Motivo del rechazo:</strong> {workflow.review.reason}</p>}
    {operation && <div className="materialization-progress" role="status"><strong>{operation.state === 'failed' && workflow.processing ? 'Esperando reintento automático' : operationLabel[operation.state] || operation.state}</strong>{operation.state !== 'failed' && <progress aria-label="Etapas del guardado definitivo" max={5} value={operationStep[operation.state] || 0} />}{operation.error_code && <p>{diagnosticLabel[operation.error_code] || 'El guardado se interrumpió. Puedes consultar la tarea y reintentar.'}</p>}{!['committed', 'cleaned'].includes(operation.state) && <p className="muted">El temporal se conserva hasta confirmar el archivo definitivo.</p>}</div>}
    {workflow.can_submit && <form className="filters" onSubmit={event => { event.preventDefault(); void act(`/documents/${document.id}/submit`, { reviewer_id: new FormData(event.currentTarget).get('reviewer') }, 'Documento enviado a revisión.', document.revision); }}><label>Revisor<select name="reviewer" defaultValue=""><option value="">Cola compartida de revisores</option>{reviewers.map(person => <option key={person.id} value={person.id}>{person.display_name}</option>)}</select></label><button disabled={busy}>{document.approval_status === 'pending_review' ? 'Actualizar envío a revisión' : 'Enviar a revisión'}</button></form>}
    {workflow.can_finalize && <div><p>Esta biblioteca permite la confirmación directa para este origen. {document.availability === 'staged' ? 'El documento se aprobará cuando termine el guardado en su carpeta definitiva.' : 'Se comprobará el archivo actual antes de registrar la aprobación.'}</p><button disabled={busy} onClick={() => void act(`/documents/${document.id}/finalize`, {}, 'Confirmación recibida.', document.revision)}>Confirmar documento</button></div>}
    {(workflow.can_approve || workflow.can_reject) && workflow.review && <form className="form-grid" onSubmit={event => { event.preventDefault(); void act(`/reviews/${workflow.review!.id}/reject`, { expected_document_revision: document.revision, reason: new FormData(event.currentTarget).get('reason') }, 'Documento rechazado. El motivo queda en el historial.'); }}><p>La decisión corresponde al documento y la clasificación que se muestran en esta ficha.</p>{workflow.can_approve && <button type="button" disabled={busy} onClick={() => void act(`/reviews/${workflow.review!.id}/approve`, { expected_document_revision: document.revision }, 'Aprobación recibida. Si hay una carga temporal, se confirmará al terminar el guardado.')}>Aprobar documento</button>}{workflow.can_reject && <><label>Motivo del rechazo<textarea name="reason" required maxLength={1000} /></label><button className="secondary" disabled={busy}>Rechazar documento</button></>}</form>}
    {workflow.can_retry && operation && (operation.state === 'failed' || operation.error_code) && <form className="filters" onSubmit={event => { event.preventDefault(); void act(`/materializations/${operation.id}/retry`, { reason: new FormData(event.currentTarget).get('reason') }, 'Reintento solicitado.'); }}><label>Motivo del reintento<input name="reason" required maxLength={1000} /></label><button disabled={busy}>Reintentar guardado</button></form>}
  </section>;
}
export function Reviews({ library }: { library: Library }) {
  const queue = usePage<Review>(`/libraries/${library.id}/reviews`); const [selected, setSelected] = useState<Document | null>(null); const [error, setError] = useState('');
  return <section className="card"><div className="section-heading"><div><h2>Pendientes de revisión</h2><p className="muted">Solicitudes asignadas a ti y a la cola compartida de esta biblioteca.</p></div><button className="secondary" onClick={queue.reload}>Actualizar pendientes</button></div><Notice text={error || queue.error} />
    {queue.items.map(review => <article className="review-row" key={review.id}><div><strong>{review.title}</strong><small>Enviado por {review.author_name} · {formatDate(review.requested_at)}</small></div><button className="secondary" onClick={() => void api<Document>(`/documents/${review.document_id}`).then(setSelected).catch(error => setError(message(error)))}>Revisar documento</button></article>)}<PageEnd {...queue} empty={queue.items.length === 0} />
    {selected && <DocumentViewer document={selected} canRemove={library.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={queue.reload} />}
  </section>;
}
