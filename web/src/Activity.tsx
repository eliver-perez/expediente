import { useState, type FormEvent } from 'react';
import { api, formatDate, message, type Attempt, type AuditEvent } from './api';
import { SessionTable } from './Account';
import { Notice, PageEnd, usePage } from './components';

function Filters({ onApply, children }: { onApply: (query: string) => void; children?: React.ReactNode }) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); const query = new URLSearchParams();
    for (const [key, value] of data.entries()) {
      if (typeof value !== 'string' || !value) continue;
      query.set(key, key === 'from' || key === 'to' ? new Date(value).toISOString() : value);
    }
    onApply(query.toString());
  }
  return <form className="filters" onSubmit={submit}>
    <label>Desde<input type="datetime-local" name="from" /></label>
    <label>Hasta (sin incluir)<input type="datetime-local" name="to" /></label>
    {children}<button className="secondary">Filtrar</button>
  </form>;
}

function Attempts({ query }: { query: string }) {
  const entries = usePage<Attempt>(`/authentication-attempts?${query}`);
  const outcomes: Record<string, string> = { success: 'Correcto', failed: 'Fallido', rate_limited: 'Limitado' };
  return <><Notice text={entries.error} /><div className="table-wrap"><table><thead><tr><th>Fecha</th><th>Identificador intentado</th><th>IP observada</th><th>Navegador</th><th>Resultado</th></tr></thead><tbody>
    {entries.items.map(attempt => <tr key={attempt.id}><td>{formatDate(attempt.occurred_at)}</td><td>{attempt.attempted_identifier || '(vacío)'}</td><td className="mono">{attempt.observed_ip}</td><td className="agent">{attempt.user_agent}</td><td><span className={`badge ${attempt.outcome === 'success' ? 'positive' : ''}`}>{outcomes[attempt.outcome]}</span></td></tr>)}
  </tbody></table></div><PageEnd {...entries} empty={entries.items.length === 0} /></>;
}

export function Access() {
  const [tab, setTab] = useState('sessions');
  const [query, setQuery] = useState('');
  return <>
    <header className="page-heading"><p className="eyebrow">AUDITORÍA</p><h1>Historial de accesos</h1><p>Consulta inicios, cierres e intentos de autenticación de la organización.</p></header>
    <section className="card"><div className="tabs" role="tablist" aria-label="Historial de accesos"><button role="tab" aria-selected={tab === 'sessions'} className={tab === 'sessions' ? 'active' : ''} onClick={() => { setTab('sessions'); setQuery(''); }}>Sesiones</button><button role="tab" aria-selected={tab === 'attempts'} className={tab === 'attempts' ? 'active' : ''} onClick={() => { setTab('attempts'); setQuery(''); }}>Intentos de acceso</button></div>
      <Filters key={tab} onApply={setQuery}>{tab === 'attempts' && <label>Resultado<select name="outcome"><option value="">Todos</option><option value="success">Correctos</option><option value="failed">Fallidos</option><option value="rate_limited">Limitados</option></select></label>}</Filters>
      {tab === 'sessions' ? <SessionTable endpoint={`/sessions?${query}`} /> : <Attempts query={query} />}
    </section>
  </>;
}

const eventNames: Record<string, string> = {
  'installation.bootstrapped': 'Administrador inicial creado', 'authentication.success': 'Autenticación correcta',
  'authentication.failed': 'Autenticación fallida', 'authentication.rate_limited': 'Intentos de acceso limitados',
  'session.started': 'Sesión iniciada', 'session.closed': 'Sesión cerrada', 'user.created': 'Usuario creado',
  'user.updated': 'Usuario actualizado', 'permissions.global_changed': 'Permisos globales modificados',
  'user.sessions_revoked': 'Sesiones revocadas', 'password.changed': 'Contraseña cambiada',
  'password.reset': 'Contraseña restablecida', 'password.local_recovery': 'Recuperación administrativa local'
};

export function Events() {
  const [query, setQuery] = useState('');
  const list = usePage<AuditEvent>(`/audit-events?${query}`);
  const [detail, setDetail] = useState<AuditEvent | null>(null);
  const [error, setError] = useState('');
  async function view(eventID: string) {
    setError(''); setDetail(null);
    try { setDetail(await api<AuditEvent>(`/audit-events/${eventID}`)); } catch (error) { setError(message(error)); }
  }
  return <>
    <header className="page-heading"><p className="eyebrow">TRAZABILIDAD</p><h1>Visor de eventos</h1><p>Bitácora de las acciones de usuarios y del sistema.</p></header>
    <section className="card"><Filters onApply={query => { setQuery(query); setDetail(null); }}><label>Tipo de evento<select name="event_type"><option value="">Todos</option>{Object.entries(eventNames).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label></Filters>
      <Notice text={list.error || error} /><div className="table-wrap"><table><thead><tr><th>Fecha</th><th>Evento</th><th>Actor</th><th>IP observada</th><th>Detalle</th></tr></thead><tbody>{list.items.map(event => <tr key={event.id}>
        <td>{formatDate(event.occurred_at)}</td><td>{eventNames[event.event_type] ?? event.event_type}</td><td>{event.actor_kind === 'system' ? 'Sistema' : event.actor_kind === 'anonymous' ? 'Sin sesión' : event.actor_user_id}</td><td>{event.observed_ip ?? '—'}</td><td><button className="text-button" onClick={() => void view(event.id)}>Ver detalle</button></td>
      </tr>)}</tbody></table></div><PageEnd {...list} empty={list.items.length === 0} />
    </section>
    {detail && <section className="card"><div className="section-heading"><h2>{eventNames[detail.event_type] ?? detail.event_type}</h2><button className="secondary" onClick={() => setDetail(null)}>Cerrar detalle</button></div><p className="muted">{formatDate(detail.occurred_at)}</p><dl className="event-detail">{Object.entries(detail.details ?? {}).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{typeof value === 'string' ? value : JSON.stringify(value)}</dd></div>)}</dl><small>Referencia: {detail.request_id}</small></section>}
  </>;
}
