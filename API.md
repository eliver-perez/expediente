# Contratos HTTP internos propuestos

Base: `/api/v1`, mismo origen que React. **H2/H3 implementados:** identidad, cuenta,
usuarios, roles globales, sesiones, intentos, bitácora y bibliotecas vinculadas.
H4–H7 siguen como contratos de diseño; H3 incorpora asignaciones por biblioteca. No confundir esta API con los cuatro endpoints HTTPS externos
`/v1/activations...` del [contrato de licencias](LICENSE_CONTRACT.md), que no cambian.

## Convenciones

JSON UTF-8; UUID opacos; instantes RFC 3339 UTC, páginas desde 1. Cookies de sesión
y CSRF según [seguridad](SECURITY.md). Todas las rutas salvo login/health requieren
sesión en DB; permisos y licencia se validan en Go. Recursos ajenos devuelven 404.
`GET /health/live` solo responde estado de proceso, sin rutas, licencia ni usuarios.

Mutaciones con versión usan `If-Match: "<revision>"`; sin precondición 428, versión
obsoleta 412 `VERSION_CONFLICT`. Creaciones/comandos reintentables usan
`Idempotency-Key` asociado a actor/operación y digest del cuerpo; misma clave/cuerpo
devuelve mismo recurso, distinta petición 409 `IDEMPOTENCY_CONFLICT`. No cachear
cookies de login ni autorizar replay si actor perdió permisos. Operaciones en cola
responden 202 con `operation_id` o `job_id`; no equivalen a aprobación completada.

Listas usan `limit` (1–100, por defecto 50), cursor opaco ligado a filtros/orden/
ámbito autorizado y orden estable con ID como desempate. Sin offset sobre todo el
corpus. Límite propuesto de consulta: 2.000 caracteres; JSON común 64 KiB salvo
endpoint configurado; PDF/lote con límites propios, 413 al excederlos. Validación
de forma estricta; campos no reconocidos se rechazan para detectar errores de cliente.

```json
{
  "error": {
    "code": "SESSION_REPLACED",
    "message": "Tu sesión fue cerrada porque se inició sesión con tu cuenta desde otro equipo o navegador",
    "request_id": "UUID"
  }
}
```

400 entrada inválida (`INVALID_REQUEST`, `SEARCH_QUERY_INVALID`), 401 sesión
(`AUTHENTICATION_FAILED`, `SESSION_REPLACED`, `SESSION_EXPIRED`, `SESSION_REVOKED`),
403 permiso/CSRF/licencia (`FORBIDDEN`, `CSRF_INVALID`, `LICENSE_READ_ONLY`,
`LICENSE_RECOVERY_REQUIRED`, `FEATURE_DISABLED`), 404 recurso, 409 conflicto de
dominio, 412 versión, 413 tamaño, 422 clasificación, 429 límites con `Retry-After`,
503 disponibilidad operativa. Nunca paths temporales, claves ni stack traces.

## H2 — Identidad, cuenta y auditoría

