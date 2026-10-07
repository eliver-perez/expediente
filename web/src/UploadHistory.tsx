import { useEffect } from 'react';
import { formatDate } from './api';
import { Notice } from './components';
import { CursorControls, useCursorPage } from './CursorPage';
import { formatSize } from './ExplorerEntries';
import { FileIcon } from './FileIcon';
import { indexReasonLabel, processingStateLabel } from './libraryTypes';
import type { UploadItem } from './organizationTypes';

export const uploadStateLabel: Record<string,string> = { receiving:'Recibiendo', staged:'Borrador privado', retained:'Cancelado, conservado', failed:'Carga incompleta', cleaned:'Temporal retirado' };
export function UploadHistoryFiles({ batchID, revision, open }: {batchID:string;revision:number;open:(id:string)=>void}) {
 const files=useCursorPage<UploadItem>(`/upload-batches/${batchID}/items?refresh=${revision}`);
 useEffect(()=>{
  if(files.busy||!files.items.some(item=>item.status==='receiving'||['pending','analysing','validating','extracting','ocr','waiting_ocr','indexing','metadata','retrying'].includes(item.processing_state)))return;
  const timer=setTimeout(files.reload,3000);return()=>clearTimeout(timer);
 },[files.busy,files.items,files.reload]);
 return <><Notice text={files.error}/>{files.busy&&!files.items.length&&<p role="status">Consultando archivos…</p>}
 <div className="table-wrap"><table aria-label="Archivos de la carga"><thead><tr><th>Nombre del archivo</th><th>Estado</th><th>Tipo</th><th>Tamaño</th><th>Acción</th></tr></thead><tbody>{files.items.map(item=><tr key={item.id}>
 <td><span className="explorer-name"><FileIcon format={item.format}/><span>{item.original_filename}</span></span><small>{formatDate(item.created_at)}</small></td>
 <td><span className={`badge ${item.processing_state==='completed'?'positive':''}`}>{processingStateLabel[item.processing_state]||indexReasonLabel[item.index_block_reason]||uploadStateLabel[item.status]||item.status}</span>{item.processing_state&&<small>{uploadStateLabel[item.status]||item.status}</small>}{item.error_code&&<p className="diagnostic">{item.error_code==='PROCESS_RESTARTED'?'La transferencia se interrumpió al reiniciar. Puedes volver a cargar el archivo.':'La transferencia no se completó. Puedes volver a cargar el archivo.'}</p>}</td>
 <td>{item.format?item.format.toUpperCase():'Sin detectar'}</td><td>{formatSize(item.size_bytes)}</td><td>{item.can_view?<button className="secondary compact" aria-label={`Ver detalles de ${item.original_filename}`} onClick={()=>open(item.document_id)}>Ver detalles</button>:<span className="muted">Sin ficha disponible</span>}</td></tr>)}</tbody></table></div>
 {!files.busy&&!files.error&&!files.items.length&&<p className="empty-state">Esta carga no contiene archivos.</p>}<CursorControls page={files} label="Archivos de la carga"/></>;
}
