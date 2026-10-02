import { useState } from 'react';
import { api } from './api';
import type { Root } from './libraryTypes';

const units = [{ seconds: 1, name: 'Segundos' }, { seconds: 60, name: 'Minutos' }, { seconds: 3600, name: 'Horas' }, { seconds: 86400, name: 'Días' }];
export function RootInterval({ root, busy, save }: { root: Root; busy: boolean; save: (action: () => Promise<unknown>, notice: string) => Promise<void> }) {
  const initialUnit = [...units].reverse().find(unit => root.reconcile_interval_seconds % unit.seconds === 0)?.seconds || 1;
  const [unit, setUnit] = useState(initialUnit);
  const [value, setValue] = useState(String(root.reconcile_interval_seconds / initialUnit));
  return <form className="filters" onSubmit={event => {
    event.preventDefault(); const data = new FormData(event.currentTarget);
    void save(() => api(`/roots/${root.id}`, { method: 'PATCH', revision: root.revision, body: { enabled: data.get('enabled') === 'on', reconcile_interval_seconds: Math.round(Number(value) * unit) } }), 'Configuración de vigilancia guardada.');
  }}>
    <label className="check"><input type="checkbox" name="enabled" defaultChecked={root.status === 'active' || root.status === 'inaccessible'} />Verificación habilitada</label>
    <label>Comprobar cada<input type="number" min={10 / unit} max={30 * 86400 / unit} step="any" value={value} onChange={event => setValue(event.target.value)} required /></label>
    <label>Unidad del intervalo<select value={unit} onChange={event => { const next = Number(event.target.value); setValue(String(Number(value) * unit / next)); setUnit(next); }}>{units.map(item => <option key={item.seconds} value={item.seconds}>{item.name}</option>)}</select></label>
    <button disabled={busy}>Guardar verificación</button>
    <small>Entre 10 segundos y 30 días. El watcher detecta cambios entre recorridos mientras la vigilancia nativa esté disponible.</small>
  </form>;
}
