# AIBID V2 — revisión final de QA y endurecimiento

Revisión del 7 de octubre de 2026 para **2.0.0-rc.1-r1**, esquema 14. Se revisó
primero el código y las pruebas existentes. Los cambios se limitan a defectos
reproducidos, los ajustes de red/cursor solicitados y la transición visible a
AIBID. No se modificaron los datos de una instalación real, ni se activaron o
desactivaron licencias comerciales durante esta revisión.

La evidencia ejecutada, cantidades de pruebas, omisiones y hashes de los paquetes
se entrega en `dist/aibid-2.0-rc.1/VALIDACION-RC1.json` y sus logs. La compilación
cruzada y las simulaciones de PowerShell no sustituyen una instalación en Windows.

## Problemas encontrados y corregidos

| Problema comprobado | Corrección y prueba de regresión |
| --- | --- |
| El prechequeo LAN usaba conexiones de 250 ms; Windows puede tardar segundos en rechazar un puerto cerrado. El firewall solo se consultaba: no existía creación automática de reglas. | Tabla TCP nativa de listeners, apertura real y reversión conservadas. El instalador elevado prepara una regla limitada a ejecutable, servicio, puerto, subred y perfiles privados/dominio. Pruebas de tabla, cambio desde una petición activa, rollback y NetSecurity simulado. [Diagnóstico completo](AIBID-2.0-RC1.md). El fallo concreto del equipo del usuario aún necesita confirmación nativa. |
| “Mostrar más” podía terminar después de cambiar un filtro y añadir registros del filtro anterior. | Cancelación de la petición y comprobación de su ámbito antes de modificar filas, cursor o estado de carga. Prueba de Chrome reteniendo la respuesta antigua: falla antes y pasa después. |
| Una petición sin respuesta mantenía controles/listados cargando indefinidamente. Un error JSON con `error: null` producía un mensaje técnico de JavaScript. | Límite de 45 s para API/importación y 120 s para cargas; errores de contrato muestran mensajes comprensibles. Cancelación al navegar conservada. Un timeout no repite escrituras automáticamente: se pide actualizar el estado antes de reintentar. Pruebas de Chrome con reloj controlado y respuesta inválida: fallan antes y pasan después. |
| Se ignoraba un fallo al escribir el estado final de un trabajo o el error de una preview. Podían quedar como activos hasta reiniciar aunque el procesamiento hubiera terminado. | Se reintenta únicamente la escritura del estado, con espera progresiva de 1 a 30 s y cancelación al cerrar el servicio. Se conserva el control de vigencia del trabajo. No se repite extracción/conversión ni se consume otro intento. La recuperación inicial de previews también reintenta si falla. Pruebas con triggers que rechazan temporalmente la escritura, restauración del almacenamiento, una sola extracción/conversión y cierre durante el reintento. |
| En una raíz administrada, el primer archivo imposible de leer interrumpía la verificación de los restantes. | Los errores de lectura individuales se registran por ruta y la verificación continúa. Resultado parcial explícito; el error se limpia tras verificar correctamente ese archivo. La pérdida de raíz, errores de base de datos o cancelación siguen deteniendo la operación. Regresión con dos documentos: el primero inaccesible y el segundo modificado; falla antes y pasa después. |
| Los controles deshabilitados mostraban cursor de espera; persistían nombres visibles de pruebas. | Cursor `not-allowed` y marca visible AIBID. Prueba de estilo en Chrome y comprobación de los paquetes. Se conservan identificadores técnicos por compatibilidad, documentados en la guía RC1. |

Durante la revisión del script nuevo también se corrigió su actualización de regla:
`Set-NetFirewallRule -Name` necesita `-NewDisplayName`, mientras la creación usa
`-DisplayName`. La simulación valida conjuntos de parámetros, además del alcance,
la idempotencia y la retirada de la regla propia.

Los fallos de escritura del estado se registran una vez por incidente con
`WORKER_STATE_WRITE_FAILED` y al recuperarse con `WORKER_STATE_RECOVERED`.
El registro usa códigos e identificadores saneados: no copia SQL, nombres de
archivos, contraseñas, claves de licencia o contenido documental. Se escribe al
log del servicio porque SQLite puede ser precisamente el recurso no disponible.
Un fallo persistente de disco requiere intervención; el reintento no lo oculta.

