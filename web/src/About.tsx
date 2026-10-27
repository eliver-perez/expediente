import { useEffect, useRef, useState } from 'react';
import { api, formatDate, message } from './api';
import { Brand } from './Brand';
import { Notice } from './components';
import { LegalMarkdown } from './LegalMarkdown';

interface AboutData { product: string; full_name: string; version: string; os: string; installation_id: string }
interface Contract { version: string; accepted_version: string; accepted_at: string; accepted: boolean; draft: boolean; text: string }

export function About() {
  const [about, setAbout] = useState<AboutData | null>(null);
  const [contract, setContract] = useState<Contract | null>(null);
  const [error, setError] = useState('');
  const modal = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const controller = new AbortController();
    void Promise.all([api<AboutData>('/about', { signal: controller.signal }), api<Contract>('/eula', { signal: controller.signal })])
      .then(([details, legal]) => { setAbout(details); setContract(legal); })
      .catch(cause => { if (!controller.signal.aborted) setError(message(cause)); });
    return () => controller.abort();
  }, []);
  return <><header className="page-heading"><p className="eyebrow">INFORMACIÓN DEL PRODUCTO</p><h1>Acerca de AIDEX</h1></header>
    <section className="card about-card"><Brand variant="horizontal" /><h2>Aplicación de Indexación Digital de Expedientes</h2><p>Tu biblioteca digital, ordenada y al alcance.</p><Notice text={error} />
      {!about || !contract ? <p role="status">Consultando información de esta instalación…</p> : <>
        <dl className="license-facts"><div><dt>Versión instalada</dt><dd>{about.version}</dd></div><div><dt>Sistema operativo</dt><dd>{about.os}</dd></div><div><dt>Contrato</dt><dd>{contract.accepted ? 'Aceptado' : 'Pendiente de aceptación'}</dd></div><div><dt>Versión aceptada</dt><dd>{contract.accepted_version || 'Ninguna'}</dd></div><div><dt>Fecha de aceptación</dt><dd>{contract.accepted_at ? formatDate(contract.accepted_at) : '—'}</dd></div><div><dt>Referencia para soporte</dt><dd className="path-text">{about.installation_id || 'No disponible'}</dd></div></dl>
        {contract.accepted_version && !contract.accepted && <p className="notice">La versión aceptada anteriormente es distinta del contrato actual.</p>}
        <div className="actions"><button className="secondary" onClick={() => modal.current?.showModal()}>Ver contrato de licencia</button><a className="button secondary" href="https://aidex.com.mx" target="_blank" rel="noopener noreferrer">Sitio oficial de AIDEX ↗</a></div>
      </>}
    </section>
    <dialog className="contract-dialog" ref={modal} aria-labelledby="contract-title"><header><h2 id="contract-title">Contrato de Licencia de Usuario Final</h2><button type="button" className="secondary" onClick={() => modal.current?.close()}>Cerrar</button></header><p><strong>AIDEX</strong> · {contract?.version} · {contract?.draft ? 'Borrador pendiente de revisión jurídica' : 'Versión vigente'}</p><p>Estado: {contract?.accepted ? 'Aceptado' : 'Pendiente de aceptación'}{contract?.accepted_at ? ` · ${formatDate(contract.accepted_at)}` : ''}</p><div className="eula-text" tabIndex={0}>{contract && <LegalMarkdown text={contract.text} />}</div><footer><button type="button" onClick={() => modal.current?.close()}>Cerrar</button></footer></dialog>
  </>;
}
