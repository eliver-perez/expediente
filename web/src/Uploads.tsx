import { useState } from 'react';
import { api, APIError, formatDate, message, uploadPDF } from './api';
import { Notice, PageEnd, usePage } from './components';
import { DocumentViewer } from './Documents';
import type { Document, Library } from './libraryTypes';
import type { UploadBatch, UploadItem } from './organizationTypes';

interface PendingFile { file: File; key: string; percent: number; status: 'waiting' | 'uploading' | 'staged' | 'failed'; error: string; item?: UploadItem; replaceKey?: boolean }
export function Uploads({ library }: { library: Library }) {
  const [files, setFiles] = useState<PendingFile[]>([]); const [batchKey, setBatchKey] = useState(() => crypto.randomUUID()); const [batchID, setBatchID] = useState('');
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [selected, setSelected] = useState<Document | null>(null); const [batch, setBatch] = useState<UploadBatch | null>(null);
  const batches = usePage<UploadBatch>(`/libraries/${library.id}/upload-batches`);
  function addFiles(incoming: File[]) { if (busy) return; if (files.length + incoming.length > 100) { setError('Cada lote admite hasta 100 archivos. Inicia otro lote.'); return; } if (incoming.some(file => !file.name.toLowerCase().endsWith('.pdf'))) { setError('Selecciona únicamente archivos PDF.'); return; } setFiles(previous => [...previous, ...incoming.map(file => ({ file, key: crypto.randomUUID(), percent: 0, status: 'waiting' as const, error: '' }))]); setError(''); }
  async function open(documentID: string) { try { setSelected(await api<Document>(`/documents/${documentID}`)); } catch (error) { setError(message(error)); } }
  async function send() {
    setBusy(true); setError('');
    try {
      const identifier = batchID || (await api<{ id: string }>(`/libraries/${library.id}/upload-batches`, { method: 'POST', body: { client_batch_id: batchKey } })).id; setBatchID(identifier);
      for (const pending of files.filter(file => file.status === 'waiting' || file.status === 'failed')) {
        const key = pending.replaceKey ? crypto.randomUUID() : pending.key;
        const update = (patch: Partial<PendingFile>) => setFiles(previous => previous.map(file => file.key === pending.key || file.key === key ? { ...file, ...patch, key } : file));
        update({ status: 'uploading', error: '', percent: 0 });
        try { const item = await uploadPDF<UploadItem>(identifier, key, pending.file, percent => update({ percent })); update({ status: 'staged', item, percent: 100 }); }
        catch (error) { update({ status: 'failed', error: message(error), replaceKey: error instanceof APIError && error.status !== 0 && error.status !== 429 }); if (error instanceof APIError && error.status === 401) break; }
      }
      batches.reload();
    } catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  return <><section className="card"><h2>Cargar documentos PDF</h2><p className="muted">Se guardan como borradores privados. Solo tú y los revisores autorizados pueden consultarlos antes de su incorporación definitiva.</p><Notice text={error} />
    <div className="upload-dropzone" onDragOver={event => event.preventDefault()} onDrop={event => { event.preventDefault(); addFiles(Array.from(event.dataTransfer.files)); }}><strong>Arrastra aquí uno o varios PDF</strong><span>o selecciónalos desde tu equipo</span><label>Archivos PDF<input type="file" multiple accept="application/pdf,.pdf" disabled={busy} onChange={event => { addFiles(Array.from(event.target.files || [])); event.target.value = ''; }} /></label></div>
    {files.length > 0 && <><div className="upload-list">{files.map(file => <article className="upload-row" key={file.key}><div><strong>{file.file.name}</strong><small>{(file.file.size / 1024).toFixed(0)} KiB · {file.status === 'waiting' ? 'Pendiente' : file.status === 'uploading' ? file.percent === 100 ? 'Validando PDF…' : `Enviando ${file.percent} %` : file.status === 'staged' ? 'Borrador guardado' : 'Carga incompleta'}</small>{file.status === 'uploading' && <progress value={file.percent} max={100} aria-label={`Carga de ${file.file.name}`} />}{file.error && <p role="alert" className="diagnostic">{file.error}</p>}</div>{file.item && <button className="secondary" onClick={() => void open(file.item!.document_id)}>Clasificar {file.file.name}</button>}{file.status === 'waiting' && <button className="text-button" disabled={busy} onClick={() => setFiles(files.filter(item => item.key !== file.key))}>Quitar {file.file.name}</button>}</article>)}</div><div className="actions"><button disabled={busy || !files.some(file => file.status === 'waiting' || file.status === 'failed')} onClick={() => void send()}>{busy ? 'Cargando…' : 'Guardar borradores'}</button><button className="secondary" disabled={busy} onClick={() => { setFiles([]); setBatchID(''); setBatchKey(crypto.randomUUID()); }}>Iniciar otro lote</button></div></>}
  </section><section className="card"><div className="section-heading"><h2>Mis lotes</h2><button className="secondary" onClick={batches.reload}>Actualizar lotes</button></div><Notice text={batches.error} />{batches.items.map(item => <button className="text-button batch-link" key={item.id} onClick={async () => { try { setBatch(await api<UploadBatch>(`/upload-batches/${item.id}`)); } catch (error) { setError(message(error)); } }}>Lote del {formatDate(item.created_at)}</button>)}<PageEnd {...batches} empty={batches.items.length === 0} />
    {batch && <div className="upload-list">{batch.items.map(item => <div className="upload-row" key={item.id}><div><strong>{item.original_filename}</strong><small>{({ receiving: 'Recibiendo', staged: 'Borrador privado', retained: 'Cancelado, conservado', failed: 'Carga incompleta', cleaned: 'Temporal retirado' } as Record<string, string>)[item.status]}</small>{item.error_code && <p className="diagnostic">{item.error_code === 'PROCESS_RESTARTED' ? 'La transferencia se interrumpió al reiniciar. Puedes volver a cargar el archivo.' : 'La transferencia no se completó. Puedes volver a cargar el archivo.'}</p>}</div>{item.document_id && <button className="secondary" onClick={() => void open(item.document_id)}>Ver {item.original_filename}</button>}</div>)}</div>}
  </section>{selected && <DocumentViewer document={selected} canRemove={library.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={batches.reload} />}</>;
}
