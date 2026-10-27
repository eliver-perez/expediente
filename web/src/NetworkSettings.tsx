import { useToastState } from './feedback';
import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';
import { Accordion } from './Accordion';

interface Diagnostic {
  mode: string; listen_address: string; port: number; revision: string; editable: boolean;
  listening: boolean; local_url: string; platform: string; interfaces_error: boolean;
  interfaces: { name: string; address: string; url: string; suggested: boolean }[];
  firewall: { state: string; message: string; checks: string[]; instructions: string };
}
const modes: Record<string, string> = { local: 'Solo este equipo', lan: 'Permitir acceso desde la red local', custom: 'Configuración personalizada' };

export function NetworkSettings() {
  const [status, setStatus] = useState<Diagnostic | null>(null);
  const [mode, setMode] = useState('local');
  const [port, setPort] = useState(18090);
  const [error, setError] = useState('');
  const [notice, setNotice] = useToastState();
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    api<Diagnostic>('/system/network', { signal: controller.signal }).then(value => { setStatus(value); setMode(value.mode); setPort(value.port); })
      .catch(cause => { if (!controller.signal.aborted) setError(message(cause)); });
    return () => controller.abort();
  }, []);
  async function run(save: boolean) {
    if (!status) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const value = await api<Diagnostic>('/system/network', save ? { method: 'PUT', body: { mode, port, revision: status.revision } } : {});
      setStatus(value); setMode(value.mode); setPort(value.port);
      if (save && value.port !== status.port) {
        const localBrowser = ['127.0.0.1', 'localhost'].includes(window.location.hostname);
        if (value.mode === 'lan' || localBrowser) {
          const next = new URL(window.location.href);
          next.port = String(value.port);
          window.location.assign(next.href);
          return;
        }
        setNotice(`Puerto guardado. Desde el equipo de AIDEX abre ${value.local_url}.`);
        return;
      }
      if (save) setNotice(value.mode === 'local' ? 'Acceso local guardado. Continúa desde el equipo donde está instalado AIDEX.' : 'Acceso en red guardado. Prueba una de las direcciones indicadas desde otro equipo de tu red.');
    } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  return <><header className="page-heading"><p className="eyebrow">CONFIGURACIÓN DE AIDEX</p><h2>Acceso y red</h2><p>Elige desde qué equipos se puede acceder a esta instalación.</p></header>
    <Notice text={error} /><Notice text={notice} kind="success" />
    {!status ? <p role="status">Consultando el servicio y el firewall…</p> : <>
      <section className="card"><h2>Acceso a AIDEX</h2>
        {status.editable ? <form className="form-grid" onSubmit={event => { event.preventDefault(); void run(true); }}>
          <label>Modo de acceso<select value={mode} disabled={busy} onChange={event => setMode(event.target.value)}><option value="local">Solo este equipo</option><option value="lan">Permitir acceso desde la red local</option></select></label>
          <label>Puerto TCP<input type="number" inputMode="numeric" min={1} max={65535} step={1} value={port} disabled={busy} onChange={event => setPort(Number(event.target.value))} required /></label>
          <p>Dirección local actual: <strong>{status.local_url}</strong>. Si cambias el puerto, el navegador abrirá la dirección nueva al terminar.</p>
          {mode === 'lan' && <p>Utiliza esta opción en una red de confianza: el acceso HTTP no cifra la conexión. Cada usuario necesita iniciar sesión. AIDEX no configura el router ni publica la instalación en Internet.</p>}
          {mode === 'local' && status.mode === 'lan' && <p>Al guardar, los demás equipos dejarán de tener acceso. Podrás continuar desde el equipo donde está instalado AIDEX.</p>}
          <button disabled={busy || mode === status.mode && port === status.port}>{busy ? 'Comprobando…' : 'Guardar servidor y conexión'}</button><small>El cambio se aplica al guardar y permanece después de reiniciar. Los trabajos documentales continúan. En Windows, los cambios de puerto LAN requieren «Configurar servidor AIDEX» para actualizar el firewall con permisos de administrador.</small>
        </form> : <p>Esta instalación utiliza HTTPS o una configuración de red personalizada. Se conserva su configuración; los cambios deben realizarlos su administrador.</p>}
      </section>
      <section className="card"><div className="section-heading"><h2>Diagnóstico de conexión</h2><button className="secondary" disabled={busy} onClick={() => void run(false)}>Ejecutar diagnóstico</button></div>
        <dl className="network-facts"><dt>Estado del servicio</dt><dd>{status.listening ? 'Correcto · puerto en escucha' : 'No se pudo confirmar la escucha'}</dd><dt>Modo actual</dt><dd>{modes[status.mode]}</dd><dt>Dirección de escucha</dt><dd>{status.listen_address}</dd><dt>Puerto</dt><dd>{status.port}</dd><dt>{status.editable ? 'Acceso desde el equipo de AIDEX' : 'URL configurada'}</dt><dd className="path-text">{status.local_url}</dd></dl>
        {status.mode === 'lan' && <><h3>Acceso desde la red</h3>{status.interfaces.filter(item => item.suggested).map(item => <p key={item.name+item.address}><a href={item.url}>{item.url}</a> <span className="muted">· {item.name}</span></p>)}
          {!status.interfaces.some(item => item.suggested) && <p>No se detectaron direcciones IPv4 recomendadas. Revisa las conexiones de red del equipo.</p>}</>}
        {status.interfaces_error && <p>No se pudieron consultar las interfaces de red.</p>}
        <Accordion title="Interfaces IPv4 disponibles">{status.interfaces.map(item => <p key={item.name+item.address}>{item.name} · {item.address}{!item.suggested && ' · Posible adaptador virtual o VPN'}</p>)}{!status.interfaces.length && <p>No hay interfaces IPv4 conectadas para acceso en red.</p>}</Accordion>
        <h3>Firewall</h3><p role="status">{status.firewall.message}</p>{status.firewall.checks.length > 0 && <ul>{status.firewall.checks.map(check => <li key={check}>{check}</li>)}</ul>}<p>{status.firewall.instructions}</p>
        {status.mode === 'lan' && <p>El acceso desde la red está habilitado, pero el firewall del sistema puede impedir conexiones al puerto {status.port}. Este diagnóstico local no sustituye una prueba desde otro equipo.</p>}
      </section>
    </>}
  </>;
}
