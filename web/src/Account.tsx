import { useState, type FormEvent } from 'react';
import { api, closeReasons, formatDate, message, type SessionEntry, type User } from './api';
import { Notice, PageEnd, usePage } from './components';

export function SessionTable({ endpoint }: { endpoint: string }) {
  const entries = usePage<SessionEntry>(endpoint);
  return <>
    <Notice text={entries.error} />
    <div className="table-wrap"><table><thead><tr><th>Inicio</th><th>Última actividad</th><th>IP observada</th><th>Navegador</th><th>Estado</th></tr></thead>
      <tbody>{entries.items.map(session => <tr key={session.id}>
        <td>{formatDate(session.started_at)}</td><td>{formatDate(session.last_activity_at)}</td>
        <td className="mono">{session.observed_ip}</td><td className="agent">{session.user_agent || 'No informado'}</td>
        <td><span className={`badge ${session.closed_at ? '' : 'positive'}`}>{session.is_current ? 'Sesión actual' : session.close_reason ? closeReasons[session.close_reason] : 'Activa'}</span>
          {session.closed_at && <small>{formatDate(session.closed_at)}</small>}</td>
      </tr>)}</tbody></table></div>
    <PageEnd {...entries} empty={entries.items.length === 0} />
  </>;
}

export function Account({ user, onPasswordChanged }: { user: User; onPasswordChanged: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function changePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError('');
    const form = event.currentTarget;
    const data = new FormData(form);
    if (data.get('new_password') !== data.get('confirmation')) { setError('Las contraseñas nuevas no coinciden.'); return; }
    setBusy(true);
    try {
      await api('/account/password', { method: 'POST', body: { current_password: data.get('current_password'), new_password: data.get('new_password') } });
      form.reset(); onPasswordChanged();
    } catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  return <>
    <header className="page-heading"><p className="eyebrow">CUENTA PERSONAL</p><h1>Mi cuenta</h1><p>Administra tu contraseña y revisa tus accesos recientes.</p></header>
    <div className="account-grid">
      <section className="card profile"><div className="avatar">{user.display_name.slice(0, 1).toUpperCase()}</div><h2>{user.display_name}</h2><p>@{user.username}</p><span className="badge positive">Cuenta habilitada</span><dl><dt>Cuenta creada</dt><dd>{formatDate(user.created_at)}</dd><dt>Sesiones simultáneas</dt><dd>Una sesión por cuenta</dd></dl></section>
      <section className="card"><h2>Cambiar contraseña</h2><p className="muted">Al cambiarla, se cerrará tu sesión y deberás volver a ingresar.</p><Notice text={error} />
        <form onSubmit={changePassword} className="form-grid">
          <label>Contraseña actual<input name="current_password" type="password" autoComplete="current-password" required maxLength={128} /></label>
          <div><label>Nueva contraseña<input name="new_password" type="password" autoComplete="new-password" required minLength={6} maxLength={128} aria-describedby="password-requirements" /></label><small id="password-requirements">De 6 a 128 caracteres. Puedes utilizar espacios.</small></div>
          <label>Repetir nueva contraseña<input name="confirmation" type="password" autoComplete="new-password" required minLength={6} maxLength={128} /></label>
          <div><button disabled={busy}>{busy ? 'Guardando…' : 'Actualizar contraseña'}</button></div>
        </form>
      </section>
    </div>
    <section className="card"><div className="section-heading"><div><h2>Mis accesos</h2><p className="muted">La IP y el navegador ayudan a revisar actividad; no identifican por sí solos a una persona.</p></div></div><SessionTable endpoint="/account/sessions" /></section>
  </>;
}
