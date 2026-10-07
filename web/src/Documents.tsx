import { ReindexDocument } from './ReindexDocument';
import { DocumentPreview } from './DocumentPreview';
import { FileIcon } from './FileIcon';
import { ExtractionDetails } from './ExtractionDetails';
import { Accordion } from './Accordion';
import { DocumentHistory } from './DocumentHistory';
import { useEffect, useRef, useState } from 'react';
import { api, message } from './api';
import { WorkflowPanel } from './Workflow';
import { ClassificationForm } from './ClassificationForm';
import { approvalLabel } from './organizationTypes';
import { Notice } from './components';
import { availabilityLabel, indexReasonLabel, type Document } from './libraryTypes';

export function DocumentList({ documents, open }: { documents: Document[]; open: (document: Document, page?: number) => void }) {
  return <div className="document-list">{documents.map(document => <article className="document-row" key={document.id}>
    <FileIcon format={document.format} /><div className="document-summary">
      <button className="text-button document-title" onClick={() => open(document)}>{document.title || document.original_filename}</button>
      <small className="path-text">{document.relative_path}</small>
      <div className="document-badges"><span className="badge">{document.format?.toUpperCase()}</span><span className={`badge ${document.availability === 'available' ? 'positive' : ''}`}>{availabilityLabel[document.availability]}</span><span className="badge">{approvalLabel[document.approval_status]}</span><span className="muted">{document.format === 'pdf' ? `${document.page_count} ${document.page_count === 1 ? 'página' : 'páginas'}` : `${document.unit_count || 0} unidades de texto`}</span>{document.extraction_freshness === 'stale' && <span className="badge">Texto anterior conservado</span>}{document.extraction_freshness === 'none' && <span className="badge">{indexReasonLabel[document.index_block_reason] || 'Pendiente de extracción'}</span>}</div>
      {document.matches.map(match => <button className="snippet" key={match.page_number} onClick={() => open(document, match.page_number)}><span className="page-number">{match.context_label || (match.unit_kind === 'page' || document.format === 'pdf' ? `Pág. ${match.page_number}` : `Unidad ${match.page_number}`)}</span>{match.segments.map((segment, index) => segment.highlighted ? <mark key={index}>{segment.text}</mark> : <span key={index}>{segment.text}</span>)}</button>)}
    </div><button className="secondary compact" onClick={() => open(document)}>Ver ficha</button>
  </article>)}</div>;
}

