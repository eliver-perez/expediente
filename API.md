# AIBID — API HTTP interna

Base: `/api/v1`, mismo origen que React. **H2–H5 implementados:** identidad, cuenta,
usuarios, permisos, sesiones, auditoría, bibliotecas, búsqueda, cargas y expedientes.
H6–H7 siguen como contratos de diseño. No confundir esta API con los cuatro endpoints HTTPS externos
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
PATCH de usuario y PUT de roles usan If-Match. Altas de biblioteca y cargas usan claves persistentes de idempotencia. `/libraries/:id/members/:user_id` requiere
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
| `PUT /libraries/:id/members/:user_id` | `{role_ids}` | `permissions.manage_library`; validar elevación y ámbito |
| `PUT /users/:id/global-roles` | `{role_ids}` + versión | `permissions.manage_global` |

Alta inicial se realiza por CLI local, no por `POST /users` anónimo. No hay API que
devuelva contraseñas, hashes, cookie original ni clave privada. Intentos fallidos
generan evento aunque no haya sesión. Un error de login no revela usuario existente.

## H3 — Bibliotecas, raíces, explorador y búsqueda

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `GET, POST /libraries` | Alta `{name,mode,manager_user_id}` + `Idempotency-Key` | Lista autorizada/201; alta `libraries.create` y `permissions.manage_global`; `linked`, `managed` o `hybrid` |
| `GET /libraries/:id` | — | Nombre, modalidad, revisión, idiomas OCR, settings y permisos efectivos |
| `PATCH /libraries/:id` | `{name,ocr_languages,mode?,settings?}` + versión | 204; `libraries.configure`; ampliación a híbrida |
| `POST /storage/path-inspections` | `{library_id,storage_source,server_path}` | Resultado de accesibilidad/canonicalización; `storage.manage_roots` |
| `POST /libraries/:id/root-plans` | `{server_path,storage_source}` | Plan: `distinct/equal/descendant/ancestor/conflict`, alternativas permitidas |
| `POST /libraries/:id/roots` | `{plan_id,expected_configuration_revision}` | 202 raíz; escaneo solo linked; revalida plan/identidad/capacidad |
| `GET /libraries/:id/roots` | — | Raíces/salud/progreso; rutas solo con permiso |
| `PATCH /roots/:id` | `{enabled,reconcile_interval_seconds}` + versión | Configurar vigilancia, sin borrar documentos |
| `POST /roots/:id/retirement-plans` | — | Plan con revisión, referencias y `affected_case_count`; invalida si cambian asociaciones |
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
| `POST /documents/:id/remove-index` | `{reason,expected_case_id?}` + versión | Baja lógica/retirar FTS; `documents.remove_index` |
| `GET /notifications` | — | Hasta 100 avisos autorizados pendientes, más recientes primero |
| `POST /notifications/:id/acknowledge` | — | Acuse idempotente del usuario; no borra evento |
| `GET /jobs/:id` | — | Estado/progreso/diagnóstico saneado, ámbito autorizado |
| `POST /jobs/:id/retry` | `{reason}` | Nuevo job enlazado; `indexing.retry` |

Implementación H3: `POST /libraries` también exige `manager_user_id` explícito e
`Idempotency-Key`. `PATCH /libraries/:id` admite `name` y `ocr_languages`
(`spa`, `eng`, `spa+eng`), `mode` y `settings` en H4. El resultado de
inspección contiene un plan con `id`, `relation`, `related_root_ids`, `server_path`
y `expected_configuration_revision`. Confirmaciones repetidas del mismo plan
recuperan su raíz, con permisos actuales revalidados.

Endpoints adicionales H3:

- `GET /libraries/:id/folder-views`, `/folders`, `/members`, `/member-candidates?q=`.
- `PUT /libraries/:id/members/:user_id`: `{role_ids}`; lector, gestor, auditor, colaborador o revisor.
- `GET /libraries/:id/jobs`, `/audit-events`: páginas de historial con cursor.
- `GET /search-events`: detalle exacto, filtrado antes de paginar por permisos en
  **todos** sus ámbitos; independiente de `audit.read_global`.

