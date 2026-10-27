import { useRef, useState, type ChangeEvent, type KeyboardEvent } from 'react';

const alphabet = /^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]*$/;
const fullCode = /^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{3}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}$/;

export function formatLicenseKey(raw: string): string {
  const value = raw.toUpperCase().replaceAll('-', '');
  if (value.length > 15 || !alphabet.test(value)) return '';
  return [value.slice(0, 3), value.slice(3, 7), value.slice(7, 11), value.slice(11, 15)].filter(Boolean).join('-');
}

export function LicenseKeyField({ disabled, value, onChange }: { disabled: boolean; value: string; onChange: (value: string) => void }) {
  const input = useRef<HTMLInputElement>(null);
  const [error, setError] = useState('');
  function change(event: ChangeEvent<HTMLInputElement>) {
    const raw = event.target.value.trim();
    const upper = raw.toUpperCase();
    const parts = upper.split('-');
    const separatorsValid = parts.length === 1 || parts.length <= 4 && parts.slice(0, -1).every((part, index) => part.length === [3, 4, 4][index]);
    const formatted = separatorsValid ? formatLicenseKey(raw) : '';
    if (!formatted && raw !== '') { onChange(''); setError('Usa 15 caracteres del alfabeto permitido, en grupos 3-4-4-4. No se aceptan I, L, O, 0 ni 1.'); return; }
    const beforeCursor = raw.slice(0, event.target.selectionStart ?? raw.length).replaceAll('-', '').length;
    // Count symbols instead of raw offsets so editing between groups preserves the caret.
    let seen = 0; let target = 0;
    while (target < formatted.length && seen < beforeCursor) { if (formatted[target] !== '-') seen++; target++; }
    while (target < formatted.length && formatted[target] === '-') target++;
    setError(''); onChange(formatted);
    requestAnimationFrame(() => { if (input.current === document.activeElement) input.current?.setSelectionRange(target, target); });
  }
  function keyDown(event: KeyboardEvent<HTMLInputElement>) {
    const field = event.currentTarget;
    if (event.key !== 'Backspace' || field.selectionStart !== field.selectionEnd || !field.selectionStart || value[field.selectionStart - 1] !== '-') return;
    event.preventDefault();
    const before = value.slice(0, field.selectionStart - 2);
    const after = value.slice(field.selectionStart);
    const next = formatLicenseKey(before + after);
    onChange(next); setError('');
    requestAnimationFrame(() => field.setSelectionRange(before.length, before.length));
  }
  return <label>Clave de licencia<input ref={input} aria-label="Clave de licencia" aria-describedby="license-key-help license-key-error" type="text" name="license_key" value={value} onChange={change} onKeyDown={keyDown} autoComplete="off" autoCapitalize="characters" spellCheck={false} inputMode="text" required maxLength={128} disabled={disabled} />
    <small id="license-key-help">Código de 15 caracteres, agrupado 3-4-4-4. Puedes pegarlo con o sin guiones.</small>{(error || value && !fullCode.test(value)) && <small id="license-key-error" role="alert">{error || 'Completa los cuatro grupos antes de activar.'}</small>}</label>;
}

export function validLicenseKey(value: string) { return fullCode.test(value); }
