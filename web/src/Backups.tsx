import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { api, formatDate, inspectBackup, message } from './api';
import { Notice } from './components';
import { confirmAction, toast } from './feedback';

interface RootSummary { id: string; library_id: string; source: 'linked' | 'managed'; path: string; status: string }
interface Manifest { product: string; format: number; schema: number; build: string; kind: 'basic' | 'full'; created_at: string; tables: Record<string, number>; documents: number; physical_files: number; roots: RootSummary[]; entries: { name: string; size: number }[] }
interface Operation { id: string; kind: string; state: string; stage: string; created_at: string; updated_at: string; bytes: number; error?: string; manifest?: Manifest; destination?: string; recovery_user?: string }
const states: Record<string, string> = { running: 'En curso', complete: 'Completado', failed: 'Falló', interrupted: 'Interrumpido', inspecting: 'Inspeccionando', inspected: 'Inspeccionado', pending_restart: 'Pendiente de reinicio', applying: 'Aplicando', attention: 'Requiere atención', cancelled: 'Cancelado' };
const size = (value: number) => value >= 1024 ** 3 ? `${(value / 1024 ** 3).toFixed(1)} GB` : value >= 1024 ** 2 ? `${(value / 1024 ** 2).toFixed(1)} MB` : `${Math.round(value / 1024)} KB`;

