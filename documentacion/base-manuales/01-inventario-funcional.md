# AIBID — Inventario funcional

**Fecha de elaboración: 2026-10-09.** Actualización de la base elaborada el 8 de octubre; nombres y procedimientos contrastados con la transición de producción. Base de trabajo para los futuros manuales; no es un manual final.

## Criterios de evidencia y alcance

- **Identificado en código:** rutas, campos, reglas y llamadas inspeccionadas. No se ejecutó el procedimiento durante esta investigación.
- **Verificado en pruebas:** un escenario concreto tiene resultado PASS en informes previos del proyecto; el nombre y su fuente se citan por funcionalidad. No implica haber observado todos los pasos en la interfaz o certificado el sistema operativo instalado.
- **Verificado en interfaz:** reservado para observación directa de la aplicación actual. **Ninguna funcionalidad recibe aquí esta clasificación.** Los PNG históricos son evidencia visual parcial de ejecuciones anteriores.
- **Pendiente de validar:** falta ejecutar/observar una combinación, plataforma o escenario; se registra en [pendientes](07-pendientes-validacion.md).

La investigación original fue de lectura. Esta actualización añade compilación, pruebas aisladas y ejecución de un binario extraído con estado ficticio; no se instalaron servicios ni se abrieron bases reales. No se usaron documentos de clientes ni claves comerciales pegadas en conversaciones anteriores.

Como evidencia histórica, la revisión previa de desarrollo está en [dist/aibid-2.0-rc.1/VALIDACION-RC1.json](../../dist/aibid-2.0-rc.1/VALIDACION-RC1.json) y [docs/QA-V2-RC1.md](../../docs/QA-V2-RC1.md). Las rutas de artefactos conservan sus nombres internos; **los títulos documentales usan AIBID**. Informes JSONL de QA del 7 de octubre se leyeron en `/private/tmp/aibid-rc1-qa-final-go.jsonl`, `/private/tmp/aibid-rc1-qa-final-race.jsonl` y `/private/tmp/aibid-rc1-qa-production.jsonl`; son archivos temporales y no se debe depender de su permanencia para editar futuros manuales. La evidencia estable de resumen está enlazada arriba. No se presenta esa evidencia histórica como certificación de los paquetes nuevos; resultados actuales en [informe de transición](../../docs/TRANSICION-PRODUCCION.md).

## Lectura del inventario