## Puntos revisados que ya tenían cobertura

| Área | Implementación y evidencia existente conservada |
| --- | --- |
| Estados, errores y vacíos | Estados de documento/procesamiento/preview, mensajes por códigos, Dashboard vacío y recuentos sin inventar progreso. `TestObservabilityEmptyAndLibraryPagination`, `TestObservabilityErrorLifecyclePrivacyPaginationAndRetention`; pruebas de interfaz y el caso nuevo de timeout. |
| Timeouts, cancelación y aislamiento | Límites configurados por trabajo, presupuesto de reintentos, cancelación, contención de panic y cierre de procesos externos. `TestWorkerTimeoutDoesNotBlockOtherDocuments`, `TestPanicIsContainedAndDoesNotExposeContent`, `TestConversionDeadlineKillsProgram`. Un PDF inválido no bloquea otros descubrimientos. |
| Acciones duplicadas y concurrencia | Revisión optimista e idempotencia en publicación/reindexación/cargas; deduplicación por raíz, cuatro reconciliaciones independientes y límites de extracción/OCR. `TestRuntimeReconcilesDifferentRootsConcurrentlyWithPerRootExclusion`, `TestRootRequestsCoalesceAndRecoverAfterRestart`, `TestNativeAndOCRWorkersBoundedAndRestartable`. |
| Filtros y paginación | Orden estable, cursores asociados al ámbito, filtros combinados y recuentos sujetos a permisos. `TestSearchScopePaginationAndPrivateQueryAudit`, `TestSearchFormatFiltersCountsCursorAndVisibility`, `TestAuditCombinedFiltersSortingAndScope`. Se corrigió únicamente la carrera del cliente descrita arriba. |
| Nombres, rutas y formato | Rutas relativas controladas, rechazo de enlaces fuera de raíz, SQLite con caracteres especiales y unidad Windows, nombres congelados y colisiones al publicar. `TestSQLiteSpecialFilenameRoundTripAndReadOnlyConnection`, `TestSQLiteWindowsDriveIsNotURIAuthority`, `TestRootPlansSymlinksAndSearchInput`, `TestH5PublicationCollisionAndFrozenNaming`. La regresión de recuperación usa también un nombre con `ñ`, espacios, `#` y `%`. |
| Duplicados y hash | SHA-256, cambios de versión, caché de identidad/tamaño/fecha con revalidación, copias visibles según permisos, hard links y reaparición. `TestDuplicateGroupsRespectLibraryPermissions`, `TestHardLinksReappearanceAndRetirement`, `TestLinkedRetentionRenameConsolidationAndChange`. Un índice fallido no sustituye texto publicado válido. |
| Sesiones y permisos | Revalidación por operación, CSRF/Origin/Host, sesión reemplazada atómicamente, reseteos concurrentes, bloqueo por intentos, expiración sin prolongación por polling. Suites `identity`, `httpapi`, pruebas de privacidad de cargas, previews, expedientes y búsqueda. |
| Previews | Generación bajo demanda sobre copia privada, eliminación de contenido activo/enlaces externos, permisos revalidados, límites de entrada/salida, timeout, LRU/expiración e invalidación por hash/generador. Suites `previews` y `libraries/previews_test.go`, HTTP Range y conversiones reales de fixtures con LibreOffice local. |
| Reinicios y colas | Recuperación de leases, intentos y checkpoints, protección contra publicación obsoleta, pausa por licencia sin consumir intentos. `TestQueueRetryLimitRestartAndPublicationFence`, `TestRootRequestsCoalesceAndRecoverAfterRestart`, `TestH6WorkerPausesWithoutBurningAttemptsAndResumesAfterModuleRenewal`. |
| Temporales, caché y disco | Limpieza de espacios de trabajo propios al arrancar; límite/expiración de caché; comprobación de espacio para materializar; journal recuperable antes/después de publicar y confirmar. `TestPreviewLRUExpiryAndClearDuringGeneration`, `TestH4UploadValidationRecoveryAndRootOverlap`, `TestH5MaterializationFailureRecovery`. La corrección nueva cubre el fallo de persistencia del resultado. |
| Logs y soporte | Catálogo de errores, saneamiento de contexto, retención/deduplicación y acceso administrativo. `TestObservabilityHTTPAuthorizationCSRFAndSanitizedFailure`, `TestObservabilityErrorLifecyclePrivacyPaginationAndRetention`; la prueba nueva confirma que la causa SQL privada no aparece en el log. |
| Licenciamiento | Firma JWS, instalación/entorno, revisiones, reloj, gracia, uso perpetuo, revocación, reactivación, TLS y ausencia de redirecciones. Suites `licensing`, `library/license` y HTTP con simulador; no se toca el servidor comercial en QA. |
| Instalar, actualizar, desinstalar | Migraciones aditivas con datos, manifiestos y conservación de archivos ajenos/modificados, borrado interno explícito sin eliminar documentos originales. `storage/*upgrade*`, `TestErasePreservesPhysicalDocumentsAndAllowsFreshInstall`, `TestEraseRefusesLinksBeforeDeletingAnything`; parser PowerShell, paquete extraído y actualización aislada del binario macOS desde fase 8. |
| Volumen | `TestLargeLibrary9000`: 9.000 archivos pequeños con cabecera PDF, interrupción tras los primeros registros, reapertura de SQLite, reanudación, nueva pasada incremental, 9.000 trabajos únicos e `integrity_check`. No es una medición de 9.000 conversiones/OCR ni de una red SMB real. |

