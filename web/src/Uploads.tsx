import { newRequestID } from './requestID';
import { useState } from 'react';
import { api, APIError, formatDate, message, uploadDocument } from './api';
import { Notice } from './components';
import { CursorControls, useCursorPage } from './CursorPage';
import { UploadHistoryFiles } from './UploadHistory';
import { formatSize } from './ExplorerEntries';
import { DocumentViewer } from './Documents';
import type { Document, Library } from './libraryTypes';
import type { UploadBatch, UploadItem } from './organizationTypes';

interface PendingFile { file: File; key: string; percent: number; status: 'waiting' | 'uploading' | 'staged' | 'failed'; error: string; item?: UploadItem; replaceKey?: boolean }
export function Uploads({ library }: { library: Library }) {
  const [files, setFiles] = useState<PendingFile[]>([]); const [batchKey, setBatchKey] = useState(() => newRequestID()); const [batchID, setBatchID] = useState('');
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [selected, setSelected] = useState<Document | null>(null); const [batch, setBatch] = useState(''); const [historyRevision,setHistoryRevision]=useState(0);
  const batches = useCursorPage<UploadBatch>(`/libraries/${library.id}/upload-batches`);
  function addFiles(incoming: File[]) { if (busy) return; if (files.length + incoming.length > 100) { setError('Cada carga admite hasta 100 archivos. Inicia otra carga.'); return; } setFiles(previous => [...previous, ...incoming.map(file => ({ file, key: newRequestID(), percent: 0, status: 'waiting' as const, error: '' }))]); setError(''); }
  async function open(documentID: string) { try { setSelected(await api<Document>(`/documents/${documentID}`)); } catch (error) { setError(message(error)); } }
  async function send() {
    setBusy(true); setError('');
    try {
      const identifier = batchID || (await api<{ id: string }>(`/libraries/${library.id}/upload-batches`, { method: 'POST', body: { client_batch_id: batchKey } })).id; setBatchID(identifier);
      for (const pending of files.filter(file => file.status === 'waiting' || file.status === 'failed')) {
        const key = pending.replaceKey ? newRequestID() : pending.key;
        const update = (patch: Partial<PendingFile>) => setFiles(previous => previous.map(file => file.key === pending.key || file.key === key ? { ...file, ...patch, key } : file));
        update({ status: 'uploading', error: '', percent: 0 });
        try { const item = await uploadDocument<UploadItem>(identifier, key, pending.file, percent => update({ percent })); update({ status: 'staged', item, percent: 100 }); }
        catch (error) { update({ status: 'failed', error: message(error), replaceKey: error instanceof APIError && error.status !== 0 && error.status !== 429 }); if (error instanceof APIError && error.status === 401) break; }
      }
      batches.reload(); setBatch(identifier); setHistoryRevision(value=>value+1);
    } catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  return <><section className="card"><h2>Cargar documentos</h2><p className="muted">Se guardan como borradores privados. Solo tú y los revisores autorizados pueden consultarlos antes de su incorporación definitiva.</p><p>Formatos admitidos: PDF, DOCX, XLSX, TXT y CSV, según las reglas de esta biblioteca. El servidor verifica el contenido real. La extracción de contenido depende de las reglas de indexación; el OCR se aplica a PDF.</p><Notice text={error} />
    <div className="upload-dropzone" onDragOver={event => event.preventDefault()} onDrop={event => { event.preventDefault(); addFiles(Array.from(event.dataTransfer.files)); }}><strong>Arrastra aquí uno o varios documentos</strong><span>o selecciónalos desde tu equipo</span><label>Archivos<input type="file" multiple disabled={busy} onChange={event => { addFiles(Array.from(event.target.files || [])); event.target.value = ''; }} /></label></div>
    {files.length > 0 && <><div className="upload-list">{files.map(file => <article className="upload-row" key={file.key}><div><strong>{file.file.name}</strong><small>{formatSize(file.file.size)} · {file.status === 'waiting' ? 'Pendiente' : file.status === 'uploading' ? file.percent === 100 ? 'Validando documento…' : `Enviando ${file.percent} %` : file.status === 'staged' ? 'Borrador guardado' : 'Carga incompleta'}</small>{file.status === 'uploading' && <progress value={file.percent} max={100} aria-label={`Carga de ${file.file.name}`} />}{file.error && <p role="alert" className="diagnostic">{file.error}</p>}</div>{file.item && <button className="secondary" onClick={() => void open(file.item!.document_id)}>Clasificar {file.file.name}</button>}{file.status === 'waiting' && <button className="text-button" disabled={busy} onClick={() => setFiles(files.filter(item => item.key !== file.key))}>Quitar {file.file.name}</button>}</article>)}</div><div className="actions"><button disabled={busy || !files.some(file => file.status === 'waiting' || file.status === 'failed')} onClick={() => void send()}>{busy ? 'Cargando…' : 'Guardar borradores'}</button><button className="secondary" disabled={busy} onClick={() => { setFiles([]); setBatchID(''); setBatchKey(newRequestID()); }}>Iniciar otra carga</button></div></>}
  </section><section className="card"><div className="section-heading"><h2>Historial de cargas</h2><button className="secondary" onClick={()=>{batches.reload();setHistoryRevision(value=>value+1);}}>Actualizar historial</button></div><Notice text={batches.error}/>
    <div className="upload-history-layout"><section aria-label="Historial de cargas"><div className="table-wrap"><table aria-label="Historial de cargas"><thead><tr><th>Carga</th><th>Fecha de carga</th><th>Archivos</th></tr></thead><tbody>{batches.items.map(item=><tr key={item.id} className={batch===item.id?'selected-row':''}><td><button className="text-button upload-reference" title={item.id} aria-label={`Seleccionar carga ${item.id}`} aria-pressed={batch===item.id} onClick={()=>setBatch(item.id)}>{item.id}</button></td><td>{formatDate(item.created_at)}</td><td>{item.file_count}</td></tr>)}</tbody></table></div>{batches.busy&&<p role="status">Consultando historial…</p>}{!batches.busy&&!batches.error&&!batches.items.length&&<p className="empty-state">Todavía no has realizado cargas en esta biblioteca.</p>}<CursorControls page={batches} label="Cargas"/></section>
    <section aria-label="Archivos seleccionados"><h3>Archivos de la carga</h3>{batch?<UploadHistoryFiles batchID={batch} revision={historyRevision} open={id=>void open(id)}/>:<p className="empty-state">Selecciona una carga para consultar sus archivos.</p>}</section></div>
  </section>{selected && <DocumentViewer document={selected} canRemove={library.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={()=>{batches.reload();setHistoryRevision(value=>value+1);}} />}</>;
}
