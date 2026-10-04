import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';

interface Policy { paused: boolean; maximum_attempts: number; retry_delay_seconds: number; timeout_seconds: number; revision: number }
export function ProcessingPolicy() {
  const [policy, setPolicy] = useState<Policy | null>(null);
  const [draft, setDraft] = useState<Policy | null>(null);
  const [error, setError] = useState(''); const [success, setSuccess] = useState(''); const [busy, setBusy] = useState(false);
  async function load() { const value = await api<Policy>('/system/processing-policy'); setPolicy(value); setDraft(value); }
  useEffect(() => { void load().catch(cause => setError(message(cause))); }, []);
  async function save(value: Policy) {
    setBusy(true); setError(''); setSuccess('');
    try {
      const saved = await api<Policy>('/system/processing-policy', { method: 'PUT', body: value });
      setPolicy(saved); setDraft(saved); setSuccess(saved.paused ? 'Procesamiento pausado. Los trabajos activos pueden terminar.' : 'Procesamiento habilitado. La cola continuará con los trabajos pendientes.');
    } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  return <section className="card"><div className="section-heading"><h2>Cola y límites de procesamiento</h2><button className="secondary" disabled={busy} onClick={() => void load().catch(cause => setError(message(cause)))}>Actualizar configuración</button></div><Notice text={error}/><Notice text={success} kind="success"/>
    {policy && draft && <><p role="status">{policy.paused ? 'Procesamiento pausado' : 'Procesamiento habilitado'}</p><p>La pausa impide iniciar trabajos nuevos. Conserva la cola y permite terminar los trabajos activos, consultar y buscar documentos. Se mantiene al reiniciar AIBID.</p>
      <button className="secondary" disabled={busy} onClick={() => void save({ ...policy, paused: !policy.paused })}>{policy.paused ? 'Reanudar procesamiento' : 'Pausar procesamiento'}</button>
      <form onSubmit={event => { event.preventDefault(); void save({ ...draft, paused: policy.paused, revision: policy.revision }); }}>
        <div className="filters">
          <label>Máximo de intentos<input type="number" min={1} max={5} required value={draft.maximum_attempts} onChange={event => setDraft({ ...draft, maximum_attempts: Number(event.target.value) })}/></label>
          <label>Espera inicial entre intentos (segundos)<input type="number" min={1} max={3600} required value={draft.retry_delay_seconds} onChange={event => setDraft({ ...draft, retry_delay_seconds: Number(event.target.value) })}/></label>
          <label>Tiempo máximo por documento (segundos)<input type="number" min={10} max={3600} required value={draft.timeout_seconds} onChange={event => setDraft({ ...draft, timeout_seconds: Number(event.target.value) })}/></label>
        </div><p className="muted">Los valores se fijan al primer intento del trabajo. La espera se duplica en cada reintento hasta una hora. Solo se reintentan automáticamente errores transitorios. El tiempo del documento incluye lectura, extracción, espera de OCR y publicación; cada página OCR conserva además su límite propio. Los recorridos de carpetas continúan por separado.</p>
        <button disabled={busy}>Guardar límites de procesamiento</button>
      </form></>}
  </section>;
}