`/explorer` y `/folders` admiten `root_id`, `view_id`, `prefix` y cursor; prefijos
se comparan por componente. `POST /search` admite tipo `name`, `ocr`, `general`, `identifier`
y filtros `root_id`, `view_id`, `prefix`, `availability`, `approval_status`.
H4 añade `case_id`, `category_id`, `document_type_id`, `exercise`,
`storage_source` y `unassigned`.
`ocr` consulta tanto texto nativo como reconocido. Los resultados identifican
cada documento con `id` y ofrecen `original_filename`, `relative_path`, `page_count`
y `sha256`; `original_path` exige `storage.view_paths`. Las consultas exactas nunca
aparecen en el detalle global ordinario de auditoría.

`POST /documents/:id/remove-index` recibe `{reason,expected_case_id?}` e `If-Match`;
si está asociado exige el ID actual del expediente como confirmación explícita. `retire` devuelve 204 al completar la
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
prefijo, disponibilidad y aprobación. H4 incorpora categoría, tipo, expediente, ejercicio y origen.
`identifier` busca el identificador; `general` incluye esa coincidencia además de
nombre/contenido. `query` se conserva exactamente, sin trim ni
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

## H4 — Cargas, clasificación y expedientes (implementado)

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `GET, POST /libraries/:id/upload-batches` | Alta `{client_batch_id}`; lista con cursor | 201 `{id}`; lista propia; `documents.upload` |
| `POST /upload-batches/:id/files` | Multipart: primero `client_file_id`, después un `file` PDF | 201 item/documento/estado/tamaño; autor del lote |
| `GET /upload-batches/:id` | — | Items del lote; autor/revisor autorizado |
| `PATCH /documents/:id/classification` | `{case_id,category_id,document_type_id,title,metadata}` + versión | 204; borrador/rechazado; `documents.classify` |
| `POST /documents/:id/associate` | Clasificación + versión | 204; linked libre; `documents.associate`; sin mover ni extraer |
| `POST /documents/:id/reassign` | Clasificación y `reason` + versión | 204; misma biblioteca; `documents.reassign`; evento con ambos expedientes |
| `POST /documents/:id/cancel` | `{reason}` + versión | 204; temporal propio draft/rejected; `documents.cancel_own`; conserva bytes |
| `GET, POST /libraries/:id/cases` | Alta `{identifier,template_version_id?,exercise?}`; lista `q/cursor` | Lista autorizada/201 `{id}`; escritura `cases.manage` |
| `GET, PATCH /cases/:id` | Patch `{identifier,exercise?}` + versión | Detalle/200 `{id}`; no renombra disco |
| `GET /cases/:id/requirements` | — | Items con conteos derivados, `progress_percent`, `initialized` |
| `PATCH /cases/:id/requirements/:requirement_id` | `{mandatory,required_count}` + versión | 204; `requirements.edit`, solo ese expediente |
| `POST /cases/:id/initialize-requirements` | `{template_version_id}` | 204; inicialización única/idempotente, sin sobrescribir |
| `GET, POST /libraries/:id/categories` | Alta `{name}` | `{items}`/201 `{id}`; escritura `catalogs.manage` |
| `PATCH /categories/:id` | `{name,archived}` + versión | 200 `{id}`; no cambia rutas históricas |
| `GET, POST /libraries/:id/document-types` | `{category_id,name,allows_multiple,requires_descriptive_title}` | `{items}`/201 `{id}`; `catalogs.manage` |
| `PATCH /document-types/:id` | Campos de tipo/archived + versión | 200 `{id}`; categoría fija; snapshots existentes conservados |
| `GET, POST /libraries/:id/templates` | `{name,requirements:[...]}` | `{items}` con versiones/201 `{version_id}`; `templates.manage` |
| `POST /templates/:id/versions` | `{requirements:[...]}` + versión vigente | 201 `{version_id}`; nueva versión inmutable |
| `POST /libraries/:id/naming-preview` | `{identifier,exercise,category,document_type,original_filename}` | 200 ruta relativa de configuración guardada; `libraries.configure` |

Configuración en `PATCH /libraries/:id`, mediante `settings`:
`identifier_label`, `exercise_enabled`, `cases_enabled`, `structure_pattern`,
`filename_pattern`, `filename_prefix`, `default_managed_root_id`. La conversión
solo amplía a híbrida. No se implementan las antiguas propuestas de rutas
`/storage-preview` o `PUT /configuration`; las operaciones reales son las anteriores.

Requerimiento de plantilla: `{category_id,document_type_id,required_count,mandatory}`.
La respuesta incluye `allows_multiple` del snapshot. Tipos únicos exigen cantidad
uno; cada versión admite hasta 200 requisitos; cantidades entre 1 y 1000.
Catálogos/plantillas devuelven sus items completos; expedientes y lotes usan páginas
de 50. No existe edición manual de recibidos ni propagación masiva de plantilla.

