import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { api, APIError, message, setCSRF, type SessionResponse, type User } from './api';
import { Notice } from './components';
import { Libraries } from './Libraries';
import { Search } from './Search';
import { Account } from './Account';
import { Users } from './Users';
import { Access, Events } from './Activity';

export function App() {
  const [user, setUser] = useState<User | null>(null);
  const [initializing, setInitializing] = useState(true);
  const [path, setPath] = useState(window.location.pathname);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const navigate = useCallback((destination: string) => { window.history.pushState({}, '', destination); setPath(destination); }, []);
  const endSession = useCallback((text: string) => { setCSRF(''); setUser(null); setNotice(text); setError(''); navigate('/login'); }, [navigate]);
  const refreshSession = useCallback(async () => {
    const session = await api<SessionResponse>('/auth/session');
    setCSRF(session.csrf_token); setUser(session.user);
  }, []);
  useEffect(() => {
    function sessionEnded(event: Event) {
      const failure = (event as CustomEvent<APIError>).detail;
      endSession(failure.code === 'SESSION_REQUIRED' ? '' : failure.message);
    }
    const pop = () => setPath(window.location.pathname);
    window.addEventListener('session-ended', sessionEnded); window.addEventListener('popstate', pop);
    refreshSession().catch(error => { if (!(error instanceof APIError && error.status === 401)) setError(message(error)); }).finally(() => setInitializing(false));
    return () => { window.removeEventListener('session-ended', sessionEnded); window.removeEventListener('popstate', pop); };
  }, [endSession, refreshSession]);
  useEffect(() => {
    if (!user) return;
    const check = () => { void refreshSession().catch(error => { if (!(error instanceof APIError && error.status === 401)) setError(message(error)); }); };
    const interval = window.setInterval(check, 30_000);
    let lastActivity = Date.now();
    const activity = () => {
      if (Date.now() - lastActivity < 60_000) return;
      lastActivity = Date.now(); void api('/account').catch(() => { /* API handles session invalidation. */ });
    };
    window.addEventListener('focus', check); window.addEventListener('pointerdown', activity); window.addEventListener('keydown', activity);
    return () => { window.clearInterval(interval); window.removeEventListener('focus', check); window.removeEventListener('pointerdown', activity); window.removeEventListener('keydown', activity); };
  }, [user?.id, refreshSession]);
  async function login(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError('');
    const form = event.currentTarget; const data = new FormData(form);
    try {
      const session = await api<SessionResponse>('/auth/login', { method: 'POST', body: { username: data.get('username'), password: data.get('password') } });
      form.reset(); setCSRF(session.csrf_token); setUser(session.user); setNotice(''); navigate('/account');
    } catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  async function logout() {
    setBusy(true); setError('');
    try { await api('/auth/logout', { method: 'POST' }); endSession('Has cerrado tu sesión.'); }
    catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  if (initializing) return <main className="loading" role="status">Preparando tu espacio de trabajo…</main>;
  if (!user) return <main className="login-layout">
    <section className="login-intro"><div className="brand"><span className="brand-mark" aria-hidden="true">G</span><span>Gestor<br />documental</span></div><div><p className="eyebrow">ESPACIO DE TRABAJO LOCAL</p><h1>Un lugar para<br />trabajar en orden.</h1><p>Accede con tu cuenta personal para continuar.</p></div><p className="intro-footer">Documentos · Personas · Trazabilidad</p></section>
    <section className="login-form"><div className="login-box"><p className="eyebrow">BIENVENIDO</p><h2>Iniciar sesión</h2><p className="muted">Introduce los datos de tu cuenta.</p><Notice text={notice} kind={notice.includes('otro equipo') ? 'error' : 'success'} /><Notice text={error} />
      <form onSubmit={login} className="form-grid"><label>Usuario<input name="username" autoComplete="username" autoFocus required maxLength={80} /></label><label>Contraseña<input name="password" type="password" autoComplete="current-password" required maxLength={128} /></label><button disabled={busy}>{busy ? 'Ingresando…' : 'Entrar'}</button></form>
      <p className="login-help">Si olvidaste tu contraseña, contacta al administrador de la instalación.</p>
    </div></section>
  </main>;
  const can = (permission: string) => user.permissions.includes(permission);
  const links = [{ path: '/libraries', label: 'Bibliotecas', available: true }, { path: '/search', label: 'Buscar documentos', available: true }, { path: '/account', label: 'Mi cuenta', available: true },
    { path: '/admin/users', label: 'Usuarios', available: can('users.manage') },
    { path: '/admin/access', label: 'Accesos', available: can('sessions.read_all') && can('authentication_attempts.read') },
    { path: '/admin/events', label: 'Eventos', available: can('audit.read_global') }];
  let content = <Account user={user} onPasswordChanged={() => endSession('Contraseña actualizada. Inicia sesión con tu nueva contraseña.')} />;
  if (path === '/libraries') content = <Libraries user={user} />;
  else if (path === '/search') content = <Search />;
  else if (path === '/admin/users' && can('users.manage')) content = <Users currentUser={user} refreshSession={refreshSession} />;
  else if (path === '/admin/access' && can('sessions.read_all') && can('authentication_attempts.read')) content = <Access />;
  else if (path === '/admin/events' && can('audit.read_global')) content = <Events />;
  else if (path.startsWith('/admin/')) content = <Notice text="No tienes permiso para consultar esta página." />;
  return <div className="app-layout"><aside className="sidebar"><div className="brand"><span className="brand-mark" aria-hidden="true">G</span><span>Gestor<br />documental</span></div><p className="nav-label">ESPACIO DE TRABAJO</p><nav aria-label="Principal">{links.filter(link => link.available).map(link => <a key={link.path} href={link.path} aria-current={path === link.path ? 'page' : undefined} onClick={event => { event.preventDefault(); setError(''); navigate(link.path); }}>{link.label}</a>)}</nav><div className="sidebar-footer"><span className="status-dot" /> Instalación local</div></aside>
    <div className="workspace"><header className="topbar"><span>Gestor documental</span><div><span className="user-name">{user.display_name}</span><button className="secondary" disabled={busy} onClick={() => void logout()}>Cerrar sesión</button></div></header><main className="main-content"><Notice text={error} />{content}</main><footer className="workspace-footer">Gestor documental · Bibliotecas y acceso controlado</footer></div>
  </div>;
}
