import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';

export interface FilePolicy {
  store_formats: string[];
  index_formats: string[];
  maximum_file_mb: number;
  maximum_index_mb: number;
  unknown_policy: string;
  non_indexable_policy: string;
}
type Overrides = { [Key in keyof FilePolicy]: FilePolicy[Key] | null };
export interface FileConfiguration {
  defaults: FilePolicy; global: FilePolicy; effective: FilePolicy; overrides: Overrides;
  revision: number; global_revision: number;
  formats: { id: string; mime: string; extractor_available: boolean }[];
}

export function FileSettings({ libraryID }: { libraryID?: string }) {
  const endpoint = libraryID ? `/libraries/${libraryID}/file-settings` : '/system/files';
  const [saved, setSaved] = useState<FileConfiguration | null>(null);
  const [policy, setPolicy] = useState<FilePolicy | null>(null);
  const [overrides, setOverrides] = useState<Overrides | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [busy, setBusy] = useState(false);
  const [refresh, setRefresh] = useState(0);
  function receive(value: FileConfiguration) { setSaved(value); setPolicy(value.effective); setOverrides(value.overrides); }
  useEffect(() => {
    const abort = new AbortController();
    api<FileConfiguration>(endpoint, { signal: abort.signal }).then(receive).catch(cause => { if (!abort.signal.aborted) setError(message(cause)); });
    return () => abort.abort();
  }, [endpoint, refresh]);
  function change<Key extends keyof FilePolicy>(key: Key, value: FilePolicy[Key]) {
    if (!policy || !overrides) return;
    const updated = { ...policy, [key]: value };
    const exceptions = { ...overrides, [key]: value };
    if (key === 'store_formats') {
      updated.index_formats = (libraryID && overrides.index_formats === null ? saved!.global.index_formats : updated.index_formats).filter(format => updated.store_formats.includes(format));
      if (!libraryID || overrides.index_formats !== null) exceptions.index_formats = updated.index_formats;
    }
    if (key === 'maximum_file_mb' && libraryID && overrides.maximum_index_mb === null) updated.maximum_index_mb = Math.min(saved!.global.maximum_index_mb, updated.maximum_file_mb);
    setPolicy(updated); setOverrides(exceptions);
  }
  function inherit(key: keyof FilePolicy, checked: boolean) {
    if (!policy || !overrides || !saved) return;
    const next = { ...overrides, [key]: checked ? null : policy[key] };
    const effective = { ...saved.global };
    for (const field of Object.keys(next) as (keyof FilePolicy)[]) {
      if (next[field] !== null) Object.assign(effective, { [field]: next[field] });
    }
    if (next.index_formats === null) effective.index_formats = effective.index_formats.filter(format => effective.store_formats.includes(format));
    if (next.maximum_index_mb === null) effective.maximum_index_mb = Math.min(effective.maximum_index_mb, effective.maximum_file_mb);
    setOverrides(next); setPolicy(effective);
  }
  const inherited = (key: keyof FilePolicy) => !!libraryID && overrides?.[key] === null;
  const inheritance = (key: keyof FilePolicy, label: string) => libraryID && <label className="check"><input type="checkbox" checked={inherited(key)} onChange={event => inherit(key, event.target.checked)} />Heredar {label}</label>;
  return <>
    {!libraryID && <header className="page-heading"><p className="eyebrow">CONFIGURACIÓN GENERAL</p><h2>Archivos y procesamiento</h2><p>Define qué documentos puede incorporar AIBID y cuáles pueden aportar texto a las búsquedas.</p></header>}
    <section className="card"><div className="section-heading"><h2>{libraryID ? 'Archivos y procesamiento' : 'Reglas de archivos'}</h2><button className="secondary" disabled={busy} onClick={() => { setError(''); setSuccess(''); setRefresh(refresh + 1); }}>Actualizar reglas</button></div>
      <Notice text={error} /><Notice text={success} kind="success" />
      {libraryID && <p>Hereda cada regla de la instalación o define una excepción para esta biblioteca. Los formatos bloqueados por seguridad no pueden habilitarse.</p>}
      <p>Almacenar permite registrar, clasificar y descargar el original. Indexar permite extraer su contenido para buscarlo. PDF dispone de extracción de texto y OCR. DOCX, XLSX, TXT y CSV disponen de extracción de contenido estático al habilitar su indexación.</p>
      {policy && saved && <form className="form-grid" onSubmit={async event => {
        event.preventDefault(); setBusy(true); setError(''); setSuccess('');
        try {
          receive(await api<FileConfiguration>(endpoint, { method: 'PUT', body: { configuration: policy, overrides, revision: saved.revision, global_revision: saved.global_revision } }));
          setSuccess('Reglas guardadas. Se aplican a nuevas incorporaciones y al procesamiento pendiente. Los documentos y el texto ya conservados no se eliminan.');
        } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
      }}>
        <fieldset className="settings-section"><legend>Formatos permitidos</legend>
          {inheritance('store_formats', 'formatos de almacenamiento')}{inheritance('index_formats', 'formatos de indexación')}
          <div className="table-wrap"><table><thead><tr><th>Formato</th><th>Almacenar</th><th>Indexar contenido</th><th>Extracción disponible</th></tr></thead><tbody>
            {saved.formats.map(format => <tr key={format.id}><th scope="row">{format.id.toUpperCase()}</th>
              {(['store_formats', 'index_formats'] as const).map(key => <td key={key}><input type="checkbox" aria-label={`${key === 'store_formats' ? 'Almacenar' : 'Indexar'} ${format.id.toUpperCase()}`} checked={policy[key].includes(format.id)} disabled={busy || inherited(key) || key === 'index_formats' && !policy.store_formats.includes(format.id)} onChange={event => change(key, event.target.checked ? [...policy[key], format.id] : policy[key].filter(id => id !== format.id))} /></td>)}
              <td>{format.extractor_available ? format.id === 'pdf' ? 'Texto y OCR de PDF' : 'Texto estructurado' : 'No disponible todavía'}</td></tr>)}
          </tbody></table></div><p className="muted">La lista de indexación heredada se limita a los formatos que esta biblioteca permite almacenar.</p>
        </fieldset>
        <fieldset className="settings-section"><legend>Tamaños máximos</legend>
          {(['maximum_file_mb', 'maximum_index_mb'] as const).map(key => <div key={key}>
            {inheritance(key, key === 'maximum_file_mb' ? 'tamaño máximo de archivo' : 'tamaño máximo de indexación')}
            <label>{key === 'maximum_file_mb' ? 'Tamaño máximo por archivo (MiB)' : 'Tamaño máximo para indexación (MiB)'}<input type="number" min={1} max={key === 'maximum_file_mb' ? 4096 : policy.maximum_file_mb} required disabled={busy || inherited(key)} value={policy[key]} onChange={event => change(key, Number(event.target.value))} /></label>
          </div>)}
          <p className="muted">El límite de indexación heredado se ajusta al tamaño máximo de archivo. Un MiB equivale a 1 048 576 bytes.</p>
        </fieldset>
        <fieldset className="settings-section"><legend>Documentos sin indexación</legend>
          {inheritance('unknown_policy', 'política de extensiones desconocidas')}
          <label>Extensión no reconocida<select disabled={busy || inherited('unknown_policy')} value={policy.unknown_policy} onChange={event => change('unknown_policy', event.target.value)}><option value="reject">Rechazar (recomendado)</option><option value="store_text">Conservar solo si es texto Unicode validado, sin indexarlo</option></select></label>
          <p className="muted">Los binarios desconocidos y los archivos prohibidos siempre se rechazan. La excepción de texto requiere permitir TXT. Una extensión conocida con contenido distinto se rechaza.</p>
          {inheritance('non_indexable_policy', 'comportamiento sin indexación')}
          <label>Archivo almacenable que no puede indexarse<select disabled={busy || inherited('non_indexable_policy')} value={policy.non_indexable_policy} onChange={event => change('non_indexable_policy', event.target.value)}><option value="store">Conservar original y metadatos (recomendado)</option><option value="reject">Rechazar su incorporación</option></select></label>
          <p className="muted">Incluye formatos sin extractor y archivos que superan el límite de indexación. En bibliotecas vinculadas, un rechazo omite la incorporación a AIBID y conserva el archivo externo intacto.</p>
        </fieldset>
        <button disabled={busy}>{busy ? 'Guardando…' : 'Guardar reglas de archivos'}</button>
      </form>}
    </section>
  </>;
}