En H2, credenciales/alta no se guardan para replay idempotente: username único
evita duplicados; ante respuesta perdida consultar la lista antes de repetir.
PATCH de usuario y PUT de roles usan If-Match. Los comandos documentales tendrán
su idempotencia persistente en H3/H4. `/libraries/:id/members/:user_id` requiere
la entidad biblioteca y está implementado en H3. `GET /system/status` (system.configure)
expone etapa, versión SQLite y estado de la política de licencia, sin rutas.
Listas aceptan `limit` y `cursor`; sesiones: user_id/close_reason/from/to,
intentos: outcome/from/to, eventos: event_type/actor_user_id/from/to. Fechas RFC3339.
GET /roles requiere permissions.manage_global. H2 solo devuelve eventos globales
(library_id NULL), para no abrir acceso a bibliotecas futuras.

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `POST /auth/login` | `{username,password}`; JSON y Origin válido | 200 `{user,session_expires_at,csrf_token}` + cookie; invalida sesión anterior atómicamente |
| `GET /auth/session` | Cookie | 200 `{user,session_expires_at,csrf_token}` o 401 estable; sondeo no renueva inactividad |
| `POST /auth/logout` | CSRF | 204; cierra sesión actual |
| `GET /account` | — | Perfil propio; nunca hash/token |
| `POST /account/password` | `{current_password,new_password}` + CSRF | 204; invalida todas las sesiones, incluida actual |
| `GET /account/sessions` | Cursor | Historial propio: UTC, IP observada, agente y motivo |
| `GET, POST /users` | Alta `{username,display_name,password}` | Lista/201; `users.manage` |
| `PATCH /users/:id` | `{display_name,disabled}` + versión | 200; `users.manage`; deshabilitar revoca sesiones |
| `POST /users/:id/password-reset` | `{new_password}` | 204; `users.reset_password`, revoca sesiones |
| `POST /users/:id/revoke-sessions` | `{reason}` | 204; `sessions.revoke` |
| `GET /sessions` | Filtros usuario/fecha/cierre | Historial organización; `sessions.read_all` |
| `GET /authentication-attempts` | Fecha/resultado, cursor | `authentication_attempts.read`, identificador saneado |
| `GET /audit-events` | Fecha/tipo/actor/biblioteca/recurso | Eventos filtrados a permisos; sin detalle sensible por defecto |
| `GET /audit-events/:id` | — | Evento y detalle de consulta solo con permisos adicionales |
| `GET /roles` | Ámbito | Plantillas de permisos visibles al administrador autorizado |
| `PUT /libraries/:id/members/:user_id` | `{role_ids}` + versión | `permissions.manage_library`; validar elevación y ámbito |
| `PUT /users/:id/global-roles` | `{role_ids}` + versión | `permissions.manage_global` |

Alta inicial se realiza por CLI local, no por `POST /users` anónimo. No hay API que
devuelva contraseñas, hashes, cookie original ni clave privada. Intentos fallidos
generan evento aunque no haya sesión. Un error de login no revela usuario existente.

## H3 — Bibliotecas, raíces, explorador y búsqueda

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `GET, POST /libraries` | Alta `{name,mode,manager_user_id}` + `Idempotency-Key` | Lista autorizada/201; alta `libraries.create` y `permissions.manage_global`; H3 solo `linked` |
| `GET /libraries/:id` | — | Nombre, modalidad, revisión, idiomas OCR y permisos efectivos |
| `PATCH /libraries/:id` | `{name,ocr_languages}` + versión | 204; `libraries.configure`; conversión a híbrida en H4 |
| `POST /storage/path-inspections` | `{library_id,storage_source,server_path}` | Resultado de accesibilidad/canonicalización; `storage.manage_roots` |
| `POST /libraries/:id/root-plans` | `{server_path,storage_source}` | Plan: `distinct/equal/descendant/ancestor/conflict`, alternativas permitidas |
| `POST /libraries/:id/roots` | `{plan_id,expected_configuration_revision}` | 202 raíz + escaneo; revalida plan/identidad/capacidad |
| `GET /libraries/:id/roots` | — | Raíces/salud/progreso; rutas solo con permiso |
| `PATCH /roots/:id` | `{enabled,reconcile_interval_seconds}` + versión | Configurar vigilancia, sin borrar documentos |
| `POST /roots/:id/retirement-plans` | — | Plan con revisión y recuento de referencias; expedientes en H4 |
| `POST /roots/:id/retire` | `{plan_id,reference_policy}` | 204; `retain_index` o `remove_index_references`; confirmación explícita de UI |
| `POST /libraries/:id/root-consolidations` | `{plan_id,expected_configuration_revision}` | 202 raíz y escaneo; consolidación de raíces propias, sin mover originales |
| `POST /libraries/:id/folder-views` | `{root_id,relative_prefix,name}` | 201 vista en biblioteca propietaria, sin archivos nuevos |
| `POST /libraries/:id/verify` | `{root_id?}` | 202 job; `indexing.run` |
| `GET /libraries/:id/explorer` | `root_id/view_id`, prefijo, filtros/cursor | Documentos autorizados; subcarpetas mediante `/folders`, no filesystem arbitrario |
| `POST /search` | Contrato inferior | Resultados por documento/página + evento con consulta exacta |
| `GET /documents/:id` | — | Ficha, disponibilidad/aprobación/OCR, revision, acciones permitidas |
| `GET /documents/:id/content` | `Range` opcional | PDF inline 200/206 y evento de apertura; 409 `DOCUMENT_UNAVAILABLE` si ausente |
| `GET /documents/:id/download` | — | PDF attachment + evento; `documents.download` |
| `GET /documents/:id/pages/:page_number/text` | — | Texto retenido/método/frescura; `documents.read` |
| `GET /documents/:id/history` | Cursor | Historia autorizada, sin secretos/ruta temporal |
| `GET /libraries/:id/duplicates` | — | Hasta 100 grupos por SHA-256 en la biblioteca autorizada, sin fusionar |
| `POST /documents/:id/remove-index` | `{reason}` + versión | Baja lógica/retirar FTS; `documents.remove_index` |
| `GET /notifications` | — | Hasta 100 avisos autorizados pendientes, más recientes primero |
| `POST /notifications/:id/acknowledge` | — | Acuse idempotente del usuario; no borra evento |
| `GET /jobs/:id` | — | Estado/progreso/diagnóstico saneado, ámbito autorizado |
| `POST /jobs/:id/retry` | `{reason}` | Nuevo job enlazado; `indexing.retry` |