export function DocumentViewer({ document: initial, initialPage, preferredCaseID, canRemove, close, changed }: { document: Document; initialPage?: number; preferredCaseID?: string; canRemove: boolean; close: () => void; changed: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [expanded,setExpanded]=useState(false);
  const [document, setDocument] = useState(initial);
  const [page, setPage] = useState(initialPage || 1);
  const [text, setText] = useState('');
  const [method, setMethod] = useState('');
 const [contextLabel, setContextLabel] = useState('');
 const unitCount = document.unit_count || document.page_count;
  const [error, setError] = useState('');
  const [showHistory, setShowHistory] = useState(false);
  useEffect(() => { const element = dialog.current; element?.showModal(); return () => element?.close(); }, []);
  useEffect(() => { const controller = new AbortController(); api<Document>(`/documents/${initial.id}`, { signal: controller.signal }).then(value => { setDocument(value); setPage(initialPage || value.first_text_unit || 1); }).catch(error => { if (!controller.signal.aborted) setError(message(error)); }); return () => controller.abort(); }, [initial.id, initialPage]);
  useEffect(() => {
    setText(''); setMethod(''); setContextLabel(''); if (!unitCount) return;
    const controller = new AbortController();
    api<{ text: string; extraction_method: string; context_label: string }>(`/documents/${document.id}/pages/${page}/text`, { signal: controller.signal })
      .then(value => { setText(value.text); setMethod(value.extraction_method); setContextLabel(value.context_label); }).catch(error => { if (!controller.signal.aborted) setError(message(error)); });
    return () => controller.abort();
  }, [document.id, unitCount, document.revision, document.processing?.id, page]);
  return <dialog ref={dialog} className={`document-dialog${expanded ? ' pdf-expanded' : ''}`} onCancel={event=>{event.preventDefault();if(expanded)setExpanded(false);else close();}} aria-labelledby="document-heading">
    <div className="section-heading"><div><p className="eyebrow">DOCUMENTO {document.storage_source === 'managed' ? 'ADMINISTRADO' : 'VINCULADO'}</p><h2 id="document-heading">{document.title || document.original_filename}</h2></div><button className="secondary" onClick={close}>Cerrar ficha</button></div>
    <Notice text={error} /><div className="document-badges">{document.can_preview_original&&<button className="secondary compact" aria-pressed={expanded} onClick={()=>setExpanded(!expanded)}>{expanded?'Volver a vista dividida':'Ampliar PDF'}</button>}<span className="badge">{availabilityLabel[document.availability]}</span>{document.extraction_freshness === 'stale' && <span className="badge">Texto anterior conservado</span>}{document.can_download && <a className="button-link" href={`/api/v1/documents/${document.id}/download`}>Descargar {document.format?.toUpperCase() || 'original'}</a>}</div>
    <p>{approvalLabel[document.approval_status]}{document.case_identifier ? ` · Expediente ${document.case_identifier}` : ' · Sin expediente'}</p>
    {(document.category_name || document.document_type_name) && <p>{document.category_name} · {document.document_type_name}</p>}
    {document.original_path && <p className="path-text muted">{document.storage_source === 'managed' ? 'Ruta definitiva' : 'Ruta original'}: {document.original_path}</p>}
    <p className="muted">Formato: {document.format?.toUpperCase()} · MIME: {document.detected_mime}{document.extension_mismatch && ' · Extensión distinta: documento conservado según la política de la biblioteca.'}</p><div className="document-preview-grid">
      {document.can_preview_original ? <iframe title="Vista previa del PDF" src={`/api/v1/documents/${document.id}/content#page=${page}`} /> : ['docx','xlsx'].includes(document.format) ? <DocumentPreview documentID={document.id} hash={document.sha256} format={document.format}/> : <div className="unavailable-preview"><h3>{document.format !== 'pdf' ? 'Vista previa no disponible para este formato' : 'Original no disponible'}</h3><p>{document.format !== 'pdf' ? 'Puedes descargar el original con el permiso correspondiente.' : 'El texto extraído permanece disponible para consulta.'}</p></div>}
      <section className="retained-text"><div className="section-heading"><h3>Texto extraído</h3><label>{document.format === 'pdf' ? 'Página' : 'Unidad de texto'}<input type="number" min={1} max={Math.max(unitCount, 1)} value={page} onChange={event => { const number = Number(event.target.value); if (number >= 1 && number <= unitCount) setPage(number); }} /></label><span>de {unitCount} {document.format === 'pdf' ? 'páginas' : 'unidades'}</span></div><div className="actions"><button className="secondary compact" disabled={page <= 1} onClick={() => setPage(page - 1)}>Anterior</button><button className="secondary compact" disabled={page >= unitCount} onClick={() => setPage(page + 1)}>Siguiente</button></div>{contextLabel && <p>{contextLabel}</p>}<small>{method === 'ocr' ? 'Reconocimiento OCR' : method === 'native' ? 'Texto nativo' : ''}</small><pre>{text || (unitCount || document.processing?.status.startsWith('complete') ? 'Esta unidad está vacía. Consulta otra unidad con los botones Anterior y Siguiente.' : indexReasonLabel[document.index_block_reason] || 'La extracción está pendiente.')}</pre></section>
    </div>
    <ExtractionDetails run={document.processing} />
    {document.can_reindex && <ReindexDocument key={document.id} document={document} updated={value => { setDocument(value); setPage(value.first_text_unit || 1); changed(); }} />}
    {document.approval_status === 'pending_review' && document.can_classify && <p className="muted">Cambiar la clasificación invalida el envío actual y requiere enviarlo nuevamente a revisión.</p>}
    {['draft', 'rejected', 'pending_review', 'needs_review'].includes(document.approval_status) && (document.can_classify || document.can_associate || document.can_reassign) && <ClassificationForm key={document.revision} document={document} preferredCaseID={preferredCaseID} saved={updated => { setDocument(updated); changed(); }} />}
    <WorkflowPanel document={document} saved={updated => { setDocument(updated); changed(); }} />
    {document.can_cancel && <Accordion title="Cancelar mi carga"><form className="filters" onSubmit={async event => { event.preventDefault(); try { await api(`/documents/${document.id}/cancel`, { method: 'POST', revision: document.revision, body: { reason: new FormData(event.currentTarget).get('reason') } }); setDocument(await api<Document>(`/documents/${document.id}`)); changed(); } catch (error) { setError(message(error)); } }}><label>Motivo de cancelación<input name="reason" required maxLength={1000} /></label><button className="secondary">Cancelar carga y conservar evidencia</button></form></Accordion>}
    <Accordion title="Huella y trazabilidad"><p className="path-text">SHA-256: {document.sha256}</p><button className="secondary" onClick={()=>setShowHistory(true)}>Consultar historial</button>{showHistory && <DocumentHistory key={document.processing?.id} documentID={document.id} />}</Accordion>
    {canRemove && document.approval_status !== 'materializing' && <details className="remove-index document-accordion"><summary>Retirar del índice</summary><p>Oculta el documento de las búsquedas y del listado de la biblioteca. Elimina el texto del índice de búsqueda. Conserva el archivo físico, los registros históricos de texto y OCR y la auditoría. Volver a recorrer la carpeta no lo incorpora automáticamente otra vez.</p><form className="filters" onSubmit={async event => { event.preventDefault(); const data = new FormData(event.currentTarget); try { await api(`/documents/${document.id}/remove-index`, { method: 'POST', revision: document.revision, body: { reason: data.get('reason'), expected_case_id: document.case_id } }); changed(); close(); } catch (error) { setError(message(error)); } }}>{document.case_id && <label className="check"><input type="checkbox" required />Entiendo que este archivo dejará de contar en el expediente {document.case_identifier}.</label>}<label className="check"><input type="checkbox" required />Confirmo que se retirará del índice de AIBID. El archivo físico permanecerá intacto.</label><label>Motivo<input name="reason" required maxLength={1000} /></label><button className="secondary">Confirmar retiro del índice</button></form></details>}
  </dialog>;
}
