import { useEffect, useRef, useState } from 'react';
import { api, message } from './api';
import { FolderIcon, ModalCloseButton } from './ActionButtons';
import { Notice } from './components';
import { useModalDialog } from './useModalDialog';

type Browse = { roots?: {id: string; name: string}[]; directories?: {name: string; relative_path: string}[]; server_path?: string; next_cursor?: number };
export function FolderSelector({ libraryID, disabled, select }: {libraryID: string; disabled: boolean; select: (path: string) => void}) {
  const [open, setOpen] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState('');
  const [native, setNative] = useState<{ticket: string; url: string} | null>(null);
  const [browse, setBrowse] = useState<Browse>({}), [location, setLocation] = useState({root: '', relative: '.'});
  const dialog = useRef<HTMLDialogElement>(null), generation = useRef(0);
  function close() { generation.current++; setOpen(false); setNative(null); setBusy(false); }
  useModalDialog(dialog, close, open);
  useEffect(() => { close(); }, [libraryID]);
  useEffect(() => {
    if (!native || !open) return;
    let stopped = false, working = false;
    const deadline = Date.now() + 130_000;
    const timer = setInterval(async () => {
      if (working || stopped) return;
      if (Date.now() > deadline) { setNative(null); setError('La selección no se completó. Puedes explorar las carpetas habilitadas o introducir la ruta manualmente.'); clearInterval(timer); return; }
      working = true;
      try {
        const result = await api<{done: boolean; server_path: string; code: string}>(`/folder-selections/${native.ticket}`);
        if (stopped || !result.done) return;
        clearInterval(timer); setNative(null);
        if (!result.code) { close(); select(result.server_path); }
        else if (result.code === 'cancelled') { setError('Selección cancelada. Puedes volver a intentarlo.'); }
        else setError('El selector del sistema no está disponible. Puedes explorar las carpetas habilitadas o introducir la ruta manualmente.');
      } catch (cause) { if (!stopped) { setError(message(cause)); setNative(null); clearInterval(timer); } }
      finally { working = false; }
    }, 1000);
    return () => { stopped = true; clearInterval(timer); };
  }, [native, open]);
  async function load(root = '', relative = '.', cursor = 0) {
    const version = generation.current; setBusy(true); setError(''); setNative(null);
    try {
      const result = await api<Browse>(`/libraries/${libraryID}/folder-browser`, {method: 'POST', body: {root_id: root, relative_path: relative, cursor}});
      if (version !== generation.current) return;
      setLocation({root, relative}); setBrowse(previous => cursor ? {...result, directories: [...(previous.directories || []), ...(result.directories || [])]} : result);
    } catch (cause) { if (version === generation.current) setError(message(cause)); }
    finally { if (version === generation.current) setBusy(false); }
  }
  async function start() {
    generation.current++; const version = generation.current;
    setOpen(true); setBusy(true); setError(''); setBrowse({}); setNative(null);
    try {
      const result = await api<{mode: string; ticket: string; launch_url: string}>('/folder-selections', {method: 'POST', body: {library_id: libraryID}});
      if (version !== generation.current) return;
      if (result.mode === 'native') {setNative({ticket: result.ticket, url: result.launch_url}); window.location.href = result.launch_url;}
      else await load();
    } catch (cause) { if (version === generation.current) setError(message(cause)); }
    finally { if (version === generation.current) setBusy(false); }
  }
  return <><button type="button" className="secondary icon-label-button" disabled={disabled} onClick={() => void start()}><FolderIcon />Seleccionar carpeta</button>
    <dialog ref={dialog} className="contract-dialog folder-access-dialog" aria-labelledby="folder-selector-title" onCancel={close}>
      <header><h2 id="folder-selector-title">Seleccionar carpeta del servidor</h2><ModalCloseButton onClick={close} /></header>
      <Notice text={error} />
      {native ? <><p role="status">Selecciona una carpeta en la ventana del sistema. Al terminar, AIDEX comprobará el acceso.</p><p><a href={native.url}>Abrir selector del sistema</a></p><button className="secondary" onClick={() => void load()}>Explorar carpetas habilitadas del servidor</button></> : <>
        <p className="muted">Aquí aparecen solo las ubicaciones habilitadas para esta biblioteca. También puedes cerrar esta ventana e introducir una ruta del servidor.</p>
        {busy && <p role="status">Cargando carpetas…</p>}
        {!busy && !browse.roots && !browse.server_path && <button type="button" className="secondary" onClick={() => void load()}>Explorar carpetas habilitadas del servidor</button>}
        {browse.roots?.length === 0 && <p>No hay ubicaciones habilitadas. El administrador puede habilitarlas en la configuración del servidor; también puedes introducir la ruta manualmente.</p>}
        <div className="folder-browser-list">{browse.roots?.map(root => <button type="button" className="secondary icon-label-button" disabled={busy} key={root.id} onClick={() => void load(root.id)}><FolderIcon />{root.name}</button>)}
          {browse.directories?.map(directory => <button type="button" className="secondary icon-label-button" disabled={busy} key={directory.relative_path} onClick={() => void load(location.root, directory.relative_path)}><FolderIcon />{directory.name}</button>)}</div>
        {browse.server_path && <><p className="path-text">{browse.server_path}</p><div className="actions folder-access-actions"><button type="button" disabled={busy} onClick={() => { const path = browse.server_path!; close(); select(path); }}>Seleccionar esta carpeta</button><button type="button" className="secondary" disabled={busy} onClick={() => { const parts = location.relative.split(/[\\/]/); parts.pop(); void load(location.relative === '.' ? '' : location.root, parts.join('/') || '.'); }}>Subir</button>{!!browse.next_cursor && <button type="button" className="secondary" disabled={busy} onClick={() => void load(location.root, location.relative, browse.next_cursor)}>Ver más carpetas</button>}</div></>}
      </>}
    </dialog>
  </>;
}