Clave de lote de 16–100 bytes por autor/biblioteca; clave de archivo por lote.
Repetir clave/nombre/bytes recupera documento; cambiar contenido produce 409
`IDEMPOTENCY_CONFLICT`. Un intento fallido exige otra clave (`UPLOAD_RETRY_KEY`).
Hasta 100 intentos por lote, dos recepciones simultáneas (`UPLOAD_BUSY`), máximo
`indexing.maximum_file_mb` por PDF (`FILE_SIZE_LIMIT` 413), falta de espacio 507.
Transferencia incompleta o PDF inválido no publica documento. Partes extra se
rechazan antes de commit. Recepción con límite de cinco minutos y margen adicional
para validación PDF/respuesta. [Almacenamiento y recuperación](docs/H4.md).

Temporales: ficha/bytes/texto/historial/búsqueda/jobs/conteos solo para autor/revisor;
DTO sin ruta/localizador, incluso con permiso de rutas. Conflictos relevantes:
`DOCUMENT_ALREADY_ASSOCIATED`, `DOCUMENT_TYPE_OCCUPIED`, `CASE_IDENTIFIER_EXISTS`,
`CASE_CONFIRMATION_REQUIRED`, `VERSION_CONFLICT`. Título/clasificación inválidos
responden `INVALID_REQUEST` 422. Ninguna asociación permite cruzar bibliotecas.

## H5 — Revisión y materialización (implementado)

| Método/ruta relativa | Entrada | Respuesta / autorización |
| --- | --- | --- |
| `GET /documents/:id/workflow` | — | Política, permisos de acciones, revisión, operación y `processing`; misma privacidad documental |
| `POST /documents/:id/submit` | `{reviewer_id}` (vacío: cola compartida), `If-Match` | 201 `{review_id}`; `documents.submit` |
| `POST /documents/:id/finalize` | `If-Match` | 202 `{operation_id}` para temporal; 200 ID vacío si ya definitivo/linked; `documents.finalize` |
| `GET /libraries/:id/reviews` | `cursor` | Pendientes asignados al usuario/cola compartida, páginas de 50; `documents.review` |
| `GET /reviews/pending` | `library_id` obligatorio, `cursor` | Alias de la bandeja por biblioteca |
| `GET /libraries/:id/reviewers` | — | Revisores activos de esa biblioteca; `documents.read` |
| `POST /reviews/:id/approve` | `{expected_document_revision}` | 202/200 `{operation_id}`; `documents.approve` |
| `POST /reviews/:id/reject` | `{expected_document_revision,reason}` | 200; `documents.reject`, motivo de hasta 1000 caracteres |
| `GET /materializations/:id` | — | Estado/diagnóstico/configuración por ID; nunca rutas privadas |
| `POST /materializations/:id/retry` | `{reason}` | 202 `{job_id}`; gestor con `indexing.retry` o revisor que autorizó con `documents.approve` |

`settings` incorpora `review_managed`, `review_linked` (false por defecto) y
`retention_days` (0 conserva; máximo 3650). `Document` incluye `integrity_status`.
`POST /libraries/:id/verify` verifica ahora ambos orígenes; las tareas distinguen
`scan`, `extract`, `verify_managed` y `materialize`. El reintento general de una
tarea de materialización delega al mismo protocolo y no cambia su ruta.

Se comprueban sesión/CSRF, biblioteca, privacidad, asignación y separación de
responsabilidades D-04. Una clasificación nueva invalida la solicitud pendiente.
La aprobación asíncrona se confirma solo después de verificar el destino y volver
a autorizar al actor. El temporal se limpia después; `approved` no equivale a
`cleaned`. Repetir una petición con revisión original conserva la operación si
no hay cambios posteriores; una revisión obsoleta recibe 412.

Errores relevantes: `CLASSIFICATION_REQUIRED`, `REVIEW_REQUIRED`,
`REVIEW_NOT_REQUIRED`, `REVIEW_POLICY_CHANGED`, `SELF_APPROVAL_FORBIDDEN`,
`REVIEW_ALREADY_DECIDED`, `CONTENT_CHANGED`, `MANAGED_ROOT_REQUIRED` y diagnósticos
`STORAGE_*` en la operación. Las rutas definitivas se consultan en la ficha con
`storage.view_paths`. [Recuperación y límites H5](docs/H5.md).

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
