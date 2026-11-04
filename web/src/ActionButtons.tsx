import type { ButtonHTMLAttributes } from 'react';

export function ShieldCheckIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 3 4 6v6c0 4 4 7.5 8 9 4-1.5 8-5 8-9V6l-8-3Z"/><path d="m8 12 3 3 5-6"/></svg>; }
export function SettingsIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="m9 3-.6 2.4-2 .9L4.2 5.6 2 9.4l1.8 1.7v1.8L2 14.6l2.2 3.8 2.2-.7 2 .9L9 21h6l.6-2.4 2-.9 2.2.7 2.2-3.8-1.8-1.7v-1.8L22 9.4l-2.2-3.8-2.2.7-2-.9L15 3H9Z"/><circle cx="12" cy="12" r="3"/></svg>; }
export function RefreshIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 6.1A8 8 0 0 1 20 12M4 12a8 8 0 0 0 13.9 5.9"/></svg>; }

export function EyeIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"/><circle cx="12" cy="12" r="2.7"/></svg>; }
export function TrashIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M4 7h16M9 7V4h6v3m-9 0 1 13h10l1-13M10 11v6m4-6v6"/></svg>; }
export function DocumentIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M6 2.5h8l4 4V21H6zM14 2.5V7h4M9 12h6M9 16h6"/></svg>; }
export function CloseIcon() { return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="m5 5 14 14M19 5 5 19"/></svg>; }

type ActionProps = ButtonHTMLAttributes<HTMLButtonElement>;
export function VisualizeButton(props: ActionProps) { return <button type="button" {...props} className={`secondary compact icon-label-button ${props.className || ''}`.trim()}><EyeIcon />Visualizar</button>; }
export function AuditDetailsButton(props: ActionProps) { return <button type="button" {...props} className={`secondary compact icon-only-button ${props.className || ''}`.trim()} aria-label={props['aria-label'] || 'Ver detalles del evento'} title={props.title || 'Ver detalles del evento'}><EyeIcon /></button>; }
export function RemoveButton(props: ActionProps) { return <button type="button" {...props} className={`secondary compact icon-label-button remove-button ${props.className || ''}`.trim()}><TrashIcon />Remover</button>; }
export function ModalCloseButton(props: ActionProps) { return <button type="button" {...props} className={`secondary compact icon-only-button modal-close-button ${props.className || ''}`.trim()} aria-label={props['aria-label'] || 'Cerrar ventana'} title={props.title || 'Cerrar ventana'}><CloseIcon /></button>; }
