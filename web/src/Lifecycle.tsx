import { useEffect, useRef, useState } from 'react';
import { api, message, type User } from './api';
import { Notice, PageEnd, usePage } from './components';
import { confirmAction, dialogs, intervention, toast } from './feedback';
import type { Library, Root } from './libraryTypes';

export interface Deletion { id: string; kind: string; name: string; library_name: string; state: string; counts: Record<string, number>; locations: string[]; pending_cleanup: number; cleanup_issues: number; cleanup_diagnostics?: string[] }
const changed = () => window.dispatchEvent(new Event('aidex-deletion-updated'));
function resultNotice(result: Deletion) {
  const text = result.state === 'complete' ? 'Eliminación completada. Los documentos definitivos permanecen intactos.' : result.state === 'pending' ? 'Eliminación pendiente. Puedes seguir el estado en Eliminaciones y limpieza.' : result.state === 'running' ? 'Eliminación en proceso. Puedes seguir el estado en Eliminaciones y limpieza.' : result.state === 'failed' ? 'La eliminación falló sin borrar registros; revisa el diagnóstico y reintenta la misma operación.' : `Los registros se eliminaron. La limpieza sigue pendiente (${result.pending_cleanup} archivos; ${result.cleanup_issues} incidencias). Consulta Eliminaciones y limpieza.`;
  toast(text, result.state === 'complete' ? 'success' : 'warning');
}
function deletionSummary(plan: Deletion) {
  const container = document.createElement('div'); container.className = 'deletion-summary';
  const paragraph = (text: string) => { const p = document.createElement('p'); p.textContent = text; container.append(p); };
  paragraph(plan.kind === 'library' ? 'Estás a punto de eliminar esta biblioteca de AIDEX. Esta acción es definitiva y no se puede deshacer. Se eliminarán el registro de la biblioteca y la información asociada que corresponda.' : 'Se eliminará definitivamente el registro de esta carpeta y sus referencias exclusivas. No podrás reanudar ese registro.');
  paragraph('Tus documentos definitivos NO se borrarán. Permanecerán en sus ubicaciones físicas, incluso si pertenecen a una biblioteca administrada. Los temporales y borradores pendientes de incorporación exclusivos podrán eliminarse durante la limpieza.');
  if (plan.kind === 'library') paragraph('Si solamente deseas ocultar la biblioteca temporalmente, utiliza la opción Archivar.');
  paragraph(`Biblioteca: ${plan.library_name}`); paragraph(`Seleccionada: ${plan.name}`);
  const labels: Record<string, string> = { shared_references: 'Documentos compartidos que conservarán su registro', roots: 'Carpetas', documents: 'Referencias documentales', preserved_documents: 'Documentos que se conservarán', cases: 'Expedientes', affected_cases: 'Expedientes con referencias afectadas', categories: 'Categorías', document_types: 'Tipos documentales', templates: 'Plantillas', views: 'Vistas lógicas', jobs: 'Tareas registradas', pending_uploads: 'Cargas pendientes', temporary_candidates: 'Registros temporales candidatos a limpieza', preserved_private_uploads: 'Cargas privadas conservadas' };
  const list = document.createElement('ul'); for (const [key, label] of Object.entries(labels)) { if (plan.counts[key] !== undefined) { const li = document.createElement('li'); li.textContent = `${label}: ${plan.counts[key]}`; list.append(li); } } container.append(list);
  paragraph('La limpieza comprobará cada archivo. Candidatos ausentes, compartidos o cuya pertenencia no pueda garantizarse no se borrarán.');
  paragraph('Ubicaciones registradas donde se conservarán los documentos:');
  for (const path of plan.locations) paragraph(path);
  if (!plan.locations.length) paragraph('No hay ubicaciones documentales registradas.');
  const manifest = document.createElement('a'); manifest.className = 'button-link compact'; manifest.href = `/api/v1/deletions/${encodeURIComponent(plan.id)}/manifest`; manifest.textContent = '↓ Descargar manifiesto de documentos'; manifest.download = 'documentos-conservados.jsonl'; container.append(manifest);
  return container;
}
export function DeleteEntity({ kind, id, done, disabled = false }: { kind: 'root' | 'library'; id: string; done: () => void; disabled?: boolean }) {
  const [busy, setBusy] = useState(false); const lock = useRef(false);
  const [watchID, setWatchID] = useState('');
  const onDone = useRef(done); onDone.current = done;
  useEffect(() => {
    if (!watchID) return;
    const controller = new AbortController(); let timer: ReturnType<typeof setTimeout>; let failures = 0;
    const check = async () => {
      try {
        const current = await api<Deletion>(`/deletions/${watchID}`, { signal: controller.signal });
        if (controller.signal.aborted) return;
        failures = 0;
        if (['complete', 'cleanup_pending', 'failed'].includes(current.state)) {
          setWatchID(''); onDone.current();
          if (current.state === 'failed') resultNotice(current);
          return;
        }
      } catch {
        if (controller.signal.aborted) return;
        if (++failures >= 3) { setWatchID(''); toast('No se pudo consultar el estado de la eliminación. Revísalo en Eliminaciones y limpieza.', 'warning'); return; }
      }
      timer = setTimeout(() => void check(), 3000);
    };
    timer = setTimeout(() => void check(), 1500);
    return () => { controller.abort(); clearTimeout(timer); };
  }, [watchID]);
  async function remove() {
    if (lock.current) return; lock.current = true; setBusy(true);
    try {
      const plan = await api<Deletion>(`/${kind === 'library' ? 'libraries' : 'roots'}/${id}/deletion-plans`, { method: 'POST' });
      const isLibrary = kind === 'library';
      const result = await dialogs.fire<Deletion>({
        titleText: isLibrary ? 'Eliminar biblioteca definitivamente' : 'Eliminar carpeta de AIDEX', html: deletionSummary(plan), icon: 'warning',
        input: isLibrary ? 'text' : undefined, inputLabel: isLibrary ? 'Escribe exactamente el nombre de la biblioteca' : undefined, inputAttributes: { autocomplete: 'off', spellcheck: 'false' },
        confirmButtonText: isLibrary ? 'Eliminar biblioteca definitivamente' : 'Eliminar registro de carpeta', showLoaderOnConfirm: true,
        allowOutsideClick: () => !dialogs.isLoading(), allowEscapeKey: () => !dialogs.isLoading(),
        didOpen: () => { if (isLibrary) { const button = dialogs.getConfirmButton()!; const input = dialogs.getInput()!; button.disabled = true; input.addEventListener('input', () => { button.disabled = input.value !== plan.name; }); } },
        preConfirm: async value => {
          if (isLibrary && value !== plan.name) { dialogs.showValidationMessage('El nombre debe coincidir exactamente.'); return false; }
          try { return await api<Deletion>(`/deletions/${plan.id}/confirm`, { method: 'POST', body: { confirmation_name: plan.name } }); }
          catch (error) { const safe = document.createElement('span'); safe.textContent = message(error); dialogs.showValidationMessage(safe.innerHTML); return false; }
        },
      });
      if (result.isConfirmed && result.value) { resultNotice(result.value); done(); if (['pending', 'running'].includes(result.value.state)) setWatchID(result.value.id); }
    } catch (error) { await intervention(message(error)); }
    finally { lock.current = false; setBusy(false); changed(); }
  }
  return <button type="button" className="secondary danger-action" disabled={busy || !!watchID || disabled} aria-busy={busy} onClick={() => void remove()}>{busy ? 'Revisando eliminación…' : watchID ? 'Eliminación en proceso…' : kind === 'library' ? 'Eliminar biblioteca' : 'Eliminar carpeta de AIDEX'}</button>;
}
export function LibraryLifecycle({ library, user, done }: { library: Library; user: User; done: () => void }) {
  const [busy, setBusy] = useState(false); const lock = useRef(false);
  const canArchive = user.permissions.includes('permissions.manage_global') || library.permissions.includes('libraries.configure');
  async function archive() {
    if (lock.current) return; lock.current = true; setBusy(true);
    const restore = library.archived;
    try {
      if (!await confirmAction(restore ? 'Restaurar biblioteca' : 'Archivar biblioteca', restore ? `Se comprobarán la configuración y las carpetas de «${library.name}» antes de reanudar sus operaciones. Sus documentos permanecerán intactos.` : `«${library.name}» dejará de aparecer en el espacio de trabajo habitual y sus operaciones quedarán suspendidas. Podrás restaurarla posteriormente. Sus documentos permanecerán intactos.`, restore ? 'Restaurar biblioteca' : 'Archivar biblioteca')) return;
      await api(`/libraries/${library.id}/${restore ? 'restore' : 'archive'}`, { method: 'POST', revision: library.revision });
      toast(restore ? 'La biblioteca se restauró correctamente. Las verificaciones se programaron.' : 'La biblioteca se archivó correctamente.'); done();
    } catch (error) { await intervention(message(error)); } finally { lock.current = false; setBusy(false); }
  }
  return <div className="actions lifecycle-actions">{canArchive && <button className="secondary" disabled={busy} aria-busy={busy} onClick={() => void archive()}>{busy ? 'Procesando…' : library.archived ? 'Restaurar biblioteca' : 'Archivar biblioteca'}</button>}{user.permissions.includes('libraries.delete') && <DeleteEntity kind="library" id={library.id} done={done} disabled={busy} />}</div>;
}
export function ResumeRoot({ root, done, disabled }: { root: Root; done: () => void; disabled: boolean }) {
  const [busy, setBusy] = useState(false); const lock = useRef(false);
  async function resume() {
    if (lock.current) return; lock.current = true; setBusy(true);
    try { if (!await confirmAction('Reanudar supervisión', `Se comprobará el acceso a ${root.server_path || 'la carpeta'} y se programará una verificación. Los originales permanecerán intactos.`, 'Reanudar supervisión')) return;
      await api(`/roots/${root.id}/resume`, { method: 'POST', revision: root.revision }); toast('La supervisión se reanudó correctamente. La verificación quedó programada.'); done();
    } catch (error) { await intervention(message(error)); } finally { lock.current = false; setBusy(false); }
  }
  return <button className="secondary" disabled={disabled || busy} aria-busy={busy} onClick={() => void resume()}>{busy ? 'Comprobando carpeta…' : 'Reanudar supervisión'}</button>;
}
export function DeletionHistory() {
  const list = usePage<Deletion>('/deletions'); const [busy, setBusy] = useState(''); const [error, setError] = useState('');
  useEffect(() => { const reload = () => list.reload(); window.addEventListener('aidex-deletion-updated', reload); return () => window.removeEventListener('aidex-deletion-updated', reload); }, [list.reload]);
  useEffect(() => { if (!list.items.some(item => ['pending', 'running', 'cleanup_pending'].includes(item.state)) || list.busy) return; const timer = setTimeout(list.reload, 5000); return () => clearTimeout(timer); }, [list.items, list.busy, list.reload]);
  return <details className="card document-accordion deletion-history"><summary>Eliminaciones y limpieza</summary><p>Registro permanente de eliminaciones autorizadas. Una limpieza pendiente conserva los archivos que requieren revisión; el servicio vuelve a intentarla de forma segura.</p><Notice text={error || list.error} /><button className="secondary" disabled={list.busy} onClick={list.reload}>Actualizar eliminaciones</button>{list.items.map(item => <article key={item.id} className="root-row"><strong className="path-text">{item.name}</strong><p>{item.state === 'pending' ? 'Eliminación pendiente' : item.state === 'running' ? 'Eliminación en proceso' : item.state === 'failed' ? 'Eliminación fallida; los registros permanecen intactos' : item.state === 'complete' ? 'Eliminación completada' : `Limpieza pendiente: ${item.pending_cleanup} archivos · ${item.cleanup_issues} incidencias`}</p>{!!item.cleanup_diagnostics?.length && <p>Diagnóstico: {item.cleanup_diagnostics.map(code => ({ CLEANUP_CHANGED: 'El archivo o su identidad cambió; se conserva para revisión.', CLEANUP_REFERENCED: 'Hay referencias compartidas; el archivo se conserva.', CLEANUP_STORAGE_UNAVAILABLE: 'Almacenamiento no disponible o permisos insuficientes.', CLEANUP_UNSAFE_PATH: 'No se pudo comprobar una ruta segura.' }[code] || code)).join(' ')}</p>}<small>Referencia de soporte: {item.id}</small><div className="actions"><a className="button-link compact" href={`/api/v1/deletions/${item.id}/manifest`} download>↓ Descargar manifiesto</a>{['failed', 'cleanup_pending'].includes(item.state) && <button className="secondary" disabled={!!busy} onClick={async () => { if (busy) return; setBusy(item.id); setError(''); try { const result = await api<Deletion>(`/deletions/${item.id}/retry`, { method: 'POST' }); resultNotice(result); list.reload(); } catch (cause) { setError(message(cause)); } finally { setBusy(''); } }}>{busy === item.id ? 'Reintentando…' : item.state === 'failed' ? 'Reintentar eliminación' : 'Reintentar limpieza'}</button>}</div></article>)}<PageEnd {...list} empty={!list.items.length} /></details>;
}
