import { useState, type FormEvent } from 'react';
import { formatDate, type Attempt } from './api';
import { AuditTable } from './AuditTable';
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

export function Events() {
 return <><header className="page-heading"><p className="eyebrow">TRAZABILIDAD</p><h1>Auditoría</h1><p>Consulta las acciones de usuarios y del sistema.</p></header><AuditTable /></>;
}
