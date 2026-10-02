import { Accordion } from './Accordion';
import { useEffect, useState } from 'react';
import { Notice, usePolling } from './components';
import { CursorControls, useCursorPage } from './CursorPage';
import { DocumentViewer } from './Documents';
import { formatDate } from './api';
import { availabilityLabel, type Document, type Library } from './libraryTypes';
import { formatSize } from './ExplorerEntries';

interface Group { sha256: string; count: number; size_bytes: number; library_count: number }
interface Summary { groups: number; files: number }
export function DuplicateSummary({ library, open }: { library: Library; open: () => void }) {
  const summary = usePolling<Summary>(`/libraries/${library.id}/duplicates`, 10000);
  return <section className="card"><div className="section-heading"><div><h2>Contenido duplicado</h2><p>{summary.data ? `${summary.data.groups} grupos · ${summary.data.files} archivos involucrados` : 'Consultando duplicados…'}</p><p className="muted">Son archivos independientes cuyo contenido es idéntico. Puedes abrirlos y comparar sus ubicaciones.</p><Notice text={summary.error} /></div><button className="secondary" onClick={open}>Revisar duplicados</button></div></section>;
}
export function Duplicates({ library }: { library: Library }) {
  const groups = useCursorPage<Group>(`/libraries/${library.id}/duplicates`);
  const [selected, setSelected] = useState<Group | null>(null);
  useEffect(() => { if (!groups.busy) setSelected(previous => groups.items.find(item => item.sha256 === previous?.sha256) || groups.items[0] || null); }, [groups.items, groups.busy]);
  return <section className="card"><h2>Revisión de duplicados</h2><p>Contenido idéntico en {library.name} y las bibliotecas que puedes consultar. Cada archivo conserva su ubicación e historial; esta vista no elimina archivos.</p><div className="duplicates-layout">
    <section aria-label="Grupos de duplicados"><h3>Grupos</h3><Notice text={groups.error} /><div className="table-wrap"><table className="duplicate-groups"><thead><tr><th>Huella digital</th><th>Tamaño</th><th>Archivos</th></tr></thead><tbody>{groups.items.map(group => <tr key={group.sha256} className={selected?.sha256 === group.sha256 ? 'selected-row' : ''}><td><button className="text-button" aria-pressed={selected?.sha256 === group.sha256} onClick={() => setSelected(group)}>{group.sha256.slice(0, 12)}…</button></td><td>{formatSize(group.size_bytes)}</td><td>{group.count}<small className="group-library-count">{group.library_count} {group.library_count === 1 ? 'biblioteca' : 'bibliotecas'}</small></td></tr>)}</tbody></table></div>
      {groups.busy ? <p role="status">Consultando grupos…</p> : !groups.items.length && <p>No hay grupos de duplicados.</p>}<CursorControls page={groups} label="Grupos" />
    </section>
    {selected ? <DuplicateFiles key={selected.sha256} library={library} group={selected} /> : <p>Selecciona un grupo para consultar sus archivos.</p>}
  </div></section>;
}
function DuplicateFiles({ library, group }: { library: Library; group: Group }) {
  const documents = useCursorPage<Document>(`/libraries/${library.id}/duplicates/${group.sha256}`);
  const [selected, setSelected] = useState<Document | null>(null);
  return <section aria-label="Archivos del grupo"><h3>{group.count} archivos con el mismo contenido</h3><Accordion title="Ver huella completa del grupo"><p className="path-text">SHA-256: {group.sha256}</p></Accordion><Notice text={documents.error} />
    <div className="duplicate-files">{documents.items.map(document => <article className="duplicate-file" key={document.id}><button className="text-button document-title" onClick={() => setSelected(document)}>{document.title || document.original_filename}</button><p>{document.library_name || library.name}</p><p className="path-text">{document.original_path || document.relative_path || 'Ubicación privada'}</p><small>{formatSize(document.size_bytes)} · {formatDate(document.modified_at || null)} · {availabilityLabel[document.availability]}</small><button className="secondary compact" onClick={() => setSelected(document)}>Ver ficha</button></article>)}</div>
    {documents.busy && <p role="status">Consultando archivos…</p>}<CursorControls page={documents} label="Archivos" />
    {selected && <DocumentViewer document={selected} close={() => setSelected(null)} changed={documents.reload} canRemove={false} />}
  </section>;
}