Implementación H3: `POST /libraries` también exige `manager_user_id` explícito e
`Idempotency-Key`. `PATCH /libraries/:id` admite `name` y `ocr_languages`
(`spa`, `eng`, `spa+eng`); conversión de modalidad sigue en H4. El resultado de
inspección contiene un plan con `id`, `relation`, `related_root_ids`, `server_path`
y `expected_configuration_revision`. Confirmaciones repetidas del mismo plan
recuperan su raíz, con permisos actuales revalidados.

Endpoints adicionales H3:

- `GET /libraries/:id/folder-views`, `/folders`, `/members`, `/member-candidates?q=`.
- `PUT /libraries/:id/members/:user_id`: `{role_ids}`; lector, gestor o auditor.
- `GET /libraries/:id/jobs`, `/audit-events`: páginas de historial con cursor.
- `GET /search-events`: detalle exacto, filtrado antes de paginar por permisos en
  **todos** sus ámbitos; independiente de `audit.read_global`.

`/explorer` y `/folders` admiten `root_id`, `view_id`, `prefix` y cursor; prefijos
se comparan por componente. `POST /search` H3 admite tipo `name`, `ocr`, `general`
y filtros `root_id`, `view_id`, `prefix`, `availability`, `approval_status`.
Los campos de expedientes/clasificación/ejercicio aún no están disponibles.
`ocr` consulta tanto texto nativo como reconocido. Los resultados identifican
cada documento con `id` y ofrecen `original_filename`, `relative_path`, `page_count`
y `sha256`; `original_path` exige `storage.view_paths`. Las consultas exactas nunca
aparecen en el detalle global ordinario de auditoría.

`POST /documents/:id/remove-index` H3 recibe `{reason}` e `If-Match`; no hay
expediente que desvincular en esta etapa. `retire` devuelve 204 al completar la
transacción de metadatos, no una promesa de trabajo pendiente. Las lecturas y
escrituras documentales requieren el build de desarrollo hasta H6.

Conflictos de raíz: 409 `ROOT_ALREADY_REGISTERED`, `ROOT_IS_DESCENDANT`,
`ROOT_CONSOLIDATION_REQUIRED`, `ROOT_AUTHORIZATION_CONFLICT`, `ROOT_STORAGE_OVERLAP`,
`ROOT_PLAN_STALE`. Respuesta nunca revela biblioteca/ruta ajena solo por colisión.

