import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { api, formatDate, importLicense, message, type LicenseSummary } from './api';
import { Notice } from './components';

export interface LicenseStatus extends LicenseSummary {
  license_type: string; license_id: string; activation_id: string; license_revision: number;
  installation_id: string; installation_public_key: string; fingerprint_hash: string;
  features: Record<string, boolean>; expires_at: string | null; grace_until: string | null;
  maintenance_until: string | null; entitled_release_until: string;
  development: boolean; online_configured: boolean; last_contact_at: string; last_error_code: string;
  diagnostic: string; offline_deactivation_pending: boolean;
  pending_operations: { request_id: string; action: string; created_at: string }[];
}
interface Artifact { id: string; direction: string; action: string; request_id: string; created_at: string; sha256: string }
export const licenseStates: Record<string, string> = { active: 'Activa', grace: 'En tolerancia', expired: 'Vencida · solo lectura', revoked: 'Revocada o desactivada · solo lectura', invalid: 'Requiere recuperación', unactivated: 'Sin activar', development: 'Desarrollo sin licencia comercial' };
const features: Record<string, string> = { linked_libraries: 'Bibliotecas vinculadas', managed_libraries: 'Bibliotecas administradas', ocr: 'Extracción y OCR', expedientes: 'Expedientes', review_workflow: 'Revisión y aprobación' };
const actions: Record<string, string> = { activate: 'Activación', renew: 'Renovación', refresh: 'Renovación', deactivate: 'Desactivación', offline_import: 'Importación' };
export function License({ onChange }: { onChange: () => Promise<void> }) {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  const [action, setAction] = useState('activate'); const [confirm, setConfirm] = useState(false);
  const [requestID, setRequestID] = useState<string>(() => crypto.randomUUID());
  const offlineIDs = useRef<Record<string, string>>({});
  const restoredRequest = useRef(false);
  const reload = useCallback(async () => {
    const [license, history] = await Promise.all([api<LicenseStatus>('/license'), api<{ items: Artifact[] }>('/license/artifacts')]);
    setStatus(license); setArtifacts(history.items);
    if (!restoredRequest.current) {
      restoredRequest.current = true;
      const pending = license.pending_operations[0];
      if (pending) { setRequestID(pending.request_id); setAction(pending.action); setNotice('Hay una solicitud pendiente. Puedes reintentarla; para activar, introduce la misma clave comercial.'); }
    }
  }, []);
  useEffect(() => { const load = () => { void reload().catch(cause => setError(message(cause))); }; load(); const timer = window.setInterval(load, 30_000); return () => window.clearInterval(timer); }, [reload]);
  async function run(operation: () => Promise<unknown>, success: string) {
    setBusy(true); setError(''); setNotice('');
    try { await operation(); setNotice(success); await reload(); await onChange(); }
    catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  async function online(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const key = new FormData(form).get('license_key');
    form.reset();
    await run(async () => {
      await api(`/license/${action}`, { method: 'POST', body: { request_id: requestID, license_key: action === 'activate' ? key : '', confirm } });
      setRequestID(crypto.randomUUID()); setConfirm(false);
    }, action === 'deactivate' ? 'El servidor confirmó la desactivación. Tus documentos se conservan.' : 'La licencia se verificó y quedó guardada.');
  }
  async function offline(selected: string) {
    await run(async () => {
      const id = offlineIDs.current[selected] ??= crypto.randomUUID();
      const artifact = await api<Artifact>('/license/offline-requests', { method: 'POST', body: { action: selected, request_id: id } });
      const anchor = document.createElement('a'); anchor.href = `/api/v1/license/artifacts/${artifact.id}`; anchor.download = ''; anchor.click();
    }, selected === 'deactivate' ? 'Solicitud creada. La plaza sigue ocupada hasta que el proveedor confirme su procesamiento.' : 'Solicitud descargada. Entrégala al proveedor para recibir el archivo .lic.');
  }
  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const file = new FormData(form).get('file');
    if (!(file instanceof File) || !file.size) return;
    await run(async () => { await importLicense<LicenseStatus>(file); form.reset(); }, 'Licencia importada y verificada.');
  }
  return <>
    <header className="page-heading"><p className="eyebrow">ADMINISTRACIÓN</p><h1>Licencia de AIBID</h1><p>Activa esta instalación, renueva su licencia y consulta los módulos disponibles.</p></header>
    <Notice text={error} /><Notice text={notice} kind="success" />
    {!status ? <p role="status">Consultando licencia…</p> : <>
      <section className="card"><div className="section-heading"><div><h2>Estado de la instalación</h2><span className={`badge ${status.write_allowed ? 'positive' : ''}`} data-testid="license-state">{licenseStates[status.state] || status.state}</span></div><button className="secondary" disabled={busy} onClick={() => void run(reload, '')}>Actualizar estado</button></div>
        {status.development && <p className="muted">Esta compilación de desarrollo permite pruebas sin licencia. Al importar o activar una licencia se aplicarán sus condiciones.</p>}
        {(status.state === 'expired' || status.state === 'revoked') && <p>La consulta y descarga de documentos siguen disponibles según los permisos de cada usuario. Las modificaciones están suspendidas.</p>}
        {status.state === 'grace' && <p>Renueva antes del {formatDate(status.grace_until)} para continuar modificando documentos.</p>}
        <Notice text={status.clock_warning ? 'Se detectó un retroceso del reloj. Revisa la fecha del equipo; la tolerancia no se prolonga por este cambio.' : ''} />
        <Notice text={status.diagnostic ? `Diagnóstico de recuperación: ${status.diagnostic}.` : ''} />
        {status.offline_deactivation_pending && <p className="notice">Desactivación sin conexión pendiente de confirmación del proveedor.</p>}
        <dl className="license-facts"><div><dt>Modalidad</dt><dd>{status.license_type === 'perpetual' ? 'Perpetua · uso sin vencimiento' : status.license_type === 'subscription' ? 'Suscripción' : 'Sin licencia comercial'}</dd></div><div><dt>Vencimiento de uso</dt><dd>{status.license_type === 'perpetual' ? 'No vence' : formatDate(status.expires_at)}</dd></div><div><dt>Fin de tolerancia</dt><dd>{formatDate(status.grace_until)}</dd></div><div><dt>Mantenimiento contratado hasta</dt><dd>{formatDate(status.maintenance_until)}</dd></div><div><dt>Versiones publicadas incluidas hasta</dt><dd>{formatDate(status.entitled_release_until)}</dd></div><div><dt>Último contacto confirmado</dt><dd>{formatDate(status.last_contact_at)}</dd></div></dl>
        {status.license_type === 'perpetual' && <p className="muted">El mantenimiento y las nuevas versiones se adquieren por separado. Su vencimiento no impide usar la versión instalada.</p>}
        {status.last_error_code && <p>Último contacto no completado: <code>{status.last_error_code}</code>. Un fallo de conexión no invalida una licencia vigente.</p>}
        <ul className="license-features">{Object.entries(features).map(([key, name]) => <li key={key}><span>{name}</span><span className={`badge ${status.features[key] ? 'positive' : ''}`}>{status.features[key] ? 'Habilitado' : 'No habilitado'}</span></li>)}</ul>
      </section>
      <div className="account-grid license-columns"><section className="card"><h2>Activación y renovación en línea</h2>
        {!status.online_configured && <p className="muted">No hay un servidor de licencias configurado. Puedes utilizar archivos sin conexión.</p>}
        {status.pending_operations.length > 0 && <label>Retomar solicitud pendiente<select aria-label="Retomar solicitud pendiente" value={status.pending_operations.some(item => item.request_id === requestID) ? requestID : ''} disabled={busy} onChange={event => { const pending = status.pending_operations.find(item => item.request_id === event.target.value); if (pending) { setRequestID(pending.request_id); setAction(pending.action); setConfirm(false); } }}><option value="">Seleccionar solicitud</option>{status.pending_operations.map(item => <option key={item.request_id} value={item.request_id}>{actions[item.action]} · {formatDate(item.created_at)}</option>)}</select></label>}
        <form className="form-grid" onSubmit={online}><label>Operación<select aria-label="Operación" value={action} disabled={busy} onChange={event => { if (event.target.value !== action) { setAction(event.target.value); setRequestID(crypto.randomUUID()); setConfirm(false); } }}><option value="activate">Activar con clave comercial</option><option value="refresh">Renovar / consultar con el servidor</option><option value="deactivate">Desactivar esta instalación</option></select></label>
          {action === 'activate' && <label>Clave comercial<input aria-label="Clave comercial" aria-describedby="license-key-help" type="password" name="license_key" autoComplete="off" required maxLength={1024} disabled={busy || !status.online_configured} /><small id="license-key-help">Se envía únicamente al activar. Si debes reintentar, introduce la misma clave.</small></label>}
          {action === 'deactivate' && <label className="checkbox-label"><input type="checkbox" checked={confirm} required onChange={event => setConfirm(event.target.checked)} />Confirmo la desactivación. La instalación quedará en consulta hasta recibir una nueva activación.</label>}
          <button disabled={busy || !status.online_configured || (action !== 'activate' && !status.activation_id)}>{busy ? 'Procesando…' : actions[action] + ' en línea'}</button>
          <small>Si falla la conexión, el botón reintenta la misma solicitud.</small><button type="button" className="secondary" disabled={busy} onClick={() => { setRequestID(crypto.randomUUID()); setNotice('Se preparó una nueva solicitud.'); }}>Preparar otra solicitud</button>
        </form>
      </section><section className="card"><h2>Operar sin conexión</h2><p className="muted">Descarga una solicitud y entrégala al proveedor. Después importa el archivo de licencia que recibas.</p>
        <div className="actions license-offline"><button className="secondary" disabled={busy} onClick={() => void offline('activate')}>Solicitar activación</button><button className="secondary" disabled={busy || !status.activation_id} onClick={() => void offline('renew')}>Solicitar renovación</button><button className="secondary" disabled={busy || !status.activation_id} onClick={() => void offline('deactivate')}>Solicitar desactivación</button></div>
        <p className="muted">Una solicitud de desactivación no libera por sí sola la plaza de la licencia.</p>
        <form onSubmit={upload} className="form-grid"><label>Archivo de licencia (.lic)<input type="file" name="file" accept=".lic" required disabled={busy} /></label><small>Máximo 64 KiB. Se verifica antes de reemplazar la licencia actual.</small><button disabled={busy}>Importar licencia</button></form>
      </section></div>
      <section className="card"><h2>Archivos de licencia</h2><p className="muted">Últimas 100 solicitudes y respuestas archivadas para esta instalación.</p><div className="table-wrap"><table><thead><tr><th>Fecha</th><th>Operación</th><th>Archivo</th></tr></thead><tbody>{artifacts.map(item => <tr key={item.id}><td>{formatDate(item.created_at)}</td><td>{actions[item.action] || item.action} · {item.direction === 'request' ? 'Solicitud' : 'Respuesta'}</td><td><a href={`/api/v1/license/artifacts/${item.id}`} download>Descargar archivo</a></td></tr>)}</tbody></table></div>{!artifacts.length && <p>No hay archivos de licencia.</p>}</section>
      <section className="card"><details><summary>Identificación para soporte</summary><dl className="license-identity"><dt>Instalación</dt><dd>{status.installation_id || 'Identidad no disponible'}</dd><dt>Licencia / activación</dt><dd>{status.license_id || '—'} / {status.activation_id || '—'}</dd><dt>Revisión aceptada</dt><dd>{status.license_revision}</dd><dt>Clave pública de instalación</dt><dd>{status.installation_public_key || '—'}</dd><dt>Huella minimizada del equipo</dt><dd>{status.fingerprint_hash || '—'}</dd></dl></details></section>
    </>}
  </>;
}
