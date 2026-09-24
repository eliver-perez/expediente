import { useState, type FormEvent } from 'react';
import { api, formatDate, message, type User } from './api';
import { Notice, PageEnd, usePage } from './components';
import { DocumentList, DocumentViewer } from './Documents';
import { Roots, LibraryJobs, LibraryMembers } from './LibrarySettings';
import type { Library, Document, Root, FolderView } from './libraryTypes';

export function Libraries({ user }: { user: User }) {
  const libraries = usePage<Library>('/libraries');
  const [selected, setSelected] = useState(''); const [creating, setCreating] = useState(false);
  const [requestKey, setRequestKey] = useState(() => crypto.randomUUID()); const [error, setError] = useState(''); const [busy, setBusy] = useState(false);
  const current = libraries.items.find(library => library.id === selected);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); const data = new FormData(event.currentTarget);
    try { const result = await api<{ id: string }>('/libraries', { method: 'POST', idempotencyKey: requestKey, body: { name: data.get('name'), mode: 'linked', manager_user_id: user.id } }); setSelected(result.id); setCreating(false); libraries.reload(); setRequestKey(crypto.randomUUID()); }
    catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  if (current) return <LibraryWorkspace library={current} user={user} back={() => setSelected('')} refresh={libraries.reload} />;
  return <><header className="page-heading row"><div><p className="eyebrow">ESPACIO DOCUMENTAL</p><h1>Bibliotecas</h1><p>Organiza las carpetas que necesitas consultar en un solo lugar.</p></div>{user.permissions.includes('libraries.create') && <button onClick={() => { setCreating(!creating); setError(''); setRequestKey(crypto.randomUUID()); }}>Nueva biblioteca</button>}</header>
    <Notice text={error || libraries.error} />
    {creating && <section className="card"><h2>Crear biblioteca vinculada</h2><form className="form-grid" onSubmit={create}><label>Nombre de la biblioteca<input name="name" required maxLength={160} autoFocus /></label><p className="muted">Los documentos se consultan desde sus carpetas originales.</p><label className="check"><input type="checkbox" required />Asignarme como gestor de esta biblioteca</label><div className="actions"><button disabled={busy}>Crear biblioteca</button><button type="button" className="secondary" onClick={() => setCreating(false)}>Cancelar</button></div></form></section>}
    <div className="library-grid">{libraries.items.map(library => <button className="library-card" key={library.id} onClick={() => setSelected(library.id)}><span className="folder-symbol" aria-hidden="true">▰</span><span className="badge">Vinculada</span><h2>{library.name}</h2><p>{library.permissions.includes('storage.manage_roots') ? 'Gestionar carpetas y consultar documentos' : library.permissions.includes('documents.read') ? 'Explorar documentos' : 'Administrar permisos y auditoría'}</p><span className="library-enter">Abrir biblioteca →</span></button>)}</div>
    {!libraries.busy && libraries.items.length === 0 && <section className="card empty-state"><h2>Tu espacio está listo</h2><p>{user.permissions.includes('libraries.create') ? 'Crea una biblioteca y añade una carpeta para comenzar.' : 'Las bibliotecas aparecerán cuando un gestor te conceda acceso.'}</p></section>}
  </>;
}
function LibraryWorkspace({ library, user, back, refresh }: { library: Library; user: User; back: () => void; refresh: () => void }) {
  const can = (permission: string) => library.permissions.includes(permission);
  const tabs = [{ id: 'explorer', label: 'Documentos', enabled: can('documents.read') }, { id: 'roots', label: 'Carpetas', enabled: can('storage.manage_roots') }, { id: 'jobs', label: 'Procesamiento', enabled: can('indexing.run') }, { id: 'members', label: 'Personas', enabled: can('permissions.manage_library') || user.permissions.includes('permissions.manage_global') }, { id: 'settings', label: 'Configuración', enabled: can('libraries.configure') }, { id: 'events', label: 'Auditoría', enabled: can('audit.read_library') }].filter(tab => tab.enabled);
  const [tab, setTab] = useState(tabs[0]?.id || ''); const [error, setError] = useState(''); const [success, setSuccess] = useState('');
  return <><button className="text-button back-link" onClick={back}>← Todas las bibliotecas</button><header className="page-heading row"><div><p className="eyebrow">BIBLIOTECA VINCULADA</p><h1>{library.name}</h1><p>Documentos originales, texto extraído y trazabilidad.</p></div>{can('indexing.run') && <button className="secondary" onClick={async () => { try { await api(`/libraries/${library.id}/verify`, { method: 'POST', body: {} }); setSuccess('Verificación en cola. Consulta su avance en Procesamiento.'); setError(''); } catch (error) { setError(message(error)); } }}>Verificar biblioteca ahora</button>}</header><Notice text={error} /><Notice text={success} kind="success" />
    <div className="tabs" role="tablist" aria-label="Biblioteca">{tabs.map(item => <button key={item.id} role="tab" aria-selected={tab === item.id} className={tab === item.id ? 'active' : ''} onClick={() => { setTab(item.id); setError(''); setSuccess(''); }}>{item.label}</button>)}</div>
    {tab === 'explorer' && can('documents.read') && <Explorer library={library} />}
    {tab === 'roots' && can('storage.manage_roots') && <Roots library={library} refresh={refresh} />}
    {tab === 'jobs' && can('indexing.run') && <LibraryJobs library={library} />}
    {tab === 'members' && <LibraryMembers library={library} />}
    {tab === 'settings' && can('libraries.configure') && <section className="card"><h2>Configuración de biblioteca</h2><form className="form-grid" onSubmit={async event => { event.preventDefault(); const data = new FormData(event.currentTarget); try { await api(`/libraries/${library.id}`, { method: 'PATCH', revision: library.revision, body: { name: data.get('name'), ocr_languages: data.get('languages') } }); refresh(); setSuccess('Configuración guardada.'); } catch (error) { setError(message(error)); } }}><label>Nombre de la biblioteca<input name="name" defaultValue={library.name} maxLength={160} required /></label><label>Idioma del OCR<select name="languages" defaultValue={library.ocr_languages}><option value="spa">Español</option><option value="eng">Inglés</option><option value="spa+eng">Español e inglés</option></select></label><button>Guardar configuración</button></form></section>}
    {tab === 'events' && can('audit.read_library') && <LibraryAudit library={library} />}
  </>;
}
function Explorer({ library }: { library: Library }) {
  const roots = usePage<Root>(`/libraries/${library.id}/roots`); const views = usePage<FolderView>(`/libraries/${library.id}/folder-views`);
  const [root, setRoot] = useState(''); const [view, setView] = useState(''); const [prefix, setPrefix] = useState(''); const [availability, setAvailability] = useState('');
  const parameters = new URLSearchParams({ root_id: root, view_id: view, prefix, availability });
  const documents = usePage<Document>(`/libraries/${library.id}/explorer?${parameters}`);
  const folders = usePage<{ name: string; relative_prefix: string }>(`/libraries/${library.id}/folders?${parameters}`);
  const duplicates = usePage<{ sha256: string; count: number }>(`/libraries/${library.id}/duplicates`);
  const [selected, setSelected] = useState<{ document: Document; page: number } | null>(null);
  return <><section className="card"><div className="section-heading"><h2>Explorador</h2><button className="secondary" onClick={() => { documents.reload(); folders.reload(); roots.reload(); views.reload(); duplicates.reload(); }}>Actualizar documentos</button></div><Notice text={documents.error || roots.error || views.error || folders.error} />
    <div className="filters"><label>Carpeta raíz<select value={root} onChange={event => { setRoot(event.target.value); setView(''); setPrefix(''); }}><option value="">Todas las raíces</option>{roots.items.filter(root => root.status !== 'superseded').map((root, index) => <option key={root.id} value={root.id}>{root.server_path || `Raíz ${index + 1}`}</option>)}</select></label><label>Vista lógica<select value={view} onChange={event => { setView(event.target.value); setPrefix(''); setRoot(views.items.find(view => view.id === event.target.value)?.root_id || ''); }}><option value="">Sin vista</option>{views.items.map(view => <option key={view.id} value={view.id}>{view.name}</option>)}</select></label><label>Disponibilidad<select value={availability} onChange={event => setAvailability(event.target.value)}><option value="">Todas</option><option value="available">Disponible</option><option value="missing">No disponible</option><option value="unknown">Raíz sin acceso</option></select></label></div>
    <div className="breadcrumbs"><button className="text-button" onClick={() => { setPrefix(''); setView(''); }}>Inicio de carpeta</button>{prefix.split('/').filter(Boolean).map((component, index, parts) => <span key={index}> / <button className="text-button" onClick={() => setPrefix(parts.slice(0, index + 1).join('/'))}>{component}</button></span>)}</div>
    {folders.items.length > 0 && <div className="folder-grid">{folders.items.map(folder => <button className="folder-button" key={folder.relative_prefix} onClick={() => { setPrefix(folder.relative_prefix); setView(''); }}><span aria-hidden="true">▰</span> {folder.name}</button>)}{folders.cursor && <button className="secondary" onClick={() => void folders.more()}>Más carpetas</button>}</div>}
    <DocumentList documents={documents.items} open={(document, page = 1) => setSelected({ document, page })} /><PageEnd {...documents} empty={documents.items.length === 0} />
  </section>{duplicates.items.length > 0 && <section className="card"><h2>Contenido duplicado</h2><p className="muted">Los archivos conservan sus registros independientes.</p>{duplicates.items.map(group => <details key={group.sha256}><summary>{group.count} archivos con el mismo contenido</summary><p className="path-text">SHA-256: {group.sha256}</p></details>)}</section>}
    {selected && <DocumentViewer document={selected.document} initialPage={selected.page} canRemove={library.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={documents.reload} />}
  </>;
}
function LibraryAudit({ library }: { library: Library }) {
  const events = usePage<{ id: string; event_type: string; occurred_at: string; details_json: string }>(`/libraries/${library.id}/audit-events`);
  const queries = usePage<{ id: string; occurred_at: string; exact_query_text: string; search_type: string; result_count: number; library_ids: string[]; filters: Record<string, unknown> }>('/search-events');
  return <><section className="card"><div className="section-heading"><h2>Eventos de biblioteca</h2><button className="secondary" onClick={events.reload}>Actualizar eventos</button></div><Notice text={events.error} />{events.items.map(event => <details key={event.id}><summary>{formatDate(event.occurred_at)} · {event.event_type}</summary><pre className="audit-json">{JSON.stringify(JSON.parse(event.details_json), null, 2)}</pre></details>)}<PageEnd {...events} empty={events.items.length === 0} /></section>
    {library.permissions.includes('audit.search_details') && <section className="card"><h2>Consultas exactas</h2><p className="muted">Solo se muestran búsquedas cuyos ámbitos completos puedes auditar.</p><Notice text={queries.error} />{queries.items.filter(query => query.library_ids.includes(library.id)).map(query => <details key={query.id}><summary>{formatDate(query.occurred_at)} · {query.result_count} resultados</summary><pre className="exact-query">{query.exact_query_text}</pre><p>Tipo: {query.search_type}</p><pre className="audit-json">{JSON.stringify(query.filters, null, 2)}</pre></details>)}<PageEnd {...queries} empty={queries.items.length === 0} /></section>}
  </>;
}
