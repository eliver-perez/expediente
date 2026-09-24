import { useState, type FormEvent } from 'react';
import { api, message } from './api';
import { Notice, usePage } from './components';
import { DocumentList, DocumentViewer } from './Documents';
import type { Document, Library, SearchInput, SearchResult } from './libraryTypes';

export function Search() {
  const libraries = usePage<Library>('/libraries');
  const notices = usePage<{ id: string; document_id: string; kind: string }>('/notifications');
  const [result, setResult] = useState<SearchResult | null>(null); const [request, setRequest] = useState<SearchInput | null>(null);
  const [selected, setSelected] = useState<{ document: Document; page: number } | null>(null);
  const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  async function search(input: SearchInput, append = false) {
    setBusy(true); setError('');
    try { const next = await api<SearchResult>('/search', { method: 'POST', body: input }); setResult(previous => append && previous ? { ...next, items: [...previous.items, ...next.items] } : next); setRequest(input); }
    catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); const library = String(data.get('library') || '');
    void search({ query: String(data.get('query') || ''), search_type: String(data.get('type')), library_ids: library ? [library] : [], filters: { availability: String(data.get('availability') || '') }, limit: 50 });
  }
  return <><header className="page-heading"><p className="eyebrow">CONSULTA DOCUMENTAL</p><h1>Buscar documentos</h1><p>Busca por nombre o por el texto de las páginas en tus bibliotecas.</p></header><Notice text={error || libraries.error} />
    {notices.items.length > 0 && <section className="card change-notices"><h2>Cambios en archivos vinculados</h2><p>{notices.items.length} cambios pendientes de revisar. El historial documental se conserva.</p>{notices.items.map((notice, index) => <div className="notice-row" key={notice.id}><button className="text-button" onClick={() => void api<Document>(`/documents/${notice.document_id}`).then(document => setSelected({ document, page: 1 })).catch(error => setError(message(error)))}>Ver documento modificado {index + 1}</button><button className="secondary compact" onClick={() => void api(`/notifications/${notice.id}/acknowledge`, { method: 'POST' }).then(notices.reload).catch(error => setError(message(error)))}>Marcar como leído</button></div>)}</section>}
    <section className="card"><form onSubmit={submit}><div className="search-line"><label className="search-query">Texto a buscar<input name="query" type="search" maxLength={2000} placeholder={'Nombre, código o "frase exacta"'} autoFocus /></label><button disabled={busy}>{busy ? 'Buscando…' : 'Buscar documentos'}</button></div><div className="filters search-filters"><label>Biblioteca<select name="library"><option value="">Todas mis bibliotecas</option>{libraries.items.filter(library => library.permissions.includes('search.execute')).map(library => <option key={library.id} value={library.id}>{library.name}</option>)}</select></label><label>Buscar en<select name="type"><option value="general">Nombre y contenido</option><option value="name">Nombre de archivo</option><option value="ocr">Texto nativo y OCR</option></select></label><label>Disponibilidad<select name="availability"><option value="">Todas</option><option value="available">Disponible</option><option value="missing">No disponible</option><option value="unknown">Raíz sin acceso</option></select></label></div></form></section>
    {result ? <section className="card"><div className="section-heading"><h2>{result.result_count} {result.result_count === 1 ? 'documento encontrado' : 'documentos encontrados'}</h2><span className="muted">{result.consulted_library_ids.length} {result.consulted_library_ids.length === 1 ? 'biblioteca consultada' : 'bibliotecas consultadas'}</span></div><DocumentList documents={result.items} open={(document, page = 1) => setSelected({ document, page })} />{result.result_count === 0 && <p className="empty-state">No hay coincidencias con estos filtros.</p>}{result.next_cursor && request && <div className="page-end"><button disabled={busy} className="secondary" onClick={() => void search({ ...request, cursor: result.next_cursor }, true)}>Mostrar más resultados</button></div>}</section> : <section className="search-empty"><span className="folder-symbol" aria-hidden="true">⌕</span><h2>El contenido también cuenta</h2><p>Encuentra una frase dentro de un PDF y abre directamente la página encontrada.</p></section>}
    {selected && <DocumentViewer document={selected.document} initialPage={selected.page} canRemove={!!libraries.items.find(library => library.id === selected.document.library_id)?.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={() => { if (request) void search({ ...request, cursor: undefined }); }} />}
  </>;
}
