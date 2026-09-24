import { useState, type FormEvent } from 'react';
import { api, formatDate, message, type User } from './api';
import { Notice, PageEnd, usePage } from './components';

export function Users({ currentUser, refreshSession }: { currentUser: User; refreshSession: () => Promise<void> }) {
  const list = usePage<User>('/users');
  const [selected, setSelected] = useState<User | null>(null);
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const can = (permission: string) => currentUser.permissions.includes(permission);
  async function operation(action: () => Promise<unknown>, successText: string) {
    setBusy(true); setError(''); setSuccess('');
    try { await action(); setSuccess(successText); list.reload(); await refreshSession(); }
    catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    void operation(async () => {
      await api('/users', { method: 'POST', body: { username: data.get('username'), display_name: data.get('display_name'), password: data.get('password') } });
      form.reset(); setCreating(false);
    }, 'Usuario creado. Puedes asignarle permisos desde Editar.');
  }
  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!selected) return;
    const data = new FormData(event.currentTarget);
    void operation(async () => {
      const updated = await api<User>(`/users/${selected.id}`, { method: 'PATCH', revision: selected.revision, body: { display_name: data.get('display_name'), disabled: data.get('disabled') === 'on' } });
      setSelected(updated);
    }, 'Datos actualizados.');
  }
  function roles(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!selected) return;
    const data = new FormData(event.currentTarget);
    void operation(async () => {
      const updated = await api<User>(`/users/${selected.id}/global-roles`, { method: 'PUT', revision: selected.revision, body: { role_ids: data.getAll('role_ids') } });
      setSelected(updated);
    }, 'Permisos actualizados.');
  }
  function password(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!selected) return;
    const form = event.currentTarget; const data = new FormData(form);
    void operation(async () => {
      await api(`/users/${selected.id}/password-reset`, { method: 'POST', body: { new_password: data.get('new_password') } });
      form.reset(); setSelected(null);
    }, 'Contraseña restablecida y sesiones cerradas.');
  }
  return <>
    <header className="page-heading row"><div><p className="eyebrow">ADMINISTRACIÓN</p><h1>Usuarios</h1><p>Cuentas personales y permisos de la instalación.</p></div><button onClick={() => { setCreating(!creating); setSelected(null); setError(''); }}>Nuevo usuario</button></header>
    <Notice text={error || list.error} /><Notice text={success} kind="success" />
    {creating && <section className="card"><h2>Crear usuario</h2><form onSubmit={create} className="form-grid">
      <label>Nombre visible<input name="display_name" required maxLength={120} autoFocus /></label>
      <label>Nombre de usuario<input name="username" autoComplete="off" required minLength={3} maxLength={80} /></label>
      <label>Contraseña inicial<input name="password" type="password" autoComplete="new-password" required minLength={6} maxLength={128} /></label>
      <div className="actions"><button disabled={busy}>Crear cuenta</button><button type="button" className="secondary" onClick={() => setCreating(false)}>Cancelar</button></div>
    </form></section>}
    <section className="card"><div className="table-wrap"><table><thead><tr><th>Usuario</th><th>Permisos globales</th><th>Creación</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
      {list.items.map(user => <tr key={user.id}><td><strong>{user.display_name}</strong><small>@{user.username}{user.id === currentUser.id ? ' · Tú' : ''}</small></td>
        <td>{user.role_ids.map(role => role === 'installation_admin' ? 'Administración' : 'Auditoría de accesos').join(', ') || 'Cuenta personal'}</td>
        <td>{formatDate(user.created_at)}</td><td><span className={`badge ${user.disabled ? '' : 'positive'}`}>{user.disabled ? 'Deshabilitado' : 'Habilitado'}</span></td>
        <td><button className="text-button" onClick={() => { setSelected(user); setCreating(false); setError(''); setSuccess(''); }}>Editar <span className="sr-only">a {user.username}</span></button></td></tr>)}
    </tbody></table></div><PageEnd {...list} empty={list.items.length === 0} /></section>
    {selected && <section className="card" key={selected.id + ':' + selected.revision} aria-label={`Editar ${selected.username}`}>
      <div className="section-heading"><h2>Editar a {selected.display_name}</h2><button className="secondary" onClick={() => setSelected(null)}>Cerrar edición</button></div>
      <div className="editor-grid"><form onSubmit={save} className="form-grid">
        <h3>Datos de la cuenta</h3><label>Nombre visible<input name="display_name" defaultValue={selected.display_name} required maxLength={120} /></label>
        <label className="check"><input name="disabled" type="checkbox" defaultChecked={selected.disabled} />Deshabilitar cuenta y cerrar sesiones</label><button disabled={busy}>Guardar datos</button>
      </form>
      {can('permissions.manage_global') && <form onSubmit={roles} className="form-grid"><h3>Permisos globales</h3>
        <label className="check"><input name="role_ids" type="checkbox" value="installation_admin" defaultChecked={selected.role_ids.includes('installation_admin')} />Administrador de instalación</label>
        <label className="check"><input name="role_ids" type="checkbox" value="access_auditor" defaultChecked={selected.role_ids.includes('access_auditor')} />Auditor de accesos</label>
        <p className="muted">La administración de cuentas y la consulta de bitácoras se conceden por separado.</p><button disabled={busy}>Guardar permisos</button>
      </form>}
      {can('users.reset_password') && <form onSubmit={password} className="form-grid"><h3>Restablecer contraseña</h3>
        <label>Nueva contraseña<input name="new_password" type="password" autoComplete="new-password" required minLength={6} maxLength={128} /></label>
        <p className="muted">Esta acción cierra las sesiones del usuario.</p><button className="secondary" disabled={busy}>Restablecer contraseña</button>
      </form>}
      </div>
      {can('sessions.revoke') && <div className="revoke-row"><button className="secondary" disabled={busy} onClick={() => {
        if (window.confirm(`¿Cerrar todas las sesiones de ${selected.display_name}?`)) void operation(() => api(`/users/${selected.id}/revoke-sessions`, { method: 'POST', body: { reason: 'Cierre solicitado desde administración de usuarios' } }), 'Sesiones cerradas.');
      }}>Cerrar sesiones de este usuario</button></div>}
    </section>}
  </>;
}
