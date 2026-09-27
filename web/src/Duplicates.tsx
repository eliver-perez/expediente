import { useState } from 'react';
import { Notice, PageEnd, usePage, usePolling } from './components';
import { DocumentList, DocumentViewer } from './Documents';
import type { Document, Library } from './libraryTypes';

interface Group { sha256: string; count: number }
interface Summary { groups: number; files: number }
export function DuplicateSummary({ library, open }: { library: Library; open: () => void }) {
  const summary = usePolling<Summary>(`/libraries/${library.id}/duplicates`, 10000);
  return <section className="card"><div className="section-heading"><div><h2>Contenido duplicado</h2><p>{summary.data ? `${summary.data.groups} grupos · ${summary.data.files} archivos involucrados` : 'Consultando duplicados…'}</p><p className="muted">Son archivos independientes cuyo contenido es idéntico. Puedes abrirlos y comparar sus ubicaciones.</p><Notice text={summary.error} /></div><button className="secondary" onClick={open}>Revisar duplicados</button></div></section>;
}
export function Duplicates({ library }: { library: Library }) {
  const groups = usePage<Group>(`/libraries/${library.id}/duplicates`); const [selected, setSelected] = useState<Group | null>(null);
  return <><section className="card"><h2>Revisión de duplicados</h2><p>Grupos de {library.name} con copias idénticas en las bibliotecas que puedes consultar. Cada archivo conserva su registro, ubicación e historial. Esta vista no elimina archivos.</p><Notice text={groups.error} />
    <div className="duplicate-groups">{groups.items.map((group, index) => <button className={selected?.sha256 === group.sha256 ? '' : 'secondary'} key={group.sha256} onClick={() => setSelected(group)}>Grupo {index + 1} · {group.count} archivos</button>)}</div><PageEnd {...groups} empty={!groups.items.length} /></section>
    {selected && <DuplicateFiles key={selected.sha256} library={library} group={selected} />}</>;
}
function DuplicateFiles({ library, group }: { library: Library; group: Group }) {
  const documents = usePage<Document>(`/libraries/${library.id}/duplicates/${group.sha256}`); const [selected, setSelected] = useState<Document | null>(null);
  return <section className="card"><h2>{group.count} archivos con el mismo contenido</h2><p>Biblioteca: {library.name}</p><Notice text={documents.error} /><DocumentList documents={documents.items} open={document => setSelected(document)} />
    <ul className="duplicate-paths">{documents.items.map(document => <li key={document.id}><strong>{document.original_filename}</strong><span> · {document.library_name || library.name}</span><p className="path-text">{document.original_path || document.relative_path || 'Ubicación privada'}</p></li>)}</ul>
    <PageEnd {...documents} empty={!documents.items.length} /><details><summary>Identificador técnico del contenido</summary><p className="path-text">SHA-256: {group.sha256}</p></details>
    {selected && <DocumentViewer document={selected} close={() => setSelected(null)} changed={documents.reload} canRemove={false} />}</section>;
}