```json
{
  "query": "Plano PR-008 \"estructura metálica\"",
  "search_type": "general",
  "library_ids": ["UUID"],
  "filters": {"root_id": "UUID", "availability": "missing"},
  "limit": 50,
  "cursor": null
}
```

Omitir `library_ids` significa todas las autorizadas. Solicitar explícitamente una
ajena da 403 sin detalle; no registrar que se consultó. Filtros H3: raíz/vista,
prefijo, disponibilidad y aprobación. Categoría, tipo, expediente/identificador,
ejercicio y origen se incorporarán con H4/H5. `query` se conserva exactamente, sin trim ni
normalización en auditoría; la compilación de búsqueda usa una copia. Rechazar
longitudes inválidas antes de ejecutar. `result_count` cuenta documentos distintos
totales de esa búsqueda, no filas de páginas ni tamaño de página; registrar también
cantidad devuelta y cursor en detalle cuando corresponda.

```json
{
  "items": [{
    "id": "UUID", "title": "Plano general",
    "availability": "missing", "approval_status": "draft",
    "extraction_freshness": "current",
    "matches": [{"page_number": 2, "segments": [
      {"text": "estructura metálica", "highlighted": true}
    ]}],
    "can_preview_original": false, "can_download": false
  }],
  "result_count": 1, "consulted_library_ids": ["UUID"]
}
```

El ejemplo omite metadatos adicionales de la ficha. `next_cursor` solo aparece si
hay otra página; el identificador de petición está en `X-Request-ID`.
`original_path` solo se agrega con permiso. Disponibilidad no se confunde con
frescura OCR. No hay enlace `file://`.
La página del visor es estado frontend; el endpoint entrega únicamente un PDF
autorizado, con Range. H3 registra cada petición de contenido; la agrupación de
varias peticiones Range de una misma apertura queda para el endurecimiento H7.

## H4/H5 — Cargas, clasificación, expedientes y revisión

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `POST /libraries/:id/upload-batches` | `{client_batch_id}` | 201 lote; `documents.upload` |
| `POST /upload-batches/:id/files` | Multipart `file` + `client_file_id` | 201 documento/item temporal, hash/estado; sin ruta privada |
| `GET /upload-batches/:id` | — | Progreso individual/lote para autor/revisor autorizado |
| `PATCH /documents/:id/classification` | `{case_id,category_id,document_type_id,title,metadata}` + versión | 200; tipo Varios exige título |
| `POST /documents/:id/associate` | `{case_id,category_id,document_type_id,title?}` + versión | Asocia libre, revisión inicial según política; no mueve ni extrae de nuevo |
| `POST /documents/:id/reassign` | `{case_id,category_id,document_type_id,reason}` + versión | `documents.reassign`; misma biblioteca, evento y ambos avances recalculados |
| `POST /documents/:id/submit` | `{reviewer_id?}` + versión | 201 solicitud; valida expediente/clasificación/política |
| `POST /documents/:id/finalize` | Versión | 202 materialización sin revisión habilitada; `documents.finalize` |
| `POST /documents/:id/cancel` | `{reason}` + versión | Cancelación trazable y retención configurada |
| `GET /reviews/pending` | Biblioteca/cursor | Solo responsabilidades del usuario |
| `POST /reviews/:id/approve` | `{expected_document_revision}` | 202 managed temporal; 200 linked/managed definitivo verificado; `documents.approve` |
| `POST /reviews/:id/reject` | `{expected_document_revision,reason}` | 200; motivo no vacío y `documents.reject` |
| `GET /materializations/:id` | — | Estado/errores y ruta final solo cuando sea definitiva y autorizada |
| `POST /materializations/:id/retry` | `{reason}` | 202; operación idempotente, permiso/capacidad revalidados |
| `GET, POST /libraries/:id/cases` | Alta `{identifier,template_version_id,exercise?}` | 201 expediente con requerimientos copiados; `cases.manage` |
| `GET, PATCH /cases/:id` | Patch `{identifier,exercise?}` + versión | Detalle/200; no renombra disco |
| `GET /cases/:id/requirements` | — | Cantidad requerida y conteos derivados por estado/avance |
| `PATCH /cases/:id/requirements/:requirement_id` | `{mandatory,required_count}` + versión | `requirements.edit`, solo ese expediente |
| `POST /cases/:id/initialize-requirements` | `{template_version_id}` | Inicialización idempotente de legado, no sobreescribir existentes |
| `GET, POST /libraries/:id/categories` | Alta `{name}` | Catálogo propio; escritura `catalogs.manage` |
| `PATCH /categories/:id` | `{name,archived}` + versión | No renombra rutas históricas |
| `GET, POST /libraries/:id/document-types` | `{category_id,name,allows_multiple,requires_descriptive_title}` | Tipos; `catalogs.manage` |
| `PATCH /document-types/:id` | Campos de tipo/archived + versión | Validar requisitos existentes, sin cambios masivos silenciosos |
| `GET, POST /libraries/:id/templates` | `{name,requirements:[...]}` | Plantilla/version inicial; `templates.manage` |
| `POST /templates/:id/versions` | `{requirements:[...]}` | Nueva versión; afecta solo expedientes nuevos |
| `POST /libraries/:id/storage-preview` | `{structure_pattern,filename_pattern,sample_fields}` | Vista previa segura; configuración autorizada |
| `PUT /libraries/:id/configuration` | Configuración de campos, destinos/políticas + versión | Nueva revisión; solo incorporaciones futuras |

