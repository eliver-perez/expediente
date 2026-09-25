export interface User {
  id: string; username: string; display_name: string; disabled: boolean;
  revision: number; created_at: string; role_ids: string[]; permissions: string[];
}
export interface SessionResponse { license?: LicenseSummary; user: User; csrf_token: string; session_expires_at: string }
export interface Page<T> { items: T[]; next_cursor: string | null }
export interface SessionEntry {
  id: string; user_id: string; started_at: string; last_activity_at: string;
  closed_at: string | null; close_reason: string | null; observed_ip: string;
  user_agent: string; is_current: boolean;
}
export interface AuditEvent {
  id: string; occurred_at: string; actor_kind: string; actor_user_id: string | null;
  event_type: string; observed_ip: string | null; request_id: string;
  details?: Record<string, unknown>;
}
export interface Attempt { id: string; occurred_at: string; attempted_identifier: string; observed_ip: string; user_agent: string; outcome: string }

let csrfToken = '';
export function setCSRF(value: string) { csrfToken = value; }
export class APIError extends Error {
  constructor(public code: string, message: string, public status: number) { super(message); }
}
export async function api<T>(path: string, options: { method?: string; body?: unknown; revision?: number; idempotencyKey?: string; signal?: AbortSignal } = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.method && options.method !== 'GET') headers['X-CSRF-Token'] = csrfToken;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  if (options.revision) headers['If-Match'] = `"${options.revision}"`;
  let response: Response;
  try {
    response = await fetch(`/api/v1${path}`, { method: options.method ?? 'GET', headers,
      credentials: 'same-origin', cache: 'no-store', signal: options.signal,
      body: options.body === undefined ? undefined : JSON.stringify(options.body) });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error;
    throw new APIError('NETWORK_ERROR', 'No se pudo conectar con el servidor. Inténtalo nuevamente.', 0);
  }
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: { code: 'SERVER_ERROR', message: 'El servidor no pudo completar la operación.' } }));
    const failure = new APIError(payload.error.code, payload.error.message, response.status);
    if (response.status === 401 && path !== '/auth/login') {
      setCSRF(''); window.dispatchEvent(new CustomEvent('session-ended', { detail: failure }));
    }
    throw failure;
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
export const formatDate = (value: string | null) => value ? new Intl.DateTimeFormat('es-MX', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '—';
export function message(error: unknown) { return error instanceof Error ? error.message : 'No se pudo completar la operación.'; }
export const closeReasons: Record<string, string> = {
  logout: 'Cierre voluntario', expired: 'Sesión expirada', new_login: 'Otro inicio de sesión',
  password_changed: 'Contraseña modificada', admin_revoked: 'Cerrada por administración'
};

export function uploadPDF<T>(batchID: string, clientID: string, file: File, progress: (percent: number) => void): Promise<T> {
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    request.open('POST', `/api/v1/upload-batches/${batchID}/files`);
    request.setRequestHeader('X-CSRF-Token', csrfToken);
    request.upload.onprogress = event => { if (event.lengthComputable) progress(Math.round(100 * event.loaded / event.total)); };
    request.onerror = () => reject(new APIError('NETWORK_ERROR', 'Se interrumpió la conexión. Puedes reintentar el archivo.', 0));
    request.onload = () => {
      let payload;
      try { payload = JSON.parse(request.responseText); } catch { reject(new Error('El servidor no pudo completar la carga.')); return; }
      if (request.status >= 200 && request.status < 300) { resolve(payload as T); return; }
      const failure = new APIError(payload.error?.code || 'UPLOAD_FAILED', payload.error?.message || 'La carga no se completó.', request.status);
      if (request.status === 401) { setCSRF(''); window.dispatchEvent(new CustomEvent('session-ended', { detail: failure })); }
      reject(failure);
    };
    const body = new FormData(); body.append('client_file_id', clientID); body.append('file', file); request.send(body);
  });
}

export interface LicenseSummary { state: string; write_allowed: boolean; read_allowed: boolean; clock_warning: boolean }
export async function importLicense<T>(file: File): Promise<T> {
  if (file.size > 65536) throw new APIError('REQUEST_TOO_LARGE', 'El archivo no debe exceder 64 KiB.', 413);
  const body = new FormData(); body.append('file', file);
  let response: Response;
  try { response = await fetch('/api/v1/license/import', { method: 'POST', body, headers: { 'X-CSRF-Token': csrfToken }, credentials: 'same-origin', cache: 'no-store' }); }
  catch { throw new APIError('NETWORK_ERROR', 'No se pudo conectar con el servidor.', 0); }
  const payload = await response.json();
  if (!response.ok) {
    const failure = new APIError(payload.error?.code || 'IMPORT_FAILED', payload.error?.message || 'No fue posible importar la licencia.', response.status);
    if (response.status === 401) { setCSRF(''); window.dispatchEvent(new CustomEvent('session-ended', { detail: failure })); }
    throw failure;
  }
  return payload as T;
}
