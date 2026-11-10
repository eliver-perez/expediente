import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';
import { LegalMarkdown } from './LegalMarkdown';

interface EulaDocument { version: string; sha256: string; text: string; draft: boolean; accepted: boolean }

export function Eula({ onAccepted, onRejected, canAccept }: { onAccepted: () => Promise<void>; onRejected: () => Promise<void>; canAccept: boolean }) {
  const [document, setDocument] = useState<EulaDocument | null>(null);
  const [checked, setChecked] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => { void api<EulaDocument>('/eula').then(setDocument).catch(cause => setError(message(cause))); }, []);
  async function accept() {
    if (!document || !checked || busy) return;
    setBusy(true); setError('');
    try { await api('/eula/accept', { method: 'POST', body: { version: document.version, sha256: document.sha256, accept: true } }); await onAccepted(); setDocument({ ...document, accepted: true }); }
    catch (cause) { setError(message(cause)); }
    finally { setBusy(false); }
  }
  return <section className="card eula-card"><h1>Contrato de licencia de AIDEX</h1><p className="muted">Consulta la versión vigente y su estado de aceptación. La instalación conserva cada aceptación con fecha, usuario y huella del texto.</p><Notice text={error} />
    {!document ? <p role="status">Cargando contrato…</p> : <><p><strong>Versión:</strong> {document.version} · {document.draft ? 'Borrador pendiente de revisión jurídica' : 'Contrato vigente'}</p><div className="eula-text" tabIndex={0} aria-label="Texto del contrato"><LegalMarkdown text={document.text} /></div>
      {document.accepted ? <p className="notice success">Esta instalación ya registró la aceptación de esta versión.</p> : canAccept ? <><label className="checkbox-label"><input type="checkbox" checked={checked} onChange={event => setChecked(event.target.checked)} />He leído el texto mostrado y acepto expresamente esta versión.</label><div className="actions"><button disabled={!checked || busy} onClick={() => void accept()}>{busy ? 'Registrando…' : 'Aceptar y continuar'}</button><button type="button" className="secondary" disabled={busy} onClick={() => void onRejected()}>Rechazar y cerrar sesión</button></div></> : <><p>Solicita al administrador que lea y acepte esta versión para continuar.</p><button type="button" className="secondary" onClick={() => void onRejected()}>Cerrar sesión</button></>}</>}
  </section>;
}
