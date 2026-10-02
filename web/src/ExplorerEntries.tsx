import { CursorControls, useCursorPage } from './CursorPage';
import { Notice } from './components';
import { formatDate } from './api';
import { availabilityLabel, type Document } from './libraryTypes';

export function formatSize(bytes?: number | null) {
  if (bytes == null) return '—';
  return bytes < 1024 ? `${bytes} B` : bytes < 1024 ** 2 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1024 ** 2).toFixed(1)} MB`;
}
export function FolderIcon() {
  return <svg className="folder-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M3 6a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6Z" fill="currentColor" opacity=".16" /><path d="M3 9h18M3 6a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6Z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" /></svg>;
}
export function ExplorerEntries({ libraryID, parameters, mode, navigate, open }: { libraryID: string; parameters: string; mode: string; navigate: (prefix: string) => void; open: (document: Document) => void }) {
  const folders = useCursorPage<{ name: string; relative_prefix: string }>(`/libraries/${libraryID}/folders?${parameters}`);
  const documents = useCursorPage<Document>(`/libraries/${libraryID}/explorer?${parameters}`);
  return <><Notice text={folders.error || documents.error} />
    {mode === 'files' ? <div className="file-explorer-grid">
      {folders.items.map(folder => <button className="file-tile folder-tile" key={folder.relative_prefix} onClick={() => navigate(folder.relative_prefix)}><FolderIcon /><strong>{folder.name}</strong><small>Carpeta</small></button>)}
      {documents.items.map(document => <button className="file-tile" key={document.id} onClick={() => open(document)}><span className="pdf-symbol" aria-hidden="true">PDF</span><strong>{document.title || document.original_filename}</strong><small>{document.page_count || '—'} páginas · {formatSize(document.size_bytes)} · {availabilityLabel[document.availability]}</small></button>)}
    </div> : <div className="table-wrap"><table className="explorer-details"><thead><tr><th>Nombre</th><th>Incorporación</th><th>Páginas</th><th>Estado</th><th>Tamaño</th><th>Consulta</th></tr></thead><tbody>
      {folders.items.map(folder => <tr key={folder.relative_prefix}><td><button className="text-button explorer-name" onClick={() => navigate(folder.relative_prefix)}><FolderIcon />{folder.name}</button></td><td>—</td><td>—</td><td>Carpeta</td><td>—</td><td>—</td></tr>)}
      {documents.items.map(document => <tr key={document.id}><td><button className="text-button explorer-name" onClick={() => open(document)}><span className="pdf-symbol" aria-hidden="true">PDF</span>{document.title || document.original_filename}</button></td><td>{formatDate(document.created_at || null)}</td><td>{document.page_count || '—'}</td><td>{availabilityLabel[document.availability]}</td><td>{formatSize(document.size_bytes)}</td><td><button className="secondary compact" onClick={()=>open(document)}>Ver ficha</button></td></tr>)}
    </tbody></table></div>}
    {(folders.busy || documents.busy) && <p role="status">Consultando carpeta…</p>}
    {!folders.busy && !documents.busy && !folders.items.length && !documents.items.length && <p>No hay documentos o carpetas con estos filtros.</p>}
    {(folders.canBack || folders.canNext) && <CursorControls page={folders} label="Carpetas" />}
    {(documents.canBack || documents.canNext) && <CursorControls page={documents} label="Documentos" />}
  </>;
}
