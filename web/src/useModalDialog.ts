import { useEffect, useRef, type RefObject } from 'react';

interface OpenModal { element: HTMLDialogElement; escape: () => void }
const openModals: OpenModal[] = [];
let listenerInstalled = false;

function topModal() { return openModals[openModals.length - 1]; }
function onEscapeKey(event: KeyboardEvent) {
  if (event.key !== 'Escape' || event.isComposing) return;
  if (document.body.classList.contains('swal2-shown')) return;
  const top = topModal();
  if (!top?.element.open) return;
  event.preventDefault();
  event.stopImmediatePropagation();
  top.escape();
}
function onNativeCancel(event: Event) {
  if (!(event.target instanceof HTMLDialogElement)) return;
  event.preventDefault();
  event.stopImmediatePropagation();
  if (topModal()?.element === event.target) topModal()?.escape();
}
function listen() {
  if (listenerInstalled) return;
  document.addEventListener('keydown', onEscapeKey, true);
  document.addEventListener('cancel', onNativeCancel, true);
  listenerInstalled = true;
}
function unlisten() {
  if (openModals.length || !listenerInstalled) return;
  document.removeEventListener('keydown', onEscapeKey, true);
  document.removeEventListener('cancel', onNativeCancel, true);
  listenerInstalled = false;
}

export function useModalDialog(dialog: RefObject<HTMLDialogElement | null>, escape: () => void, active = true) {
  const escapeRef = useRef(escape);
  escapeRef.current = escape;
  useEffect(() => {
    const element = dialog.current;
    if (!active || !element) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const entry: OpenModal = { element, escape: () => escapeRef.current() };
    openModals.push(entry);
    listen();
    if (!element.open) element.showModal();
    return () => {
      const index = openModals.indexOf(entry);
      if (index >= 0) openModals.splice(index, 1);
      if (element.open) element.close();
      unlisten();
      previousFocus?.focus();
    };
  }, [dialog, active]);
}
