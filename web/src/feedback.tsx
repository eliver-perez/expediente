import { useCallback, useEffect, useRef, useState, useSyncExternalStore, type Dispatch, type SetStateAction } from 'react';
import { createPortal } from 'react-dom';
import Swal from 'sweetalert2';
import 'sweetalert2/dist/sweetalert2.min.css';

type Kind = 'success' | 'error' | 'warning' | 'info';
interface Toast { id: number; text: string; kind: Kind }
let sequence = 0;
let entries: Toast[] = [];
const listeners = new Set<() => void>();
function publish() { listeners.forEach(listener => listener()); }
export function toast(text: string, kind: Kind = 'success') {
  if (!text) return;
  entries = [...entries.filter(item => item.text !== text || item.kind !== kind), { id: ++sequence, text, kind }].slice(-3); publish();
}
function close(id: number) { entries = entries.filter(item => item.id !== id); publish(); }
function ToastMessage({ item, paused }: { item: Toast; paused: boolean }) {
  const [hover, setHover] = useState(false);
  useEffect(() => {
    if (paused || hover) return;
    const timer = setTimeout(() => close(item.id), item.kind === 'error' ? 15000 : 9000);
    return () => clearTimeout(timer);
  }, [paused, hover, item]);
  const labels: Record<Kind, string> = { success: 'Éxito', error: 'Error', warning: 'Atención', info: 'Información' };
  return <div className={`aidex-toast toast-${item.kind}`} onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)} onFocus={() => setHover(true)} onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget)) setHover(false); }}>
    <div role={item.kind === 'error' ? 'alert' : 'status'}><strong>{labels[item.kind]}</strong><p>{item.text}</p></div><button type="button" className="text-button" aria-label="Cerrar notificación" onClick={() => close(item.id)}>×</button>
  </div>;
}
const subscribe = (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener); }; };
export function ToastHost() {
  const items = useSyncExternalStore(subscribe, () => entries);
  const [dialog, setDialog] = useState(false);
  useEffect(() => {
    const check = () => setDialog(document.body.classList.contains('swal2-shown'));
    const observer = new MutationObserver(check); observer.observe(document.body, { attributes: true, attributeFilter: ['class'] }); check();
    return () => observer.disconnect();
  }, []);
  return createPortal(<aside className="toast-host" aria-label="Notificaciones" hidden={dialog}>{items.slice(0, 3).map(item => <ToastMessage key={item.id} item={item} paused={dialog} />)}</aside>, document.body);
}
// Every completed action emits once, even when its message matches an earlier action.
// Existing field errors remain inline; success notices move into the shared viewport.
export function useToastState(): [string, Dispatch<SetStateAction<string>>] {
  const previous = useRef('');
  const set = useCallback<Dispatch<SetStateAction<string>>>(value => {
    const text = typeof value === 'function' ? value(previous.current) : value;
    previous.current = text; if (text) toast(text);
  }, []);
  return ['', set];
}
export const dialogs = Swal.mixin({
  customClass: { popup: 'aidex-dialog', confirmButton: 'aidex-confirm', cancelButton: 'secondary' },
  buttonsStyling: false, showCancelButton: true, cancelButtonText: 'Cancelar', focusCancel: true,
  returnFocus: true, reverseButtons: true,
});
export async function confirmAction(title: string, text: string, confirmButtonText: string) {
  return (await dialogs.fire({ titleText: title, text, icon: 'warning', confirmButtonText })).isConfirmed;
}
export async function intervention(text: string) { await dialogs.fire({ titleText: 'No se pudo completar la operación', text, icon: 'error', showCancelButton: false, confirmButtonText: 'Entendido' }); }