127 funcionalidades. Cada entrada contiene procedimiento, campos, validación, permisos, resultado, peculiaridades, pantalla/captura y fuente. Los pasos son instrucciones derivadas de implementación, **no acciones ejecutadas aquí**. Para escrituras se suma autorización de sesión, CSRF, licencia/módulos, ámbito y estado, incluso cuando la UI ofrece el botón. Los límites UI son los publicados por el formulario; algunas validaciones backend usan bytes o runes: véase [VAL-021](07-pendientes-validacion.md#val-021).

Los roles completos están en [pantallas y permisos](02-pantallas-y-permisos.md), la API y persistencia en [inventario técnico](04-inventario-tecnico.md), y las plataformas en [instalación y operación](03-instalacion-y-operacion.md).

## Índice por módulo

| Módulo | Funcionalidades |
| --- | --- |
| Identidad | [FUN-001](#fun-001), [FUN-002](#fun-002), [FUN-003](#fun-003), [FUN-004](#fun-004), [FUN-005](#fun-005), [FUN-006](#fun-006), [FUN-007](#fun-007), [FUN-008](#fun-008), [FUN-009](#fun-009), [FUN-010](#fun-010), [FUN-011](#fun-011) |
| Seguridad | [FUN-012](#fun-012) |
| Bibliotecas | [FUN-013](#fun-013), [FUN-014](#fun-014), [FUN-015](#fun-015), [FUN-017](#fun-017), [FUN-018](#fun-018), [FUN-019](#fun-019), [FUN-020](#fun-020), [FUN-021](#fun-021), [FUN-072](#fun-072) |
| Permisos | [FUN-016](#fun-016) |
| Almacenamiento | [FUN-022](#fun-022), [FUN-023](#fun-023), [FUN-024](#fun-024), [FUN-025](#fun-025), [FUN-026](#fun-026), [FUN-027](#fun-027), [FUN-028](#fun-028) |
| Documentos | [FUN-029](#fun-029), [FUN-030](#fun-030), [FUN-031](#fun-031), [FUN-032](#fun-032), [FUN-033](#fun-033), [FUN-037](#fun-037) |
| Extracción | [FUN-034](#fun-034), [FUN-080](#fun-080), [FUN-081](#fun-081) |
| Previews | [FUN-035](#fun-035) |
| Procesamiento | [FUN-036](#fun-036), [FUN-073](#fun-073), [FUN-074](#fun-074), [FUN-075](#fun-075), [FUN-076](#fun-076), [FUN-077](#fun-077), [FUN-078](#fun-078), [FUN-079](#fun-079), [FUN-082](#fun-082), [FUN-083](#fun-083) |
| Organización | [FUN-038](#fun-038), [FUN-039](#fun-039), [FUN-040](#fun-040), [FUN-041](#fun-041), [FUN-042](#fun-042), [FUN-043](#fun-043), [FUN-044](#fun-044), [FUN-045](#fun-045), [FUN-046](#fun-046), [FUN-051](#fun-051), [FUN-052](#fun-052), [FUN-053](#fun-053) |
| Cargas | [FUN-047](#fun-047), [FUN-048](#fun-048), [FUN-049](#fun-049), [FUN-050](#fun-050), [FUN-054](#fun-054) |
| Revisión | [FUN-055](#fun-055), [FUN-056](#fun-056), [FUN-057](#fun-057), [FUN-058](#fun-058), [FUN-059](#fun-059), [FUN-060](#fun-060) |
| Búsqueda | [FUN-061](#fun-061), [FUN-062](#fun-062), [FUN-063](#fun-063), [FUN-064](#fun-064), [FUN-065](#fun-065) |
| Duplicados | [FUN-066](#fun-066) |
| Auditoría | [FUN-067](#fun-067), [FUN-068](#fun-068), [FUN-069](#fun-069), [FUN-070](#fun-070), [FUN-071](#fun-071) |
| Configuración | [FUN-084](#fun-084), [FUN-085](#fun-085), [FUN-086](#fun-086), [FUN-087](#fun-087), [FUN-088](#fun-088), [FUN-089](#fun-089), [FUN-090](#fun-090), [FUN-091](#fun-091), [FUN-092](#fun-092), [FUN-093](#fun-093), [FUN-094](#fun-094), [FUN-095](#fun-095) |
| Red | [FUN-096](#fun-096), [FUN-097](#fun-097) |
| Observabilidad | [FUN-098](#fun-098), [FUN-099](#fun-099), [FUN-100](#fun-100), [FUN-101](#fun-101), [FUN-102](#fun-102) |
| Licenciamiento | [FUN-103](#fun-103), [FUN-104](#fun-104), [FUN-105](#fun-105), [FUN-106](#fun-106), [FUN-107](#fun-107), [FUN-108](#fun-108), [FUN-109](#fun-109) |
| Instalación | [FUN-110](#fun-110), [FUN-111](#fun-111), [FUN-112](#fun-112), [FUN-113](#fun-113), [FUN-114](#fun-114), [FUN-115](#fun-115), [FUN-116](#fun-116), [FUN-117](#fun-117), [FUN-118](#fun-118), [FUN-119](#fun-119), [FUN-120](#fun-120), [FUN-121](#fun-121) |
| Operación | [FUN-122](#fun-122), [FUN-123](#fun-123), [FUN-124](#fun-124), [FUN-125](#fun-125), [FUN-126](#fun-126), [FUN-127](#fun-127) |

<a id="fun-001"></a>

## FUN-001 — Iniciar sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Iniciar sesión |
| Objetivo y resultado esperado | Se crea una sesión autenticada; cualquier sesión anterior de esa cuenta queda cerrada. |
| Descripción y comportamiento especial | Una sola sesión abierta por usuario. El primer acceso administrativo se crea en consola, no desde este formulario. |
| Ubicación | [PAN-001](02-pantallas-y-permisos.md#pan-001) — /login |
| Roles / permisos requeridos | Cuenta habilitada; sin rol previo |
| Condiciones previas | Cuenta creada; navegador admite cookies del origen. |
| Campos involucrados | Usuario; Contraseña |
| Validaciones | Usuario normalizado a minúsculas; contraseña 6–128 caracteres para alta/cambio; login rechaza contraseña incorrecta sin revelar existencia de cuenta. |
| Acciones disponibles | Iniciar sesión |
| Estados relacionados | Sesión activa / reemplazada |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-001](05-catalogo-capturas.md#cap-001) |
| Implementación | [web/src/App.tsx](../../web/src/App.tsx); [internal/identity/login.go](../../internal/identity/login.go); [internal/httpapi/identity_handlers.go](../../internal/httpapi/identity_handlers.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestConcurrentLoginsReplaceAtomically`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir la URL de AIBID.
2. Escribir Usuario y Contraseña.
3. Pulsar Iniciar sesión.
4. Confirmar que aparece Mi cuenta y el menú permitido.

<a id="fun-002"></a>

## FUN-002 — Reconocer acceso incorrecto y limitación de intentos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Reconocer acceso incorrecto y limitación de intentos |
| Objetivo y resultado esperado | El acceso fallido queda registrado sin crear sesión. |
| Descripción y comportamiento especial | No hay enlace funcional para recuperar contraseña por correo. Los límites se persisten y sobreviven al reinicio. |
| Ubicación | [PAN-001](02-pantallas-y-permisos.md#pan-001) — /login |
| Roles / permisos requeridos | Público |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Usuario; Contraseña |
| Validaciones | Ventana predeterminada 15 min, límite por identificador 5 y por IP 30; configuración admite otros límites; 429 entrega Retry-After. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Fallido / limitado |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-002](05-catalogo-capturas.md#cap-002) |
| Implementación | [web/src/App.tsx](../../web/src/App.tsx); [internal/identity/login.go](../../internal/identity/login.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestFailedLoginPersistsWithoutSessionOrSecretsAndRateLimitSurvivesRestart`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Intentar iniciar sesión.
2. Leer el aviso del formulario.
3. Si hay límite de intentos, esperar y reintentar.
4. Si se olvidó la contraseña, contactar al administrador.

<a id="fun-003"></a>

## FUN-003 — Cerrar sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Cerrar sesión |
| Objetivo y resultado esperado | Cierra la sesión en servidor y elimina la cookie. |
| Descripción y comportamiento especial | Cerrar una pestaña no sustituye Cerrar sesión. |
| Ubicación | [PAN-002](02-pantallas-y-permisos.md#pan-002) — /account |
| Roles / permisos requeridos | Sesión activa |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin campos |
| Validaciones | Solicita origen y CSRF válidos; sesión anterior ya cerrada no recupera acceso. |
| Acciones disponibles | Cerrar sesión en la cabecera |
| Estados relacionados | Cerrada |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-003](05-catalogo-capturas.md#cap-003) |
| Implementación | [web/src/Account.tsx](../../web/src/Account.tsx); [internal/httpapi/identity_handlers.go](../../internal/httpapi/identity_handlers.go); [internal/identity/service.go](../../internal/identity/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestHTTPReplacementCookiesCSRFAndPasswordChange`, fuente [internal/httpapi/server_test.go](../../internal/httpapi/server_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Pulsar Cerrar sesión en la cabecera.
2. Comprobar que vuelve Inicio de sesión.

<a id="fun-004"></a>

## FUN-004 — Cambiar la contraseña propia

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Cambiar la contraseña propia |
| Objetivo y resultado esperado | Se cambia el hash y se cierran las sesiones de la cuenta, incluida la actual. |
| Descripción y comportamiento especial | No publicar ejemplos que parezcan credenciales utilizables. Restablecimiento administrativo es otra operación. |
| Ubicación | [PAN-002](02-pantallas-y-permisos.md#pan-002) — /account |
| Roles / permisos requeridos | Sesión activa |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Contraseña actual; Nueva contraseña; Repetir nueva contraseña |
| Validaciones | Nueva contraseña 6–128 caracteres; no exige mayúsculas/números/símbolos; coincidencia de confirmación comprobada en UI. |
| Acciones disponibles | Actualizar contraseña |
| Estados relacionados | Sesiones cerradas por cambio |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-004](05-catalogo-capturas.md#cap-004) |
| Implementación | [web/src/Account.tsx](../../web/src/Account.tsx); [internal/identity/password.go](../../internal/identity/password.go); [internal/identity/password_change.go](../../internal/identity/password_change.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPasswordChangeAndResetInvalidateSessions`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Mi cuenta.
2. Escribir contraseña actual.
3. Capturar nueva contraseña dos veces.
4. Pulsar Actualizar contraseña.
5. Volver a iniciar sesión con la nueva contraseña.

<a id="fun-005"></a>

## FUN-005 — Consultar Mi cuenta y Mis accesos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Consultar Mi cuenta y Mis accesos |
| Objetivo y resultado esperado | Muestra sesiones propias, fechas, actividad, IP, navegador y motivo de cierre. |
| Descripción y comportamiento especial | IP y navegador son datos de soporte; ocultarlos en capturas definitivas. |
| Ubicación | [PAN-002](02-pantallas-y-permisos.md#pan-002) — /account |
| Roles / permisos requeridos | Sesión propia |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Paginación de accesos |
| Validaciones | El endpoint de cuenta no permite consultar otra identidad por un parámetro libre. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Activa / cerrada |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-005](05-catalogo-capturas.md#cap-005) |
| Implementación | [web/src/Account.tsx](../../web/src/Account.tsx); [internal/httpapi/identity_handlers.go](../../internal/httpapi/identity_handlers.go); [internal/identity/service.go](../../internal/identity/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAuditRolePaginationAndImmutableEvents`, fuente [internal/httpapi/server_test.go](../../internal/httpapi/server_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Mi cuenta.
2. Revisar datos y fecha de alta.
3. Leer Mis accesos.
4. Usar paginación cuando haya más registros.

<a id="fun-006"></a>

## FUN-006 — Comprender vencimiento y reemplazo de sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Comprender vencimiento y reemplazo de sesión |
| Objetivo y resultado esperado | La expiración o reemplazo vuelve a exigir autenticación. |
| Descripción y comportamiento especial | App consulta sesión cada 30 s y al recuperar foco sin prolongar inactividad; actividad humana actualiza cuenta con límite de frecuencia. |
| Ubicación | [PAN-002](02-pantallas-y-permisos.md#pan-002) — /account |
| Roles / permisos requeridos | Sesión activa |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin formulario; tiempos en config.json |
| Validaciones | Por defecto inactividad 30 min y duración absoluta 12 h; no confundir sondeo con actividad humana. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Expirada / reemplazada |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-006](05-catalogo-capturas.md#cap-006) |
| Implementación | [web/src/Account.tsx](../../web/src/Account.tsx); [internal/identity/service.go](../../internal/identity/service.go); [web/src/App.tsx](../../web/src/App.tsx) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPollingDoesNotExtendIdleSessionAndExpiryIsRecorded`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-014](07-pendientes-validacion.md#val-014) |

**Procedimiento:**

1. Usar normalmente la aplicación.
2. Ante sesión terminada, leer el aviso.
3. Volver a iniciar sesión.
4. Si ocurre por reemplazo, comprobar que no hay otro acceso con la misma cuenta.

<a id="fun-007"></a>

## FUN-007 — Crear usuario

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Crear usuario |
| Objetivo y resultado esperado | Se crea cuenta habilitada; los roles no se conceden automáticamente por existir. |
| Descripción y comportamiento especial | El aviso tras crear invita a asignar permisos desde Editar. No hay registro público. |
| Ubicación | [PAN-004](02-pantallas-y-permisos.md#pan-004) — Usuarios → formulario en línea |
| Roles / permisos requeridos | users.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre visible; Nombre de usuario; Contraseña inicial |
| Validaciones | Nombre visible máximo 120; usuario 3–80, ASCII minúsculas/dígitos y . _ @ - con primer carácter alfanumérico; nombre único; contraseña 6–128. |
| Acciones disponibles | Nuevo usuario; Crear cuenta |
| Estados relacionados | Habilitado |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-007](05-catalogo-capturas.md#cap-007) |
| Implementación | [web/src/Users.tsx](../../web/src/Users.tsx); [internal/identity/users.go](../../internal/identity/users.go); [internal/identity/service.go](../../internal/identity/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestBootstrapLastAdministratorAndRevisionGuards`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-021](07-pendientes-validacion.md#val-021) |

**Procedimiento:**

1. Ir a Usuarios.
2. Pulsar Nuevo usuario.
3. Completar los tres campos.
4. Pulsar Crear cuenta.
5. Abrir Editar si necesita asignar permisos.

<a id="fun-008"></a>

## FUN-008 — Editar nombre y deshabilitar cuenta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Editar nombre y deshabilitar cuenta |
| Objetivo y resultado esperado | Guarda datos con revisión y cierra sesiones al deshabilitar. |
| Descripción y comportamiento especial | Ante VERSION_CONFLICT actualizar página y revisar cambios ajenos antes de guardar. |
| Ubicación | [PAN-004](02-pantallas-y-permisos.md#pan-004) — Usuarios → formulario en línea |
| Roles / permisos requeridos | users.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre visible; Deshabilitar cuenta y cerrar sesiones |
| Validaciones | Revisión vigente; no puede dejarse la instalación sin al menos un administrador habilitado. |
| Acciones disponibles | Editar en la cuenta; Guardar datos |
| Estados relacionados | Habilitado / deshabilitado |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-008](05-catalogo-capturas.md#cap-008) |
| Implementación | [web/src/Users.tsx](../../web/src/Users.tsx); [internal/identity/users.go](../../internal/identity/users.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestBootstrapLastAdministratorAndRevisionGuards`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Usuarios.
2. Pulsar Editar en la cuenta.
3. Cambiar nombre o marcar deshabilitación.
4. Pulsar Guardar datos.
5. Comprobar estado de la cuenta.

<a id="fun-009"></a>

## FUN-009 — Asignar perfiles globales

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Asignar perfiles globales |
| Objetivo y resultado esperado | Permisos globales actualizados y auditados. |
| Descripción y comportamiento especial | Administración global no concede lectura de todas las bibliotecas. Perfiles combinables; no hay editor UI de roles personalizados. |
| Ubicación | [PAN-004](02-pantallas-y-permisos.md#pan-004) — Usuarios → formulario en línea |
| Roles / permisos requeridos | permissions.manage_global |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Administrador de instalación; Auditor de accesos |
| Validaciones | Solo roles globales definidos; control de revisión y último administrador. |
| Acciones disponibles | Guardar permisos |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-009](05-catalogo-capturas.md#cap-009) |
| Implementación | [web/src/Users.tsx](../../web/src/Users.tsx); [internal/identity/users.go](../../internal/identity/users.go); [db/migrations/0001_identity.up.sql](../../db/migrations/0001_identity.up.sql) |
| Verificación | Verificado en pruebas (escenario parcial): `TestEveryWriteRechecksPermissionsAndSession`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Editar en Usuarios.
2. Seleccionar perfiles globales.
3. Pulsar Guardar permisos.
4. Revisar menús visibles tras actualizar sesión.

<a id="fun-010"></a>

## FUN-010 — Restablecer contraseña de otro usuario

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Restablecer contraseña de otro usuario |
| Objetivo y resultado esperado | Cambia credencial y revoca sesiones existentes. |
| Descripción y comportamiento especial | No existe bandera implementada de cambio obligatorio en primer acceso; no prometerla en el manual. |
| Ubicación | [PAN-004](02-pantallas-y-permisos.md#pan-004) — Usuarios → formulario en línea |
| Roles / permisos requeridos | users.reset_password |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nueva contraseña |
| Validaciones | 6–128 caracteres; autorización revalidada dentro de la transacción. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sesiones invalidadas |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-010](05-catalogo-capturas.md#cap-010) |
| Implementación | [web/src/Users.tsx](../../web/src/Users.tsx); [internal/identity/password_change.go](../../internal/identity/password_change.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestConcurrentResetNeverLeavesOldCredentialSessionActive`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Usuarios.
2. Editar la cuenta.
3. Capturar contraseña en restablecimiento.
4. Enviar la acción de restablecer.
5. Comunicar al usuario por un canal adecuado que debe iniciar otra vez.

<a id="fun-011"></a>

## FUN-011 — Revocar sesiones de una cuenta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Identidad — Revocar sesiones de una cuenta |
| Objetivo y resultado esperado | Sesiones abiertas cerradas con motivo auditado. |
| Descripción y comportamiento especial | La persona puede iniciar una sesión nueva con sus mismas credenciales. |
| Ubicación | [PAN-003](02-pantallas-y-permisos.md#pan-003) — /admin/users |
| Roles / permisos requeridos | sessions.revoke |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Confirmación del navegador |
| Validaciones | Revalida sesión y permiso; no deshabilita usuario. |
| Acciones disponibles | Cerrar sesiones de este usuario |
| Estados relacionados | Cerrada por revocación |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-011](05-catalogo-capturas.md#cap-011) |
| Implementación | [web/src/Users.tsx](../../web/src/Users.tsx); [internal/identity/users.go](../../internal/identity/users.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPasswordChangeAndResetInvalidateSessions`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En Usuarios localizar cuenta.
2. Pulsar Cerrar sesiones de este usuario.
3. Confirmar el diálogo.
4. Revisar sesiones en Accesos si se dispone del permiso.

<a id="fun-012"></a>

## FUN-012 — Consultar sesiones e intentos de acceso

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Seguridad — Consultar sesiones e intentos de acceso |
| Objetivo y resultado esperado | Se consulta el historial con alcance administrativo y paginación. |
| Descripción y comportamiento especial | Un rol que solo tuviera uno de los permisos podría usar su endpoint autorizado, pero la pantalla exige ambos. |
| Ubicación | [PAN-005](02-pantallas-y-permisos.md#pan-005) — /admin/access |
| Roles / permisos requeridos | UI: sessions.read_all y authentication_attempts.read; backend permisos separados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Desde; Hasta (sin incluir); resultado |
| Validaciones | Fechas convertidas a ISO; resultado de intentos correcto/fallido/limitado. |
| Acciones disponibles | Filtrar |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-012](05-catalogo-capturas.md#cap-012) |
| Implementación | [web/src/Activity.tsx](../../web/src/Activity.tsx); [internal/httpapi/identity_handlers.go](../../internal/httpapi/identity_handlers.go); [internal/httpapi/lists.go](../../internal/httpapi/lists.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestHTTPAuthorizationFailedAttemptsAndNoSecretDisclosure`, fuente [internal/httpapi/server_test.go](../../internal/httpapi/server_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-014](07-pendientes-validacion.md#val-014) |

**Procedimiento:**

1. Abrir Accesos.
2. Elegir Sesiones o Intentos de acceso.
3. Definir fechas y resultado cuando aplica.
4. Pulsar Filtrar.
5. Usar Mostrar más.

<a id="fun-013"></a>

## FUN-013 — Listar y abrir bibliotecas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Listar y abrir bibliotecas |
| Objetivo y resultado esperado | Se abre el espacio de trabajo según los permisos y modalidad. |
| Descripción y comportamiento especial | Quien administra permisos puede ver una biblioteca y gestionar Personas sin acceso documental. Recargar puede perder selección/tab. |
| Ubicación | [PAN-008](02-pantallas-y-permisos.md#pan-008) — /libraries |
| Roles / permisos requeridos | Asignación a biblioteca o permissions.manage_global para administración |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin formulario; selección de tarjeta |
| Validaciones | La selección usa estado local dentro de /libraries, no una ruta web por ID. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-013](05-catalogo-capturas.md#cap-013) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/service.go](../../internal/libraries/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSearchScopePaginationAndPrivateQueryAudit`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Bibliotecas.
2. Revisar modalidad de las tarjetas.
3. Abrir la biblioteca deseada.
4. Elegir una pestaña disponible.

<a id="fun-014"></a>

## FUN-014 — Crear biblioteca vinculada

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Crear biblioteca vinculada |
| Objetivo y resultado esperado | Crea biblioteca destinada a indexar originales externos. |
| Descripción y comportamiento especial | La UI asigna al propio usuario como gestor; API permite otro gestor habilitado. No copia ni mueve archivos. |
| Ubicación | [PAN-009](02-pantallas-y-permisos.md#pan-009) — Bibliotecas → creación en línea |
| Roles / permisos requeridos | libraries.create + permissions.manage_global en backend; módulo linked |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; modalidad Vinculada; gestor inicial |
| Validaciones | Nombre hasta 160; gestor habilitado; clave de idempotencia evita doble alta. |
| Acciones disponibles | Nueva biblioteca; Crear biblioteca |
| Estados relacionados | Biblioteca vinculada activa |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-014](05-catalogo-capturas.md#cap-014) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/service.go](../../internal/libraries/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRootPlansSymlinksAndSearchInput`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Bibliotecas.
2. Pulsar Nueva biblioteca.
3. Escribir Nombre.
4. Elegir Vinculada.
5. Confirmar gestor inicial.
6. Pulsar Crear biblioteca.
7. Abrir Carpetas para registrar una ruta.

<a id="fun-015"></a>

## FUN-015 — Crear biblioteca administrada o híbrida

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Crear biblioteca administrada o híbrida |
| Objetivo y resultado esperado | Biblioteca administrada recibe cargas; híbrida combina cargas y rutas vinculadas. |
| Descripción y comportamiento especial | Expedientes iniciales condicionados al módulo de licencia; una biblioteca no necesita expedientes para almacenar documentos. |
| Ubicación | [PAN-009](02-pantallas-y-permisos.md#pan-009) — Bibliotecas → creación en línea |
| Roles / permisos requeridos | libraries.create + permissions.manage_global; módulos correspondientes |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; Administrada/Híbrida; gestor |
| Validaciones | Licencia y módulos de modalidad deben permitir creación. |
| Acciones disponibles | Nueva biblioteca en Bibliotecas |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-015](05-catalogo-capturas.md#cap-015) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/service.go](../../internal/libraries/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4LinkedConversionAssociationAndReassignmentPreservePhysicalIndex`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Pulsar Nueva biblioteca en Bibliotecas.
2. Completar Nombre.
3. Elegir Administrada o Híbrida.
4. Confirmar gestor inicial.
5. Crear biblioteca.
6. Registrar destino administrado y configurar organización.

<a id="fun-016"></a>

## FUN-016 — Asignar perfiles por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Permisos — Asignar perfiles por biblioteca |
| Objetivo y resultado esperado | Une permisos de perfiles seleccionados en esa biblioteca. |
| Descripción y comportamiento especial | Quitar todos los perfiles retira la asignación. Auditor de biblioteca no concede por sí mismo lectura documental. |
| Ubicación | [PAN-023](02-pantallas-y-permisos.md#pan-023) — Biblioteca → Personas |
| Roles / permisos requeridos | permissions.manage_library o permissions.manage_global |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Buscar persona; Lector; Colaborador; Revisor; Gestor; Auditor |
| Validaciones | Identidades y roles deben pertenecer al ámbito admitido; revalidación backend. |
| Acciones disponibles | Guardar permisos de biblioteca |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-016](05-catalogo-capturas.md#cap-016) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/service.go](../../internal/libraries/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestEveryWriteRechecksPermissionsAndSession`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-014](07-pendientes-validacion.md#val-014) |

**Procedimiento:**

1. Abrir biblioteca.
2. Personas.
3. Buscar una cuenta habilitada.
4. Seleccionarla o pulsar Editar.
5. Marcar perfiles.
6. Pulsar Guardar permisos de biblioteca.
7. Revisar asignación.

<a id="fun-017"></a>

## FUN-017 — Cambiar nombre y ampliar modalidad

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Cambiar nombre y ampliar modalidad |
| Objetivo y resultado esperado | Actualiza configuración sin recrear documentos. |
| Descripción y comportamiento especial | No hay botón general para eliminar una biblioteca; retiro de carpeta/documento se documenta aparte. |
| Ubicación | [PAN-024](02-pantallas-y-permisos.md#pan-024) — Biblioteca → Configuración |
| Roles / permisos requeridos | libraries.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; modalidad |
| Validaciones | Control de revisión; la ampliación a híbrida está permitida, las conversiones reductoras no. |
| Acciones disponibles | Guardar configuración |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-017](05-catalogo-capturas.md#cap-017) |
| Implementación | [web/src/ManagedSettings.tsx](../../web/src/ManagedSettings.tsx); [internal/libraries/settings.go](../../internal/libraries/settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4LinkedConversionAssociationAndReassignmentPreservePhysicalIndex`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configuración de biblioteca.
2. Cambiar nombre o elegir Híbrida si procede.
3. Pulsar Guardar configuración.
4. Revisar las pestañas.

<a id="fun-018"></a>

## FUN-018 — Habilitar expedientes y ejercicio

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Habilitar expedientes y ejercicio |
| Objetivo y resultado esperado | Controla organización y formularios futuros. |
| Descripción y comportamiento especial | Desactivar ejercicio normaliza estructura/nombres incompatibles. No afirmar que borra expedientes existentes. |
| Ubicación | [PAN-024](02-pantallas-y-permisos.md#pan-024) — Biblioteca → Configuración |
| Roles / permisos requeridos | libraries.configure; módulo expedientes; biblioteca no vinculada |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Utilizar expedientes; etiqueta del identificador; ejercicio |
| Validaciones | Etiqueta hasta 60; configuración compatible con modalidad/licencia. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-018](05-catalogo-capturas.md#cap-018) |
| Implementación | [web/src/ManagedSettings.tsx](../../web/src/ManagedSettings.tsx); [internal/libraries/settings.go](../../internal/libraries/settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configuración.
2. Activar o desactivar Utilizar expedientes.
3. Definir etiqueta del identificador.
4. Elegir uso de ejercicio.
5. Guardar configuración.
6. Revisar Expedientes.

<a id="fun-019"></a>

## FUN-019 — Seleccionar destino y reglas de almacenamiento

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Seleccionar destino y reglas de almacenamiento |
| Objetivo y resultado esperado | Define rutas para futuras incorporaciones aprobadas. |
| Descripción y comportamiento especial | No mueve documentos ya incorporados. Sufijo estable de identidad evita colisiones; materialización congela versión de configuración. |
| Ubicación | [PAN-024](02-pantallas-y-permisos.md#pan-024) — Biblioteca → Configuración |
| Roles / permisos requeridos | libraries.configure; destino administrado activo |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Destino; estructura; formato de nombre; prefijo |
| Validaciones | Estructuras y nombres deben ser opciones admitidas; prefijo hasta 40; configuración versionada. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-019](05-catalogo-capturas.md#cap-019) |
| Implementación | [web/src/ManagedSettings.tsx](../../web/src/ManagedSettings.tsx); [internal/libraries/settings.go](../../internal/libraries/settings.go); [internal/libraries/materialization.go](../../internal/libraries/materialization.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5PublicationCollisionAndFrozenNaming`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-021](07-pendientes-validacion.md#val-021) |

**Procedimiento:**

1. Registrar antes destino en Carpetas.
2. Abrir Configuración.
3. Elegir destino administrado.
4. Elegir estructura y formato de nombre.
5. Completar prefijo si procede.
6. Guardar configuración.

<a id="fun-020"></a>

## FUN-020 — Consultar ejemplo de nombre y ruta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Consultar ejemplo de nombre y ruta |
| Objetivo y resultado esperado | Calcula ejemplo con valores persistidos. |
| Descripción y comportamiento especial | Los cambios sin guardar del formulario principal no alteran el ejemplo; el original de ejemplo no se materializa. |
| Ubicación | [PAN-025](02-pantallas-y-permisos.md#pan-025) — Configuración → Vista previa del formato guardado |
| Roles / permisos requeridos | libraries.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Identificador; Ejercicio; Categoría; Tipo |
| Validaciones | Datos de catálogo y estructura consistentes. |
| Acciones disponibles | Ver ejemplo |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-020](05-catalogo-capturas.md#cap-020) |
| Implementación | [web/src/ManagedSettings.tsx](../../web/src/ManagedSettings.tsx); [internal/libraries/settings.go](../../internal/libraries/settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5PublicationCollisionAndFrozenNaming`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Guardar primero configuración.
2. Abrir Vista previa del formato guardado.
3. Completar ejemplo.
4. Pulsar Ver ejemplo.
5. Leer ruta y nombre derivados.

<a id="fun-021"></a>

## FUN-021 — Configurar revisión y retención de cargas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Configurar revisión y retención de cargas |
| Objetivo y resultado esperado | Define publicación directa o con revisión y futura limpieza de temporales. |
| Descripción y comportamiento especial | Aprobación vinculada mantiene original. Retención no es borrado de los documentos confirmados ni del historial. |
| Ubicación | [PAN-024](02-pantallas-y-permisos.md#pan-024) — Biblioteca → Configuración |
| Roles / permisos requeridos | libraries.configure; módulo revisión si se habilita |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Revisión administrada; revisión inicial vinculada en híbridas; conservación días |
| Validaciones | Retención 0–3650 días; 0 conserva indefinidamente. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-021](05-catalogo-capturas.md#cap-021) |
| Implementación | [web/src/ManagedSettings.tsx](../../web/src/ManagedSettings.tsx); [internal/libraries/settings.go](../../internal/libraries/settings.go); [internal/libraries/workflow.go](../../internal/libraries/workflow.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5ReviewAssignmentDecisionsAndRetention`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configuración.
2. Definir requisitos de revisión por origen.
3. Indicar conservación de cargas canceladas/rechazadas.
4. Guardar.
5. Explicar a los revisores el flujo elegido.

<a id="fun-022"></a>

## FUN-022 — Inspeccionar una ruta de carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Inspeccionar una ruta de carpeta |
| Objetivo y resultado esperado | Obtiene plan de alta/vista/consolidación o aviso de raíz existente. |
| Descripción y comportamiento especial | No es un selector de carpetas del navegador del cliente. En Windows usar UNC de red, no unidades mapeadas de sesión interactiva. |
| Ubicación | [PAN-020](02-pantallas-y-permisos.md#pan-020) — Biblioteca → Carpetas |
| Roles / permisos requeridos | storage.manage_roots |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Origen de la carpeta; Ruta de la carpeta en el servidor |
| Validaciones | Ruta accesible bajo cuenta del servicio; se rechazan enlaces, rutas internas privadas y solapamientos incompatibles. |
| Acciones disponibles | Inspeccionar carpeta |
| Estados relacionados | Plan pendiente / error de acceso |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-022](05-catalogo-capturas.md#cap-022) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/roots.go](../../internal/libraries/roots.go); [internal/libraries/filesystem.go](../../internal/libraries/filesystem.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRootPlansSymlinksAndSearchInput`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Carpetas.
2. Elegir origen cuando aparezca selector.
3. Escribir ruta absoluta del equipo servidor.
4. Pulsar Inspeccionar carpeta.
5. Leer el tipo de plan y sus condiciones.

<a id="fun-023"></a>

## FUN-023 — Registrar raíz vinculada

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Registrar raíz vinculada |
| Objetivo y resultado esperado | Registra raíz y encola descubrimiento. |
| Descripción y comportamiento especial | El procesamiento es asíncrono; detectar una raíz no equivale a terminar OCR de todos los documentos. |
| Ubicación | [PAN-021](02-pantallas-y-permisos.md#pan-021) — Carpetas → Inspeccionar carpeta |
| Roles / permisos requeridos | storage.manage_roots; módulo linked |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Plan y revisión; ruta |
| Validaciones | El plan se revalida; si cambió configuración/carpeta hay que inspeccionar otra vez. |
| Acciones disponibles | Registrar e indexar |
| Estados relacionados | Raíz activa; trabajo queued/running |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-023](05-catalogo-capturas.md#cap-023) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/roots.go](../../internal/libraries/roots.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestExtractionStartsWhileScanStillRunning`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Inspeccionar carpeta con origen Vinculada.
2. Revisar que es independiente.
3. Pulsar Registrar e indexar.
4. Revisar raíz en Carpetas.
5. Consultar Procesamiento.

<a id="fun-024"></a>

## FUN-024 — Registrar destino administrado

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Registrar destino administrado |
| Objetivo y resultado esperado | Registra destino apto para escrituras futuras. |
| Descripción y comportamiento especial | No incorpora por sí solo archivos externos ya presentes como cargas confirmadas. |
| Ubicación | [PAN-021](02-pantallas-y-permisos.md#pan-021) — Carpetas → Inspeccionar carpeta |
| Roles / permisos requeridos | storage.manage_roots; módulo managed |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Origen administrado; plan |
| Validaciones | Prueba de escritura/lectura y rechazo de raíces solapadas; revalidación de plan. |
| Acciones disponibles | Registrar destino administrado |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-024](05-catalogo-capturas.md#cap-024) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/roots.go](../../internal/libraries/roots.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4UploadValidationRecoveryAndRootOverlap`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Inspeccionar carpeta como destino administrado.
2. Revisar ausencia de solapamientos.
3. Pulsar Registrar destino administrado.
4. Ir a Configuración.
5. Seleccionarlo como destino.

<a id="fun-025"></a>

## FUN-025 — Crear vista lógica dentro de una raíz

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Crear vista lógica dentro de una raíz |
| Objetivo y resultado esperado | Crea filtro lógico por raíz y prefijo. |
| Descripción y comportamiento especial | No agrega un segundo escaneo de la misma subcarpeta ni mueve archivos. |
| Ubicación | [PAN-021](02-pantallas-y-permisos.md#pan-021) — Carpetas → Inspeccionar carpeta |
| Roles / permisos requeridos | storage.manage_roots |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre de vista |
| Validaciones | Nombre hasta 160; prefijo relativo seguro; unicidad biblioteca/raíz/prefijo. |
| Acciones disponibles | Crear vista lógica |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-025](05-catalogo-capturas.md#cap-025) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/roots.go](../../internal/libraries/roots.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestLinkedRetentionRenameConsolidationAndChange`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Inspeccionar una subcarpeta de raíz existente.
2. Leer propuesta de vista.
3. Escribir Nombre de vista.
4. Pulsar Crear vista lógica.
5. Seleccionarla en Documentos.

<a id="fun-026"></a>

## FUN-026 — Consolidar raíces hijas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Consolidar raíces hijas |
| Objetivo y resultado esperado | Amplía cobertura conservando identidades, extracción e historial. |
| Descripción y comportamiento especial | Raíces antiguas se sustituyen lógicamente; no se reescriben originales ni se fusionan registros por hash. |
| Ubicación | [PAN-021](02-pantallas-y-permisos.md#pan-021) — Carpetas → Inspeccionar carpeta |
| Roles / permisos requeridos | storage.manage_roots |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Plan antecesor; confirmación |
| Validaciones | Solo conjunto de raíces/versión comprobados; planes caducados rechazados. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-026](05-catalogo-capturas.md#cap-026) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/roots.go](../../internal/libraries/roots.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestLinkedRetentionRenameConsolidationAndChange`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Inspeccionar carpeta antecesora.
2. Revisar raíces hijas listadas.
3. Confirmar consolidación.
4. Revisar nueva raíz y vistas en Explorador.

<a id="fun-027"></a>

## FUN-027 — Programar verificación periódica de carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Programar verificación periódica de carpeta |
| Objetivo y resultado esperado | Persiste intervalo por raíz. |
| Descripción y comportamiento especial | Desactivar watcher no equivale a desactivar la verificación periódica. La programación coalesce solicitudes para una misma raíz. |
| Ubicación | [PAN-022](02-pantallas-y-permisos.md#pan-022) — Carpetas → Configurar carpeta |
| Roles / permisos requeridos | storage.manage_roots |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Verificación habilitada; Comprobar cada; unidad |
| Validaciones | Equivalente de 10 segundos a 30 días; revisión de raíz. |
| Acciones disponibles | Guardar verificación |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-027](05-catalogo-capturas.md#cap-027) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go); [internal/libraries/runtime.go](../../internal/libraries/runtime.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestUpgradeCoalescesLegacyRequestsAndPreservesConfiguredInterval`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En Carpetas abrir Configurar carpeta.
2. Activar verificación si procede.
3. Elegir cantidad y segundos/minutos/horas/días.
4. Pulsar Guardar verificación.
5. Revisar estado.

<a id="fun-028"></a>

## FUN-028 — Dejar de supervisar una carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Almacenamiento — Dejar de supervisar una carpeta |
| Objetivo y resultado esperado | Detiene supervisión y aplica política de referencias. |
| Descripción y comportamiento especial | No borra originales; conservar índice mantiene consulta, retirar referencias deja evidencia histórica. |
| Ubicación | [PAN-022](02-pantallas-y-permisos.md#pan-022) — Carpetas → Configurar carpeta |
| Roles / permisos requeridos | storage.manage_roots |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Política conservar índice o retirar referencias; confirmación |
| Validaciones | Plan y revisión vigentes; confirmación explícita; impacto de expedientes. |
| Acciones disponibles | Dejar de supervisar esta carpeta…; Confirmar retiro de carpeta o Cancelar |
| Estados relacionados | Raíz retirada |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-028](05-catalogo-capturas.md#cap-028) |
| Implementación | [web/src/LibrarySettings.tsx](../../web/src/LibrarySettings.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRetirementAndIndexRemovalPreserveOriginalAndHistory`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configurar carpeta.
2. Pulsar Dejar de supervisar esta carpeta….
3. Leer impacto del plan.
4. Elegir política.
5. Confirmar.
6. Pulsar Confirmar retiro de carpeta o Cancelar.

<a id="fun-029"></a>

## FUN-029 — Navegar carpetas y vistas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Navegar carpetas y vistas |
| Objetivo y resultado esperado | Muestra hijos directos de la carpeta y documentos del alcance elegido. |
| Descripción y comportamiento especial | Cambiar raíz/vista reinicia prefijo y cursor. Las vistas impiden subir fuera de su prefijo base. |
| Ubicación | [PAN-011](02-pantallas-y-permisos.md#pan-011) — Biblioteca → Documentos |
| Roles / permisos requeridos | documents.read |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Carpeta raíz; Vista lógica; migas |
| Validaciones | Raíz/vista de la biblioteca; prefijo relativo validado. |
| Acciones disponibles | Mostrar más si hay más carpetas |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-029](05-catalogo-capturas.md#cap-029) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/search.go](../../internal/libraries/search.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestGroupedSearchTotalsScopeAndDirectChildren`, fuente [internal/libraries/grouped_search_test.go](../../internal/libraries/grouped_search_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-011](07-pendientes-validacion.md#val-011) |

**Procedimiento:**

1. Abrir Documentos.
2. Elegir Carpeta raíz o Vista lógica.
3. Abrir una carpeta.
4. Usar migas, Inicio de carpeta o Carpeta superior.
5. Pulsar Mostrar más si hay más carpetas.

<a id="fun-030"></a>

## FUN-030 — Cambiar presentación y filtros del explorador

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Cambiar presentación y filtros del explorador |
| Objetivo y resultado esperado | Lista/cuadrícula muestran registros con iconos SVG por formato. |
| Descripción y comportamiento especial | Preferencia visual en sessionStorage. Actualizar documentos recarga DB; no realiza recorrido de disco. |
| Ubicación | [PAN-011](02-pantallas-y-permisos.md#pan-011) — Biblioteca → Documentos |
| Roles / permisos requeridos | documents.read |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Formato; Disponibilidad; Vista de lista / Vista de cuadrícula |
| Validaciones | Formatos PDF, Word (DOCX), Excel (XLSX), Texto (TXT), CSV; disponibilidad Todas/available/missing/unknown/staged según datos. |
| Acciones disponibles | Actualizar documentos para recargar |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-030](05-catalogo-capturas.md#cap-030) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/search.go](../../internal/libraries/search.go); [web/src/ExplorerEntries.tsx](../../web/src/ExplorerEntries.tsx) |
| Verificación | Verificado en pruebas (escenario parcial): `TestReindexHTTPAuthorizationIdempotencyAndExplorerFormat`, fuente [internal/httpapi/reindex_test.go](../../internal/httpapi/reindex_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Documentos.
2. Seleccionar formato y disponibilidad.
3. Cambiar Vista de lista o Vista de cuadrícula.
4. Revisar registros.
5. Pulsar Actualizar documentos para recargar.

<a id="fun-031"></a>

## FUN-031 — Buscar desde una carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Buscar desde una carpeta |
| Objetivo y resultado esperado | Filtra documentos con ámbito explícito y conserva referencia de contexto. |
| Descripción y comportamiento especial | Búsqueda con subcarpetas difiere del listado de hijos directos; coincidencias pueden remitir a unidad concreta. |
| Ubicación | [PAN-011](02-pantallas-y-permisos.md#pan-011) — Biblioteca → Documentos |
| Roles / permisos requeridos | search.execute en biblioteca |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Tipo de búsqueda; texto; alcance |
| Validaciones | Texto hasta 2000; sintaxis FTS construida/validada por backend, no SQL libre. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-031](05-catalogo-capturas.md#cap-031) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/search.go](../../internal/libraries/search.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestGroupedSearchTotalsScopeAndDirectChildren`, fuente [internal/libraries/grouped_search_test.go](../../internal/libraries/grouped_search_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Documentos.
2. Elegir búsqueda general/nombre/identificador/texto OCR.
3. Escribir texto.
4. Elegir alcance biblioteca o carpeta y subcarpetas.
5. Enviar búsqueda.
6. Abrir coincidencia o Cerrar búsqueda.

<a id="fun-032"></a>

## FUN-032 — Consultar ficha y original

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Consultar ficha y original |
| Objetivo y resultado esperado | Muestra metadatos y original PDF cuando disponible. |
| Descripción y comportamiento especial | Cerrar/Escape colapsa primero PDF ampliado. Disponibilidad del original y vigencia del texto son estados distintos. |
| Ubicación | [PAN-012](02-pantallas-y-permisos.md#pan-012) — Explorador / Búsqueda / expediente / Cargas → Ver ficha o Clasificar |
| Roles / permisos requeridos | documents.read y visibilidad |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin edición inicial |
| Validaciones | Private staged exige autor/revisor autorizado; paths completos solo storage.view_paths; contenido autorizado en cada solicitud. |
| Acciones disponibles | Ver ficha en un documento |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-032](05-catalogo-capturas.md#cap-032) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go); [internal/httpapi/library_handlers.go](../../internal/httpapi/library_handlers.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4MultipartAuthorizationRangesAndNoTemporaryDisclosure`, fuente [internal/httpapi/organization_test.go](../../internal/httpapi/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-011](07-pendientes-validacion.md#val-011) |

**Procedimiento:**

1. Pulsar Ver ficha en un documento.
2. Revisar disponibilidad, formato y estado.
3. Consultar panel original.
4. Cerrar ficha o ampliar PDF.

<a id="fun-033"></a>

## FUN-033 — Descargar archivo original

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Descargar archivo original |
| Objetivo y resultado esperado | Descarga original con MIME y nombre seguros. |
| Descripción y comportamiento especial | No convertir a PDF al descargar Word/Excel; preview es copia derivada distinta. |
| Ubicación | [PAN-012](02-pantallas-y-permisos.md#pan-012) — Explorador / Búsqueda / expediente / Cargas → Ver ficha o Clasificar |
| Roles / permisos requeridos | documents.download; documento accesible |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin formulario; botón Descargar FORMATO |
| Validaciones | Backend diferencia endpoint content (lectura) de download (descarga); no basta botón visible. |
| Acciones disponibles | Descargar PDF/DOCX/XLSX/TXT/CSV según formato |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-033](05-catalogo-capturas.md#cap-033) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/httpapi/library_handlers.go](../../internal/httpapi/library_handlers.go); [internal/libraries/documents.go](../../internal/libraries/documents.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5WorkflowHTTPAuthorizationPublicationAndRanges`, fuente [internal/httpapi/workflow_test.go](../../internal/httpapi/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha.
2. Comprobar disponibilidad.
3. Pulsar Descargar PDF/DOCX/XLSX/TXT/CSV según formato.
4. Revisar descarga del navegador.

<a id="fun-034"></a>

## FUN-034 — Consultar texto por unidad y resumen de extracción

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Extracción — Consultar texto por unidad y resumen de extracción |
| Objetivo y resultado esperado | Texto por página PDF, párrafo DOCX, hoja XLSX o bloque TXT/CSV. |
| Descripción y comportamiento especial | En DOCX se abre primera unidad no vacía cuando procede; una unidad vacía no implica documento entero vacío. XLSX no calcula fórmulas. |
| Ubicación | [PAN-013](02-pantallas-y-permisos.md#pan-013) — Ficha → Texto extraído / Último procesamiento |
| Roles / permisos requeridos | documents.read |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Página o Unidad de texto |
| Validaciones | Unidad de 1 al número disponible; respuesta de texto respeta permisos del documento. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Texto inexistente / vigente / desactualizado; run completo / advertencias / fallido |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-034](05-catalogo-capturas.md#cap-034) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go); [internal/extraction/document.go](../../internal/extraction/document.go); [web/src/ExtractionDetails.tsx](../../web/src/ExtractionDetails.tsx) |
| Verificación | Verificado en pruebas (escenario parcial): `TestLegacyWordRunOpensFirstNonemptyUnit`, fuente [internal/libraries/document_extractors_test.go](../../internal/libraries/document_extractors_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-019](07-pendientes-validacion.md#val-019) |

**Procedimiento:**

1. Abrir ficha.
2. Revisar Texto extraído.
3. Seleccionar unidad o usar Anterior/Siguiente.
4. Abrir Último procesamiento.
5. Leer extractor, resumen y advertencias.

<a id="fun-035"></a>

## FUN-035 — Solicitar y consultar preview de Word o Excel

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Previews — Solicitar y consultar preview de Word o Excel |
| Objetivo y resultado esperado | Genera/consulta PDF derivado bajo caché privada. |
| Descripción y comportamiento especial | No hay preview visual equivalente para TXT/CSV. No disponer de LibreOffice no impide extracción ni búsqueda. |
| Ubicación | [PAN-014](02-pantallas-y-permisos.md#pan-014) — Ficha → panel de vista previa |
| Roles / permisos requeridos | documents.read; previews habilitadas; LibreOffice disponible |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin formulario; botones de estado |
| Validaciones | Fuente dentro límite; convertidor local; output máximo 64 MiB y conversión 120 s. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | queued / generating / ready / error / obsolete / unavailable |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-035](05-catalogo-capturas.md#cap-035) |
| Implementación | [web/src/DocumentPreview.tsx](../../web/src/DocumentPreview.tsx); [internal/libraries/preview_worker.go](../../internal/libraries/preview_worker.go); [internal/previews/converter.go](../../internal/previews/converter.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPreviewOnDemandReuseClearAndPrivateAccess`, fuente [internal/libraries/previews_test.go](../../internal/libraries/previews_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-018](07-pendientes-validacion.md#val-018) |

**Procedimiento:**

1. Abrir ficha DOCX/XLSX.
2. Esperar creación solicitada automáticamente si no está lista.
3. Revisar estado.
4. Ampliar vista previa / Reducir vista previa o Comprobar vigencia.
5. Reintentar cuando la UI lo ofrezca.

<a id="fun-036"></a>

## FUN-036 — Reindexar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Reindexar un documento |
| Objetivo y resultado esperado | Nueva extracción usa configuración vigente sin cambiar identidad del documento. |
| Descripción y comportamiento especial | Fallo no sustituye índice válido previo; puede cerrarse ficha mientras sigue trabajo. No confundir reindexación con nueva carga. |
| Ubicación | [PAN-015](02-pantallas-y-permisos.md#pan-015) — Ficha → Reindexar documento |
| Roles / permisos requeridos | indexing.run; reglas de formato y licencia |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Idempotency-Key generado por UI |
| Validaciones | Estado admisible; no cancelled/materializing; restricciones de origen, tamaño y módulo OCR. |
| Acciones disponibles | Reindexar documento |
| Estados relacionados | queued/running/completed/failed |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-036](05-catalogo-capturas.md#cap-036) |
| Implementación | [web/src/ReindexDocument.tsx](../../web/src/ReindexDocument.tsx); [internal/libraries/reindex.go](../../internal/libraries/reindex.go); [internal/libraries/extraction.go](../../internal/libraries/extraction.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestManualReindexRulesPrivateVisibilityAndFailedAttemptRetainText`, fuente [internal/libraries/reindex_test.go](../../internal/libraries/reindex_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha.
2. Pulsar Reindexar documento.
3. Seguir progreso.
4. Consultar nuevo Último procesamiento.
5. Buscar de nuevo el texto esperado.

<a id="fun-037"></a>

## FUN-037 — Retirar documento del índice

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Documentos — Retirar documento del índice |
| Objetivo y resultado esperado | Oculta documento de listados/búsquedas y despublica FTS; conserva evidencia. |
| Descripción y comportamiento especial | No hay función de restauración desde UI; escaneo periódico no reimporta automáticamente el registro retirado. Vista Duplicados no ofrece esta acción. |
| Ubicación | [PAN-019](02-pantallas-y-permisos.md#pan-019) — Ficha → Retirar del índice |
| Roles / permisos requeridos | documents.remove_index |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Motivo; confirmaciones |
| Validaciones | Motivo requerido hasta 1000; revisión y expediente esperado; materialización activa impide retiro. |
| Acciones disponibles | Confirmar retiro del índice |
| Estados relacionados | Retirado lógicamente |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-037](05-catalogo-capturas.md#cap-037) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRetirementAndIndexRemovalPreserveOriginalAndHistory`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha.
2. Desplegar Retirar del índice.
3. Leer conservación de original/historial.
4. Escribir Motivo.
5. Confirmar índice y expediente cuando se pidan.
6. Pulsar Confirmar retiro del índice.

<a id="fun-038"></a>

## FUN-038 — Crear, editar y archivar categorías

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Crear, editar y archivar categorías |
| Objetivo y resultado esperado | Catálogo propio de la biblioteca. |
| Descripción y comportamiento especial | Archivar no borra referencias históricas ni documentos; no hay acción de borrado físico del catálogo. |
| Ubicación | [PAN-028](02-pantallas-y-permisos.md#pan-028) — Biblioteca no vinculada → Catálogos |
| Roles / permisos requeridos | documents.read para consulta; catalogs.manage para cambios |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; archivado |
| Validaciones | Nombre hasta 160; revisión/ámbito; solo no vinculadas para organización. |
| Acciones disponibles | Crear categoría o Guardar categoría |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-038](05-catalogo-capturas.md#cap-038) |
| Implementación | [web/src/Catalogs.tsx](../../web/src/Catalogs.tsx); [internal/libraries/catalogs.go](../../internal/libraries/catalogs.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Catálogos.
2. Revisar Categorías.
3. Crear categoría o editar una existente.
4. Escribir Nombre y estado.
5. Pulsar Crear categoría o Guardar categoría.
6. Revisar disponibilidad en clasificación.

<a id="fun-039"></a>

## FUN-039 — Configurar tipos documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Configurar tipos documentales |
| Objetivo y resultado esperado | Define restricciones de clasificación y requisitos. |
| Descripción y comportamiento especial | Tipo con una sola instancia afecta multiplicidad y cantidades; campos obligatorios se aplican en backend. |
| Ubicación | [PAN-028](02-pantallas-y-permisos.md#pan-028) — Biblioteca no vinculada → Catálogos |
| Roles / permisos requeridos | catalogs.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; categoría; múltiples; título descriptivo; archivado |
| Validaciones | Categoría del tipo no se cambia libremente en edición; referencias de la misma biblioteca. |
| Acciones disponibles | Crear tipo o Guardar tipo |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-039](05-catalogo-capturas.md#cap-039) |
| Implementación | [web/src/Catalogs.tsx](../../web/src/Catalogs.tsx); [internal/libraries/catalogs.go](../../internal/libraries/catalogs.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Catálogos.
2. Crear tipo.
3. Seleccionar categoría.
4. Escribir Nombre.
5. Definir si admite varios y título descriptivo.
6. Pulsar Crear tipo o Guardar tipo.
7. Editar/archivar cuando proceda.

<a id="fun-040"></a>

## FUN-040 — Crear plantilla y nuevas versiones

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Crear plantilla y nuevas versiones |
| Objetivo y resultado esperado | Publica snapshot de requisitos para futuros expedientes. |
| Descripción y comportamiento especial | Versiones anteriores inmutables; copiar última versión no actualiza expedientes ya inicializados. |
| Ubicación | [PAN-029](02-pantallas-y-permisos.md#pan-029) — Catálogos → Plantillas |
| Roles / permisos requeridos | templates.manage; expedientes habilitados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre; Incluir; cantidad; Obligatorio |
| Validaciones | Nombre hasta 160; cantidades 1–1000, o 1 para tipo no múltiple; tipos válidos. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-040](05-catalogo-capturas.md#cap-040) |
| Implementación | [web/src/Catalogs.tsx](../../web/src/Catalogs.tsx); [internal/libraries/catalogs.go](../../internal/libraries/catalogs.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4TemplateSnapshotsInitializationAndDerivedCounts`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En Catálogos abrir Plantillas.
2. Crear plantilla.
3. Escribir nombre.
4. Seleccionar tipos, cantidades y obligatoriedad.
5. Crear plantilla o Guardar nueva versión según formulario.
6. Para cambios pulsar Crear nueva versión de {nombre} y guardar versión nueva.

<a id="fun-041"></a>

## FUN-041 — Listar y filtrar expedientes

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Listar y filtrar expedientes |
| Objetivo y resultado esperado | Lista expedientes del ámbito autorizado. |
| Descripción y comportamiento especial | No es búsqueda OCR dentro de expedientes; filtro busca la entidad expediente. |
| Ubicación | [PAN-030](02-pantallas-y-permisos.md#pan-030) — Biblioteca → Expedientes |
| Roles / permisos requeridos | documents.read; casos habilitados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Consulta de identificador |
| Validaciones | Consulta hasta 160; cursor se reinicia al cambiar filtro. |
| Acciones disponibles | Filtrar expedientes |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-041](05-catalogo-capturas.md#cap-041) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Expedientes.
2. Escribir filtro.
3. Pulsar Filtrar expedientes.
4. Usar Mostrar más si corresponde.
5. Abrir expediente.

<a id="fun-042"></a>

## FUN-042 — Crear expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Crear expediente |
| Objetivo y resultado esperado | Crea expediente y copia requisitos de versión elegida cuando hay plantilla. |
| Descripción y comportamiento especial | La lista inicial privilegia versión más reciente de plantilla. Sin plantilla no equivale a cero progreso automáticamente. |
| Ubicación | [PAN-030](02-pantallas-y-permisos.md#pan-030) — Biblioteca → Expedientes |
| Roles / permisos requeridos | cases.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Identificador; Ejercicio; Plantilla inicial |
| Validaciones | Identificador hasta 160; ejercicio hasta 40; restricciones de unicidad/ámbito. |
| Acciones disponibles | Nuevo expediente; Crear expediente |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-042](05-catalogo-capturas.md#cap-042) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4TemplateSnapshotsInitializationAndDerivedCounts`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-021](07-pendientes-validacion.md#val-021) |

**Procedimiento:**

1. Abrir Expedientes.
2. Pulsar Nuevo expediente.
3. Completar identificador.
4. Completar ejercicio si se utiliza.
5. Elegir plantilla inicial o Sin plantilla.
6. Pulsar Crear expediente.
7. Abrir detalle.

<a id="fun-043"></a>

## FUN-043 — Editar identificador y ejercicio

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Editar identificador y ejercicio |
| Objetivo y resultado esperado | Actualiza metadatos con revisión. |
| Descripción y comportamiento especial | No asumir renombrado físico retroactivo de documentos administrados: nomenclatura se congela por incorporación. |
| Ubicación | [PAN-031](02-pantallas-y-permisos.md#pan-031) — Expedientes → expediente seleccionado |
| Roles / permisos requeridos | cases.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Identificador; Ejercicio |
| Validaciones | Formato/longitud y unicidad de expediente; datos vigentes. |
| Acciones disponibles | Guardar expediente; Actualizar expediente |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-043](05-catalogo-capturas.md#cap-043) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir expediente.
2. Desplegar Editar identificador.
3. Modificar campos.
4. Pulsar Guardar expediente.
5. Pulsar Actualizar expediente.

<a id="fun-044"></a>

## FUN-044 — Inicializar requisitos desde plantilla

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Inicializar requisitos desde plantilla |
| Objetivo y resultado esperado | Copia snapshot de versión al expediente. |
| Descripción y comportamiento especial | No hay editor UI para agregar un requisito arbitrario nuevo en un expediente ya inicializado; sí ajustar los existentes. |
| Ubicación | [PAN-031](02-pantallas-y-permisos.md#pan-031) — Expedientes → expediente seleccionado |
| Roles / permisos requeridos | requirements.edit; expediente sin requisitos inicializados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Versión de plantilla |
| Validaciones | No permite reinicialización destructiva sobre requisitos ya inicializados. |
| Acciones disponibles | Inicializar requerimientos |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-044](05-catalogo-capturas.md#cap-044) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4TemplateSnapshotsInitializationAndDerivedCounts`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir expediente sin requisitos.
2. Seleccionar versión de plantilla.
3. Pulsar Inicializar requerimientos.
4. Revisar tarjetas de requisitos.

<a id="fun-045"></a>

## FUN-045 — Ajustar cantidades y obligatoriedad del expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Ajustar cantidades y obligatoriedad del expediente |
| Objetivo y resultado esperado | Edita requisito específico sin cambiar plantilla. |
| Descripción y comportamiento especial | Cambiar plantilla global no reemplaza estos ajustes locales. |
| Ubicación | [PAN-031](02-pantallas-y-permisos.md#pan-031) — Expedientes → expediente seleccionado |
| Roles / permisos requeridos | requirements.edit |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Cantidad requerida; Obligatorio |
| Validaciones | 1–1000; tipo de una instancia limita cantidad; revisión vigente. |
| Acciones disponibles | Guardar requisito |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-045](05-catalogo-capturas.md#cap-045) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4TemplateSnapshotsInitializationAndDerivedCounts`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir expediente.
2. Localizar requisito.
3. Cambiar Cantidad y/o Obligatorio.
4. Pulsar Guardar requisito.
5. Actualizar expediente.
6. Revisar avance.

<a id="fun-046"></a>

## FUN-046 — Interpretar avance y documentos de un expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Interpretar avance y documentos de un expediente |
| Objetivo y resultado esperado | Presenta avance derivado de documentos válidos y requisitos obligatorios. |
| Descripción y comportamiento especial | Documentos vinculados aprobados no pierden aprobación por simple inaccesibilidad; administrados alterados/no disponibles no se consideran válidos para avance. |
| Ubicación | [PAN-031](02-pantallas-y-permisos.md#pan-031) — Expedientes → expediente seleccionado |
| Roles / permisos requeridos | documents.read |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin escritura; selección de documento |
| Validaciones | No hay porcentaje cuando no hay requisitos obligatorios; mínimo requerido limita contribución por requisito. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-046](05-catalogo-capturas.md#cap-046) |
| Implementación | [web/src/Cases.tsx](../../web/src/Cases.tsx); [internal/libraries/cases.go](../../internal/libraries/cases.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir expediente.
2. Revisar porcentaje y totales por estado.
3. Leer cada requisito.
4. Abrir documentos asociados.
5. Comprobar disponibilidad y aprobación.

<a id="fun-047"></a>

## FUN-047 — Preparar carga múltiple

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Cargas — Preparar carga múltiple |
| Objetivo y resultado esperado | Prepara lote e inicia envíos individuales. |
| Descripción y comportamiento especial | Quitar archivo antes de enviar no es cancelación de carga ya persistida. |
| Ubicación | [PAN-032](02-pantallas-y-permisos.md#pan-032) — Biblioteca no vinculada → Cargas |
| Roles / permisos requeridos | documents.upload; biblioteca administrada/híbrida |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Selector de archivos o arrastrar; lista de lote |
| Validaciones | Hasta 100 seleccionados en UI; reglas efectivas por biblioteca; servidor limita ingresos simultáneos. |
| Acciones disponibles | Guardar borradores |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-047](05-catalogo-capturas.md#cap-047) |
| Implementación | [web/src/Uploads.tsx](../../web/src/Uploads.tsx); [internal/libraries/uploads.go](../../internal/libraries/uploads.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4PrivateUploadsScopeOCRAndIdempotency`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Cargas.
2. Seleccionar o arrastrar documentos.
3. Revisar nombres y formatos.
4. Quitar de lista los no deseados.
5. Pulsar Guardar borradores.

<a id="fun-048"></a>

## FUN-048 — Seguir carga y continuar tras error individual

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Cargas — Seguir carga y continuar tras error individual |
| Objetivo y resultado esperado | Cada documento válido queda borrador privado y un fallo no descarta otros. |
| Descripción y comportamiento especial | UI envía secuencialmente; backend admite dos ingresos. 100% de transferencia no significa validación/índice completo. |
| Ubicación | [PAN-032](02-pantallas-y-permisos.md#pan-032) — Biblioteca no vinculada → Cargas |
| Roles / permisos requeridos | documents.upload |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Lista individual; porcentajes; reintento de lote |
| Validaciones | MIME/bytes y seguridad, tamaño; idempotencia por archivo; HTTP 401 interrumpe continuidad por sesión. |
| Acciones disponibles | Clasificar en documento recibido |
| Estados relacionados | waiting / uploading / staged / failed |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-048](05-catalogo-capturas.md#cap-048) |
| Implementación | [web/src/Uploads.tsx](../../web/src/Uploads.tsx); [internal/libraries/uploads.go](../../internal/libraries/uploads.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRejectedUploadRecordsDetectionAndCleansTemporary`, fuente [internal/libraries/document_formats_test.go](../../internal/libraries/document_formats_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Enviar Guardar borradores.
2. Seguir cada porcentaje.
3. Si indica Validando documento…, esperar validación.
4. Revisar fallidos.
5. Reintentar los pendientes/fallidos disponibles.
6. Pulsar Clasificar en documento recibido.

<a id="fun-049"></a>

## FUN-049 — Iniciar otra carga conservando borradores previos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Cargas — Iniciar otra carga conservando borradores previos |
| Objetivo y resultado esperado | Reinicia selección de navegador y genera otro lote. |
| Descripción y comportamiento especial | Borradores siguen privados con sus identidades; no reutilizar UUID a propósito para otro archivo. |
| Ubicación | [PAN-032](02-pantallas-y-permisos.md#pan-032) — Biblioteca no vinculada → Cargas |
| Roles / permisos requeridos | documents.upload |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nuevo lote |
| Validaciones | No borra lotes ya persistidos. |
| Acciones disponibles | Iniciar otra carga |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-049](05-catalogo-capturas.md#cap-049) |
| Implementación | [web/src/Uploads.tsx](../../web/src/Uploads.tsx); [internal/libraries/uploads.go](../../internal/libraries/uploads.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestUploadHistoryCountsPaginationAndIsolation`, fuente [internal/libraries/upload_history_test.go](../../internal/libraries/upload_history_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Completar/revisar lote actual.
2. Pulsar Iniciar otra carga.
3. Seleccionar nuevos archivos.
4. Revisar historial para volver al lote anterior.

<a id="fun-050"></a>

## FUN-050 — Consultar historial de cargas y detalle de lote

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Cargas — Consultar historial de cargas y detalle de lote |
| Objetivo y resultado esperado | Expone conteos, estados, formatos y contexto del lote. |
| Descripción y comportamiento especial | Tener rol gestor no convierte todos los borradores ajenos en visibles. Estados activos se actualizan aproximadamente cada 3 s. |
| Ubicación | [PAN-033](02-pantallas-y-permisos.md#pan-033) — Cargas → historial |
| Roles / permisos requeridos | Autor; revisores habilitados según alcance |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Selección de lote; Mostrar más |
| Validaciones | Aislamiento de lotes privados; permisos de lectura del documento comprobados al abrir. |
| Acciones disponibles | Ver detalles si se permite |
| Estados relacionados | receiving / staged / retained / failed / cleaned |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-050](05-catalogo-capturas.md#cap-050) |
| Implementación | [web/src/Uploads.tsx](../../web/src/Uploads.tsx); [web/src/UploadHistory.tsx](../../web/src/UploadHistory.tsx); [internal/libraries/upload_history.go](../../internal/libraries/upload_history.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestUploadHistoryPrivateBatches`, fuente [internal/libraries/upload_history_test.go](../../internal/libraries/upload_history_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Cargas.
2. Revisar Historial de cargas.
3. Seleccionar lote.
4. Ver tabla de archivos.
5. Pulsar Ver detalles si se permite.
6. Paginar lotes/archivos.

<a id="fun-051"></a>

## FUN-051 — Clasificar borrador

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Clasificar borrador |
| Objetivo y resultado esperado | Persisten clasificación y asociación sin publicar automáticamente el borrador. |
| Descripción y comportamiento especial | Cambiar clasificación puede invalidar revisión pendiente; valores archivados existentes se conservan donde el formulario lo permite. |
| Ubicación | [PAN-016](02-pantallas-y-permisos.md#pan-016) — Ficha → Clasificación |
| Roles / permisos requeridos | documents.classify; visibilidad del autor/revisor |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Expediente; Categoría; Tipo; Título; metadatos |
| Validaciones | Título hasta 250; hasta 20 pares, clave 60 y valor 500; tipo/categoría de biblioteca; multiplicidad; revisión. |
| Acciones disponibles | Guardar clasificación |
| Estados relacionados | draft / rejected / pending_review / needs_review |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-051](05-catalogo-capturas.md#cap-051) |
| Implementación | [web/src/ClassificationForm.tsx](../../web/src/ClassificationForm.tsx); [internal/libraries/classification.go](../../internal/libraries/classification.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4ClassificationScopeMultiplicityAndProgress`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-021](07-pendientes-validacion.md#val-021) |

**Procedimiento:**

1. Desde Cargas pulsar Clasificar.
2. Abrir Clasificación.
3. Localizar/elegir expediente si aplica.
4. Elegir Categoría y Tipo.
5. Completar título si se requiere.
6. Añadir metadatos necesarios.
7. Pulsar Guardar clasificación.

<a id="fun-052"></a>

## FUN-052 — Asociar vinculado a expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Asociar vinculado a expediente |
| Objetivo y resultado esperado | Añade referencia y organización al archivo externo. |
| Descripción y comportamiento especial | Conserva original, hash, texto e identidad física; no lo convierte en administrado. |
| Ubicación | [PAN-016](02-pantallas-y-permisos.md#pan-016) — Ficha → Clasificación |
| Roles / permisos requeridos | documents.associate; biblioteca híbrida; documentos visibles |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Expediente; Categoría; Tipo; Título |
| Validaciones | Documento pertenece a misma biblioteca; expediente y tipos válidos; revisión vigente. |
| Acciones disponibles | Asociar archivo |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-052](05-catalogo-capturas.md#cap-052) |
| Implementación | [web/src/ClassificationForm.tsx](../../web/src/ClassificationForm.tsx); [internal/libraries/classification.go](../../internal/libraries/classification.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4LinkedConversionAssociationAndReassignmentPreservePhysicalIndex`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir expediente.
2. Localizar documentos vinculados sin expediente.
3. Abrir ficha.
4. En Clasificación elegir expediente y clasificación.
5. Pulsar Asociar archivo.
6. Volver al expediente.

<a id="fun-053"></a>

## FUN-053 — Reasignar documento a otro expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Organización — Reasignar documento a otro expediente |
| Objetivo y resultado esperado | Cambia referencia documental y recuentos derivados. |
| Descripción y comportamiento especial | No cambia la biblioteca del documento ni renombra retroactivamente original por traslado lógico. |
| Ubicación | [PAN-016](02-pantallas-y-permisos.md#pan-016) — Ficha → Clasificación |
| Roles / permisos requeridos | documents.reassign |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nuevo expediente; Motivo |
| Validaciones | Motivo requerido hasta 1000; restricciones de multiplicidad/ámbito y revisión. |
| Acciones disponibles | Reasignar archivo |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-053](05-catalogo-capturas.md#cap-053) |
| Implementación | [web/src/ClassificationForm.tsx](../../web/src/ClassificationForm.tsx); [internal/libraries/classification.go](../../internal/libraries/classification.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH4LinkedConversionAssociationAndReassignmentPreservePhysicalIndex`, fuente [internal/libraries/organization_test.go](../../internal/libraries/organization_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha con expediente actual.
2. Abrir Clasificación.
3. Localizar nuevo expediente.
4. Revisar categoría/tipo.
5. Escribir Motivo.
6. Pulsar Reasignar archivo.
7. Revisar ambos expedientes.

<a id="fun-054"></a>

## FUN-054 — Cancelar carga propia

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Cargas — Cancelar carga propia |
| Objetivo y resultado esperado | Cancela flujo con evidencia; limpieza física temporal queda sujeta a retención. |
| Descripción y comportamiento especial | No es cancelar transferencia desde el browser ni eliminar documento confirmado. |
| Ubicación | [PAN-034](02-pantallas-y-permisos.md#pan-034) — Ficha privada → cancelación |
| Roles / permisos requeridos | documents.cancel_own; autor |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Motivo de cancelación |
| Validaciones | Solo carga propia y estados admisibles; motivo hasta 1000. |
| Acciones disponibles | Cancelar carga y conservar evidencia |
| Estados relacionados | cancelled; temporal retenido/cleaned |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-054](05-catalogo-capturas.md#cap-054) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/libraries/classification.go](../../internal/libraries/classification.go); [internal/libraries/workflow_queries.go](../../internal/libraries/workflow_queries.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5ReviewAssignmentDecisionsAndRetention`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-020](07-pendientes-validacion.md#val-020) |

**Procedimiento:**

1. Abrir ficha del borrador/carga admisible.
2. Leer cancelación y conservación.
3. Escribir Motivo.
4. Pulsar Cancelar carga y conservar evidencia.
5. Consultar historial.

<a id="fun-055"></a>

## FUN-055 — Enviar a revisión y actualizar envío

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Enviar a revisión y actualizar envío |
| Objetivo y resultado esperado | Solicitud conserva revisión y versión de contenido esperadas. |
| Descripción y comportamiento especial | No se publica borrador por enviarlo. Asignación exclusiva restringe quién puede decidir. |
| Ubicación | [PAN-017](02-pantallas-y-permisos.md#pan-017) — Ficha → acciones de WorkflowPanel |
| Roles / permisos requeridos | documents.submit; revisión habilitada |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Revisor o Cola compartida de revisores |
| Validaciones | Revisor habilitado y con permisos; no se aprueba versión diferente sin nueva comprobación. |
| Acciones disponibles | Enviar a revisión |
| Estados relacionados | pending_review |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-055](05-catalogo-capturas.md#cap-055) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/workflow.go](../../internal/libraries/workflow.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5ReviewAssignmentDecisionsAndRetention`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha clasificada.
2. Elegir revisor distinto del autor o cola compartida.
3. Pulsar Enviar a revisión.
4. Si se modifica envío, revisar estado y usar Actualizar envío a revisión.
5. Consultar Revisiones.

<a id="fun-056"></a>

## FUN-056 — Consultar revisiones pendientes

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Consultar revisiones pendientes |
| Objetivo y resultado esperado | Lista solicitudes visibles asignadas/compartidas. |
| Descripción y comportamiento especial | También hay endpoint global de pendientes sin una ruta de menú independiente; documentar API sin inventar pantalla. |
| Ubicación | [PAN-035](02-pantallas-y-permisos.md#pan-035) — Biblioteca → Revisiones |
| Roles / permisos requeridos | documents.review y documents.read efectivo |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin campos específicos |
| Validaciones | No reemplaza comprobaciones del backend al decidir. |
| Acciones disponibles | Actualizar pendientes; Revisar |
| Estados relacionados | pending |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-056](05-catalogo-capturas.md#cap-056) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/workflow_queries.go](../../internal/libraries/workflow_queries.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5WorkflowHTTPAuthorizationPublicationAndRanges`, fuente [internal/httpapi/workflow_test.go](../../internal/httpapi/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir biblioteca.
2. Revisiones.
3. Pulsar Actualizar pendientes.
4. Revisar autor y fecha.
5. Pulsar Revisar.
6. Inspeccionar ficha y clasificación antes de decidir.

<a id="fun-057"></a>

## FUN-057 — Aprobar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Aprobar un documento |
| Objetivo y resultado esperado | Vinculado queda aprobado sin mover original; administrado se confirma tras materialización. |
| Descripción y comportamiento especial | Aprobar no equivale a terminar copia; no enseñar a cerrar una incidencia de materialización como si estuviera disponible. |
| Ubicación | [PAN-017](02-pantallas-y-permisos.md#pan-017) — Ficha → acciones de WorkflowPanel |
| Roles / permisos requeridos | documents.approve; revisor autorizado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Confirmación de acción |
| Validaciones | Autor no se autoaprueba por perfil ordinario; excepción explícita y auditada si permiso especial existe; versión/revisión esperadas. |
| Acciones disponibles | Aprobar documento |
| Estados relacionados | approved o materializing |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-057](05-catalogo-capturas.md#cap-057) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/workflow.go](../../internal/libraries/workflow.go); [internal/libraries/materialization.go](../../internal/libraries/materialization.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5LinkedApprovalKeepsOriginalAndApprovalOnExternalChange`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir revisión pendiente.
2. Leer original, texto y clasificación.
3. Pulsar Aprobar documento.
4. Seguir incorporación si administrado.
5. Comprobar estado final.

<a id="fun-058"></a>

## FUN-058 — Rechazar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Rechazar un documento |
| Objetivo y resultado esperado | Rechaza solicitud y devuelve flujo a estado revisable. |
| Descripción y comportamiento especial | Temporal permanece hasta retención; clasificación puede corregirse y volver a enviar si permitido. |
| Ubicación | [PAN-017](02-pantallas-y-permisos.md#pan-017) — Ficha → acciones de WorkflowPanel |
| Roles / permisos requeridos | documents.reject; revisor autorizado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Motivo |
| Validaciones | Motivo requerido hasta 1000; autorización, asignación y versión comprobadas. |
| Acciones disponibles | Rechazar documento |
| Estados relacionados | rejected |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-058](05-catalogo-capturas.md#cap-058) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/workflow.go](../../internal/libraries/workflow.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5ReviewAssignmentDecisionsAndRetention`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir revisión.
2. Inspeccionar documento.
3. Escribir Motivo.
4. Pulsar Rechazar documento.
5. Ver estado en ficha/historial.

<a id="fun-059"></a>

## FUN-059 — Confirmar sin revisión

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Confirmar sin revisión |
| Objetivo y resultado esperado | Inicia publicación directa autorizada. |
| Descripción y comportamiento especial | Confirmación administrada crea copia final con verificación; carga privada se conserva hasta commit. |
| Ubicación | [PAN-017](02-pantallas-y-permisos.md#pan-017) — Ficha → acciones de WorkflowPanel |
| Roles / permisos requeridos | documents.finalize; revisión no requerida para origen |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Clasificación ya guardada |
| Validaciones | Permiso finalize, política y clasificación válida; no se ofrece como atajo a revisión requerida. |
| Acciones disponibles | Confirmar documento |
| Estados relacionados | materializing → approved |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-059](05-catalogo-capturas.md#cap-059) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/workflow.go](../../internal/libraries/workflow.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5DirectPublicationAndManagedIntegrity`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir documento clasificado.
2. Verificar destino y reglas.
3. Pulsar Confirmar documento.
4. Seguir progreso.
5. Abrir original una vez disponible.

<a id="fun-060"></a>

## FUN-060 — Seguir y reintentar incorporación administrada

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Revisión — Seguir y reintentar incorporación administrada |
| Objetivo y resultado esperado | Reanuda operación persistida sin duplicar archivo final confirmado. |
| Descripción y comportamiento especial | Fases planned/copying/verified/published/committed/cleaned/failed; avance por fases, no porcentaje de bytes. Copia no sobreescribe colisión existente. |
| Ubicación | [PAN-017](02-pantallas-y-permisos.md#pan-017) — Ficha → acciones de WorkflowPanel |
| Roles / permisos requeridos | Autorización aprobador o indexing.retry según estado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Motivo del reintento |
| Validaciones | Destino accesible, espacio, hash, configuración congelada y permisos; revalidación de licencia antes de publicar. |
| Acciones disponibles | Reintentar guardado con Motivo cuando disponible |
| Estados relacionados | planned / copying / verified / published / committed / cleaned / failed |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-060](05-catalogo-capturas.md#cap-060) |
| Implementación | [web/src/Workflow.tsx](../../web/src/Workflow.tsx); [internal/libraries/materialization.go](../../internal/libraries/materialization.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5MaterializationFailureRecovery`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-013](07-pendientes-validacion.md#val-013) |

**Procedimiento:**

1. En ficha revisar progreso de incorporación.
2. Si falla, consultar diagnóstico.
3. Corregir causa fuera del documento según soporte.
4. Pulsar Reintentar guardado con Motivo cuando disponible.
5. Confirmar estado final.

<a id="fun-061"></a>

## FUN-061 — Realizar búsqueda general

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Búsqueda — Realizar búsqueda general |
| Objetivo y resultado esperado | Consulta por nombre/contenido/identificador o OCR según tipo. |
| Descripción y comportamiento especial | No busca todas las bibliotecas por ser administrador; cuenta resultados dentro de bibliotecas autorizadas. |
| Ubicación | [PAN-038](02-pantallas-y-permisos.md#pan-038) — /search |
| Roles / permisos requeridos | search.execute en cada biblioteca |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Texto; tipo; biblioteca |
| Validaciones | Texto hasta 2000; alcance por permisos; admite filtros sin texto según modo. |
| Acciones disponibles | Buscar documentos |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-061](05-catalogo-capturas.md#cap-061) |
| Implementación | [web/src/Search.tsx](../../web/src/Search.tsx); [internal/libraries/search.go](../../internal/libraries/search.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSearchScopePaginationAndPrivateQueryAudit`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Buscar documentos.
2. Elegir biblioteca o todas las autorizadas.
3. Elegir tipo General/Nombre/Identificador/Texto OCR.
4. Escribir texto.
5. Pulsar Buscar documentos.
6. Revisar grupos de resultados.

<a id="fun-062"></a>

## FUN-062 — Aplicar filtros documentales de búsqueda

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Búsqueda — Aplicar filtros documentales de búsqueda |
| Objetivo y resultado esperado | Restringe conjunto por atributos documentales. |
| Descripción y comportamiento especial | Filtros se aplican en backend antes de contar y paginar; borradores ajenos permanecen excluidos. |
| Ubicación | [PAN-038](02-pantallas-y-permisos.md#pan-038) — /search |
| Roles / permisos requeridos | search.execute |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Formato; Disponibilidad; Origen; Estado; sin expediente |
| Validaciones | Formatos del catálogo; origen linked/managed; estados/availability válidos; revisión de cursor y consulta. |
| Acciones disponibles | Buscar documentos |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-062](05-catalogo-capturas.md#cap-062) |
| Implementación | [web/src/Search.tsx](../../web/src/Search.tsx); [internal/libraries/search.go](../../internal/libraries/search.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSearchFormatFiltersCountsCursorAndVisibility`, fuente [internal/libraries/search_formats_test.go](../../internal/libraries/search_formats_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Buscar documentos.
2. Seleccionar filtros generales.
3. Pulsar Buscar documentos.
4. Abrir grupo y revisar coincidencias.
5. Cambiar filtro y buscar otra vez.

<a id="fun-063"></a>

## FUN-063 — Buscar por catálogo, ejercicio y expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Búsqueda — Buscar por catálogo, ejercicio y expediente |
| Objetivo y resultado esperado | Búsqueda contextual por organización de la biblioteca. |
| Descripción y comportamiento especial | Catálogos archivados siguen pudiendo aportar filtros históricos; no ofrecer estos campos cuando biblioteca no los usa. |
| Ubicación | [PAN-038](02-pantallas-y-permisos.md#pan-038) — /search |
| Roles / permisos requeridos | search.execute; biblioteca no vinculada con organización |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Categoría; Tipo; Ejercicio; Localizar expediente; expediente |
| Validaciones | Tipos de categoría elegida; consulta de expediente hasta 160; ejercicio hasta 40; IDs del mismo ámbito. |
| Acciones disponibles | Buscar documentos |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-063](05-catalogo-capturas.md#cap-063) |
| Implementación | [web/src/Search.tsx](../../web/src/Search.tsx); [internal/libraries/search.go](../../internal/libraries/search.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSearchFormatFiltersCountsCursorAndVisibility`, fuente [internal/libraries/search_formats_test.go](../../internal/libraries/search_formats_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Elegir una biblioteca concreta.
2. Seleccionar Categoría y Tipo.
3. Indicar Ejercicio si aplica.
4. Localizar expediente y seleccionarlo.
5. Pulsar Buscar documentos.

<a id="fun-064"></a>

## FUN-064 — Navegar resultados agrupados y coincidencias

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Búsqueda — Navegar resultados agrupados y coincidencias |
| Objetivo y resultado esperado | Resumen por biblioteca y páginas independientes; enlace al contexto del resultado. |
| Descripción y comportamiento especial | El total agrupado no es la cantidad de filas de una página. Texto resaltado se renderiza seguro, sin HTML externo. |
| Ubicación | [PAN-038](02-pantallas-y-permisos.md#pan-038) — /search |
| Roles / permisos requeridos | search.execute / documents.read |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Grupo; tamaño de página 10/25/50; cursor |
| Validaciones | Cursores ligados a consulta y filtros; cambio reinicia posición. |
| Acciones disponibles | Ver ficha o coincidencia |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-064](05-catalogo-capturas.md#cap-064) |
| Implementación | [web/src/Search.tsx](../../web/src/Search.tsx); [internal/libraries/search.go](../../internal/libraries/search.go); [web/src/SearchResults.tsx](../../web/src/SearchResults.tsx) |
| Verificación | Verificado en pruebas (escenario parcial): `TestGroupedSearchTotalsScopeAndDirectChildren`, fuente [internal/libraries/grouped_search_test.go](../../internal/libraries/grouped_search_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-011](07-pendientes-validacion.md#val-011) |

**Procedimiento:**

1. Ejecutar búsqueda.
2. Expandir biblioteca.
3. Elegir resultados por página.
4. Usar Anterior/Siguiente del grupo.
5. Pulsar Ver ficha o coincidencia.
6. Revisar unidad de texto indicada.

<a id="fun-065"></a>

## FUN-065 — Consultar y reconocer cambios documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Búsqueda — Consultar y reconocer cambios documentales |
| Objetivo y resultado esperado | Acuse por usuario deja de mostrar el aviso para esa cuenta. |
| Descripción y comportamiento especial | Hasta 100 avisos. Validar aún combinación avisos/borradores dañados y cambios de permisos antes de capturas finales (VAL-015). |
| Ubicación | [PAN-039](02-pantallas-y-permisos.md#pan-039) — Buscar documentos → avisos |
| Roles / permisos requeridos | documents.read; licencia permite lectura |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Aviso; acuse |
| Validaciones | No autoriza modificación ni reparación; servidor verifica lectura al abrir. |
| Acciones disponibles | Ver documento modificado; Marcar como leído |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-065](05-catalogo-capturas.md#cap-065) |
| Implementación | [web/src/Search.tsx](../../web/src/Search.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH5DirectPublicationAndManagedIntegrity`, fuente [internal/libraries/workflow_test.go](../../internal/libraries/workflow_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-015](07-pendientes-validacion.md#val-015) |

**Procedimiento:**

1. Abrir Buscar documentos.
2. Revisar avisos de cambio.
3. Pulsar Ver documento modificado.
4. Examinar original/historial.
5. Pulsar Marcar como leído.

<a id="fun-066"></a>

## FUN-066 — Consultar grupos de duplicados

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Duplicados — Consultar grupos de duplicados |
| Objetivo y resultado esperado | Agrupa hashes de contenido idéntico de archivos visibles, incluso entre bibliotecas autorizadas. |
| Descripción y comportamiento especial | No elimina ni fusiona. Hard links pueden ser ubicaciones de misma identidad física, distintos de copias con misma huella. |
| Ubicación | [PAN-037](02-pantallas-y-permisos.md#pan-037) — Biblioteca → Duplicados |
| Roles / permisos requeridos | documents.read; visibilidad por biblioteca |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Selección de grupo; paginación |
| Validaciones | Solo hash actual reconocido; no equivalencia de nombre; borradores privados controlados. |
| Acciones disponibles | Ver huella completa |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-066](05-catalogo-capturas.md#cap-066) |
| Implementación | [web/src/Duplicates.tsx](../../web/src/Duplicates.tsx); [internal/libraries/duplicates.go](../../internal/libraries/duplicates.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestDuplicateGroupsRespectLibraryPermissions`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Duplicados o Revisar duplicados desde explorador.
2. Seleccionar grupo.
3. Revisar ubicaciones y cantidades.
4. Pulsar Ver huella completa.
5. Abrir ficha.
6. Mostrar más si hay más grupos.

<a id="fun-067"></a>

## FUN-067 — Consultar huella e historial documental

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Auditoría — Consultar huella e historial documental |
| Objetivo y resultado esperado | Traza descubrimiento, carga, cambios, apertura, descarga, clasificación, revisión y retiro. |
| Descripción y comportamiento especial | Hash corresponde versión de contenido, no título ni metadata; no mostrar rutas privadas de cargas. |
| Ubicación | [PAN-018](02-pantallas-y-permisos.md#pan-018) — Ficha → Huella y trazabilidad |
| Roles / permisos requeridos | documents.read; detalles según permisos |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | SHA-256; paginación |
| Validaciones | Acceso al documento y ámbito de auditoría; historial no editable. |
| Acciones disponibles | Consultar historial |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-067](05-catalogo-capturas.md#cap-067) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/libraries/documents.go](../../internal/libraries/documents.go); [web/src/DocumentHistory.tsx](../../web/src/DocumentHistory.tsx) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRetirementAndIndexRemovalPreserveOriginalAndHistory`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir ficha.
2. Desplegar Huella y trazabilidad.
3. Leer SHA-256.
4. Pulsar Consultar historial.
5. Paginar.
6. Ver detalles del evento.

<a id="fun-068"></a>

## FUN-068 — Filtrar y ordenar auditoría global

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Auditoría — Filtrar y ordenar auditoría global |
| Objetivo y resultado esperado | Lista eventos de personas y sistema con descripciones legibles. |
| Descripción y comportamiento especial | Por defecto último mes hasta ahora+1 min; información exacta de búsquedas requiere permiso adicional, no toda auditoría global la revela. |
| Ubicación | [PAN-006](02-pantallas-y-permisos.md#pan-006) — /admin/events |
| Roles / permisos requeridos | audit.read_global |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Desde; Hasta; Tipo de evento; Usuario; Buscar; orden |
| Validaciones | Fecha inicial anterior a final; texto hasta 200; paginación 25; fechas en zona del navegador. |
| Acciones disponibles | Aplicar filtros |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-068](05-catalogo-capturas.md#cap-068) |
| Implementación | [web/src/AuditTable.tsx](../../web/src/AuditTable.tsx); [internal/httpapi/audit_view.go](../../internal/httpapi/audit_view.go); [internal/httpapi/lists.go](../../internal/httpapi/lists.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAuditCombinedFiltersSortingAndScope`, fuente [internal/httpapi/audit_view_test.go](../../internal/httpapi/audit_view_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Eventos.
2. Definir fechas y filtros.
3. Pulsar Aplicar filtros.
4. Ordenar Fecha y hora/Usuario/Tipo de evento.
5. Paginar Anterior/Siguiente.
6. Usar Restablecer.

<a id="fun-069"></a>

## FUN-069 — Leer detalle de evento y registro original

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Auditoría — Leer detalle de evento y registro original |
| Objetivo y resultado esperado | Muestra datos históricos conservados en contexto autorizado. |
| Descripción y comportamiento especial | Actor automático se distingue de usuario; versiones históricas no se editan desde el diálogo. |
| Ubicación | [PAN-007](02-pantallas-y-permisos.md#pan-007) — Eventos / Auditoría / historial → Ver detalles |
| Roles / permisos requeridos | Permiso del visor; restricciones de detalle |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin edición |
| Validaciones | Campos sensibles de búsqueda se filtran por permiso; se sirven valores seguros del evento. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-069](05-catalogo-capturas.md#cap-069) |
| Implementación | [web/src/AuditTable.tsx](../../web/src/AuditTable.tsx); [internal/httpapi/audit_view.go](../../internal/httpapi/audit_view.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAuditRolePaginationAndImmutableEvents`, fuente [internal/httpapi/server_test.go](../../internal/httpapi/server_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En auditoría/historial pulsar Ver detalles.
2. Leer actor, recurso y descripción.
3. Desplegar Información avanzada · registro original.
4. Cerrar detalle.

<a id="fun-070"></a>

## FUN-070 — Consultar auditoría de biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Auditoría — Consultar auditoría de biblioteca |
| Objetivo y resultado esperado | Eventos limitados a la biblioteca seleccionada. |
| Descripción y comportamiento especial | Auditor de biblioteca puede consultar eventos sin recibir por ello acceso al original documental. |
| Ubicación | [PAN-040](02-pantallas-y-permisos.md#pan-040) — Biblioteca → Auditoría |
| Roles / permisos requeridos | audit.read_library |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Mismos filtros de AuditTable |
| Validaciones | Opciones de usuarios/tipos y detalle respetan alcance de biblioteca. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-070](05-catalogo-capturas.md#cap-070) |
| Implementación | [web/src/AuditTable.tsx](../../web/src/AuditTable.tsx); [internal/httpapi/audit_view.go](../../internal/httpapi/audit_view.go); [internal/libraries/search_audit.go](../../internal/libraries/search_audit.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAuditCombinedFiltersSortingAndScope`, fuente [internal/httpapi/audit_view_test.go](../../internal/httpapi/audit_view_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir biblioteca.
2. Auditoría.
3. Aplicar filtros.
4. Paginar.
5. Ver detalles.

<a id="fun-071"></a>

## FUN-071 — Consultar registro exacto de búsquedas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Auditoría — Consultar registro exacto de búsquedas |
| Objetivo y resultado esperado | Registro de consulta y filtros separado de auditoría general. |
| Descripción y comportamiento especial | Consultas exactas es una sección de Documentos en Libraries.tsx, junto con auditoría en otra pestaña. Solo se muestran búsquedas cuyos ámbitos completos pueden auditarse. No existe exportación masiva de consultas. |
| Ubicación | [PAN-010](02-pantallas-y-permisos.md#pan-010) — /libraries → biblioteca seleccionada (estado local) |
| Roles / permisos requeridos | audit.search_details en ámbitos solicitados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Bibliotecas; consulta histórica |
| Validaciones | API verifica permiso por biblioteca; no exhibir términos privados en capturas. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-071](05-catalogo-capturas.md#cap-071) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/search_audit.go](../../internal/libraries/search_audit.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSearchScopePaginationAndPrivateQueryAudit`, fuente [internal/libraries/service_test.go](../../internal/libraries/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-014](07-pendientes-validacion.md#val-014) |

**Procedimiento:**

1. Abrir una biblioteca con audit.search_details.
2. En Documentos revisar Consultas exactas.
3. Desplegar una consulta para leer texto, tipo y filtros.
4. Usar Mostrar más cuando esté disponible.

<a id="fun-072"></a>

## FUN-072 — Solicitar verificación manual de biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Bibliotecas — Solicitar verificación manual de biblioteca |
| Objetivo y resultado esperado | Encola escaneos vinculados y comprobaciones de integridad administradas. |
| Descripción y comportamiento especial | Hasta cuatro recorridos de raíces distintas simultáneos; una raíz nunca tiene dos verificaciones concurrentes. No confundir cola con finalización. |
| Ubicación | [PAN-010](02-pantallas-y-permisos.md#pan-010) — /libraries → biblioteca seleccionada (estado local) |
| Roles / permisos requeridos | indexing.run |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin campos; acción por biblioteca |
| Validaciones | Coalescencia de una misma raíz, no bloqueo de otras raíces. |
| Acciones disponibles | Verificar biblioteca ahora |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-072](05-catalogo-capturas.md#cap-072) |
| Implementación | [web/src/Libraries.tsx](../../web/src/Libraries.tsx); [internal/libraries/processing.go](../../internal/libraries/processing.go); [internal/libraries/runtime.go](../../internal/libraries/runtime.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRuntimeReconcilesDifferentRootsConcurrentlyWithPerRootExclusion`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir biblioteca.
2. Pulsar Verificar biblioteca ahora.
3. Leer confirmación de solicitud.
4. Abrir Procesamiento.
5. Revisar avances por raíz.

<a id="fun-073"></a>

## FUN-073 — Consultar resumen, progreso y tareas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Consultar resumen, progreso y tareas |
| Objetivo y resultado esperado | Muestra archivos lógicos, nativo/OCR, cola, intentos y actividad. |
| Descripción y comportamiento especial | Rendimiento 15 min incluye inactividad y esperas; no es benchmark de CPU. Sondeo aprox. 2.5 s. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | indexing.run |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Paginación |
| Validaciones | Cada archivo lógico contado una vez; descubrimiento total desconocido usa progreso indeterminado. |
| Acciones disponibles | Actualizar tareas |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-073](05-catalogo-capturas.md#cap-073) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/processing.go](../../internal/libraries/processing.go); [internal/libraries/processing_state.go](../../internal/libraries/processing_state.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestScanCachePartialErrorsAndTargetedRetry`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Procesamiento.
2. Revisar resumen y rendimiento.
3. Leer Reconciliación de carpetas y Trabajos activos.
4. Revisar tabla y diagnóstico.
5. Pulsar Actualizar tareas.
6. Paginar.

<a id="fun-074"></a>

## FUN-074 — Cancelar recorrido y reanudar raíz

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Cancelar recorrido y reanudar raíz |
| Objetivo y resultado esperado | Cancela cooperativamente y pausa raíz para que no reinicie sola. |
| Descripción y comportamiento especial | Recorrido incompleto no confirma eliminaciones; conserva índice previo y errores parciales. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | indexing.run según cancelación; verificación manual para reanudación |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Raíz/trabajo; motivo interno de solicitud |
| Validaciones | Solo tipos cancelables; no cancela materialización ni verificación administrada desde este botón. |
| Acciones disponibles | Cancelar recorrido |
| Estados relacionados | cancelled; root_scan_controls.paused |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-074](05-catalogo-capturas.md#cap-074) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/job_control.go](../../internal/libraries/job_control.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestCancelledScanStaysPausedAndCanResume`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En trabajos activos localizar recorrido.
2. Pulsar Cancelar recorrido.
3. Revisar raíz pausada.
4. Cuando proceda pulsar Reanudar reconciliación o solicitar verificación.
5. Seguir nuevo recorrido.

<a id="fun-075"></a>

## FUN-075 — Cancelar extracción pendiente o activa

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Cancelar extracción pendiente o activa |
| Objetivo y resultado esperado | Detiene cooperativamente extracción; no publica texto parcial. |
| Descripción y comportamiento especial | La reconciliación periódica no vuelve a lanzar extracción cancelada de la misma versión sin nueva petición. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | indexing.run; tipos cancelables |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Trabajo |
| Validaciones | Aplica queued/retry_wait/paused o running según trabajo; no cancela otros tipos indiscriminadamente. |
| Acciones disponibles | Cancelar trabajo |
| Estados relacionados | cancelled |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-075](05-catalogo-capturas.md#cap-075) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/job_control.go](../../internal/libraries/job_control.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestCancelledExtractionDoesNotReturnOnPeriodicScan`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En Procesamiento localizar extracción.
2. Pulsar Cancelar trabajo.
3. Comprobar estado.
4. Reintentar manualmente solo si es necesario.

<a id="fun-076"></a>

## FUN-076 — Reintentar tarea fallida

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Reintentar tarea fallida |
| Objetivo y resultado esperado | Solicita recuperación manual auditada. |
| Descripción y comportamiento especial | No debe usarse en bucle para error determinista de formato; reglas distinguen errores transitorios para reintentos automáticos. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | indexing.retry |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Trabajo; motivo registrado por UI |
| Validaciones | Estados failed/cancelled admitidos; mismos límites de permisos/origen/licencia. |
| Acciones disponibles | Reintentar en la tarea |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-076](05-catalogo-capturas.md#cap-076) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/job_control.go](../../internal/libraries/job_control.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPauseRetryBudgetsAndManualRecovery`, fuente [internal/libraries/processing_policy_test.go](../../internal/libraries/processing_policy_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Leer diagnóstico del trabajo.
2. Resolver causa.
3. Pulsar Reintentar en la tarea.
4. Revisar intento nuevo y resultado.

<a id="fun-077"></a>

## FUN-077 — Reintentar rutas con error y consultar omitidos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Reintentar rutas con error y consultar omitidos |
| Objetivo y resultado esperado | Reprocesa errores seleccionados/conocidos sin descartar toda biblioteca. |
| Descripción y comportamiento especial | Omitidos no generan cola repetitiva de extracción; ocultos/temporales conservan originales. Los errores por archivo permiten continuar otros. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | indexing.retry para rutas; indexing.run para vista |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Raíz y lista reciente |
| Validaciones | Hasta cien errores y cien omisiones recientes en UI; no es listado completo de todos los rechazos. |
| Acciones disponibles | reintento de rutas de la raíz |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-077](05-catalogo-capturas.md#cap-077) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/scan_cache.go](../../internal/libraries/scan_cache.go); [internal/libraries/file_skips.go](../../internal/libraries/file_skips.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestScanCachePartialErrorsAndTargetedRetry`, fuente [internal/libraries/processing_test.go](../../internal/libraries/processing_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Procesamiento.
2. Consultar errores de ruta y archivos omitidos.
3. Diferenciar error de permisos de omisión por reglas.
4. Resolver causa.
5. Pulsar reintento de rutas de la raíz.

<a id="fun-078"></a>

## FUN-078 — Comprender descubrimiento, hash y cambios vinculados

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Comprender descubrimiento, hash y cambios vinculados |
| Objetivo y resultado esperado | Recorre y registra versiones; encola extracción incremental conforme descubre. |
| Descripción y comportamiento especial | Archivo ilegible aislado produce SCAN_PARTIAL y no confirma eliminaciones sobre cobertura fallida. Desconexión completa conserva disponibilidades hasta revisión válida. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | Automático; módulo linked; raíz habilitada |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | No formulario; políticas de raíz/archivo |
| Validaciones | Hash SHA-256 sobre handle abierto; tamaño/mtime/identidad y estabilidad comprobados; caché evita leer contenido sin cambios. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-078](05-catalogo-capturas.md#cap-078) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/scanner.go](../../internal/libraries/scanner.go); [internal/libraries/scan_cache.go](../../internal/libraries/scan_cache.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestLinkedTextChangesReplaceFTSAndRetainEvidence`, fuente [internal/libraries/document_extractors_test.go](../../internal/libraries/document_extractors_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-012](07-pendientes-validacion.md#val-012); [VAL-019](07-pendientes-validacion.md#val-019) |

**Procedimiento:**

1. Registrar raíz.
2. Observar descubrimiento en Procesamiento.
3. Consultar documentos detectados.
4. Ante modificación externa, consultar nueva versión y texto.
5. Leer errores parciales si aparecen.

<a id="fun-079"></a>

## FUN-079 — Comprender vigilancia y pérdida de eventos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Comprender vigilancia y pérdida de eventos |
| Objetivo y resultado esperado | fsnotify agrupa cambios y reconciliación recupera desbordamientos. |
| Descripción y comportamiento especial | Directorio nuevo se registra; imposibilidad de watcher pasa a polling. Estado de pérdida no prueba pérdida de documentos. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | Automático; watcher configurado; sistema permite vigilancia |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Intervalo raíz; opciones watcher |
| Validaciones | Debounce 2.5 s con máximo 5 s desde primer evento; buffers finitos; generación de pérdida solo se reconoce tras barrera completa. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-079](05-catalogo-capturas.md#cap-079) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/watcher.go](../../internal/libraries/watcher.go); [internal/libraries/runtime.go](../../internal/libraries/runtime.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestOverflowGenerationNeedsCompleteBarrier`, fuente [internal/libraries/reconciliation_test.go](../../internal/libraries/reconciliation_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-012](07-pendientes-validacion.md#val-012) |

**Procedimiento:**

1. En Carpetas consultar Vigilancia nativa / Reconciliación periódica.
2. Revisar última comprobación.
3. Si se perdieron eventos, observar reconciliación.
4. Revisar diagnóstico si no recupera cobertura.

<a id="fun-080"></a>

## FUN-080 — Comprender extracción por formato

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Extracción — Comprender extracción por formato |
| Objetivo y resultado esperado | PDF usa Poppler/OCR; DOCX/XLSX OpenXML; TXT/CSV lectura Unicode con contexto. |
| Descripción y comportamiento especial | Por defecto se almacenan cinco formatos y solo PDF se indexa; habilitar Excel/Word para almacenar no basta para extraerlos. |
| Ubicación | [PAN-013](02-pantallas-y-permisos.md#pan-013) — Ficha → Texto extraído / Último procesamiento |
| Roles / permisos requeridos | Automático; formato indexable; tamaño permitido |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Política almacenar/indexar |
| Validaciones | No macros, scripts ni Office antiguos; motores disponibles no implican que política indexe formato. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-080](05-catalogo-capturas.md#cap-080) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/extraction/document.go](../../internal/extraction/document.go); [internal/documentformat/policy.go](../../internal/documentformat/policy.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestDocumentRegistryMatchesAdmissionCatalog`, fuente [internal/extraction/document_test.go](../../internal/extraction/document_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-019](07-pendientes-validacion.md#val-019) |

**Procedimiento:**

1. Habilitar formato para indexar en Ajustes o biblioteca.
2. Incorporar o reindexar archivo.
3. Revisar Texto extraído y Último procesamiento.
4. Consultar advertencias.

<a id="fun-081"></a>

## FUN-081 — Comprender OCR de PDF

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Extracción — Comprender OCR de PDF |
| Objetivo y resultado esperado | Reconoce páginas con texto nativo insuficiente. |
| Descripción y comportamiento especial | OCR no se aplica a imágenes incrustadas de DOCX/XLSX. OCR automático desactivado permite manual si OCR general sigue habilitado. |
| Ubicación | [PAN-013](02-pantallas-y-permisos.md#pan-013) — Ficha → Texto extraído / Último procesamiento |
| Roles / permisos requeridos | OCR permitido por política y licencia |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Idioma; umbral; máximo páginas; timeout |
| Validaciones | Solo PDF; conteo letras/dígitos menor a umbral; idiomas instalados; límites por página/documento. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-081](05-catalogo-capturas.md#cap-081) |
| Implementación | [web/src/Documents.tsx](../../web/src/Documents.tsx); [internal/extraction/pdf.go](../../internal/extraction/pdf.go); [internal/libraries/ocr_workers.go](../../internal/libraries/ocr_workers.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestOCRDisabledAndNativeThreshold`, fuente [internal/extraction/pdf_test.go](../../internal/extraction/pdf_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Configurar OCR e idiomas.
2. Incorporar o reindexar PDF.
3. Revisar unidades marcadas OCR.
4. Si hay aviso OCR_DISABLED, comprobar regla y módulo.
5. Reindexar tras cambiar configuración.

<a id="fun-082"></a>

## FUN-082 — Comprender recuperación de cola tras reinicio

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Comprender recuperación de cola tras reinicio |
| Objetivo y resultado esperado | Recupera trabajos huérfanos con fencing y conserva decisiones finales. |
| Descripción y comportamiento especial | Persistir resultado fallido se reintenta con backoff 1–30 s sin repetir extracción/conversión; corte al apagar. Pruebas históricas, no reinicio ejecutado durante investigación. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | Automático; bloqueo exclusivo de estado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin formulario |
| Validaciones | Un solo proceso abre estado; tentativas incompletas se cierran; cancelados no se reactivan. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-082](05-catalogo-capturas.md#cap-082) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/runtime.go](../../internal/libraries/runtime.go); [internal/libraries/job_control.go](../../internal/libraries/job_control.go); [internal/libraries/preview_worker.go](../../internal/libraries/preview_worker.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestWorkerRecoversFailedOutcomeWriteWithoutRepeatingExtraction`, fuente [internal/libraries/outcome_recovery_test.go](../../internal/libraries/outcome_recovery_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-013](07-pendientes-validacion.md#val-013); [VAL-020](07-pendientes-validacion.md#val-020) |

**Procedimiento:**

1. Ante reinicio real del servicio revisar Procesamiento.
2. Identificar PROCESS_RESTARTED si había trabajos activos.
3. Seguir retry_wait o fallo al agotar intentos.
4. Revisar materializaciones y previews pendientes.

<a id="fun-083"></a>

## FUN-083 — Verificar integridad administrada

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Procesamiento — Verificar integridad administrada |
| Objetivo y resultado esperado | Compara existencia/hash de originales administrados y conserva evidencia de cambio. |
| Descripción y comportamiento especial | No adopta automáticamente bytes alterados como aprobados; vinculados y administrados tienen políticas diferentes. |
| Ubicación | [PAN-036](02-pantallas-y-permisos.md#pan-036) — Biblioteca → Procesamiento |
| Roles / permisos requeridos | Automático; acceso físico al destino; permitido incluso solo lectura |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Raíz; sin modificaciones del original |
| Validaciones | Error individual no bloquea otros archivos; fallo de raíz completa distinto de fallo puntual. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | integrity_status verified / changed; availability available / missing |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-083](05-catalogo-capturas.md#cap-083) |
| Implementación | [web/src/Processing.tsx](../../web/src/Processing.tsx); [internal/libraries/managed_integrity.go](../../internal/libraries/managed_integrity.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestManagedIntegrityContinuesAfterUnreadableFile`, fuente [internal/libraries/managed_integrity_recovery_test.go](../../internal/libraries/managed_integrity_recovery_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-012](07-pendientes-validacion.md#val-012) |

**Procedimiento:**

1. Configurar destino administrado.
2. Consultar verificación periódica/manual.
3. Si aparece original cambiado/no disponible, abrir ficha y expediente.
4. Revisar avisos y trazabilidad.
5. Recuperar original mediante soporte autorizado.

<a id="fun-084"></a>

## FUN-084 — Configurar reglas globales de archivos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar reglas globales de archivos |
| Objetivo y resultado esperado | Define admisión y extracción por separado para cinco formatos. |
| Descripción y comportamiento especial | Defaults almacenar cinco/indexar PDF,256 MiB/rechazar desconocidos/conservar no indexables; guardar no elimina texto existente. |
| Ubicación | [PAN-042](02-pantallas-y-permisos.md#pan-042) — /admin/settings (alias /admin/files) |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Almacenar; indexar; máximo archivo/indexación; desconocidos; no indexables |
| Validaciones | Todo indexable debe almacenarse; tamaños 1–4096 MiB, índice ≤ almacenamiento; listas sin duplicados. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-084](05-catalogo-capturas.md#cap-084) |
| Implementación | [web/src/SettingsNavigation.tsx](../../web/src/SettingsNavigation.tsx); [web/src/FileSettings.tsx](../../web/src/FileSettings.tsx); [internal/libraries/file_settings.go](../../internal/libraries/file_settings.go); [internal/documentformat/policy.go](../../internal/documentformat/policy.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestFileRulesAuthorizationVersioningAndWorkerGate`, fuente [internal/libraries/document_formats_test.go](../../internal/libraries/document_formats_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Ajustes.
2. Archivos.
3. Marcar formatos a almacenar/indexar.
4. Definir límites y políticas.
5. Guardar reglas de archivos.
6. Revisar valores efectivos de bibliotecas.

<a id="fun-085"></a>

## FUN-085 — Configurar herencia de reglas de archivo

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar herencia de reglas de archivo |
| Objetivo y resultado esperado | Overrides por campo; null hereda y lista vacía prohíbe deliberadamente. |
| Descripción y comportamiento especial | Habilitar indexación puede encolar documentos antes solo almacenados; no reprocesa siempre todo contenido previo. |
| Ubicación | [PAN-026](02-pantallas-y-permisos.md#pan-026) — Biblioteca → Configuración → reglas de archivos |
| Roles / permisos requeridos | libraries.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Heredar por campo; formatos; límites; políticas |
| Validaciones | Revisión global y local vigentes; índice heredado se intersecta con almacenamiento; límite heredado se recorta a archivo. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-085](05-catalogo-capturas.md#cap-085) |
| Implementación | [web/src/FileSettings.tsx](../../web/src/FileSettings.tsx); [internal/libraries/file_settings.go](../../internal/libraries/file_settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestNewExtractorsManagedSearchAndFirstIndexAfterPolicyChange`, fuente [internal/libraries/document_extractors_test.go](../../internal/libraries/document_extractors_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configuración de biblioteca.
2. Reglas de archivos.
3. Desmarcar herencia solo en campos a personalizar.
4. Ajustar valores.
5. Guardar reglas de archivos.
6. Revisar efectivo.

<a id="fun-086"></a>

## FUN-086 — Interpretar rechazo de formato o contenido

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Interpretar rechazo de formato o contenido |
| Objetivo y resultado esperado | Distingue bloqueo de seguridad, formato desconocido, mismatch, límite y no indexable. |
| Descripción y comportamiento especial | store_text acepta únicamente texto validado con extensión desconocida; no abre excepción para ZIP genérico. PrinterSettings Office solo datos declarados, nunca ejecución. |
| Ubicación | [PAN-042](02-pantallas-y-permisos.md#pan-042) — /admin/settings (alias /admin/files) |
| Roles / permisos requeridos | Reglas efectivas; sin permiso especial |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre/bytes/tamaño del archivo |
| Validaciones | Binarios/scripts/macros/comprimidos bloqueados siempre; Office ZIP permitido solo si paquete válido seguro. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-086](05-catalogo-capturas.md#cap-086) |
| Implementación | [web/src/SettingsNavigation.tsx](../../web/src/SettingsNavigation.tsx); [web/src/FileSettings.tsx](../../web/src/FileSettings.tsx); [internal/documentformat/format.go](../../internal/documentformat/format.go); [internal/documentformat/office.go](../../internal/documentformat/office.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestDisguisedAndUnsafeFilesRejected`, fuente [internal/documentformat/format_test.go](../../internal/documentformat/format_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-019](07-pendientes-validacion.md#val-019) |

**Procedimiento:**

1. Ante carga/recorrido omitido leer motivo.
2. Comprobar extensión y formato real.
3. Revisar política de biblioteca.
4. Corregir origen o regla autorizada.
5. Volver a incorporar si es admisible.

<a id="fun-087"></a>

## FUN-087 — Configurar automatización y prioridad manual

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar automatización y prioridad manual |
| Objetivo y resultado esperado | Controla elegibilidad y prioridad de trabajos nuevos. |
| Descripción y comportamiento especial | Desactivar automático conserva pendientes; trabajo manual sigue posible si pausa general no está activa. Verificación de integridad administrada continúa. |
| Ubicación | [PAN-044](02-pantallas-y-permisos.md#pan-044) — /admin/advanced |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Procesamiento automático; Reprocesar documentos modificados; Priorizar trabajos manuales |
| Validaciones | Validación de revisión; trabajos en curso conservan parámetros. |
| Acciones disponibles | Guardar configuración avanzada |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-087](05-catalogo-capturas.md#cap-087) |
| Implementación | [web/src/AdvancedSettings.tsx](../../web/src/AdvancedSettings.tsx); [internal/libraries/advanced_settings.go](../../internal/libraries/advanced_settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAdvancedQueueAutomaticAndManualPriority`, fuente [internal/libraries/advanced_settings_test.go](../../internal/libraries/advanced_settings_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Ajustes.
2. Automatización, OCR y vigilancia.
3. Desplegar Procesamiento automático.
4. Ajustar checks.
5. Pulsar Guardar configuración avanzada.
6. Consultar cola.

<a id="fun-088"></a>

## FUN-088 — Configurar parámetros OCR globales

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar parámetros OCR globales |
| Objetivo y resultado esperado | Define condiciones de reconocimiento de PDF. |
| Descripción y comportamiento especial | Default umbral32, páginas1000, idioma spa, timeout120; conservar texto viejo al cambiar reglas; configuración no instala idiomas. |
| Ubicación | [PAN-044](02-pantallas-y-permisos.md#pan-044) — /admin/advanced |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Habilitar OCR; OCR automático; Idioma; Umbral; Máximo páginas; Tiempo por operación |
| Validaciones | Idiomas Español/Inglés/ambos; umbral y páginas 1–10000; timeout 5–300 s; datos de idioma deben existir. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-088](05-catalogo-capturas.md#cap-088) |
| Implementación | [web/src/AdvancedSettings.tsx](../../web/src/AdvancedSettings.tsx); [internal/libraries/advanced_settings.go](../../internal/libraries/advanced_settings.go); [internal/extraction/pdf.go](../../internal/extraction/pdf.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAdvancedInheritanceValidationAndPersistence`, fuente [internal/libraries/advanced_settings_test.go](../../internal/libraries/advanced_settings_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Automatización, OCR y vigilancia.
2. Reconocimiento de texto (OCR).
3. Elegir checks/idioma.
4. Definir umbral, páginas y timeout.
5. Guardar configuración avanzada.
6. Reindexar casos necesarios.

<a id="fun-089"></a>

## FUN-089 — Configurar vigilancia y archivos ignorados

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar vigilancia y archivos ignorados |
| Objetivo y resultado esperado | Controla watcher/omisiones y conserva históricos. |
| Descripción y comportamiento especial | $RECYCLE.BIN, System Volume Information y atributo Sistema Windows siempre excluidos; ignorar ocultos no es permiso de acceso. Periodicidad raíz sigue activa aunque watcher apagado. |
| Ubicación | [PAN-044](02-pantallas-y-permisos.md#pan-044) — /admin/advanced |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Vigilancia de cambios; Office temporales; ocultos; temporales; extensiones |
| Validaciones | Hasta 32 extensiones de 2–16 caracteres que inician punto, sin rutas/comodines. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-089](05-catalogo-capturas.md#cap-089) |
| Implementación | [web/src/AdvancedSettings.tsx](../../web/src/AdvancedSettings.tsx); [internal/libraries/advanced_settings.go](../../internal/libraries/advanced_settings.go); [internal/libraries/watch_attributes_windows.go](../../internal/libraries/watch_attributes_windows.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAdvancedWatcherIgnoresWithoutMarkingDocumentsMissing`, fuente [internal/libraries/advanced_settings_test.go](../../internal/libraries/advanced_settings_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Watcher y archivos ignorados.
2. Elegir checks.
3. Escribir extensiones separadas por comas.
4. Guardar.
5. Revisar omisiones en Procesamiento.

<a id="fun-090"></a>

## FUN-090 — Sobrescribir reglas avanzadas por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Sobrescribir reglas avanzadas por biblioteca |
| Objetivo y resultado esperado | Herencia granular con overrides locales y revisiones global/local/previews. |
| Descripción y comportamiento especial | Presupuesto de workers, reintentos, caché y convertidor no son por biblioteca. Desactivar previews global prevalece sobre permiso local. |
| Ubicación | [PAN-027](02-pantallas-y-permisos.md#pan-027) — Biblioteca → Configuración → reglas avanzadas |
| Roles / permisos requeridos | libraries.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Heredar por parámetro; automatización/OCR/watcher/previews |
| Validaciones | Mismas validaciones que global; null vuelve a heredar. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-090](05-catalogo-capturas.md#cap-090) |
| Implementación | [web/src/AdvancedSettings.tsx](../../web/src/AdvancedSettings.tsx); [internal/libraries/advanced_settings.go](../../internal/libraries/advanced_settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestAdvancedPreviewLibraryOverride`, fuente [internal/libraries/advanced_settings_test.go](../../internal/libraries/advanced_settings_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Configuración.
2. Reglas avanzadas de esta biblioteca.
3. Elegir qué campos Heredar.
4. Cambiar únicamente excepciones necesarias.
5. Guardar configuración avanzada.
6. Revisar efectivo.

<a id="fun-091"></a>

## FUN-091 — Consultar recursos y configurar concurrencia

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Consultar recursos y configurar concurrencia |
| Objetivo y resultado esperado | Controla operaciones de contenido simultáneas compartidas por bibliotecas. |
| Descripción y comportamiento especial | Auto conserva capacidad para otras apps. Cuatro workers de reconciliación aparte. Más procesos que CPU no implica más núcleos ni garantiza más velocidad. |
| Ubicación | [PAN-043](02-pantallas-y-permisos.md#pan-043) — /admin/processing |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Modo Automático/Manual; extracción directa; OCR; límite general |
| Validaciones | Manual: Máximo de extracción directa 1–8, Máximo de OCR 1–4, Límite general de contenido 1–8 e igual o mayor que cada límite individual. No representan porcentaje de CPU. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-091](05-catalogo-capturas.md#cap-091) |
| Implementación | [web/src/ProcessingSettings.tsx](../../web/src/ProcessingSettings.tsx); [web/src/ProcessingPolicy.tsx](../../web/src/ProcessingPolicy.tsx); [internal/libraries/processing_configuration.go](../../internal/libraries/processing_configuration.go); [internal/extraction/resources.go](../../internal/extraction/resources.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestConservativeResourceSelection`, fuente [internal/extraction/resources_test.go](../../internal/extraction/resources_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-012](07-pendientes-validacion.md#val-012); [VAL-017](07-pendientes-validacion.md#val-017) |

**Procedimiento:**

1. Abrir Ajustes.
2. Procesamiento y recursos.
3. Leer CPU/memoria detectadas.
4. Elegir Automático o Manual.
5. En Manual definir procesos.
6. Guardar procesamiento.
7. Revisar Configuración vigente.

<a id="fun-092"></a>

## FUN-092 — Pausar y reanudar procesamiento general

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Pausar y reanudar procesamiento general |
| Objetivo y resultado esperado | Pausa persistida impide iniciar trabajos nuevos y conserva cola. |
| Descripción y comportamiento especial | No modificar el fichero de DB para pausar. La pausa sobrevive al reinicio. |
| Ubicación | [PAN-043](02-pantallas-y-permisos.md#pan-043) — /admin/processing |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Pausar procesamiento / Reanudar procesamiento |
| Validaciones | No cancela automáticamente trabajos ya activos; lectura/búsqueda siguen posibles según licencia. |
| Acciones disponibles | Pausar procesamiento |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-092](05-catalogo-capturas.md#cap-092) |
| Implementación | [web/src/ProcessingSettings.tsx](../../web/src/ProcessingSettings.tsx); [web/src/ProcessingPolicy.tsx](../../web/src/ProcessingPolicy.tsx); [internal/libraries/processing_policy.go](../../internal/libraries/processing_policy.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPauseAllowsActiveDocumentToFinishAndRemainSearchable`, fuente [internal/libraries/processing_policy_test.go](../../internal/libraries/processing_policy_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Procesamiento y recursos.
2. Cola y límites de procesamiento.
3. Pulsar Pausar procesamiento.
4. Revisar aviso de trabajos activos.
5. Cuando proceda pulsar Reanudar procesamiento.

<a id="fun-093"></a>

## FUN-093 — Configurar intentos y tiempos documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar intentos y tiempos documentales |
| Objetivo y resultado esperado | Fija política al primer intento de cada trabajo. |
| Descripción y comportamiento especial | Solo transitorios se reintentan; espera duplica hasta1 h. Tiempo incluye lectura, extracción, esperaOCR y publicación; recorridos aparte. Un archivo fallido no bloquea cola. |
| Ubicación | [PAN-043](02-pantallas-y-permisos.md#pan-043) — /admin/processing |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Máximo de intentos; Espera inicial entre intentos; Tiempo máximo por documento |
| Validaciones | Intentos1–5; espera1–3600 s; timeout10–3600 s; defaults5/2/3600. |
| Acciones disponibles | Guardar límites de procesamiento |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-093](05-catalogo-capturas.md#cap-093) |
| Implementación | [web/src/ProcessingSettings.tsx](../../web/src/ProcessingSettings.tsx); [web/src/ProcessingPolicy.tsx](../../web/src/ProcessingPolicy.tsx); [internal/libraries/processing_policy.go](../../internal/libraries/processing_policy.go); [internal/libraries/runtime.go](../../internal/libraries/runtime.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestWorkerTimeoutDoesNotBlockOtherDocuments`, fuente [internal/libraries/processing_policy_test.go](../../internal/libraries/processing_policy_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Cola y límites de procesamiento.
2. Definir intentos, espera y tiempo.
3. Pulsar Guardar límites de procesamiento.
4. Consultar errores de trabajos nuevos.

<a id="fun-094"></a>

## FUN-094 — Configurar previews, convertidor y caché

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Configurar previews, convertidor y caché |
| Objetivo y resultado esperado | Política global de previews derivadas y limpieza. |
| Descripción y comportamiento especial | UI rotula MB en algunos textos; backend utiliza MiB. Limpieza por capacidad siempre limita caché; no usar una raíz vinculada sobre cache privada. |
| Ubicación | [PAN-045](02-pantallas-y-permisos.md#pan-045) — /admin/previews |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Habilitación; límite caché; días; fuente; limpieza; LibreOffice |
| Validaciones | Caché1–20480 MiB, días1–365, fuente1–512 MiB;ruta absoluta ejecutable sin argumentos. |
| Acciones disponibles | Guardar vistas previas; Actualizar estado |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-094](05-catalogo-capturas.md#cap-094) |
| Implementación | [web/src/PreviewSettings.tsx](../../web/src/PreviewSettings.tsx); [internal/libraries/preview_settings.go](../../internal/libraries/preview_settings.go); [internal/previews/converter.go](../../internal/previews/converter.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPreviewInvalidationLimitsAndPolicy`, fuente [internal/libraries/previews_test.go](../../internal/libraries/previews_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-013](07-pendientes-validacion.md#val-013); [VAL-018](07-pendientes-validacion.md#val-018) |

**Procedimiento:**

1. Abrir Vistas previas y caché.
2. Revisar convertidor disponible.
3. Ajustar política y ruta absoluta si necesaria.
4. Pulsar Guardar vistas previas.
5. Pulsar Actualizar estado.
6. Probar ficha autorizada en futura validación.

<a id="fun-095"></a>

## FUN-095 — Vaciar caché de vistas previas

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Configuración — Vaciar caché de vistas previas |
| Objetivo y resultado esperado | Elimina derivados privados de preview y invalida pendientes. |
| Descripción y comportamiento especial | No borra originales, índices, OCR ni auditoría; conversión activa no puede republicar derivado ya invalidado. |
| Ubicación | [PAN-045](02-pantallas-y-permisos.md#pan-045) — /admin/previews |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Confirmación de vaciado |
| Validaciones | Revisión/confirmación; no atraviesa enlaces ni raíces externas. |
| Acciones disponibles | Vaciar caché de vistas previas |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-095](05-catalogo-capturas.md#cap-095) |
| Implementación | [web/src/PreviewSettings.tsx](../../web/src/PreviewSettings.tsx); [internal/libraries/previews.go](../../internal/libraries/previews.go); [internal/libraries/preview_worker.go](../../internal/libraries/preview_worker.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestPreviewLRUExpiryAndClearDuringGeneration`, fuente [internal/libraries/previews_test.go](../../internal/libraries/previews_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-020](07-pendientes-validacion.md#val-020) |

**Procedimiento:**

1. Abrir Vistas previas y caché.
2. Leer alcance de borrado.
3. Marcar confirmación.
4. Pulsar Vaciar caché de vistas previas.
5. Revisar consumo.
6. Abrir fichas para regeneración bajo reglas.

<a id="fun-096"></a>

## FUN-096 — Habilitar acceso LAN o regresar a local

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Red — Habilitar acceso LAN o regresar a local |
| Objetivo y resultado esperado | Cambia listener IPv4 127.0.0.1 ↔0.0.0.0 y persiste sin reiniciar indexación. |
| Descripción y comportamiento especial | El cliente remoto pierde conexión al volver a local. LAN HTTP sin cifrado; Windows depende regla preparada por instalador, Linux/macOS diagnóstico no crea regla. |
| Ubicación | [PAN-046](02-pantallas-y-permisos.md#pan-046) — /admin/network |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Solo este equipo / Permitir acceso desde la red local |
| Validaciones | Puerto actual libre y config persistible; si falla se restaura modo previo. HTTPS/proxy personalizado no se sobrescribe desde UI. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-096](05-catalogo-capturas.md#cap-096) |
| Implementación | [web/src/NetworkSettings.tsx](../../web/src/NetworkSettings.tsx); [internal/network/service.go](../../internal/network/service.go); [internal/httpapi/network_settings.go](../../internal/httpapi/network_settings.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestLocalLANPersistsAndReturnsWithoutRestartingApplication`, fuente [internal/network/service_test.go](../../internal/network/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-002](07-pendientes-validacion.md#val-002); [VAL-009](07-pendientes-validacion.md#val-009) |

**Procedimiento:**

1. Abrir Ajustes.
2. Acceso y red.
3. Leer modo actual y puerto.
4. Elegir Permitir acceso desde la red local.
5. Guardar acceso.
6. Abrir URL sugerida desde otro equipo.
7. Para cerrar acceso, elegir Solo este equipo y guardar.

<a id="fun-097"></a>

## FUN-097 — Consultar diagnóstico de acceso y firewall

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Red — Consultar diagnóstico de acceso y firewall |
| Objetivo y resultado esperado | Diferencia modo persistido/actual, interfaz y posibles restricciones de firewall. |
| Descripción y comportamiento especial | Windows inspección de listeners evita falsos fallos de dial al conmutar; regla Domain/Private LocalSubnet no permite Public ni vence GPO. |
| Ubicación | [PAN-046](02-pantallas-y-permisos.md#pan-046) — /admin/network |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Ejecutar diagnóstico; Interfaces IPv4 |
| Validaciones | Sugerencias por interfaces reales IPv4; no garantiza conectividad; virtual/VPN pueden aparecer. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-097](05-catalogo-capturas.md#cap-097) |
| Implementación | [web/src/NetworkSettings.tsx](../../web/src/NetworkSettings.tsx); [internal/network/firewall.go](../../internal/network/firewall.go); [internal/network/portcheck_windows.go](../../internal/network/portcheck_windows.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestWindowsFirewallEffectiveRuleDiagnostics`, fuente [internal/network/firewall_report_test.go](../../internal/network/firewall_report_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-002](07-pendientes-validacion.md#val-002); [VAL-009](07-pendientes-validacion.md#val-009) |

**Procedimiento:**

1. Abrir Acceso y red.
2. Actualizar diagnóstico.
3. Revisar listener, puerto y URLs.
4. Abrir Interfaces IPv4 disponibles.
5. Revisar mensaje firewall.
6. Si falla remoto, consultar procedimiento SO en 03.

<a id="fun-098"></a>

## FUN-098 — Consultar estadísticas del Dashboard

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Observabilidad — Consultar estadísticas del Dashboard |
| Objetivo y resultado esperado | Resumen administrativo de toda la instalación. |
| Descripción y comportamiento especial | 30 días; hoy/semana lunes/mes según zona del servicio; latencia de completados24 h excluye espera inicial pero incluye ejecución/esperasOCR/publicación. |
| Ubicación | [PAN-041](02-pantallas-y-permisos.md#pan-041) — /admin/dashboard |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin filtro individual; actualización |
| Validaciones | Counts no otorgan lectura documental; datos basados en registros vigentes; cero/sin muestra diferenciados. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-098](05-catalogo-capturas.md#cap-098) |
| Implementación | [web/src/Dashboard.tsx](../../web/src/Dashboard.tsx); [internal/libraries/dashboard.go](../../internal/libraries/dashboard.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestObservabilityDashboardCountsDatesAndPublishedContent`, fuente [internal/libraries/observability_test.go](../../internal/libraries/observability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Dashboard.
2. Revisar KPIs de registros/texto/OCR/cola.
3. Consultar actividad y distribución.
4. Leer duplicados/caché/errores.
5. Actualizar o esperar sondeo.

<a id="fun-099"></a>

## FUN-099 — Consultar gráficos, tablas accesibles y detalle por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Observabilidad — Consultar gráficos, tablas accesibles y detalle por biblioteca |
| Objetivo y resultado esperado | Gráficos Chart.js locales acompañados por datos tabulares accesibles. |
| Descripción y comportamiento especial | No ApexCharts ni dependencia de CDN; valores no inventados. Captura actual de test con datos vacíos requiere sustitución. |
| Ubicación | [PAN-041](02-pantallas-y-permisos.md#pan-041) — /admin/dashboard |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Ver datos de…; Ver definiciones; Mostrar más |
| Validaciones | Valores vacíos se representan; fecha sin actividad incluye cero. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-099](05-catalogo-capturas.md#cap-099) |
| Implementación | [web/src/Dashboard.tsx](../../web/src/Dashboard.tsx); [web/src/DashboardChart.tsx](../../web/src/DashboardChart.tsx); [internal/libraries/dashboard.go](../../internal/libraries/dashboard.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestObservabilityEmptyAndLibraryPagination`, fuente [internal/libraries/observability_test.go](../../internal/libraries/observability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-011](07-pendientes-validacion.md#val-011) |

**Procedimiento:**

1. Abrir Dashboard.
2. Leer línea diaria/donut formatos/barras estados/donut orígenes.
3. Desplegar Ver datos de cada gráfico.
4. Abrir definiciones.
5. Paginar bibliotecas.

<a id="fun-100"></a>

## FUN-100 — Filtrar errores y consultar contexto

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Observabilidad — Filtrar errores y consultar contexto |
| Objetivo y resultado esperado | Lista diagnósticos sanitizados y repetición por firma. |
| Descripción y comportamiento especial | No equivale a auditoría. Las incidencias repetidas se agrupan con contador y última fecha; nuevas apariciones reabren revisadas. |
| Ubicación | [PAN-047](02-pantallas-y-permisos.md#pan-047) — /admin/errors |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Desde; Hasta; Módulo; Gravedad; Estado |
| Validaciones | Desde≤Hasta; hasta incluye día mediante límite exclusivo del siguiente; módulo/severidad/estado válidos. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-100](05-catalogo-capturas.md#cap-100) |
| Implementación | [web/src/SystemErrors.tsx](../../web/src/SystemErrors.tsx); [internal/libraries/system_errors.go](../../internal/libraries/system_errors.go); [internal/diagnostics/diagnostics.go](../../internal/diagnostics/diagnostics.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestObservabilityErrorLifecyclePrivacyPaginationAndRetention`, fuente [internal/libraries/observability_test.go](../../internal/libraries/observability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Sistema · Errores.
2. Definir filtros.
3. Aplicar filtros.
4. Abrir Contexto en incidencia.
5. Usar Mostrar más.

<a id="fun-101"></a>

## FUN-101 — Marcar diagnóstico y copiar información de soporte

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Observabilidad — Marcar diagnóstico y copiar información de soporte |
| Objetivo y resultado esperado | Produce reporte sanitizado con IDs/códigos/contexto permitido y organiza atención. |
| Descripción y comportamiento especial | Marcar revisado no repara archivo ni reintenta trabajo. Windows EventLog inicial aún requiere revisión nativa de contenido (VAL-016). |
| Ubicación | [PAN-047](02-pantallas-y-permisos.md#pan-047) — /admin/errors |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Estado revisado/pendiente; copia |
| Validaciones | Control de revisión; no logs crudos, nombres/rutas libres ni cuerpos HTTP en reporte. |
| Acciones disponibles | Copiar información de diagnóstico |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-101](05-catalogo-capturas.md#cap-101) |
| Implementación | [web/src/SystemErrors.tsx](../../web/src/SystemErrors.tsx); [internal/libraries/system_errors.go](../../internal/libraries/system_errors.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestObservabilityHTTPAuthorizationCSRFAndSanitizedFailure`, fuente [internal/httpapi/observability_test.go](../../internal/httpapi/observability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-016](07-pendientes-validacion.md#val-016) |

**Procedimiento:**

1. Abrir incidencia.
2. Pulsar Copiar información de diagnóstico.
3. Si clipboard falla, copiar texto alternativo.
4. Marcar revisado o pendiente según seguimiento.
5. Revisar estado.

<a id="fun-102"></a>

## FUN-102 — Configurar retención de diagnósticos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Observabilidad — Configurar retención de diagnósticos |
| Objetivo y resultado esperado | Controla tamaño/edad de registro operacional. |
| Descripción y comportamiento especial | Poda no afecta audit_events; guardar puede eliminar diagnósticos antiguos de inmediato, no recuperar luego desde UI. |
| Ubicación | [PAN-047](02-pantallas-y-permisos.md#pan-047) — /admin/errors |
| Roles / permisos requeridos | system.configure |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Días; máximo de eventos |
| Validaciones | 1–365 d y100–50000; defaults30/5000; revisión obligatoria. |
| Acciones disponibles | Guardar retención |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-102](05-catalogo-capturas.md#cap-102) |
| Implementación | [web/src/SystemErrors.tsx](../../web/src/SystemErrors.tsx); [internal/diagnostics/diagnostics.go](../../internal/diagnostics/diagnostics.go); [db/migrations/0014_observability.up.sql](../../db/migrations/0014_observability.up.sql) |
| Verificación | Verificado en pruebas (escenario parcial): `TestObservabilityErrorLifecyclePrivacyPaginationAndRetention`, fuente [internal/libraries/observability_test.go](../../internal/libraries/observability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Abrir Errores y diagnósticos.
2. Desplegar Retención.
3. Indicar días y máximo.
4. Pulsar Guardar retención.
5. Revisar lista después de poda.

<a id="fun-103"></a>

## FUN-103 — Consultar estado, módulos y datos de licencia

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Consultar estado, módulos y datos de licencia |
| Objetivo y resultado esperado | Distingue licencia activa, gracia, vencida, revocada/desactivada, inválida o sin activar. |
| Descripción y comportamiento especial | Perpetua sin vencimiento de uso; mantenimiento/versiones separado. Suscripción tolerancia15 d. Identidad privada jamás se muestra; datos públicos no necesitan exponerse completos en captura. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Actualizar estado; identificación soporte |
| Validaciones | Módulos linked/managed/OCR/expedientes/review y releases/maintenance no equivalen a permiso de usuario. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-103](05-catalogo-capturas.md#cap-103) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/client.go](../../internal/licensing/client.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestSubscriptionExclusiveBoundariesClockAndPerpetualOffline`, fuente [internal/licensing/contract_test.go](../../internal/licensing/contract_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Abrir Licencia.
2. Revisar estado y fechas.
3. Leer módulos y restricciones.
4. Revisar último contacto/diagnóstico.
5. Desplegar identificación de soporte.
6. Actualizar estado.

<a id="fun-104"></a>

## FUN-104 — Activar licencia en línea

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Activar licencia en línea |
| Objetivo y resultado esperado | Cliente solicita challenge, prueba identidad y acepta licencia firmada vinculada a instalación. |
| Descripción y comportamiento especial | Campo visible intencional, no contraseña. No incluir ninguna clave comercial real en documentación; aquí servidor es proyecto externo no inspeccionado. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage; servidor y confianza configurados |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Operación activar; Clave de licencia en texto |
| Validaciones | HTTPS, claves públicas por kid, binding instalación/huella/clave pública; requestID idempotente;timeout15sHTTP. |
| Acciones disponibles | Activación en línea |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-104](05-catalogo-capturas.md#cap-104) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/network.go](../../internal/licensing/network.go); [internal/licensing/contract.go](../../internal/licensing/contract.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestOnlineIdempotentRestartRevocationAndConfirmedDeactivation`, fuente [internal/licensing/online_development_test.go](../../internal/licensing/online_development_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Abrir Licencia.
2. Elegir activación.
3. Pegar Clave de licencia visible.
4. Pulsar Activación en línea.
5. Esperar respuesta.
6. Revisar estado/módulos.

<a id="fun-105"></a>

## FUN-105 — Renovar estado y retomar solicitud de licencia

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Renovar estado y retomar solicitud de licencia |
| Objetivo y resultado esperado | Reutiliza UUID y estado operativo tras fallos/reinicio. |
| Descripción y comportamiento especial | Actualizarestado UI relee estado local; renovación solicita servidor. Fallo transitorio no elimina estado firmado ni extiende artificialmente gracia. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage; activación existente |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Renovar/validar; solicitud pendiente |
| Validaciones | No seguir redirecciones; firma/respuesta inválida no reemplaza licencia válida. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-105](05-catalogo-capturas.md#cap-105) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/network.go](../../internal/licensing/network.go); [internal/licensing/client.go](../../internal/licensing/client.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestTransientFailuresKeepSignedStateAndGraceAfterRestart`, fuente [internal/licensing/availability_test.go](../../internal/licensing/availability_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Abrir Licencia.
2. Elegir operación de renovación.
3. Enviar.
4. Si resultado incierto, seleccionar en Retomar solicitud pendiente.
5. Solo preparar otra solicitud para una operación nueva.
6. Revisar último contacto.

<a id="fun-106"></a>

## FUN-106 — Desactivar licencia en línea

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Desactivar licencia en línea |
| Objetivo y resultado esperado | Desactivación confirmada por proveedor coloca instalación en solo lectura. |
| Descripción y comportamiento especial | No elimina biblioteca/usuarios/originales. Se conserva información del producto contratado con distintivos suspendidos; no confundir con licencia todavía habilitada. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage; activación actual |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Operación desactivar; confirmación |
| Validaciones | Confirmación explícita; requestID estable; mantiene evidencia de activación. |
| Acciones disponibles | Desactivación en línea |
| Estados relacionados | revoked / deactivated; read-only |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-106](05-catalogo-capturas.md#cap-106) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/network.go](../../internal/licensing/network.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestOnlineIdempotentRestartRevocationAndConfirmedDeactivation`, fuente [internal/licensing/online_development_test.go](../../internal/licensing/online_development_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Abrir Licencia.
2. Elegir desactivación.
3. Leer efecto sobre modificaciones.
4. Marcar confirmación.
5. Pulsar Desactivación en línea.
6. Revisar estado y módulos suspendidos.

<a id="fun-107"></a>

## FUN-107 — Generar solicitud de licencia sin conexión

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Generar solicitud de licencia sin conexión |
| Objetivo y resultado esperado | Genera sobre firmado localmente y registra artefacto privado. |
| Descripción y comportamiento especial | Generar desactivación offline no libera por sí solo plaza en servidor; queda pendiente hasta confirmación. Flujo del portal externo no está inspeccionado. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Activación/renovación/desactivación offline; requestID |
| Validaciones | Identidad instalada íntegra; action/requestID válidos. |
| Acciones disponibles | Solicitar activación, Solicitar renovación o Solicitar desactivación según la acción |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-107](05-catalogo-capturas.md#cap-107) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/artifacts.go](../../internal/licensing/artifacts.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestOfflineActivationRenewalAndPendingDeactivation`, fuente [internal/licensing/online_development_test.go](../../internal/licensing/online_development_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Abrir Operar sin conexión.
2. Pulsar Solicitar activación, Solicitar renovación o Solicitar desactivación según la acción.
3. Generar solicitud.
4. Descargar archivo.
5. Transferir al procedimiento autorizado del proveedor.
6. Conservar referencia para importar respuesta.

<a id="fun-108"></a>

## FUN-108 — Importar respuesta firmada y consultar artefactos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Importar respuesta firmada y consultar artefactos |
| Objetivo y resultado esperado | Valida firma y binding antes de reemplazar estado; mantiene últimos artefactos. |
| Descripción y comportamiento especial | No editar el archivo firmado ni copiar identidad privada de otro equipo; datos sensibles del artefacto no incluir en manual. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | license.manage |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Archivo .lic; historial de solicitudes/respuestas |
| Validaciones | Máximo64 KiB; contrato estricto, firma Ed25519, revisión no regresiva y ambiente correcto. |
| Acciones disponibles | Importar licencia |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-108](05-catalogo-capturas.md#cap-108) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/artifacts.go](../../internal/licensing/artifacts.go); [internal/licensing/contract.go](../../internal/licensing/contract.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRevisionFloorsImportAtomicityAndOfflineProof`, fuente [internal/licensing/contract_test.go](../../internal/licensing/contract_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Obtener respuesta del proveedor.
2. En Licencia seleccionar archivo .lic.
3. Pulsar Importar licencia.
4. Comprobar estado.
5. Abrir historial de artefactos y descargar si se necesita soporte.

<a id="fun-109"></a>

## FUN-109 — Interpretar solo lectura y avisos de reloj/identidad

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Licenciamiento — Interpretar solo lectura y avisos de reloj/identidad |
| Objetivo y resultado esperado | Aplica política de lectura/escritura y pausa de trabajos sin gastar intentos por falta de módulo. |
| Descripción y comportamiento especial | Reloj efectivo monotónico evita prolongar tolerancia por retroceso; recuperar identidad requiere nueva activación y coordinación externa. |
| Ubicación | [PAN-048](02-pantallas-y-permisos.md#pan-048) — /admin/license |
| Roles / permisos requeridos | Operación documental según estado; administración de seguridad sigue habilitada |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin edición; diagnósticos |
| Validaciones | Revocada/vencida permiten consulta/descarga autorizadas; inválida/sin activar bloquean acceso documental; seguridad/recuperación disponibles. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-109](05-catalogo-capturas.md#cap-109) |
| Implementación | [web/src/License.tsx](../../web/src/License.tsx); [internal/licensing/policy.go](../../internal/licensing/policy.go); [internal/libraries/license.go](../../internal/libraries/license.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestH6WorkerPausesWithoutBurningAttemptsAndResumesAfterModuleRenewal`, fuente [internal/libraries/license_test.go](../../internal/libraries/license_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-014](07-pendientes-validacion.md#val-014) |

**Procedimiento:**

1. Leer banner de licencia.
2. Abrir Licencia.
3. Distinguir vencida/revocada de inválida/sin activar.
4. Resolver renovación/identidad con administrador.
5. Verificar módulos tras respuesta.

<a id="fun-110"></a>

## FUN-110 — Instalar en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Instalar en Windows |
| Objetivo y resultado esperado | Instala programa/servicio virtual, datos privados, regla LAN y accesos AIBID. |
| Descripción y comportamiento especial | Servicio inicialmente manual; configuración inicial necesita bootstrap antes de arranque. Revisión de código y empaquetado, pendiente sistema nativo. |
| Ubicación | [PAN-049](02-pantallas-y-permisos.md#pan-049) — Asistente NSIS; Inicio → AIBID |
| Roles / permisos requeridos | Administrador del SO |
| Condiciones previas | Paquete correcto; administrador; puerto18090 previsto; permisos y antivirus no bloquean. |
| Campos involucrados | Instalador .exe amd64; aceptación UAC |
| Validaciones | SO/arquitectura, dueño/marker/locks y symlinks/reparse points; incompatible se conserva sin forzar migración. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Asistente instalado / anulado |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-110](05-catalogo-capturas.md#cap-110) |
| Implementación | [packaging/windows/installer.nsi](../../packaging/windows/installer.nsi); [packaging/windows/manage.ps1](../../packaging/windows/manage.ps1) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-001](07-pendientes-validacion.md#val-001); [VAL-006](07-pendientes-validacion.md#val-006); [VAL-017](07-pendientes-validacion.md#val-017) |

**Procedimiento:**

1. Verificar Windows 10+64bits.
2. Ejecutar instalador AIBID.
3. Autorizar UAC.
4. Leer Bienvenida.
5. Instalar.
6. Revisar detalle y final.
7. Abrir Configurar AIBID.

<a id="fun-111"></a>

## FUN-111 — Crear administrador e iniciar en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Crear administrador e iniciar en Windows |
| Objetivo y resultado esperado | Crea cuenta inicial, marca habilitación, configura inicio automático demorado y abre navegador. |
| Descripción y comportamiento especial | Una instalación nueva usa NT SERVICE\AIBID. Las actualizaciones conservan el SID anterior según el apartado de migración de 03. Registro inicial y arranque nativo pendientes de validación Windows. |
| Ubicación | [PAN-049](02-pantallas-y-permisos.md#pan-049) — Asistente NSIS; Inicio → AIBID |
| Roles / permisos requeridos | Administrador del SO; consola elevada |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre de usuario; nombre visible; contraseña confirmada |
| Validaciones | Contraseña6–128; diagnóstico y migración deben completar; no credencial por defecto. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | service-enabled; bootstrap listo |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-111](05-catalogo-capturas.md#cap-111) |
| Implementación | [packaging/windows/installer.nsi](../../packaging/windows/installer.nsi); [packaging/windows/manage.ps1](../../packaging/windows/manage.ps1); [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-001](07-pendientes-validacion.md#val-001) |

**Procedimiento:**

1. Inicio.
2. AIBID.
3. Configurar AIBID.
4. Autorizar elevación.
5. Esperar doctor PDF/OCR.
6. Si falta cuenta, completar bootstrap local.
7. Esperar inicio.
8. Acceder a URL del navegador.

<a id="fun-112"></a>

## FUN-112 — Abrir aplicación instalada desde acceso directo

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Abrir aplicación instalada desde acceso directo |
| Objetivo y resultado esperado | Launcher abre navegador sin pedir administrador para uso ordinario. |
| Descripción y comportamiento especial | AIBID.exe launcher no es backend aibid-server.exe. No copia claves/config privada al directorio de acceso público. |
| Ubicación | [PAN-049](02-pantallas-y-permisos.md#pan-049) — Asistente NSIS; Inicio → AIBID |
| Roles / permisos requeridos | Usuario del SO; servicio ya configurado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Sin campos; launcher.json contiene URL pública y nombre de servicio AIBID; admite la identidad anterior únicamente para actualización |
| Validaciones | URL sin credenciales/comandos; HTTP solo loopback; HTTPS permitido configurado. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-112](05-catalogo-capturas.md#cap-112) |
| Implementación | [packaging/windows/installer.nsi](../../packaging/windows/installer.nsi); [packaging/windows/manage.ps1](../../packaging/windows/manage.ps1); [cmd/aibid-launcher/main_windows.go](../../cmd/aibid-launcher/main_windows.go); [internal/launcher/target.go](../../internal/launcher/target.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestBrowserTargetRejectsCommandsCredentialsAndNonlocalHTTP`, fuente [internal/launcher/target_test.go](../../internal/launcher/target_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. En Windows abrir AIBID en escritorio o Abrir AIBID en Inicio.
2. Esperar comprobación de servicio.
3. Leer aviso si no está listo.
4. Usar navegador abierto en URL configurada.

<a id="fun-113"></a>

## FUN-113 — Desinstalar en Windows conservando datos

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Desinstalar en Windows conservando datos |
| Objetivo y resultado esperado | Detiene servicio, elimina registro/regla y archivos empaquetados; conserva estado/documentos. |
| Descripción y comportamiento especial | Conservar es predeterminado. Una reinstalación retoma estado compatible; no prometer eliminación completa de archivos ajenos. |
| Ubicación | [PAN-050](02-pantallas-y-permisos.md#pan-050) — AIBID → Desinstalar / aplicaciones Windows |
| Roles / permisos requeridos | Administrador del SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Checkbox de datos desmarcado |
| Validaciones | Hash-manifest protege archivos modificados/desconocidos; validación de rutas antes de borrar. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Datos conservados |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-113](05-catalogo-capturas.md#cap-113) |
| Implementación | [packaging/windows/installer.nsi](../../packaging/windows/installer.nsi); [packaging/windows/manage.ps1](../../packaging/windows/manage.ps1) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-001](07-pendientes-validacion.md#val-001) |

**Procedimiento:**

1. Si se cambia equipo/licencia, desactivar antes si procede.
2. Abrir Desinstalar AIBID.
3. Dejar desmarcado Eliminar los datos de AIBID al desinstalar.
4. Confirmar.
5. Revisar resultado.

<a id="fun-114"></a>

## FUN-114 — Desinstalar y reiniciar datos internos en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Desinstalar y reiniciar datos internos en Windows |
| Objetivo y resultado esperado | Elimina metadata/usuarios/índices/extracción/identidad/config internos; originales y cargas físicas preservados. |
| Descripción y comportamiento especial | No ejecutar para documentar. Las cargas privadas físicamente conservadas pierden asociación al borrar DB. No libera activación remota. |
| Ubicación | [PAN-050](02-pantallas-y-permisos.md#pan-050) — AIBID → Desinstalar / aplicaciones Windows |
| Roles / permisos requeridos | Administrador del SO; autorización explícita |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Checkbox eliminar; segunda confirmación |
| Validaciones | erase-internal-data valida todas las rutas y exclusividad antes de borrar. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Instalación nueva requerida |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-114](05-catalogo-capturas.md#cap-114) |
| Implementación | [packaging/windows/installer.nsi](../../packaging/windows/installer.nsi); [internal/maintenance/erase.go](../../internal/maintenance/erase.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestErasePreservesPhysicalDocumentsAndAllowsFreshInstall`, fuente [internal/maintenance/erase_test.go](../../internal/maintenance/erase_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-001](07-pendientes-validacion.md#val-001) |

**Procedimiento:**

1. Respaldar por procedimiento validado si se necesitan datos.
2. Desactivar licencia si desea liberar activación.
3. Abrir Desinstalar.
4. Marcar eliminación de datos.
5. Leer segunda advertencia.
6. Confirmar solo si quiere reiniciar.
7. Revisar resultado.

<a id="fun-115"></a>

## FUN-115 — Instalar en Ubuntu con paquete de arquitectura correcta

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Instalar en Ubuntu con paquete de arquitectura correcta |
| Objetivo y resultado esperado | APT instala programa, cuenta local y servicio systemd. |
| Descripción y comportamiento especial | AMD64 no ejecuta nativamente en UbuntuARM del M1. Una instalación nueva usa cuenta aibid, unidad aibid.service y rutas aibid. Las actualizaciones conservan cuenta y datos anteriores, con la unidad nueva; véase migración en 03. |
| Ubicación | [PAN-051](02-pantallas-y-permisos.md#pan-051) — Terminal → APT; aibid-admin / aibid-uninstall |
| Roles / permisos requeridos | sudo/root |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Paquete .deb amd64 o arm64 |
| Validaciones | Ubuntu 24.04 objetivo; paquetes ARM/amd64 distintos; Poppler, Tesseractspa/eng, fuentes, systemd; LibreOffice recomendado. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Pendiente instalación nativa limpia |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-115](05-catalogo-capturas.md#cap-115) |
| Implementación | [packaging/ubuntu/postinst](../../packaging/ubuntu/postinst); [packaging/ubuntu/aibid-admin](../../packaging/ubuntu/aibid-admin); [scripts/build_installers.py](../../scripts/build_installers.py) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-003](07-pendientes-validacion.md#val-003); [VAL-017](07-pendientes-validacion.md#val-017) |

**Procedimiento:**

1. Consultar dpkg --print-architecture.
2. Seleccionar paquete correspondiente.
3. Ejecutar sudo apt install ./paquete.deb.
4. Revisar dependencias.
5. Ejecutar sudo aibid-admin.
6. Completar configuración inicial.

<a id="fun-116"></a>

## FUN-116 — Crear administrador y arrancar en Ubuntu

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Crear administrador y arrancar en Ubuntu |
| Objetivo y resultado esperado | Habilita marcador e inicio systemd tras bootstrap. |
| Descripción y comportamiento especial | aibid-admin selecciona la cuenta y datos detectados, y administra aibid.service. No usar permisos de usuario interactivo como prueba de lectura de la cuenta del servicio. |
| Ubicación | [PAN-051](02-pantallas-y-permisos.md#pan-051) — Terminal → APT; aibid-admin / aibid-uninstall |
| Roles / permisos requeridos | sudo/root; wrapper corre backend como cuenta servicio |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Nombre/contraseña bootstrap |
| Validaciones | Doctor valida herramientas e idiomas; cuenta de sistema separada sin login. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-116](05-catalogo-capturas.md#cap-116) |
| Implementación | [packaging/ubuntu/postinst](../../packaging/ubuntu/postinst); [packaging/ubuntu/aibid-admin](../../packaging/ubuntu/aibid-admin); [packaging/ubuntu/aibid.service](../../packaging/ubuntu/aibid.service) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-003](07-pendientes-validacion.md#val-003) |

**Procedimiento:**

1. Ejecutar sudo aibid-admin.
2. Revisar doctor.
3. Crear cuenta si no inicializada.
4. Esperar systemctl enable --now.
5. Consultar URL127.0.0.1:18090.
6. Activar licencia.

<a id="fun-117"></a>

## FUN-117 — Preparar e instalar bundle Ubuntu sin red

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Preparar e instalar bundle Ubuntu sin red |
| Objetivo y resultado esperado | Scripts reúnen cierre de dependencias y APT temporal local. |
| Descripción y comportamiento especial | No instala repositorio APT permanente. Procedimiento implementado pero prueba offline VM limpia sigue pendiente; no se ejecutó en esta tarea. |
| Ubicación | [PAN-051](02-pantallas-y-permisos.md#pan-051) — Terminal → APT; aibid-admin / aibid-uninstall |
| Roles / permisos requeridos | sudo/root; equipo preparador Ubuntu 24.04 misma arquitectura |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Paquete; carpeta nueva, admite espacios y Unicode |
| Validaciones | Arquitectura igual, carpeta no existente; admite espacios y Unicode; hashes verificados; un paquete AIBID. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-117](05-catalogo-capturas.md#cap-117) |
| Implementación | [packaging/ubuntu/postinst](../../packaging/ubuntu/postinst); [packaging/ubuntu/aibid-admin](../../packaging/ubuntu/aibid-admin); [packaging/ubuntu/prepare-offline.sh](../../packaging/ubuntu/prepare-offline.sh); [packaging/ubuntu/install-offline.sh](../../packaging/ubuntu/install-offline.sh) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-004](07-pendientes-validacion.md#val-004) |

**Procedimiento:**

1. En equipo con red ejecutar prepare-offline.sh con paquete y carpeta nueva.
2. Conservar SHA256SUMS por canal independiente.
3. Copiar bundle completo.
4. En VM sin red ejecutar sudo ./install.sh.
5. Después configurar con aibid-admin.

<a id="fun-118"></a>

## FUN-118 — Desinstalar en Ubuntu

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Desinstalar en Ubuntu |
| Objetivo y resultado esperado | Wrapper detiene/retira paquete; borrado interno opcional validado. |
| Descripción y comportamiento especial | apt remove/purge por sí mismos no equivalen a borrar DB/config. Borrado no elimina originales ni cargas privadas; licencia remota aparte. |
| Ubicación | [PAN-051](02-pantallas-y-permisos.md#pan-051) — Terminal → APT; aibid-admin / aibid-uninstall |
| Roles / permisos requeridos | sudo/root |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Conservar por defecto; texto ELIMINAR si decide borrado |
| Validaciones | prerm detiene servicio/check; postrm conserva datos incluso purge directo. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-118](05-catalogo-capturas.md#cap-118) |
| Implementación | [packaging/ubuntu/postinst](../../packaging/ubuntu/postinst); [packaging/ubuntu/aibid-admin](../../packaging/ubuntu/aibid-admin); [packaging/ubuntu/aibid-uninstall](../../packaging/ubuntu/aibid-uninstall); [packaging/ubuntu/prerm](../../packaging/ubuntu/prerm); [packaging/ubuntu/postrm](../../packaging/ubuntu/postrm) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-003](07-pendientes-validacion.md#val-003) |

**Procedimiento:**

1. Ejecutar sudo aibid-uninstall.
2. Leer preguntas de conservación.
3. Conservar datos o confirmar explícitamente ELIMINAR.
4. Revisar apt-get remove.
5. Consultar mensajes finales.

<a id="fun-119"></a>

## FUN-119 — Preparar dependencias e instalar en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Preparar dependencias e instalar en macOS |
| Objetivo y resultado esperado | Instala binario, cuenta _aibid, datos y launchdaemon app.aibid. |
| Descripción y comportamiento especial | Firma ad hoc no es notarización. No hay paquete Intel en conjunto actual; no presentar Intel como soportado. |
| Ubicación | [PAN-052](02-pantallas-y-permisos.md#pan-052) — Installer .pkg; /Applications/AIBID/*.command |
| Roles / permisos requeridos | Administrador del SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Paquete .pkg arm64; dependencias externas |
| Validaciones | ARM64; Poppler, Tesseract+idiomas fuera del pkg; rutas típicas /opt/homebrew/bin. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-119](05-catalogo-capturas.md#cap-119) |
| Implementación | [packaging/macos/postinstall](../../packaging/macos/postinstall); [packaging/macos/admin.command](../../packaging/macos/admin.command); [scripts/build_installers.py](../../scripts/build_installers.py) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-005](07-pendientes-validacion.md#val-005); [VAL-006](07-pendientes-validacion.md#val-006); [VAL-017](07-pendientes-validacion.md#val-017) |

**Procedimiento:**

1. Verificar macOS 13+AppleSilicon.
2. Instalar dependencias externas según procedimiento en03.
3. Abrir .pkg.
4. Completar Installer.
5. Revisar /Applications/AIBID.
6. Abrir Configurar AIBID.command.

<a id="fun-120"></a>

## FUN-120 — Configurar primer acceso y servicio en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Configurar primer acceso y servicio en macOS |
| Objetivo y resultado esperado | Habilita servicio y abre interfaz local. |
| Descripción y comportamiento especial | Validación existente del binario extraído no equivale a ejecutar Installer/launchd real. |
| Ubicación | [PAN-052](02-pantallas-y-permisos.md#pan-052) — Installer .pkg; /Applications/AIBID/*.command |
| Roles / permisos requeridos | Administrador del SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Bootstrap interactivo |
| Validaciones | Marcador service-enabled; launchdaemon app.aibid bajo cuenta dedicada. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-120](05-catalogo-capturas.md#cap-120) |
| Implementación | [packaging/macos/postinstall](../../packaging/macos/postinstall); [packaging/macos/admin.command](../../packaging/macos/admin.command); [scripts/build_installers.py](../../scripts/build_installers.py) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-005](07-pendientes-validacion.md#val-005) |

**Procedimiento:**

1. Abrir Configurar AIBID.command.
2. Autorizar administración.
3. Revisar doctor.
4. Crear administrador local si falta.
5. Esperar launchd.
6. Usar Abrir AIBID.webloc o URL indicada.

<a id="fun-121"></a>

## FUN-121 — Desinstalar en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Instalación — Desinstalar en macOS |
| Objetivo y resultado esperado | Retira launchd/programa según manifest y receipt, mantiene documentos físicos. |
| Descripción y comportamiento especial | Cuenta/directorios con datos pueden permanecer. Firma/notarización y ciclo nativo pendientes. |
| Ubicación | [PAN-052](02-pantallas-y-permisos.md#pan-052) — Installer .pkg; /Applications/AIBID/*.command |
| Roles / permisos requeridos | Administrador del SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Conservar por defecto; ELIMINAR si borrado |
| Validaciones | Valida enlaces y manifiesto completo antes de borrar; preserva archivos empaquetados modificados. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-121](05-catalogo-capturas.md#cap-121) |
| Implementación | [packaging/macos/postinstall](../../packaging/macos/postinstall); [packaging/macos/admin.command](../../packaging/macos/admin.command); [packaging/macos/uninstall.command](../../packaging/macos/uninstall.command); [internal/maintenance/erase.go](../../internal/maintenance/erase.go) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-005](07-pendientes-validacion.md#val-005) |

**Procedimiento:**

1. Abrir Desinstalar AIBID.command.
2. Leer alcance.
3. Elegir conservar o confirmar ELIMINAR.
4. Esperar parada/verificación.
5. Revisar archivos preservados.

<a id="fun-122"></a>

## FUN-122 — Consultar versión y diagnóstico local

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Consultar versión y diagnóstico local |
| Objetivo y resultado esperado | Diagnóstico concreto de runtime y herramientas externas. |
| Descripción y comportamiento especial | Comandos de manual se muestran como procedimientos, no pruebas nuevas. No exponer configuration.json ni identidad privada como reporte. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Permisos del SO para configuración/estado |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | version; doctor; --config; --sample-directory |
| Validaciones | doctor no es read-only estricto: muestras/procesamiento pueden usar temporales; no se ejecutó durante documentación. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-122](05-catalogo-capturas.md#cap-122) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [internal/extraction/diagnostic.go](../../internal/extraction/diagnostic.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestRealNativeAndSpanishOCR`, fuente [internal/extraction/pdf_test.go](../../internal/extraction/pdf_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-016](07-pendientes-validacion.md#val-016) |

**Procedimiento:**

1. Abrir terminal administrativa según SO.
2. Ejecutar version.
3. Ejecutar doctor --config ruta con muestras empaquetadas cuando corresponda.
4. Leer herramientas/idiomas/PDF/OCR/config.
5. Conservar únicamente reporte sin secretos.

<a id="fun-123"></a>

## FUN-123 — Recuperar administrador local

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Recuperar administrador local |
| Objetivo y resultado esperado | Restablece cuenta administrativa existente con evidencia. |
| Descripción y comportamiento especial | No es acceso remoto anónimo; no pasar contraseña como argumento ni documentar valores reales. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Acceso administrativo SO; sin recuperación por correo |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | recover-admin; --config; credencial interactiva |
| Validaciones | Bloqueo exclusivo y contraseña válida; preserva último administrador habilitado. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-123](05-catalogo-capturas.md#cap-123) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [internal/identity/service.go](../../internal/identity/service.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestBootstrapLastAdministratorAndRevisionGuards`, fuente [internal/identity/service_test.go](../../internal/identity/service_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-010](07-pendientes-validacion.md#val-010) (captura editorial) y [VAL-022](07-pendientes-validacion.md#val-022) (vigencia de evidencia). |

**Procedimiento:**

1. Detener/controlar servicio según requisitos de bloqueo.
2. Abrir consola local administrativa.
3. Ejecutar recover-admin --config ruta.
4. Seguir datos y contraseña interactivos.
5. Volver a iniciar servicio.
6. Revisar auditoría.

<a id="fun-124"></a>

## FUN-124 — Recuperar identidad de licencia dañada

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Recuperar identidad de licencia dañada |
| Objetivo y resultado esperado | Genera identidad recuperada y conserva evidencia. |
| Descripción y comportamiento especial | No recupera automáticamente cupo del proveedor. Requiere nueva activación; no usar como método de reset de licencia válido. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Acceso local SO; confirmación explícita |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | recover-license; --confirm-license-recovery; --reason |
| Validaciones | Estado privado y journal seguros; confirmación/motivo; no reutiliza identidad secreta ajena. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-124](05-catalogo-capturas.md#cap-124) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [internal/licensing/recovery.go](../../internal/licensing/recovery.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestMissingKeyRecoveryRetainsEvidenceAndRequiresNewActivation`, fuente [internal/licensing/contract_test.go](../../internal/licensing/contract_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-008](07-pendientes-validacion.md#val-008) |

**Procedimiento:**

1. Consultar diagnóstico de identidad.
2. Coordinar con proveedor activación/transferencia.
3. Detener servicio.
4. Ejecutar recuperación local con confirmación y motivo.
5. Reiniciar.
6. Activar identidad nueva según proveedor.

<a id="fun-125"></a>

## FUN-125 — Administrar servicio y logs por plataforma

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Administrar servicio y logs por plataforma |
| Objetivo y resultado esperado | Controla backend fuera de UI con comandos nativos. |
| Descripción y comportamiento especial | No hay panel UI de inicio/stop del propio servicio ni visor libre de archivos de log. Operaciones no ejecutadas aquí. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Cuenta autorizada del SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Get-Service/systemctl/launchctl; logs |
| Validaciones | Cuenta separada, permisos mínimos y exclusivo estado; shutdown con cancelación de workers. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-125](05-catalogo-capturas.md#cap-125) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [cmd/gestor-documental/host_windows.go](../../cmd/gestor-documental/host_windows.go); [packaging/ubuntu/aibid.service](../../packaging/ubuntu/aibid.service); [scripts/build_installers.py](../../scripts/build_installers.py) |
| Verificación | Identificado en código; ejecución del procedimiento pendiente de validación. |
| Pendientes | [VAL-016](07-pendientes-validacion.md#val-016) |

**Procedimiento:**

1. Identificar plataforma.
2. Consultar estado con comando documentado en03.
3. Leer log correspondiente.
4. Detener/reiniciar solo en mantenimiento autorizado.
5. Revisar cola/diagnóstico al volver.

<a id="fun-126"></a>

## FUN-126 — Actualizar instalación conservando estado

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Actualizar instalación conservando estado |
| Objetivo y resultado esperado | Migraciones aditivas verifican checksum y conservan identidades/configuración. |
| Descripción y comportamiento especial | Antes de migración de observabilidad puede generar upgrade-backups SQLite; no reemplaza respaldo de originales/config/identidad. Compatibilidad real SO aún pendiente. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Administrador SO; paquete compatible |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | Instalador nuevo; marcadores existentes |
| Validaciones | Marcadores de versiones admitidas; lock exclusivo; no downgradear esquema ni sobrescribir configuración existente. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-126](05-catalogo-capturas.md#cap-126) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [internal/storage/database.go](../../internal/storage/database.go); [scripts/build_installers.py](../../scripts/build_installers.py) |
| Verificación | Verificado en pruebas (escenario parcial): `TestUpgradePreservesH4PrivateClassifiedUpload`, fuente [internal/storage/database_test.go](../../internal/storage/database_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-001](07-pendientes-validacion.md#val-001); [VAL-003](07-pendientes-validacion.md#val-003); [VAL-005](07-pendientes-validacion.md#val-005); [VAL-007](07-pendientes-validacion.md#val-007); [VAL-013](07-pendientes-validacion.md#val-013) |

**Procedimiento:**

1. Revisar compatibilidad y respaldo completo validado.
2. Cerrar actividad administrativa.
3. Ejecutar instalador de la misma arquitectura.
4. Revisar preflight/stop/migración.
5. Reconfigurar arranque si fue requerido.
6. Confirmar login, bibliotecas e índice.

<a id="fun-127"></a>

## FUN-127 — Comprender inicialización, migración y mantenimiento restringido

| Aspecto | Registro |
| --- | --- |
| Módulo / nombre | Operación — Comprender inicialización, migración y mantenimiento restringido |
| Objetivo y resultado esperado | CLI implementa inicialización segura, migración, diagnóstico de locks y eliminación interna protegida. |
| Descripción y comportamiento especial | No hay backup/restore funcional general. bootstrap-ready abre DB/migra pese a parecer consulta; check-state no abre SQLite pero toma lock. |
| Ubicación | [PAN-053](02-pantallas-y-permisos.md#pan-053) — Terminal → aibid-server subcomando --config ruta |
| Roles / permisos requeridos | Administrador SO |
| Condiciones previas | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. |
| Campos involucrados | init; configure-license; check-state; bootstrap-ready; migrate; rollback-empty; erase-internal-data |
| Validaciones | init no sobrescribe; rollback-empty solo instalación nunca usada; erase confirma y protege físicos. |
| Acciones disponibles | Pasos concretos descritos a continuación; controles propios de la pantalla referenciada. |
| Estados relacionados | Sin estado adicional propio: utiliza los estados de la entidad y el resultado indicado; véase catálogo común. |
| Mensajes / advertencias | Consultar catálogo común de mensajes al final y etiquetas exactas en 02. Un error requiere leer el código de diagnóstico; un aviso de solicitud en cola no indica finalización. |
| Captura necesaria | [CAP-127](05-catalogo-capturas.md#cap-127) |
| Implementación | [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); [internal/storage/database.go](../../internal/storage/database.go); [internal/maintenance/erase.go](../../internal/maintenance/erase.go) |
| Verificación | Verificado en pruebas (escenario parcial): `TestInstallerCheckStateDoesNotOpenDatabaseAndRequiresExclusiveLock`, fuente [cmd/gestor-documental/main_test.go](../../cmd/gestor-documental/main_test.go); PASS previo. Pasos/controles identificados en código, no observados aquí en vivo. |
| Pendientes | [VAL-007](07-pendientes-validacion.md#val-007) |

**Procedimiento:**

1. Consultar subcomando y ruta de config.
2. Usar init solo en estado nuevo.
3. Dejar instalación aplicar configure-license/migrate.
4. Usar check-state para exclusividad según soporte.
5. No ejecutar rollback-empty/erase sin alcance y confirmación explícitos.

## Catálogo común de estados y mensajes

| Entidad | Estados técnicos / presentación | Consecuencia para manual |
| --- | --- | --- |
| Cuenta/sesión | habilitada/deshabilitada; activa; closed_at con motivo; reemplazada, expiración, cambio/reset, revocación | No son estados de aprobación documental; cerrar pestaña no cierra sesión. |
| Biblioteca/origen | linked → Vinculada; managed → Administrada; hybrid → Híbrida; biblioteca activa | La modalidad de biblioteca y el origen de cada archivo son independientes en híbrida. |
| Disponibilidad | available → Disponible; missing → No disponible; unknown → Raíz sin acceso; staged → Temporal privado | El texto indexado puede conservarse aunque el original no esté accesible. |
| Aprobación | draft → Borrador; pending_review; approved; rejected; cancelled; needs_review; materializing | Confirmación puede iniciar copia asíncrona antes de aprobado definitivo. |
| Extracción | complete; complete_warnings; failed; superseded; freshness stale | Un nuevo run fallido no sustituye publicación válida previa. |
| Cola | queued; running; retry_wait; paused; completed; failed; cancelled | Intentos no se consumen solo por falta de licencia; paused por raíz/instalación no es cancelación. |
| Raíz/recorrido | active, inaccessible, disabled, retired, superseded; running, complete, partial, failed, superseded | Recorrido parcial/incompleto no autoriza inferir desaparición de todos los archivos. |
| Materialización | planned, copying, verified, published, committed, cleaned, failed | Fases de guardado, no progreso porcentual de bytes. |
| Preview | queued, generating, ready, error, obsolete, unavailable | Fallo de preview no es fallo de extracción. |
| Licencia | active, grace, expired, revoked, invalid, unactivated; development solo compilaciones especiales | Modo de desarrollo no es una licencia comercial ni opción normal del instalador. |
| Diagnóstico | open / reviewed; info / warning / error / critical | Marcar revisado no corrige causa; puede reabrirse por repetición. |

| Código / texto encontrado | Uso | Fuente |
| --- | --- | --- |
| SESSION_REQUIRED — Inicia sesión para continuar. | Falta sesión | [internal/httpapi/server.go](../../internal/httpapi/server.go) |
| FORBIDDEN — No tienes permiso para esta operación. | Permiso global | [internal/httpapi/server.go](../../internal/httpapi/server.go) |
| CSRF_INVALID — La solicitud no pudo validarse. Recarga la página. | CSRF/origen | [internal/httpapi/server.go](../../internal/httpapi/server.go) |
| PASSWORD_POLICY — La contraseña debe tener entre 6 y 128 caracteres. | Alta/cambio | [internal/identity/password.go](../../internal/identity/password.go) |
| LAST_ADMIN_REQUIRED — Debe conservarse al menos un administrador habilitado. | Protección última cuenta | [internal/identity/users.go](../../internal/identity/users.go) |
| VERSION_CONFLICT — El usuario cambió. Actualiza la página. | Revisión concurrente usuario | [internal/identity/users.go](../../internal/identity/users.go) |
| REVISION_CONFLICT — La configuración cambió. Actualiza antes de guardar. | Ajustes avanzados | [internal/libraries/advanced_settings.go](../../internal/libraries/advanced_settings.go) |
| ROOT_PLAN_STALE — La configuración o carpeta cambió. Vuelve a inspeccionarla. | Plan de raíz | [internal/libraries/roots.go](../../internal/libraries/roots.go) |
| FILE_TYPE_BLOCKED — El archivo pertenece a una categoría no permitida (ejecutable, script, paquete o archivo comprimido). | Seguridad formatos | [internal/documentformat/format.go](../../internal/documentformat/format.go) |
| FILE_FORMAT_DISABLED — La biblioteca no permite almacenar este formato. | Regla de biblioteca | [internal/documentformat/format.go](../../internal/documentformat/format.go) |
| UPLOAD_SPACE — No hay espacio suficiente para recibir la carga. | HTTP507 | [internal/libraries/uploads.go](../../internal/libraries/uploads.go) |
| Esta unidad está vacía. Consulta otra unidad con los botones Anterior y Siguiente. | Unidad de texto, no documento vacío | [web/src/Documents.tsx](../../web/src/Documents.tsx) |
| Verificación en cola. Consulta su avance en Procesamiento. | Solicitud asíncrona | [web/src/Libraries.tsx](../../web/src/Libraries.tsx) |
| WORKER_STATE_WRITE_FAILED / WORKER_STATE_RECOVERED / PROCESS_RESTARTED | Diagnóstico de persistencia/recuperación | [internal/libraries/job_control.go](../../internal/libraries/job_control.go) |
| SCAN_PARTIAL; WATCH_LIMIT; pérdida de eventos | Errores parciales/recuperación watcher; UI traduce códigos | [web/src/libraryTypes.ts](../../web/src/libraryTypes.ts); [internal/libraries/scanner.go](../../internal/libraries/scanner.go) |

No es un diccionario exhaustivo de toda cadena interna; los mensajes operativos se localizan además en los mapas de [web/src/libraryTypes.ts](../../web/src/libraryTypes.ts), [web/src/organizationTypes.ts](../../web/src/organizationTypes.ts), [web/src/api.ts](../../web/src/api.ts) y en el apéndice de controles del documento de pantallas. No confundir el literal UI de una incidencia con su causa técnica sin el código.
