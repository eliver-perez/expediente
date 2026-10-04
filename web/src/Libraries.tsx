import { FileSettings } from './FileSettings';
import { newRequestID } from './requestID';
import { ExplorerEntries } from './ExplorerEntries';
import { useState, type FormEvent } from 'react';
import { api, formatDate, message, type User } from './api';
import { Notice, PageEnd, usePage } from './components';
import { DocumentViewer } from './Documents';
import { Roots, LibraryMembers } from './LibrarySettings';
import { LibraryJobs } from './Processing';
import { SearchSection } from './SearchResults';
import { Duplicates, DuplicateSummary } from './Duplicates';
import { AuditTable } from './AuditTable';
import { ManagedSettings } from './ManagedSettings';
import { Catalogs } from './Catalogs';
import { Cases } from './Cases';
import { Reviews } from './Workflow';
import { Uploads } from './Uploads';
import { modeLabel } from './organizationTypes';
import type { Library, Document, Root, FolderView } from './libraryTypes';

export function Libraries({ user }: { user: User }) {
  const libraries = usePage<Library>('/libraries');
  const [selected, setSelected] = useState(''); const [creating, setCreating] = useState(false);
  const [requestKey, setRequestKey] = useState(() => newRequestID()); const [error, setError] = useState(''); const [busy, setBusy] = useState(false);
  const current = libraries.items.find(library => library.id === selected);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError(''); const data = new FormData(event.currentTarget);
    try { const result = await api<{ id: string }>('/libraries', { method: 'POST', idempotencyKey: requestKey, body: { name: data.get('name'), mode: data.get('mode'), manager_user_id: user.id } }); setSelected(result.id); setCreating(false); libraries.reload(); setRequestKey(newRequestID()); }
    catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  if (current) return <LibraryWorkspace library={current} user={user} back={() => setSelected('')} refresh={libraries.reload} />;
  return <><header className="page-heading row"><div><p className="eyebrow">ESPACIO DOCUMENTAL</p><h1>Bibliotecas</h1><p>Organiza las carpetas que necesitas consultar en un solo lugar.</p></div>{user.permissions.includes('libraries.create') && <button onClick={() => { setCreating(!creating); setError(''); setRequestKey(newRequestID()); }}>Nueva biblioteca</button>}</header>
    <Notice text={error || libraries.error} />
    {creating && <section className="card"><h2>Crear biblioteca</h2><form className="form-grid" onSubmit={create}><label>Nombre de la biblioteca<input name="name" required maxLength={160} autoFocus /></label><label>Modalidad de biblioteca<select name="mode" defaultValue="linked"><option value="linked">Vinculada: carpetas existentes</option><option value="managed">Administrada: cargas privadas</option><option value="hybrid">Híbrida: carpetas y cargas</option></select></label><label className="check"><input type="checkbox" required />Asignarme como gestor de esta biblioteca</label><div className="actions"><button disabled={busy}>Crear biblioteca</button><button type="button" className="secondary" onClick={() => setCreating(false)}>Cancelar</button></div></form></section>}
    <div className="library-grid">{libraries.items.map(library => <button className="library-card" key={library.id} onClick={() => setSelected(library.id)}><span className="folder-symbol" aria-hidden="true">▰</span><span className="badge">{modeLabel[library.mode]}</span><h2>{library.name}</h2><p>{library.permissions.includes('storage.manage_roots') ? 'Gestionar carpetas y consultar documentos' : library.permissions.includes('documents.read') ? 'Explorar documentos' : 'Administrar permisos y auditoría'}</p><span className="library-enter">Abrir biblioteca →</span></button>)}</div>
    {!libraries.busy && libraries.items.length === 0 && <section className="card empty-state"><h2>Tu espacio está listo</h2><p>{user.permissions.includes('libraries.create') ? 'Crea una biblioteca y añade una carpeta para comenzar.' : 'Las bibliotecas aparecerán cuando un gestor te conceda acceso.'}</p></section>}
  </>;
}
function LibraryWorkspace({ library, user, back, refresh }: { library: Library; user: User; back: () => void; refresh: () => void }) {
  const can = (permission: string) => library.permissions.includes(permission);
  const tabs = [{ id: 'explorer', label: 'Documentos', enabled: can('documents.read') }, { id: 'duplicates', label: 'Duplicados', enabled: can('documents.read') }, { id: 'uploads', label: 'Cargas', enabled: library.mode !== 'linked' && can('documents.upload') }, { id: 'reviews', label: 'Revisiones', enabled: can('documents.review') }, { id: 'cases', label: 'Expedientes', enabled: library.mode !== 'linked' && library.settings.cases_enabled && can('documents.read') }, { id: 'catalogs', label: 'Catálogos', enabled: library.mode !== 'linked' && can('documents.read') }, { id: 'roots', label: 'Carpetas', enabled: can('storage.manage_roots') }, { id: 'jobs', label: 'Procesamiento', enabled: can('indexing.run') }, { id: 'members', label: 'Personas', enabled: can('permissions.manage_library') || user.permissions.includes('permissions.manage_global') }, { id: 'settings', label: 'Configuración', enabled: can('libraries.configure') }, { id: 'events', label: 'Auditoría', enabled: can('audit.read_library') }].filter(tab => tab.enabled);
  const [tab, setTab] = useState(tabs[0]?.id || ''); const [error, setError] = useState(''); const [success, setSuccess] = useState('');
  return <><button className="text-button back-link" onClick={back}>← Todas las bibliotecas</button><header className="page-heading row"><div><p className="eyebrow">BIBLIOTECA {modeLabel[library.mode].toUpperCase()}</p><h1>{library.name}</h1><p>Documentos originales, texto extraído y trazabilidad.</p></div>{can('indexing.run') && <button className="secondary" onClick={async () => { try { await api(`/libraries/${library.id}/verify`, { method: 'POST', body: {} }); setSuccess('Verificación en cola. Consulta su avance en Procesamiento.'); setError(''); } catch (error) { setError(message(error)); } }}>Verificar biblioteca ahora</button>}</header><Notice text={error} /><Notice text={success} kind="success" />
    <div className="tabs" role="tablist" aria-label="Biblioteca">{tabs.map(item => <button key={item.id} role="tab" aria-selected={tab === item.id} className={tab === item.id ? 'active' : ''} onClick={() => { setTab(item.id); setError(''); setSuccess(''); }}>{item.label}</button>)}</div>
    {tab === 'explorer' && can('documents.read') && <Explorer library={library} openDuplicates={() => setTab('duplicates')} />}
    {tab === 'duplicates' && can('documents.read') && <Duplicates library={library} />}
    {tab === 'roots' && can('storage.manage_roots') && <Roots library={library} refresh={refresh} />}
    {tab === 'jobs' && can('indexing.run') && <LibraryJobs library={library} />}
    {tab === 'members' && <LibraryMembers library={library} />}
    {tab === 'settings' && can('libraries.configure') && <><ManagedSettings library={library} refresh={refresh} /><FileSettings libraryID={library.id} /></>}
    {tab === 'uploads' && can('documents.upload') && <Uploads library={library} />}
    {tab === 'cases' && library.settings.cases_enabled && <Cases library={library} />}
    {tab === 'reviews' && can('documents.review') && <Reviews library={library} />}
    {tab === 'catalogs' && <Catalogs library={library} />}
    {tab === 'events' && can('audit.read_library') && <LibraryAudit library={library} />}
  </>;
}
function Explorer({ library, openDuplicates }: { library: Library; openDuplicates: () => void }) {
  const roots = usePage<Root>(`/libraries/${library.id}/roots`); const views = usePage<FolderView>(`/libraries/${library.id}/folder-views`);
  const [root, setRoot] = useState(''); const [view, setView] = useState(''); const [prefix, setPrefix] = useState(''); const [availability, setAvailability] = useState('');
  const [navigationMode,setNavigationMode]=useState(()=>sessionStorage.getItem('aibid-explorer-view')||'list');const [browserRevision,setBrowserRevision]=useState(0);const [query,setQuery]=useState('');const [draftQuery,setDraftQuery]=useState('');const [searchType,setSearchType]=useState('general');const [scope,setScope]=useState('library');
  const basePrefix=views.items.find(item=>item.id===view)?.relative_prefix||'';
  const parameters = new URLSearchParams({ root_id: root, view_id: view, prefix, availability, direct_children:'true' });
  const [selected, setSelected] = useState<{ document: Document; page?: number } | null>(null);
  return <><section className="card"><div className="section-heading"><h2>Explorador</h2><button className="secondary" onClick={() => { setBrowserRevision(value=>value+1); roots.reload(); views.reload(); }}>Actualizar documentos</button></div><Notice text={roots.error || views.error} />
    <div className="filters explorer-filters"><label>Carpeta raíz<select value={root} onChange={event => { setRoot(event.target.value); setView(''); setPrefix(''); }}><option value="">Todas las raíces</option>{roots.items.filter(root => root.status !== 'superseded').map((root, index) => <option key={root.id} value={root.id}>{root.server_path || `Raíz ${index + 1}`}</option>)}</select></label><label>Vista lógica<select value={view} onChange={event => { setView(event.target.value); setPrefix(views.items.find(view=>view.id===event.target.value)?.relative_prefix||''); setRoot(views.items.find(view => view.id === event.target.value)?.root_id || ''); }}><option value="">Sin vista</option>{views.items.map(view => <option key={view.id} value={view.id}>{view.name}</option>)}</select></label><label>Disponibilidad<select value={availability} onChange={event => setAvailability(event.target.value)}><option value="">Todas</option><option value="available">Disponible</option><option value="missing">No disponible</option><option value="unknown">Raíz sin acceso</option><option value="staged">Temporal privado</option></select></label>{library.permissions.includes('search.execute')&&<><label>Buscar en<select value={searchType} onChange={event=>setSearchType(event.target.value)}><option value="general">Nombre, expediente y contenido</option><option value="name">Nombre de archivo</option><option value="identifier">Identificador de expediente</option><option value="ocr">Texto nativo y OCR</option></select></label><label>Ámbito<select value={scope} onChange={event=>setScope(event.target.value)}><option value="library">Toda la biblioteca con los filtros actuales</option><option value="folder">Carpeta seleccionada y subcarpetas</option></select></label></>}</div>
    {library.permissions.includes('search.execute')&&<form className="library-search" onSubmit={event=>{event.preventDefault();setQuery(String(new FormData(event.currentTarget).get('query')||'').trim());}}><div className="search-line"><label className="search-query">Buscar en esta biblioteca<input name="query" type="search" value={draftQuery} maxLength={2000} placeholder="Nombre, identificador o contenido" onChange={event=>{setDraftQuery(event.target.value);if(!event.target.value)setQuery('');}}/></label><button>Buscar en biblioteca</button></div>{query&&<button type="button" className="secondary" onClick={()=>{setDraftQuery('');setQuery('');}}>Cerrar búsqueda</button>}</form>}
    <div className="section-heading"><div className="breadcrumbs"><button className="text-button" onClick={()=>setPrefix(basePrefix)}>Inicio de carpeta</button>{prefix.split('/').filter(Boolean).map((component,index,parts)=><span key={index}> / <button className="text-button" disabled={!!basePrefix&&parts.slice(0,index+1).join('/').length<basePrefix.length} onClick={()=>setPrefix(parts.slice(0,index+1).join('/'))}>{component}</button></span>)}</div><div className="actions" role="group" aria-label="Vista de navegación"><button className="secondary compact" aria-pressed={navigationMode==='list'} onClick={()=>{setNavigationMode('list');sessionStorage.setItem('aibid-explorer-view','list');}}>Vista de lista</button><button className="secondary compact" aria-pressed={navigationMode==='files'} onClick={()=>{setNavigationMode('files');sessionStorage.setItem('aibid-explorer-view','files');}}>Vista de cuadrícula</button></div></div>
    {query ? <SearchSection key={JSON.stringify([query,searchType,scope,root,view,prefix,availability])} input={{query,search_type:searchType,library_ids:[library.id],filters:{root_id:root,view_id:view,prefix:scope==='folder'?prefix:'',availability}}} name={library.name} open={(document,page)=>setSelected({document,page})}/> : <>
    {<button className="text-button" disabled={!prefix||prefix===basePrefix} onClick={()=>setPrefix(prefix.split('/').slice(0,-1).join('/'))}>← Carpeta superior</button>}
    <ExplorerEntries key={browserRevision} libraryID={library.id} parameters={parameters.toString()} mode={navigationMode} navigate={setPrefix} open={document=>setSelected({document})}/></>}


  </section><DuplicateSummary library={library} open={openDuplicates} />
    {selected && <DocumentViewer document={selected.document} initialPage={selected.page} canRemove={library.permissions.includes('documents.remove_index')} close={() => setSelected(null)} changed={()=>setBrowserRevision(value=>value+1)} />}
  </>;
}
function LibraryAudit({ library }: { library: Library }) {
  const queries = usePage<{ id: string; occurred_at: string; exact_query_text: string; search_type: string; result_count: number; library_ids: string[]; filters: Record<string, unknown> }>('/search-events');
  return <><AuditTable libraryID={library.id} />
    {library.permissions.includes('audit.search_details') && <section className="card"><h2>Consultas exactas</h2><p className="muted">Solo se muestran búsquedas cuyos ámbitos completos puedes auditar.</p><Notice text={queries.error} />{queries.items.filter(query => query.library_ids.includes(library.id)).map(query => <details key={query.id}><summary>{formatDate(query.occurred_at)} · {query.result_count} resultados</summary><pre className="exact-query">{query.exact_query_text}</pre><p>Tipo: {query.search_type}</p><pre className="audit-json">{JSON.stringify(query.filters, null, 2)}</pre></details>)}<PageEnd {...queries} empty={queries.items.length === 0} /></section>}
  </>;
}