export function Backups({ canBackup, canRestore }: { canBackup: boolean; canRestore: boolean }) {
  const [operations, setOperations] = useState<Operation[]>([]);
  const [preview, setPreview] = useState<Operation | null>(null);
  const [kind, setKind] = useState<'basic' | 'full'>('basic');
  const [password, setPassword] = useState(''); const [restorePassword, setRestorePassword] = useState('');
  const [destination, setDestination] = useState(''); const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [mappings, setMappings] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const lastStates = useRef<Record<string, string>>({});
  const load = useCallback(async () => {
    const response = await api<{ items: Operation[] }>('/backups');
    for (const item of response.items) {
      if (lastStates.current[item.id] === 'running' && item.state === 'complete') toast('Respaldo generado y disponible para descargar.');
      if (lastStates.current[item.id] === 'running' && item.state === 'failed') toast('La generación falló. Consulta el diagnóstico.', 'error');
      if (lastStates.current[item.id] === 'pending_restart' && item.state === 'complete') toast('Restauración aplicada. Inicia sesión de nuevo.');
      lastStates.current[item.id] = item.state;
    }
    setOperations(response.items);
  }, []);
  useEffect(() => { void load().catch(cause => setError(message(cause))); const timer = window.setInterval(() => { void load().catch(() => {}); }, 4000); return () => window.clearInterval(timer); }, [load]);
  async function generate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError('');
    try { await api('/backups', { method: 'POST', body: { kind, password } }); setPassword(''); toast('Respaldo iniciado. Puedes salir de esta pantalla y consultar el estado después.', 'info'); await load(); }
    catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const file = new FormData(form).get('file'); if (!(file instanceof File) || !file.size) return;
    setBusy(true); setError(''); setUploadProgress(0);
    try { const result = await inspectBackup<Operation>(file, restorePassword, setUploadProgress); setPreview(result); setMappings({}); setRestorePassword(''); form.reset(); toast('El paquete pasó la verificación de estructura e integridad. Revisa las consecuencias antes de confirmar.', 'info'); await load(); }
    catch (cause) { setError(message(cause)); } finally { setBusy(false); setUploadProgress(null); }
  }
  async function schedule() {
    if (!preview?.manifest) return;
    const text = `Se reemplazarán los datos lógicos actuales por los del respaldo ${preview.manifest.kind === 'full' ? 'completo' : 'básico'}. Los originales actuales no se borrarán. La licencia de este equipo permanecerá intacta. La operación se aplicará al reiniciar el servicio y cerrará las sesiones.`;
    if (!await confirmAction('Confirmar restauración', text, 'Programar restauración')) return;
    setBusy(true); setError('');
    try { await api(`/restores/${preview.id}/confirm`, { method: 'POST', body: { destination, mappings: Object.fromEntries(Object.entries(mappings).filter(([, path]) => path.trim())), confirmation: 'RESTAURAR' } }); toast('Restauración programada. Reinicia el servicio AIDEX para aplicarla.', 'warning'); setPreview(null); await load(); }
    catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  async function discard() {
    if (!preview) return; setBusy(true);
    try { await api(`/restores/${preview.id}/discard`, { method: 'POST' }); setPreview(null); toast('Vista previa descartada.', 'info'); await load(); }
    catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  async function cancel(id: string) { try { await api(`/backups/${id}/cancel`, { method: 'POST' }); toast('Cancelación solicitada.', 'info'); await load(); } catch (cause) { setError(message(cause)); } }
  async function retryRestore(id: string) { if (!await confirmAction('Reintentar restauración', 'Se volverá a validar el paquete y el destino. No se tocarán originales existentes.', 'Programar de nuevo')) return; try { await api(`/restores/${id}/retry`, { method: 'POST' }); toast('Restauración programada de nuevo. Reinicia AIDEX.', 'warning'); await load(); } catch (cause) { setError(message(cause)); } }
  return <>
    <header className="page-heading"><p className="eyebrow">ADMINISTRACIÓN</p><h1>Respaldos y restauración</h1><p>Protege los datos lógicos y, si lo necesitas, los documentos administrados por AIDEX.</p></header>
    <Notice text={error} />
    <div className="account-grid backup-columns">{canBackup && <section className="card"><h2>Generar respaldo</h2>
      <form className="form-grid" onSubmit={generate}><label>Modalidad<select value={kind} disabled={busy} onChange={event => setKind(event.target.value as 'basic' | 'full')}><option value="basic">Básico · datos y referencias</option><option value="full">Completo · incluye archivos administrados</option></select></label>
        <p className="muted">{kind === 'basic' ? 'No contiene archivos físicos. Al restaurarlo, las rutas que no existan quedarán no disponibles.' : 'Incluye documentos publicados y cargas privadas recuperables. Los originales vinculados externos siguen en sus ubicaciones y no se copian. Puede requerir bastante tiempo y espacio.'}</p>
        <label>Contraseña de cifrado (opcional)<input type="password" autoComplete="new-password" value={password} maxLength={4096} disabled={busy} onChange={event => setPassword(event.target.value)} /></label>
        <small>Si cifras el respaldo, guarda la contraseña en un lugar seguro. Sin ella no podrás recuperarlo. Un respaldo sin contraseña también puede contener datos confidenciales.</small>
        <button disabled={busy}>{busy ? 'Iniciando…' : 'Generar respaldo'}</button></form>
    </section>}{canRestore && <section className="card"><h2>Restaurar respaldo</h2><p className="muted">Primero se comprueba el archivo sin modificar datos. La restauración requiere licencia válida del equipo destino y permiso de administrador.</p>
      <form className="form-grid" onSubmit={inspect}><label>Archivo .aidexbak<input type="file" name="file" accept=".aidexbak" required disabled={busy} /></label><label>Contraseña, si se cifró<input type="password" value={restorePassword} onChange={event => setRestorePassword(event.target.value)} disabled={busy} maxLength={4096} /></label><button disabled={busy}>{busy ? 'Inspeccionando…' : 'Validar y consultar contenido'}</button>{uploadProgress !== null && <small role="status">Transferencia: {uploadProgress}%. Después se verificará la integridad del paquete.</small>}</form>
    </section>}</div>
    {canRestore && preview?.manifest && <section className="card backup-preview"><h2>Vista previa del respaldo</h2><dl className="license-facts"><div><dt>Tipo</dt><dd>{preview.manifest.kind === 'full' ? 'Completo' : 'Básico'}</dd></div><div><dt>Creado</dt><dd>{formatDate(preview.manifest.created_at)}</dd></div><div><dt>Versión de origen</dt><dd>{preview.manifest.build}</dd></div><div><dt>Bibliotecas</dt><dd>{preview.manifest.tables.libraries ?? 0}</dd></div><div><dt>Expedientes</dt><dd>{preview.manifest.tables.cases ?? 0}</dd></div><div><dt>Documentos</dt><dd>{preview.manifest.documents}</dd></div><div><dt>Archivos físicos</dt><dd>{preview.manifest.physical_files}</dd></div><div><dt>Tamaño</dt><dd>{size(preview.bytes)}</dd></div></dl>
      <p>El cambio lógico sustituirá los registros actuales. Los archivos originales existentes no se eliminarán ni sobrescribirán. Las rutas vinculadas requieren comprobación en este equipo; un respaldo básico no recupera archivos faltantes.</p>
      <p className="muted">Las sumas de comprobación detectan daños, pero no prueban que el paquete provenga de una fuente confiable. Restaura solo respaldos cuyo origen conozcas.</p>
      {preview.manifest.kind === 'full' && <label className="form-grid">Directorio existente para archivos recuperados<input value={destination} onChange={event => setDestination(event.target.value)} placeholder="Ruta absoluta del equipo servidor" disabled={busy} /><small>AIDEX creará una carpeta nueva y aislada dentro de este directorio; no usará raíces documentales existentes.</small></label>}
      {(preview.manifest.roots ?? []).filter(root => root.source === 'linked' || preview.manifest?.kind === 'basic').length > 0 && <div className="form-grid"><h3>Rutas de documentos existentes</h3><p className="muted">Si la ruta original sigue accesible, déjala vacía. Si cambió de equipo o ubicación, indica la carpeta equivalente. AIDEX comprobará cada archivo y su huella antes de mostrarlo como disponible.</p>{(preview.manifest.roots ?? []).filter(root => root.source === 'linked' || preview.manifest?.kind === 'basic').map(root => <label key={root.id}>{root.source === 'linked' ? 'Vinculada' : 'Administrada'} · {root.path}<input value={mappings[root.id] ?? ''} onChange={event => setMappings(current => ({ ...current, [root.id]: event.target.value }))} placeholder="Nueva ruta absoluta (opcional)" disabled={busy} /></label>)}</div>}
      <div className="actions"><button disabled={busy || preview.manifest.kind === 'full' && !destination.trim()} onClick={() => void schedule()}>Confirmar restauración</button><button className="secondary" disabled={busy} onClick={() => void discard()}>Descartar vista previa</button></div>
    </section>}
    <section className="card"><div className="section-heading"><h2>Operaciones recientes</h2><button className="secondary" onClick={() => void load()}>Actualizar</button></div>
      {!operations.length ? <p className="muted">Aún no hay respaldos ni restauraciones.</p> : <div className="table-wrap"><table><thead><tr><th>Fecha</th><th>Operación</th><th>Estado</th><th>Detalle</th><th>Acciones</th></tr></thead><tbody>{operations.map(op => <tr key={op.id}><td>{formatDate(op.created_at)}</td><td>{op.kind === 'restore' ? 'Restauración' : op.kind === 'full' ? 'Respaldo completo' : 'Respaldo básico'}</td><td>{states[op.state] ?? op.state}</td><td>{op.stage}{op.error && <p className="backup-error">{op.error}</p>}{op.recovery_user && <p>Acceso de recuperación: <strong>{op.recovery_user}</strong> (contraseña actual del administrador que restauró)</p>}{op.state === 'complete' && op.kind !== 'restore' && <small>{size(op.bytes)}</small>}</td><td>{canBackup && op.state === 'complete' && op.kind !== 'restore' && <a href={`/api/v1/backups/${op.id}/download`} onClick={() => toast('Descarga preparada.', 'info')}>Descargar</a>}{canBackup && op.state === 'running' && <button className="secondary" onClick={() => void cancel(op.id)}>Cancelar</button>}{op.state === 'pending_restart' && <span>Reinicia AIDEX</span>}{canRestore && op.state === 'attention' && op.kind === 'restore' && <button className="secondary" onClick={() => void retryRestore(op.id)}>Reintentar</button>}</td></tr>)}</tbody></table></div>}
    </section>
  </>;
}