Conflictos: `DOCUMENT_ALREADY_ASSOCIATED`, `CLASSIFICATION_REQUIRED`,
`DESCRIPTIVE_TITLE_REQUIRED`, `MULTIPLE_FILES_NOT_ALLOWED`, `REVIEW_REQUIRED`,
`SELF_APPROVAL_FORBIDDEN`, `STORAGE_UNAVAILABLE`, `STORAGE_INTEGRITY_FAILED`.
Subir archivos de un lote por requests separadas permite progreso/reintento por
archivo sin repetir los demás; el frontend sigue ofreciendo multifile/drag-and-drop.
No hay endpoint para editar manualmente «recibidos» ni propagación de plantilla
a existentes en la primera entrega del módulo.

## H6 — Administración del cliente de licencias

Todas requieren `license.manage`, CSRF en mutaciones y acceso durante recuperación.
El cliente traduce estos comandos locales al contrato externo exacto.

| Método/ruta relativa | Entrada | Respuesta |
| --- | --- | --- |
| `GET /license` | — | Estado derivado, módulos, fechas, revisión, diagnóstico saneado |
| `POST /license/activate` | `{license_key}` | Resultado online; no persistir clave comercial en logs |
| `POST /license/refresh` | — | Resultado online/idempotente, mantiene válida ante fallo de red |
| `POST /license/deactivate` | Confirmación explícita | Resultado remoto; solo confirmación libera plaza |
| `POST /license/offline-requests` | `{action:activate\|renew\|deactivate}` | Descarga `.licreq` firmado, registro trazable |
| `POST /license/import` | Multipart `.lic` | Validación JWS/importación transaccional de revisión máxima |

Ni PDF, texto OCR, rutas ni DB se incluyen en el HTTP externo. No añadir endpoint
remoto de `hybrid`, usuarios, ventas, biblioteca o backup.

## H7 — Operaciones

`GET/PATCH /system/configuration` para escucha/proxy/recursos (sin devolver claves),
`POST /backups` y `GET /backups/:id` con `backup.manage`; planifica snapshot consistente
y reporta manifest/diagnóstico por ID. Exportación administrativa por endpoint
autorizado conserva recuperación en modo read-only. Restauración se inicia localmente
con el servicio en mantenimiento y plan verificado; no exponer reemplazo inmediato
de DB por HTTP. Contrato fino de exportación/configuración se cierra al implementar
H7, sin introducir otro motor ni almacenamiento en nube obligatorio.