## Alcance de la ejecución y pendientes de producción

- Entorno de esta revisión: macOS Apple Silicon. Suites Go de desarrollo y
  producción, detector de concurrencia, Chrome, Poppler/Tesseract y LibreOffice
  local. Las pruebas de carga se ejecutan separadas del detector de concurrencia:
  la primera combinación alcanzó el límite global de 10 minutos del ejecutor, por
  lo que se repitió con presupuesto explícito y evidencia separada.
- **Antes de aprobar Windows:** instalar RC1 sobre fase 8 con una copia recuperable
  de los datos; comprobar ida/vuelta local↔LAN desde otro equipo, persistencia al
  reiniciar, regla efectiva con la cuenta virtual, GPO y acceso a la carpeta SMB.
  También comprobar instalación limpia, desinstalación conservando datos y
  reinstalación. La prueba nativa de tabla TCP está añadida al código/CI, pero no
  se ha ejecutado en Windows desde esta Mac. PowerShell aquí es 7.x; falta ejecutar
  el instalador en Windows PowerShell 5.1 y su servicio real.
- **Antes de aprobar Ubuntu:** verificar los DEB `arm64` y `amd64` en sus sistemas
  respectivos, incluido ciclo systemd/actualización/desinstalación. Aquí se validan
  arquitectura, metadatos, contenido y compilación. El smoke macOS ejecuta el
  binario extraído y datos temporales; no modifica launchd ni instala el servicio
  de esta computadora. La instalación interactiva y sus accesos requieren prueba.
- **Carga y almacenamiento reales:** ensayar el corpus representativo de producción
  sobre SMB, una desconexión/reconexión y restauración desde backup. Las lecturas
  bloqueadas dentro del kernel/controlador de red dependen también de los timeouts
  del sistema operativo. La prueba de fallo de escritura es una inyección controlada
  en SQLite, no el llenado de un disco real. Estos límites no se presentan como
  validaciones ya realizadas.
- **Distribución:** los paquetes son candidatos sin firma comercial/notarización.
  Completar firma y prueba de confianza antes de distribución general. Se conservan
  servicio, SID, cuentas y rutas internas históricas para actualizar sin perder
  permisos ni activación; renombrarlos exigiría una migración aparte.

Esta revisión aporta correcciones y evidencia de regresión para RC1. La aprobación
de V2 para producción queda condicionada a las comprobaciones nativas anteriores;
un build satisfactorio no demuestra por sí solo que un firewall o una política de
dominio permita el acceso desde otro equipo.
