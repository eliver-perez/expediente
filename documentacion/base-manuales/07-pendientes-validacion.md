# AIBID — Pendientes de validación

**Fecha de elaboración: 2026-10-09.** Actualización de la base elaborada el 8 de octubre; nombres y procedimientos contrastados con la transición de producción. Base de trabajo para los futuros manuales; no es un manual final.

## Alcance y prioridades

**22 pendientes**. Son límites de evidencia y preparación de manuales; **no una lista de defectos demostrados**. Esta actualización corrige identidad y transporte/quoting del instalador y agrega pruebas aisladas; no incorpora funciones documentales nuevas. Se mantienen estos 22 escenarios de validación con sus IDs, sin presentar las pruebas parciales como certificación nativa. Alta: bloquea una afirmación operativa importante/publicación; Media: requiere completar precisión, combinación o calidad editorial.

La falta de respaldo/restauración general sí se verificó como **función no implementada**, y no se promete en el inventario. Las demás dudas no se convierten en afirmaciones de fallo. Código y pruebas previas no reemplazan prueba nativa de instalador/firewall ni observación visual actual.

| VAL | Pendiente | Módulo | Prioridad | FUN / PAN |
| --- | --- | --- | --- | --- |
| [VAL-001](07-pendientes-validacion.md#val-001) | Instalación, actualización y desinstalación nativas en Windows | Instalación | Alta | [FUN-110](01-inventario-funcional.md#fun-110), [FUN-111](01-inventario-funcional.md#fun-111), [FUN-113](01-inventario-funcional.md#fun-113), [FUN-114](01-inventario-funcional.md#fun-114), [FUN-126](01-inventario-funcional.md#fun-126) |
| [VAL-002](07-pendientes-validacion.md#val-002) | Conmutación LAN y regla efectiva en Windows | Red | Alta | [FUN-096](01-inventario-funcional.md#fun-096), [FUN-097](01-inventario-funcional.md#fun-097) |
| [VAL-003](07-pendientes-validacion.md#val-003) | Instalación y ciclo systemd en Ubuntu de ambas arquitecturas | Instalación | Alta | [FUN-115](01-inventario-funcional.md#fun-115), [FUN-116](01-inventario-funcional.md#fun-116), [FUN-118](01-inventario-funcional.md#fun-118), [FUN-126](01-inventario-funcional.md#fun-126) |
| [VAL-004](07-pendientes-validacion.md#val-004) | Dependencias y bundle offline Ubuntu | Instalación | Alta | [FUN-117](01-inventario-funcional.md#fun-117) |
| [VAL-005](07-pendientes-validacion.md#val-005) | Installer y launchd reales en macOS | Instalación | Alta | [FUN-119](01-inventario-funcional.md#fun-119), [FUN-120](01-inventario-funcional.md#fun-120), [FUN-121](01-inventario-funcional.md#fun-121), [FUN-126](01-inventario-funcional.md#fun-126) |
| [VAL-006](07-pendientes-validacion.md#val-006) | Distribución firmada y notarización | Instalación | Alta | [FUN-110](01-inventario-funcional.md#fun-110), [FUN-119](01-inventario-funcional.md#fun-119) |
| [VAL-007](07-pendientes-validacion.md#val-007) | Respaldo completo y restauración | Operación | Alta | [FUN-126](01-inventario-funcional.md#fun-126), [FUN-127](01-inventario-funcional.md#fun-127) |
| [VAL-008](07-pendientes-validacion.md#val-008) | Servidor externo de licencias y flujo del proveedor | Licenciamiento | Alta | [FUN-103](01-inventario-funcional.md#fun-103), [FUN-104](01-inventario-funcional.md#fun-104), [FUN-105](01-inventario-funcional.md#fun-105), [FUN-106](01-inventario-funcional.md#fun-106), [FUN-107](01-inventario-funcional.md#fun-107), [FUN-108](01-inventario-funcional.md#fun-108), [FUN-124](01-inventario-funcional.md#fun-124) |
| [VAL-009](07-pendientes-validacion.md#val-009) | TLS y proxy en entorno real | Red | Media | [FUN-096](01-inventario-funcional.md#fun-096), [FUN-097](01-inventario-funcional.md#fun-097) |
| [VAL-010](07-pendientes-validacion.md#val-010) | Capturas editoriales y revisión visual actual | Interfaz | Alta | [PAN-001](02-pantallas-y-permisos.md#pan-001), [PAN-002](02-pantallas-y-permisos.md#pan-002), [PAN-003](02-pantallas-y-permisos.md#pan-003), [PAN-004](02-pantallas-y-permisos.md#pan-004), [PAN-005](02-pantallas-y-permisos.md#pan-005), [PAN-006](02-pantallas-y-permisos.md#pan-006), [PAN-007](02-pantallas-y-permisos.md#pan-007), [PAN-008](02-pantallas-y-permisos.md#pan-008), [PAN-009](02-pantallas-y-permisos.md#pan-009), [PAN-010](02-pantallas-y-permisos.md#pan-010), [PAN-011](02-pantallas-y-permisos.md#pan-011), [PAN-012](02-pantallas-y-permisos.md#pan-012), [PAN-013](02-pantallas-y-permisos.md#pan-013), [PAN-014](02-pantallas-y-permisos.md#pan-014), [PAN-015](02-pantallas-y-permisos.md#pan-015), [PAN-016](02-pantallas-y-permisos.md#pan-016), [PAN-017](02-pantallas-y-permisos.md#pan-017), [PAN-018](02-pantallas-y-permisos.md#pan-018), [PAN-019](02-pantallas-y-permisos.md#pan-019), [PAN-020](02-pantallas-y-permisos.md#pan-020), [PAN-021](02-pantallas-y-permisos.md#pan-021), [PAN-022](02-pantallas-y-permisos.md#pan-022), [PAN-023](02-pantallas-y-permisos.md#pan-023), [PAN-024](02-pantallas-y-permisos.md#pan-024), [PAN-025](02-pantallas-y-permisos.md#pan-025), [PAN-026](02-pantallas-y-permisos.md#pan-026), [PAN-027](02-pantallas-y-permisos.md#pan-027), [PAN-028](02-pantallas-y-permisos.md#pan-028), [PAN-029](02-pantallas-y-permisos.md#pan-029), [PAN-030](02-pantallas-y-permisos.md#pan-030), [PAN-031](02-pantallas-y-permisos.md#pan-031), [PAN-032](02-pantallas-y-permisos.md#pan-032), [PAN-033](02-pantallas-y-permisos.md#pan-033), [PAN-034](02-pantallas-y-permisos.md#pan-034), [PAN-035](02-pantallas-y-permisos.md#pan-035), [PAN-036](02-pantallas-y-permisos.md#pan-036), [PAN-037](02-pantallas-y-permisos.md#pan-037), [PAN-038](02-pantallas-y-permisos.md#pan-038), [PAN-039](02-pantallas-y-permisos.md#pan-039), [PAN-040](02-pantallas-y-permisos.md#pan-040), [PAN-041](02-pantallas-y-permisos.md#pan-041), [PAN-042](02-pantallas-y-permisos.md#pan-042), [PAN-043](02-pantallas-y-permisos.md#pan-043), [PAN-044](02-pantallas-y-permisos.md#pan-044), [PAN-045](02-pantallas-y-permisos.md#pan-045), [PAN-046](02-pantallas-y-permisos.md#pan-046), [PAN-047](02-pantallas-y-permisos.md#pan-047), [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| [VAL-011](07-pendientes-validacion.md#val-011) | Contraste, accesibilidad, móvil y teclado | Interfaz | Media | [FUN-029](01-inventario-funcional.md#fun-029), [FUN-032](01-inventario-funcional.md#fun-032), [FUN-064](01-inventario-funcional.md#fun-064), [FUN-099](01-inventario-funcional.md#fun-099) |
| [VAL-012](07-pendientes-validacion.md#val-012) | Gran volumen y red SMB reales | Procesamiento | Alta | [FUN-078](01-inventario-funcional.md#fun-078), [FUN-079](01-inventario-funcional.md#fun-079), [FUN-083](01-inventario-funcional.md#fun-083), [FUN-091](01-inventario-funcional.md#fun-091) |
| [VAL-013](07-pendientes-validacion.md#val-013) | Disco lleno, cortes y recuperación del sistema | Procesamiento | Alta | [FUN-060](01-inventario-funcional.md#fun-060), [FUN-082](01-inventario-funcional.md#fun-082), [FUN-094](01-inventario-funcional.md#fun-094), [FUN-126](01-inventario-funcional.md#fun-126) |
| [VAL-014](07-pendientes-validacion.md#val-014) | UI con permisos parciales y licencia solo lectura | Permisos | Media | [FUN-012](01-inventario-funcional.md#fun-012), [FUN-016](01-inventario-funcional.md#fun-016), [FUN-071](01-inventario-funcional.md#fun-071), [FUN-109](01-inventario-funcional.md#fun-109) |
| [VAL-015](07-pendientes-validacion.md#val-015) | Avisos de cambios y privacidad excepcional | Búsqueda | Media | [FUN-065](01-inventario-funcional.md#fun-065) |
| [VAL-016](07-pendientes-validacion.md#val-016) | Logs del host y mensajes locales de soporte | Operación | Alta | [FUN-101](01-inventario-funcional.md#fun-101), [FUN-122](01-inventario-funcional.md#fun-122), [FUN-125](01-inventario-funcional.md#fun-125) |
| [VAL-017](07-pendientes-validacion.md#val-017) | Hardware, capacidad y espacio mínimo publicados | Instalación | Alta | [FUN-091](01-inventario-funcional.md#fun-091), [FUN-110](01-inventario-funcional.md#fun-110), [FUN-115](01-inventario-funcional.md#fun-115), [FUN-119](01-inventario-funcional.md#fun-119) |
| [VAL-018](07-pendientes-validacion.md#val-018) | Fuentes y fidelidad de previews | Previews | Media | [FUN-035](01-inventario-funcional.md#fun-035), [FUN-094](01-inventario-funcional.md#fun-094) |
| [VAL-019](07-pendientes-validacion.md#val-019) | Codificaciones, archivos poco comunes y corpus ampliado | Extracción | Media | [FUN-034](01-inventario-funcional.md#fun-034), [FUN-078](01-inventario-funcional.md#fun-078), [FUN-080](01-inventario-funcional.md#fun-080), [FUN-086](01-inventario-funcional.md#fun-086) |
| [VAL-020](07-pendientes-validacion.md#val-020) | Caducidad temporal, caché y limpieza del SO | Cargas | Media | [FUN-054](01-inventario-funcional.md#fun-054), [FUN-082](01-inventario-funcional.md#fun-082), [FUN-095](01-inventario-funcional.md#fun-095) |
| [VAL-021](07-pendientes-validacion.md#val-021) | Longitudes Unicode y mensajes exactos de validación | Interfaz | Media | [FUN-007](01-inventario-funcional.md#fun-007), [FUN-019](01-inventario-funcional.md#fun-019), [FUN-042](01-inventario-funcional.md#fun-042), [FUN-051](01-inventario-funcional.md#fun-051) |
| [VAL-022](07-pendientes-validacion.md#val-022) | Integridad de evidencia histórica y fidelidad al código actual | Documentación | Media |  |

<a id="val-001"></a>

## VAL-001 — Instalación, actualización y desinstalación nativas en Windows

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Instalación, actualización y desinstalación nativas en Windows; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | NSIS/manage.ps1; validación previa PE, scripts PowerShell y mocks; paquetes existentes. |
| Qué falta comprobar | Ejecutar ciclo real Windows 10/11 x64 con PowerShell5.1, UAC, SCM, ACL y archivos modificados; instalación nueva y actualización del servicio anterior. Comprobar visualmente áéíóú/ÁÉÍÓÚ/Ññ/ü/¿¡ y rutas C:\Users\José, María López, Muñoz, Documentación y Unicode adicional en instalación, ejecución, actualización y desinstalación. |
| Cómo validar con seguridad | VM desechable con snapshot y documentos ficticios; probar conservar y eliminar en copias separadas, sin intervenir producción. |
| Referencias | [FUN-110](01-inventario-funcional.md#fun-110), [FUN-111](01-inventario-funcional.md#fun-111), [FUN-113](01-inventario-funcional.md#fun-113), [FUN-114](01-inventario-funcional.md#fun-114), [FUN-126](01-inventario-funcional.md#fun-126) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-002"></a>

## VAL-002 — Conmutación LAN y regla efectiva en Windows

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Conmutación LAN y regla efectiva en Windows; Red |
| Prioridad para manuales | Alta |
| Evidencia existente | Pruebas previas network y scripts de firewall; listener IPv4 y rule scope en código. |
| Qué falta comprobar | Confirmar cuenta virtual, listener actual, Domain/Private, regla efectiva, reinicio y vuelta local con conexión keep-alive; GPO/Public pueden restringir. |
| Cómo validar con seguridad | Dos equipos/VM aislados, regla y estado antes/después exportados sin secretos; probar sin elevar backend. |
| Referencias | [FUN-096](01-inventario-funcional.md#fun-096), [FUN-097](01-inventario-funcional.md#fun-097) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-003"></a>

## VAL-003 — Instalación y ciclo systemd en Ubuntu de ambas arquitecturas

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Instalación y ciclo systemd en Ubuntu de ambas arquitecturas; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | Paquetes amd64 y arm64 inspeccionados; scripts de cuenta/systemd; funciones Go probadas. |
| Qué falta comprobar | APT limpio, bootstrap, permisos locales/LAN, upgrade, parada y conservación/borrado; cuenta nueva aibid y perfil anterior con aibid.service/drop-in. Comprobar mensajes y rutas Unicode con espacios en ambas arquitecturas. |
| Cómo validar con seguridad | VM Ubuntu 24.04 de arquitectura correspondiente con snapshot; no usar amd64 como sustituto de ARM64. |
| Referencias | [FUN-115](01-inventario-funcional.md#fun-115), [FUN-116](01-inventario-funcional.md#fun-116), [FUN-118](01-inventario-funcional.md#fun-118), [FUN-126](01-inventario-funcional.md#fun-126) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-004"></a>

## VAL-004 — Dependencias y bundle offline Ubuntu

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Dependencias y bundle offline Ubuntu; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | prepare-offline.sh/install-offline.sh verifican cierre de paquetes y SHA256. |
| Qué falta comprobar | Reproducibilidad sin red en VM limpia; tamaño real del bundle y dependencias recomendadas disponibles. |
| Cómo validar con seguridad | Preparar en VM autorizada con red y probar clon sin adaptador/red, mismo SO/arquitectura, registrar inventario. |
| Referencias | [FUN-117](01-inventario-funcional.md#fun-117) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-005"></a>

## VAL-005 — Installer y launchd reales en macOS

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Installer y launchd reales en macOS; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | Binario pkg extraído probado; scripts/manifest y plist generado examinados. |
| Qué falta comprobar | Installer con privilegios, creación cuenta, launchd automático, logs/paths, conservar/eliminar y upgrade reales. |
| Cómo validar con seguridad | Mac/VM ARM64 desechable; snapshot/respaldo del sistema de prueba; no ejecutar scripts contra Mac de trabajo. |
| Referencias | [FUN-119](01-inventario-funcional.md#fun-119), [FUN-120](01-inventario-funcional.md#fun-120), [FUN-121](01-inventario-funcional.md#fun-121), [FUN-126](01-inventario-funcional.md#fun-126) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-006"></a>

## VAL-006 — Distribución firmada y notarización

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Distribución firmada y notarización; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | Build macOS codesign ad hoc; no proceso equivalente de firma comercial certificado en evidencia. |
| Qué falta comprobar | Gatekeeper/SmartScreen y decisión de firma oficial para manual distribuible. |
| Cómo validar con seguridad | Inspección con herramientas nativas en copia de paquete; definir proceso de firma con responsable; no desactivar protecciones como instrucción normal. |
| Referencias | [FUN-110](01-inventario-funcional.md#fun-110), [FUN-119](01-inventario-funcional.md#fun-119) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-007"></a>

## VAL-007 — Respaldo completo y restauración

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Respaldo completo y restauración; Operación |
| Prioridad para manuales | Alta |
| Evidencia existente | Permisos backup/restore declarados; storage.upgrade_backup VACUUM INTO; CLI sin respaldo/restauración general. |
| Qué falta comprobar | Procedimiento integral coherente de DB/identidad/config/uploads/originales/restauración; no existe producto completo implementado. |
| Cómo validar con seguridad | Diseñar runbook externo y ensayar restauración en clon aislado y proveedor de pruebas; validar rutas/hash y binding. No inventar botones. |
| Referencias | [FUN-126](01-inventario-funcional.md#fun-126), [FUN-127](01-inventario-funcional.md#fun-127) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-008"></a>

## VAL-008 — Servidor externo de licencias y flujo del proveedor

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Servidor externo de licencias y flujo del proveedor; Licenciamiento |
| Prioridad para manuales | Alta |
| Evidencia existente | Cliente HTTPS/JWS/offline y tests locales previos; LiveProvider omitido. |
| Qué falta comprobar | Contrato operativo actual, respuestas, activación real, cupo tras offline y recovery; servidor pertenece a otro proyecto no inspeccionado. |
| Cómo validar con seguridad | Licencia exclusiva autorizada de prueba y ambiente proveedor sin datos reales; no usar claves históricas pegadas en conversación. |
| Referencias | [FUN-103](01-inventario-funcional.md#fun-103), [FUN-104](01-inventario-funcional.md#fun-104), [FUN-105](01-inventario-funcional.md#fun-105), [FUN-106](01-inventario-funcional.md#fun-106), [FUN-107](01-inventario-funcional.md#fun-107), [FUN-108](01-inventario-funcional.md#fun-108), [FUN-124](01-inventario-funcional.md#fun-124) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-009"></a>

## VAL-009 — TLS y proxy en entorno real

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | TLS y proxy en entorno real; Red |
| Prioridad para manuales | Media |
| Evidencia existente | Validación config, Host/Origin/cookies/proxies y pruebas de transport. |
| Qué falta comprobar | Instalación de certificado/proxy, URLs y redes corporativas con cookies Secure; soporte público vsLAN. |
| Cómo validar con seguridad | Entorno aislado HTTPS con proxy explícitamente confiado; capturar solo nombres de claves/config, nunca certificados privados. |
| Referencias | [FUN-096](01-inventario-funcional.md#fun-096), [FUN-097](01-inventario-funcional.md#fun-097) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-010"></a>

## VAL-010 — Capturas editoriales y revisión visual actual

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Capturas editoriales y revisión visual actual; Interfaz |
| Prioridad para manuales | Alta |
| Evidencia existente | Ocho PNG reales de pruebas reutilizados; catálogo de todas las escenas; ninguna sesión UI actual iniciada. |
| Qué falta comprobar | Sustituir pruebas/nombres/rutas, llenar estados válidos y vacíos, sin errores de framing/cortes; todos los CAP pendientes o requieren reemplazo. |
| Cómo validar con seguridad | Navegador con perfil de demostración ya autorizado; obtener captura de lectura, sin crear datos de producción; no fabricar interfaces. |
| Referencias | [PAN-001](02-pantallas-y-permisos.md#pan-001), [PAN-002](02-pantallas-y-permisos.md#pan-002), [PAN-003](02-pantallas-y-permisos.md#pan-003), [PAN-004](02-pantallas-y-permisos.md#pan-004), [PAN-005](02-pantallas-y-permisos.md#pan-005), [PAN-006](02-pantallas-y-permisos.md#pan-006), [PAN-007](02-pantallas-y-permisos.md#pan-007), [PAN-008](02-pantallas-y-permisos.md#pan-008), [PAN-009](02-pantallas-y-permisos.md#pan-009), [PAN-010](02-pantallas-y-permisos.md#pan-010), [PAN-011](02-pantallas-y-permisos.md#pan-011), [PAN-012](02-pantallas-y-permisos.md#pan-012), [PAN-013](02-pantallas-y-permisos.md#pan-013), [PAN-014](02-pantallas-y-permisos.md#pan-014), [PAN-015](02-pantallas-y-permisos.md#pan-015), [PAN-016](02-pantallas-y-permisos.md#pan-016), [PAN-017](02-pantallas-y-permisos.md#pan-017), [PAN-018](02-pantallas-y-permisos.md#pan-018), [PAN-019](02-pantallas-y-permisos.md#pan-019), [PAN-020](02-pantallas-y-permisos.md#pan-020), [PAN-021](02-pantallas-y-permisos.md#pan-021), [PAN-022](02-pantallas-y-permisos.md#pan-022), [PAN-023](02-pantallas-y-permisos.md#pan-023), [PAN-024](02-pantallas-y-permisos.md#pan-024), [PAN-025](02-pantallas-y-permisos.md#pan-025), [PAN-026](02-pantallas-y-permisos.md#pan-026), [PAN-027](02-pantallas-y-permisos.md#pan-027), [PAN-028](02-pantallas-y-permisos.md#pan-028), [PAN-029](02-pantallas-y-permisos.md#pan-029), [PAN-030](02-pantallas-y-permisos.md#pan-030), [PAN-031](02-pantallas-y-permisos.md#pan-031), [PAN-032](02-pantallas-y-permisos.md#pan-032), [PAN-033](02-pantallas-y-permisos.md#pan-033), [PAN-034](02-pantallas-y-permisos.md#pan-034), [PAN-035](02-pantallas-y-permisos.md#pan-035), [PAN-036](02-pantallas-y-permisos.md#pan-036), [PAN-037](02-pantallas-y-permisos.md#pan-037), [PAN-038](02-pantallas-y-permisos.md#pan-038), [PAN-039](02-pantallas-y-permisos.md#pan-039), [PAN-040](02-pantallas-y-permisos.md#pan-040), [PAN-041](02-pantallas-y-permisos.md#pan-041), [PAN-042](02-pantallas-y-permisos.md#pan-042), [PAN-043](02-pantallas-y-permisos.md#pan-043), [PAN-044](02-pantallas-y-permisos.md#pan-044), [PAN-045](02-pantallas-y-permisos.md#pan-045), [PAN-046](02-pantallas-y-permisos.md#pan-046), [PAN-047](02-pantallas-y-permisos.md#pan-047), [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-011"></a>

## VAL-011 — Contraste, accesibilidad, móvil y teclado

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Contraste, accesibilidad, móvil y teclado; Interfaz |
| Prioridad para manuales | Media |
| Evidencia existente | React/ARIA; pruebas browser y capturas previas; tablas accesibles de gráficas. |
| Qué falta comprobar | Confirmar pantallas/diálogos completos con zoom200%, focus, lector pantalla, teclado y tamaños móviles. |
| Cómo validar con seguridad | Recorrido de lectura en demostración con cuentas ficticias, sin cambiar configuraciones. |
| Referencias | [FUN-029](01-inventario-funcional.md#fun-029), [FUN-032](01-inventario-funcional.md#fun-032), [FUN-064](01-inventario-funcional.md#fun-064), [FUN-099](01-inventario-funcional.md#fun-099) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-012"></a>

## VAL-012 — Gran volumen y red SMB reales

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Gran volumen y red SMB reales; Procesamiento |
| Prioridad para manuales | Alta |
| Evidencia existente | TestLargeLibrary9000 previo aprobado; concurrent root exclusion y cache tests. |
| Qué falta comprobar | Rendimiento con red lenta, muchos subdirectorios, cuotas fsnotify y permisos de servicio, no benchmark extrapolable. |
| Cómo validar con seguridad | Corpus sintético incremental en volumen aislado SMB y cuenta servicio; limites de tiempo y captura de métricas, nunca duplicar clientes. |
| Referencias | [FUN-078](01-inventario-funcional.md#fun-078), [FUN-079](01-inventario-funcional.md#fun-079), [FUN-083](01-inventario-funcional.md#fun-083), [FUN-091](01-inventario-funcional.md#fun-091) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-013"></a>

## VAL-013 — Disco lleno, cortes y recuperación del sistema

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Disco lleno, cortes y recuperación del sistema; Procesamiento |
| Prioridad para manuales | Alta |
| Evidencia existente | Space checks, materialization/outcome recovery, rollback, retry tests. |
| Qué falta comprobar | Fallo ENOSPC real por plataforma, caída abrupta y fsync/SMB, extracción/conversión/desinstalación con volumen saturado. |
| Cómo validar con seguridad | Disco virtual acotado/snapshot desechable; probar archivos sintéticos con timeout y monitor; no llenar disco del usuario. |
| Referencias | [FUN-060](01-inventario-funcional.md#fun-060), [FUN-082](01-inventario-funcional.md#fun-082), [FUN-094](01-inventario-funcional.md#fun-094), [FUN-126](01-inventario-funcional.md#fun-126) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-014"></a>

## VAL-014 — UI con permisos parciales y licencia solo lectura

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | UI con permisos parciales y licencia solo lectura; Permisos |
| Prioridad para manuales | Media |
| Evidencia existente | Backend revalida cada escritura; UI uses can_*; Accesos exige ambos permisos; consultas exactas UI/endpoint. |
| Qué falta comprobar | Acciones visibles con roles combinados, retirados durante diálogo, módulos suspendidos; avisos de endpoints no permitidos y actividad de sesión en pantallas con sondeo (touch=true frente a auth/session touch=false). |
| Cómo validar con seguridad | Matriz de cuentas ficticias y licencias simuladas existentes en clone; lectura inicial y posteriores pruebas solo en sandbox. |
| Referencias | [FUN-012](01-inventario-funcional.md#fun-012), [FUN-016](01-inventario-funcional.md#fun-016), [FUN-071](01-inventario-funcional.md#fun-071), [FUN-109](01-inventario-funcional.md#fun-109) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-015"></a>

## VAL-015 — Avisos de cambios y privacidad excepcional

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Avisos de cambios y privacidad excepcional; Búsqueda |
| Prioridad para manuales | Media |
| Evidencia existente | Notifications exige documents.read y no deleted; consultas de documentos aplican visibleDocumentSQL; tests privados y workflow previos. |
| Qué falta comprobar | Combinación avisos de archivo, privacidad staged, integridad dañada, roles retirados y acuse tras cambio. |
| Cómo validar con seguridad | Escenario sintético separado con varias cuentas; examinar respuesta como cada cuenta; no exponer IDs reales. |
| Referencias | [FUN-065](01-inventario-funcional.md#fun-065) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-016"></a>

## VAL-016 — Logs del host y mensajes locales de soporte

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Logs del host y mensajes locales de soporte; Operación |
| Prioridad para manuales | Alta |
| Evidencia existente | HTTP sanitiza errores; diagnóstico allowlist; Windows host registra err.Error() en EventLog al terminar; doctor/CLI muestra errores locales. |
| Qué falta comprobar | Garantizar que errores de arranque de librerías externas no expongan rutas/valores sensibles en todos los hosts. |
| Cómo validar con seguridad | Provocar fallos sintéticos sin credenciales reales y revisar logs sanitizados; limitar acceso a logsSO; no copiar config/identidad. |
| Referencias | [FUN-101](01-inventario-funcional.md#fun-101), [FUN-122](01-inventario-funcional.md#fun-122), [FUN-125](01-inventario-funcional.md#fun-125) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-017"></a>

## VAL-017 — Hardware, capacidad y espacio mínimo publicados

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Hardware, capacidad y espacio mínimo publicados; Instalación |
| Prioridad para manuales | Alta |
| Evidencia existente | Selección conservadora de CPU/memoria y límites por archivo; bytes de cuatro paquetes existentes. |
| Qué falta comprobar | No hay requisito mínimo universal RAM/CPU/disco medido; margen paraOCR, previews, DB y originales; concurrencia real. |
| Cómo validar con seguridad | Medir corpus representativo sintético en equipos acordados, documentar rango/capacidad y margen; no convertir límites configurables en requisitos mínimos. |
| Referencias | [FUN-091](01-inventario-funcional.md#fun-091), [FUN-110](01-inventario-funcional.md#fun-110), [FUN-115](01-inventario-funcional.md#fun-115), [FUN-119](01-inventario-funcional.md#fun-119) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-018"></a>

## VAL-018 — Fuentes y fidelidad de previews

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Fuentes y fidelidad de previews; Previews |
| Prioridad para manuales | Media |
| Evidencia existente | LibreOffice probado con fixtures; originals unchanged/sanitized; PDFderivado y límites. |
| Qué falta comprobar | Fidelidad con fuentes reales corporativas, encabezados/tablas/celdas, plantillas diversas; no equivale a Word/Excel. |
| Cómo validar con seguridad | Documentos de demostración autorizados con fuentes disponibles; comparación visual sin copiar documentos privados a documentación. |
| Referencias | [FUN-035](01-inventario-funcional.md#fun-035), [FUN-094](01-inventario-funcional.md#fun-094) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-019"></a>

## VAL-019 — Codificaciones, archivos poco comunes y corpus ampliado

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Codificaciones, archivos poco comunes y corpus ampliado; Extracción |
| Prioridad para manuales | Media |
| Evidencia existente | DOCX/logo/blank, printer settings, XLSX cached formulas, TXT/CSVUnicode probados; LocalDocumentCorpus omitido en suite final. |
| Qué falta comprobar | Documentos reales autorizados múltiples, CSVambiguo/saltos, UTF16, Límites de rutas UNC/Unicode/reservados y reportes ad hoc. |
| Cómo validar con seguridad | Corpus no privado de fixtures aprobado, hashantes/después, ejecutar solo entorno de pruebas y timeout; no asumirOCRparaOffice. |
| Referencias | [FUN-034](01-inventario-funcional.md#fun-034), [FUN-078](01-inventario-funcional.md#fun-078), [FUN-080](01-inventario-funcional.md#fun-080), [FUN-086](01-inventario-funcional.md#fun-086) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-020"></a>

## VAL-020 — Caducidad temporal, caché y limpieza del SO

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Caducidad temporal, caché y limpieza del SO; Cargas |
| Prioridad para manuales | Media |
| Evidencia existente | Retención upload0–3650 y LRU/cache/hourly/clear tests previos; protección paths. |
| Qué falta comprobar | Retención cronológica a largo plazo y reinicios con temporalescreados antes del fallo; antivirus/locksWindows. |
| Cómo validar con seguridad | VM con tiempos controlados/corpus sintético, verificar originales/hash y privadas aisladas; no borrar pruebas/clientes del workspace. |
| Referencias | [FUN-054](01-inventario-funcional.md#fun-054), [FUN-082](01-inventario-funcional.md#fun-082), [FUN-095](01-inventario-funcional.md#fun-095) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-021"></a>

## VAL-021 — Longitudes Unicode y mensajes exactos de validación

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Longitudes Unicode y mensajes exactos de validación; Interfaz |
| Prioridad para manuales | Media |
| Evidencia existente | UI maxLength, backend len/rune/SQL length varían por campo; Password usa runes; sin criterio uniforme de todos campos. |
| Qué falta comprobar | BordesUnicode en nombres/títulos/identificadores vs límites rotulados; no afirmar que todo límite son caracteres Unicode. |
| Cómo validar con seguridad | Casos sintéticos de emojis/acentos al límite en clone; registrar mensaje/backend HTTP y conservar manual con límite UI hasta validar. |
| Referencias | [FUN-007](01-inventario-funcional.md#fun-007), [FUN-019](01-inventario-funcional.md#fun-019), [FUN-042](01-inventario-funcional.md#fun-042), [FUN-051](01-inventario-funcional.md#fun-051) |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

<a id="val-022"></a>

## VAL-022 — Integridad de evidencia histórica y fidelidad al código actual

| Aspecto | Registro |
| --- | --- |
| Descripción / módulo | Integridad de evidencia histórica y fidelidad al código actual; Documentación |
| Prioridad para manuales | Media |
| Evidencia existente | 413 hashes baseline de archivos de proyecto; informes QA previos; implementación revisada ahora. |
| Qué falta comprobar | Resultados QA previos no certifican cambios posteriores automáticamente; evitar marcar interfaz en vivo por PNG histórico. |
| Cómo validar con seguridad | Mantener revisión de fuente y huellas de docs; antes de manual final revisar diffdel código y capturas aprobadas, pruebas solo cuando nuevo alcance autorice. |
| Referencias | Afecta transversalmente a los siete documentos. |
| Estado | Pendiente; no ejecutado durante esta investigación; requiere entorno de demostración/validación autorizado. |
| Criterio de cierre | Adjuntar evidencia fechada, ambiente/plataforma exactos, resultado esperado/observado y límites; actualizar matriz y capturas sin borrar evidencia histórica. |

## Límites de la investigación realizada

No se abrió SQLite ni se volcaron registros; las relaciones se obtuvieron de migraciones y código. No se instaló/software, ejecutó servicio, abrió sesión, invocó cliente comercial de licencias, modificó firewall o corrió procedimientos destructivos. No se repitieron suites anteriores ni se regeneraron paquetes. No se investigó el servidor del otro proyecto ni los equipos Windows/Ubuntu del usuario. No hubo acceso de navegador a una instancia actual. Las pruebas previas omitidas LocalDocumentCorpus y LiveProvider no se convierten en PASS; pruebas de LibreOffice con fixtures sí figuran aprobadas, con su alcance limitado.

## Funciones que no deben inventarse al redactar

Respaldo/restauración completo por UI/CLI; registro público; recuperación por email; cambio obligatorio de contraseña inicial; editor de roles personalizados; eliminación general de biblioteca; restauración UI de documentos retirados; cálculo Excel; OCR sobre Word/Excel; reproducción de macros; preview visual TXT/CSV; modo nocturno configurable; instalador Mac Intel/Windows ARM; administración del portal/proveedor de licencias; regla automática LAN universal en todos los SO. No se encontraron como funciones del cliente actual. Una política o permiso declarado no basta para presentarlo como implementado.

Para renovar evidencia después de cambiar el código, consultar [matriz](06-matriz-cobertura.md), [arquitectura](04-inventario-tecnico.md) y las fuentes enlazadas; conservar fecha/ambiente de las pruebas y actualizar solo los puntos afectados.

## Comprobación final de la base documental

Revisión de solo lectura realizada el 2026-10-08 sobre esta entrega:

| Comprobación | Resultado |
| --- | --- |
| Entregables | 7%20Markdown%20y%208%20PNG%3B%20todos%20dentro%20de%20documentacion/base-manuales/. |
| Identificadores | 127%20FUN%2C%2053%20PAN%2C%20127%20CAP%20y%2022%20VAL%3B%20definici%F3n%20%FAnica%20y%20referencias%20resueltas. |
| Matriz | 127%20filas%20funcionales%3B%20cubre%20exactamente%20todas%20las%20FUN%20sin%20duplicados. |
| Fuentes%20y%20enlaces%20locales | No%20se%20detectaron%20destinos%20de%20archivo%20inexistentes%20ni%20anchors%20sin%20definici%F3n. |
| Encabezados | Identificaci%F3n%20AIBID%20y%20fecha%3B%20sin%20nombres%20de%20release%20de%20prueba%20ni%20numeraciones%20de%20producto%20en%20encabezados. |
| Bloques%20de%20c%F3digo | Sin%20fences%20Markdown%20abiertos. |
| Confidencialidad | Sin%20claves%20comerciales%20ni%20marcadores%20de%20claves%20privadas%20detectados%3B%20valores%20de%20configuraci%F3n%20privada%20no%20le%EDdos/publicados.%20Los%20hashes%20de%20artefactos%20no%20son%20secretos. |
| Capturas | 8%20copias%20byte%20a%20byte%20de%20im%E1genes%20hist%F3ricas%20de%20fixtures%2C%20requieren%20reemplazo%3B%20119%20escenas%20pendientes%3B%200%20capturas%20nuevas%20de%20UI%20actual%20y%200%20aprobadas%20para%20manual%20final. |
| Archivos%20previos%20del%20proyecto | Comparaci%F3n%20SHA-256%20de%20413%20archivos%20inventariados%20al%20inicio%3A%200%20cambiados%2C%200%20ausentes.%20Se%20preservaron%20los%20cambios%20previos%20ajenos%20a%20esta%20tarea. |
| Ejecuci%F3n | No%20se%20iniciaron%20servicios%2C%20no%20se%20abrieron%20bases%20con%20un%20motor%20SQL%2C%20no%20se%20crearon%20sesiones%20ni%20se%20probaron%20escrituras%3B%20no%20se%20instalaron%20dependencias%2C%20ejecutaron%20suites%20o%20regeneraron%20instaladores. |
| Alcance%20de%20validaci%F3n%20documental | Consistencia%20de%20archivos/referencias%20y%20fuente%20inspeccionada%3B%20no%20certificaci%F3n%20sem%E1ntica%20autom%E1tica%20de%20todos%20los%20procedimientos%20ni%20aprobaci%F3n%20editorial%20de%20capturas. |

Los informes anteriores y scripts originales permanecen en sus ubicaciones. Esta comprobación no cierra los VAL ni produce los manuales finales de instalación, usuario o técnico.

## Evidencia agregada y alcance de las correcciones

Identidad activa corregida a AIBID y 1.0.0; nombres de servicios/programas/paquetes nuevos concordantes. La compatibilidad anterior conserva datos/cuentas y rechaza duplicidad; no se cambian esquemas ni documentos. Transporte PowerShell→NSIS corregido con UTF-16LE y argumentos Windows CRT para SCM. Scripts shell/HTML revisados como UTF-8, perfiles y plantillas comprobados con datos temporales. Los resultados exactos de compilación, paquetes y pruebas están en [informe](../../docs/TRANSICION-PRODUCCION.md).

**VAL-001:** queda pendiente comprobación visual y ciclo completo nativo Windows5.1/NSIS, incluyendo locale español y nombres de usuario con acentos; las pruebas del transporte/quoting y mocks no lo sustituyen. **VAL-003/004:** falta APT/systemd y bundle offline real, tanto instalación nueva como perfil heredado, en rutas con espacios/ñ/Unicode; se retiró la restricción artificial de espacios. **VAL-005:** prueba del binario extraído y recursos plist en macOS no certifica Installer, launchd, permisos de cuenta ni desinstalación del sistema; comprobar también instalación/upgrade/uninstall con usuarios y temporales Unicode. **VAL-008:** confirmar app_version 1.0.0 en el proveedor sin modificar contrato o identidad. **VAL-006:** firmas comerciales y notarización continúan pendientes. Fabricante/contacto definitivo de empaquetado: confirmar con el responsable antes de publicación; actualmente la marca visible es AIBID.

No se generaron nuevas capturas ni manuales Word/PDF. Los 8 PNG históricos se conservan byte a byte y mantienen el estado de reemplazo. La actualización corrige lo demostrado y deja separados los pendientes nativos; no se certifica producción solo por tener paquetes compilados.
