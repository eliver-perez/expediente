import { useState } from 'react';
import { AuditDetail, actorName, eventDescription, eventLabels, type Event } from './AuditTable';
import { CursorControls, useCursorPage } from './CursorPage';
import { Notice } from './components';
import { formatDate } from './api';

export function DocumentHistory({ documentID }: { documentID: string }) {
  const history = useCursorPage<Event>(`/documents/${documentID}/history`);
  const [selected, setSelected] = useState<Event | null>(null);
  return <div><Notice text={history.error} /><div className="table-wrap"><table aria-label="Historial del documento"><thead><tr><th>Fecha y hora</th><th>Evento</th><th>Usuario</th><th>Descripción</th><th>Detalles</th></tr></thead><tbody>{history.items.map(event => <tr key={event.id}><td>{formatDate(event.occurred_at)}</td><td>{eventLabels(event).action}</td><td>{actorName(event)}</td><td>{eventDescription(event)}</td><td><button className="text-button" onClick={() => setSelected(event)}>Ver detalles</button></td></tr>)}</tbody></table></div>
    {history.busy && <p role="status">Consultando historial…</p>}
    {!history.busy && !history.items.length && <p>No hay eventos registrados.</p>}
    <CursorControls page={history} label="Historial" />
    {selected && <AuditDetail event={selected} close={() => setSelected(null)} />}
  </div>;
}
