# AIBID — Catálogo de capturas

**Fecha de elaboración: 2026-10-09.** Actualización de la base elaborada el 8 de octubre; nombres y procedimientos contrastados con la transición de producción. Base de trabajo para los futuros manuales; no es un manual final.

## Criterio y estado

**127 escenas necesarias** para los procedimientos y variantes inventariados. **8 archivos históricos reales reutilizados**, todos con estado **requiere reemplazo**. **119 escenas pendientes**. **0 capturas nuevas de la interfaz actual y 0 imágenes aprobadas para insertar en manuales finales**. No se fabricaron interfaces, no se editaron las imágenes y no se copiaron documentos de clientes.

Las imágenes vienen de `web/test-results`, ejecuciones anteriores con fixtures y usuarios ficticios. Se inspeccionaron visualmente antes de reutilizarlas. Algunas llevan nombres de prueba, puertos temporales, rutas de sandbox o encuadre fullpage; por eso se conservan como **evidencia provisional**, no como ilustración editorial aprobada. El nombre nuevo de archivo no cambia lo que aparece dentro de la imagen. Tras la transición se conservan todos los archivos e IDs CAP; cualquier marca anterior o pantalla modificada sigue requiriendo reemplazo, sin retocar imágenes históricas.

No se dispone de una herramienta de navegador conectada para obtener nuevas escenas sin iniciar una sesión. Una sesión real escribe cookies/registro de accesos y la apertura de documentos puede registrar auditoría: no se hizo durante esta tarea de solo investigación. Tampoco se ejecutó el servidor e2e, que crea datos de pruebas.

## Reglas de toma definitiva

Usar datos de demostración previamente autorizados. Excluir personas/clientes reales, contraseñas, claves comerciales, tokens, IDs completos de licencia, rutas privadas y documentos reales. No crear ni alterar datos de producción para obtener un estado, ni ejecutar confirmaciones/desactivaciones/retiros solo para capturar resultados. Mantener pendiente la escena que no exista de forma segura.

Capturar con zoom legible, sin carga ni cortes ni overlays ajenos al diálogo ilustrado. Los diagramas técnicos están en Mermaid en 04; no son pantallas simuladas.

## Evidencia visual conservada

Esta tabla permite comprobar procedencia e integridad. Las copias son byte a byte; no hubo edición, recorte ni redacción de las imágenes. Sus datos son ficticios de pruebas. Todas necesitan aprobación/reemplazo editorial antes de los manuales finales.

| CAP | Origen | Copia | Dimensiones | Bytes | SHA-256 |
| --- | --- | --- | --- | --- | --- |
| [CAP-001](05-catalogo-capturas.md#cap-001) | [web/test-results/login-mobile.png](../../web/test-results/login-mobile.png) | [capturas/CAP-001-iniciar-sesion.png](capturas/CAP-001-iniciar-sesion.png) | 390 × 844 | 35486 | `49370530adbbc289f156457672fe435051694d8302f6fce6a0a12dfa6d6e31bd` |
| [CAP-029](05-catalogo-capturas.md#cap-029) | [web/test-results/h3-explorer.png](../../web/test-results/h3-explorer.png) | [capturas/CAP-029-navegar-carpetas-y-vistas.png](capturas/CAP-029-navegar-carpetas-y-vistas.png) | 1280 × 1504 | 155080 | `3f788a827b2045c56ad3b06c30b69109dd22de7048bfa070474de69ac6df0987` |
| [CAP-035](05-catalogo-capturas.md#cap-035) | [web/test-results/v2-phase4-docx.png](../../web/test-results/v2-phase4-docx.png) | [capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png](capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png) | 1280 × 1525 | 132752 | `6ab2db467f9a5475548e3f083fe4970324b1ccd2382573a4e7cd905c6e132bc5` |
| [CAP-046](05-catalogo-capturas.md#cap-046) | [web/test-results/h4-case.png](../../web/test-results/h4-case.png) | [capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png](capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png) | 1280 × 1469 | 130818 | `713e53d0cf0d50805eaaf365adb2d46c52913f2ff8fea36271897ee179f479ce` |
| [CAP-050](05-catalogo-capturas.md#cap-050) | [web/test-results/v2-phase6-history-desktop.png](../../web/test-results/v2-phase6-history-desktop.png) | [capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png](capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png) | 1800 × 1616 | 207971 | `50e8bea3fe7445ab8bcbbca87fa8cf00ac8061943bcca0c84121086fc4e43e25` |
| [CAP-066](05-catalogo-capturas.md#cap-066) | [web/test-results/r6-duplicates.png](../../web/test-results/r6-duplicates.png) | [capturas/CAP-066-consultar-grupos-de-duplicados.png](capturas/CAP-066-consultar-grupos-de-duplicados.png) | 1280 × 1367 | 166658 | `ce2a970bd794325050e733d0a89fc0d562b8ea1029b44d4335ed13ea1e4ed10c` |
| [CAP-097](05-catalogo-capturas.md#cap-097) | [web/test-results/phase8-settings-desktop.png](../../web/test-results/phase8-settings-desktop.png) | [capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png](capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png) | 1800 × 1464 | 159866 | `80aea36728796462eb8713c9e04a3c1a2f3d7055d25544136797784772ca9722` |
| [CAP-098](05-catalogo-capturas.md#cap-098) | [web/test-results/phase8-dashboard-desktop.png](../../web/test-results/phase8-dashboard-desktop.png) | [capturas/CAP-098-consultar-estadisticas-del-dashboard.png](capturas/CAP-098-consultar-estadisticas-del-dashboard.png) | 1800 × 2367 | 244388 | `95841f2f31c57106fa90b991628b36c432ed94833bca8fca293870ea19b19125` |

## Índice de escenas

| CAP | Nombre | PAN / FUN | Manuales | Estado | Archivo sugerido |
| --- | --- | --- | --- | --- | --- |
| [CAP-001](05-catalogo-capturas.md#cap-001) | Iniciar sesión | [PAN-001](02-pantallas-y-permisos.md#pan-001) / [FUN-001](01-inventario-funcional.md#fun-001) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-001-iniciar-sesion.png](capturas/CAP-001-iniciar-sesion.png) |
| [CAP-002](05-catalogo-capturas.md#cap-002) | Reconocer acceso incorrecto y limitación de intentos | [PAN-001](02-pantallas-y-permisos.md#pan-001) / [FUN-002](01-inventario-funcional.md#fun-002) | Usuario; Técnico | pendiente | `capturas/CAP-002-reconocer-acceso-incorrecto-y-limitacion-de-intentos.png` (pendiente; no existe) |
| [CAP-003](05-catalogo-capturas.md#cap-003) | Cerrar sesión | [PAN-002](02-pantallas-y-permisos.md#pan-002) / [FUN-003](01-inventario-funcional.md#fun-003) | Usuario; Técnico | pendiente | `capturas/CAP-003-cerrar-sesion.png` (pendiente; no existe) |
| [CAP-004](05-catalogo-capturas.md#cap-004) | Cambiar la contraseña propia | [PAN-002](02-pantallas-y-permisos.md#pan-002) / [FUN-004](01-inventario-funcional.md#fun-004) | Usuario; Técnico | pendiente | `capturas/CAP-004-cambiar-la-contrasena-propia.png` (pendiente; no existe) |
| [CAP-005](05-catalogo-capturas.md#cap-005) | Consultar Mi cuenta y Mis accesos | [PAN-002](02-pantallas-y-permisos.md#pan-002) / [FUN-005](01-inventario-funcional.md#fun-005) | Usuario; Técnico | pendiente | `capturas/CAP-005-consultar-mi-cuenta-y-mis-accesos.png` (pendiente; no existe) |
| [CAP-006](05-catalogo-capturas.md#cap-006) | Comprender vencimiento y reemplazo de sesión | [PAN-002](02-pantallas-y-permisos.md#pan-002) / [FUN-006](01-inventario-funcional.md#fun-006) | Usuario; Técnico | pendiente | `capturas/CAP-006-comprender-vencimiento-y-reemplazo-de-sesion.png` (pendiente; no existe) |
| [CAP-007](05-catalogo-capturas.md#cap-007) | Crear usuario | [PAN-004](02-pantallas-y-permisos.md#pan-004) / [FUN-007](01-inventario-funcional.md#fun-007) | Usuario; Técnico | pendiente | `capturas/CAP-007-crear-usuario.png` (pendiente; no existe) |
| [CAP-008](05-catalogo-capturas.md#cap-008) | Editar nombre y deshabilitar cuenta | [PAN-004](02-pantallas-y-permisos.md#pan-004) / [FUN-008](01-inventario-funcional.md#fun-008) | Usuario; Técnico | pendiente | `capturas/CAP-008-editar-nombre-y-deshabilitar-cuenta.png` (pendiente; no existe) |
| [CAP-009](05-catalogo-capturas.md#cap-009) | Asignar perfiles globales | [PAN-004](02-pantallas-y-permisos.md#pan-004) / [FUN-009](01-inventario-funcional.md#fun-009) | Usuario; Técnico | pendiente | `capturas/CAP-009-asignar-perfiles-globales.png` (pendiente; no existe) |
| [CAP-010](05-catalogo-capturas.md#cap-010) | Restablecer contraseña de otro usuario | [PAN-004](02-pantallas-y-permisos.md#pan-004) / [FUN-010](01-inventario-funcional.md#fun-010) | Usuario; Técnico | pendiente | `capturas/CAP-010-restablecer-contrasena-de-otro-usuario.png` (pendiente; no existe) |
| [CAP-011](05-catalogo-capturas.md#cap-011) | Revocar sesiones de una cuenta | [PAN-003](02-pantallas-y-permisos.md#pan-003) / [FUN-011](01-inventario-funcional.md#fun-011) | Usuario; Técnico | pendiente | `capturas/CAP-011-revocar-sesiones-de-una-cuenta.png` (pendiente; no existe) |
| [CAP-012](05-catalogo-capturas.md#cap-012) | Consultar sesiones e intentos de acceso | [PAN-005](02-pantallas-y-permisos.md#pan-005) / [FUN-012](01-inventario-funcional.md#fun-012) | Usuario; Técnico | pendiente | `capturas/CAP-012-consultar-sesiones-e-intentos-de-acceso.png` (pendiente; no existe) |
| [CAP-013](05-catalogo-capturas.md#cap-013) | Listar y abrir bibliotecas | [PAN-008](02-pantallas-y-permisos.md#pan-008) / [FUN-013](01-inventario-funcional.md#fun-013) | Usuario; Técnico | pendiente | `capturas/CAP-013-listar-y-abrir-bibliotecas.png` (pendiente; no existe) |
| [CAP-014](05-catalogo-capturas.md#cap-014) | Crear biblioteca vinculada | [PAN-009](02-pantallas-y-permisos.md#pan-009) / [FUN-014](01-inventario-funcional.md#fun-014) | Usuario; Técnico | pendiente | `capturas/CAP-014-crear-biblioteca-vinculada.png` (pendiente; no existe) |
| [CAP-015](05-catalogo-capturas.md#cap-015) | Crear biblioteca administrada o híbrida | [PAN-009](02-pantallas-y-permisos.md#pan-009) / [FUN-015](01-inventario-funcional.md#fun-015) | Usuario; Técnico | pendiente | `capturas/CAP-015-crear-biblioteca-administrada-o-hibrida.png` (pendiente; no existe) |
| [CAP-016](05-catalogo-capturas.md#cap-016) | Asignar perfiles por biblioteca | [PAN-023](02-pantallas-y-permisos.md#pan-023) / [FUN-016](01-inventario-funcional.md#fun-016) | Usuario; Técnico | pendiente | `capturas/CAP-016-asignar-perfiles-por-biblioteca.png` (pendiente; no existe) |
| [CAP-017](05-catalogo-capturas.md#cap-017) | Cambiar nombre y ampliar modalidad | [PAN-024](02-pantallas-y-permisos.md#pan-024) / [FUN-017](01-inventario-funcional.md#fun-017) | Usuario; Técnico | pendiente | `capturas/CAP-017-cambiar-nombre-y-ampliar-modalidad.png` (pendiente; no existe) |
| [CAP-018](05-catalogo-capturas.md#cap-018) | Habilitar expedientes y ejercicio | [PAN-024](02-pantallas-y-permisos.md#pan-024) / [FUN-018](01-inventario-funcional.md#fun-018) | Usuario; Técnico | pendiente | `capturas/CAP-018-habilitar-expedientes-y-ejercicio.png` (pendiente; no existe) |
| [CAP-019](05-catalogo-capturas.md#cap-019) | Seleccionar destino y reglas de almacenamiento | [PAN-024](02-pantallas-y-permisos.md#pan-024) / [FUN-019](01-inventario-funcional.md#fun-019) | Usuario; Técnico | pendiente | `capturas/CAP-019-seleccionar-destino-y-reglas-de-almacenamiento.png` (pendiente; no existe) |
| [CAP-020](05-catalogo-capturas.md#cap-020) | Consultar ejemplo de nombre y ruta | [PAN-025](02-pantallas-y-permisos.md#pan-025) / [FUN-020](01-inventario-funcional.md#fun-020) | Usuario; Técnico | pendiente | `capturas/CAP-020-consultar-ejemplo-de-nombre-y-ruta.png` (pendiente; no existe) |
| [CAP-021](05-catalogo-capturas.md#cap-021) | Configurar revisión y retención de cargas | [PAN-024](02-pantallas-y-permisos.md#pan-024) / [FUN-021](01-inventario-funcional.md#fun-021) | Usuario; Técnico | pendiente | `capturas/CAP-021-configurar-revision-y-retencion-de-cargas.png` (pendiente; no existe) |
| [CAP-022](05-catalogo-capturas.md#cap-022) | Inspeccionar una ruta de carpeta | [PAN-020](02-pantallas-y-permisos.md#pan-020) / [FUN-022](01-inventario-funcional.md#fun-022) | Usuario; Técnico | pendiente | `capturas/CAP-022-inspeccionar-una-ruta-de-carpeta.png` (pendiente; no existe) |
| [CAP-023](05-catalogo-capturas.md#cap-023) | Registrar raíz vinculada | [PAN-021](02-pantallas-y-permisos.md#pan-021) / [FUN-023](01-inventario-funcional.md#fun-023) | Usuario; Técnico | pendiente | `capturas/CAP-023-registrar-raiz-vinculada.png` (pendiente; no existe) |
| [CAP-024](05-catalogo-capturas.md#cap-024) | Registrar destino administrado | [PAN-021](02-pantallas-y-permisos.md#pan-021) / [FUN-024](01-inventario-funcional.md#fun-024) | Usuario; Técnico | pendiente | `capturas/CAP-024-registrar-destino-administrado.png` (pendiente; no existe) |
| [CAP-025](05-catalogo-capturas.md#cap-025) | Crear vista lógica dentro de una raíz | [PAN-021](02-pantallas-y-permisos.md#pan-021) / [FUN-025](01-inventario-funcional.md#fun-025) | Usuario; Técnico | pendiente | `capturas/CAP-025-crear-vista-logica-dentro-de-una-raiz.png` (pendiente; no existe) |
| [CAP-026](05-catalogo-capturas.md#cap-026) | Consolidar raíces hijas | [PAN-021](02-pantallas-y-permisos.md#pan-021) / [FUN-026](01-inventario-funcional.md#fun-026) | Usuario; Técnico | pendiente | `capturas/CAP-026-consolidar-raices-hijas.png` (pendiente; no existe) |
| [CAP-027](05-catalogo-capturas.md#cap-027) | Programar verificación periódica de carpeta | [PAN-022](02-pantallas-y-permisos.md#pan-022) / [FUN-027](01-inventario-funcional.md#fun-027) | Usuario; Técnico | pendiente | `capturas/CAP-027-programar-verificacion-periodica-de-carpeta.png` (pendiente; no existe) |
| [CAP-028](05-catalogo-capturas.md#cap-028) | Dejar de supervisar una carpeta | [PAN-022](02-pantallas-y-permisos.md#pan-022) / [FUN-028](01-inventario-funcional.md#fun-028) | Usuario; Técnico | pendiente | `capturas/CAP-028-dejar-de-supervisar-una-carpeta.png` (pendiente; no existe) |
| [CAP-029](05-catalogo-capturas.md#cap-029) | Navegar carpetas y vistas | [PAN-011](02-pantallas-y-permisos.md#pan-011) / [FUN-029](01-inventario-funcional.md#fun-029) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-029-navegar-carpetas-y-vistas.png](capturas/CAP-029-navegar-carpetas-y-vistas.png) |
| [CAP-030](05-catalogo-capturas.md#cap-030) | Cambiar presentación y filtros del explorador | [PAN-011](02-pantallas-y-permisos.md#pan-011) / [FUN-030](01-inventario-funcional.md#fun-030) | Usuario; Técnico | pendiente | `capturas/CAP-030-cambiar-presentacion-y-filtros-del-explorador.png` (pendiente; no existe) |
| [CAP-031](05-catalogo-capturas.md#cap-031) | Buscar desde una carpeta | [PAN-011](02-pantallas-y-permisos.md#pan-011) / [FUN-031](01-inventario-funcional.md#fun-031) | Usuario; Técnico | pendiente | `capturas/CAP-031-buscar-desde-una-carpeta.png` (pendiente; no existe) |
| [CAP-032](05-catalogo-capturas.md#cap-032) | Consultar ficha y original | [PAN-012](02-pantallas-y-permisos.md#pan-012) / [FUN-032](01-inventario-funcional.md#fun-032) | Usuario; Técnico | pendiente | `capturas/CAP-032-consultar-ficha-y-original.png` (pendiente; no existe) |
| [CAP-033](05-catalogo-capturas.md#cap-033) | Descargar archivo original | [PAN-012](02-pantallas-y-permisos.md#pan-012) / [FUN-033](01-inventario-funcional.md#fun-033) | Usuario; Técnico | pendiente | `capturas/CAP-033-descargar-archivo-original.png` (pendiente; no existe) |
| [CAP-034](05-catalogo-capturas.md#cap-034) | Consultar texto por unidad y resumen de extracción | [PAN-013](02-pantallas-y-permisos.md#pan-013) / [FUN-034](01-inventario-funcional.md#fun-034) | Usuario; Técnico | pendiente | `capturas/CAP-034-consultar-texto-por-unidad-y-resumen-de-extraccion.png` (pendiente; no existe) |
| [CAP-035](05-catalogo-capturas.md#cap-035) | Solicitar y consultar preview de Word o Excel | [PAN-014](02-pantallas-y-permisos.md#pan-014) / [FUN-035](01-inventario-funcional.md#fun-035) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png](capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png) |
| [CAP-036](05-catalogo-capturas.md#cap-036) | Reindexar un documento | [PAN-015](02-pantallas-y-permisos.md#pan-015) / [FUN-036](01-inventario-funcional.md#fun-036) | Usuario; Técnico | pendiente | `capturas/CAP-036-reindexar-un-documento.png` (pendiente; no existe) |
| [CAP-037](05-catalogo-capturas.md#cap-037) | Retirar documento del índice | [PAN-019](02-pantallas-y-permisos.md#pan-019) / [FUN-037](01-inventario-funcional.md#fun-037) | Usuario; Técnico | pendiente | `capturas/CAP-037-retirar-documento-del-indice.png` (pendiente; no existe) |
| [CAP-038](05-catalogo-capturas.md#cap-038) | Crear, editar y archivar categorías | [PAN-028](02-pantallas-y-permisos.md#pan-028) / [FUN-038](01-inventario-funcional.md#fun-038) | Usuario; Técnico | pendiente | `capturas/CAP-038-crear-editar-y-archivar-categorias.png` (pendiente; no existe) |
| [CAP-039](05-catalogo-capturas.md#cap-039) | Configurar tipos documentales | [PAN-028](02-pantallas-y-permisos.md#pan-028) / [FUN-039](01-inventario-funcional.md#fun-039) | Usuario; Técnico | pendiente | `capturas/CAP-039-configurar-tipos-documentales.png` (pendiente; no existe) |
| [CAP-040](05-catalogo-capturas.md#cap-040) | Crear plantilla y nuevas versiones | [PAN-029](02-pantallas-y-permisos.md#pan-029) / [FUN-040](01-inventario-funcional.md#fun-040) | Usuario; Técnico | pendiente | `capturas/CAP-040-crear-plantilla-y-nuevas-versiones.png` (pendiente; no existe) |
| [CAP-041](05-catalogo-capturas.md#cap-041) | Listar y filtrar expedientes | [PAN-030](02-pantallas-y-permisos.md#pan-030) / [FUN-041](01-inventario-funcional.md#fun-041) | Usuario; Técnico | pendiente | `capturas/CAP-041-listar-y-filtrar-expedientes.png` (pendiente; no existe) |
| [CAP-042](05-catalogo-capturas.md#cap-042) | Crear expediente | [PAN-030](02-pantallas-y-permisos.md#pan-030) / [FUN-042](01-inventario-funcional.md#fun-042) | Usuario; Técnico | pendiente | `capturas/CAP-042-crear-expediente.png` (pendiente; no existe) |
| [CAP-043](05-catalogo-capturas.md#cap-043) | Editar identificador y ejercicio | [PAN-031](02-pantallas-y-permisos.md#pan-031) / [FUN-043](01-inventario-funcional.md#fun-043) | Usuario; Técnico | pendiente | `capturas/CAP-043-editar-identificador-y-ejercicio.png` (pendiente; no existe) |
| [CAP-044](05-catalogo-capturas.md#cap-044) | Inicializar requisitos desde plantilla | [PAN-031](02-pantallas-y-permisos.md#pan-031) / [FUN-044](01-inventario-funcional.md#fun-044) | Usuario; Técnico | pendiente | `capturas/CAP-044-inicializar-requisitos-desde-plantilla.png` (pendiente; no existe) |
| [CAP-045](05-catalogo-capturas.md#cap-045) | Ajustar cantidades y obligatoriedad del expediente | [PAN-031](02-pantallas-y-permisos.md#pan-031) / [FUN-045](01-inventario-funcional.md#fun-045) | Usuario; Técnico | pendiente | `capturas/CAP-045-ajustar-cantidades-y-obligatoriedad-del-expediente.png` (pendiente; no existe) |
| [CAP-046](05-catalogo-capturas.md#cap-046) | Interpretar avance y documentos de un expediente | [PAN-031](02-pantallas-y-permisos.md#pan-031) / [FUN-046](01-inventario-funcional.md#fun-046) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png](capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png) |
| [CAP-047](05-catalogo-capturas.md#cap-047) | Preparar carga múltiple | [PAN-032](02-pantallas-y-permisos.md#pan-032) / [FUN-047](01-inventario-funcional.md#fun-047) | Usuario; Técnico | pendiente | `capturas/CAP-047-preparar-carga-multiple.png` (pendiente; no existe) |
| [CAP-048](05-catalogo-capturas.md#cap-048) | Seguir carga y continuar tras error individual | [PAN-032](02-pantallas-y-permisos.md#pan-032) / [FUN-048](01-inventario-funcional.md#fun-048) | Usuario; Técnico | pendiente | `capturas/CAP-048-seguir-carga-y-continuar-tras-error-individual.png` (pendiente; no existe) |
| [CAP-049](05-catalogo-capturas.md#cap-049) | Iniciar otra carga conservando borradores previos | [PAN-032](02-pantallas-y-permisos.md#pan-032) / [FUN-049](01-inventario-funcional.md#fun-049) | Usuario; Técnico | pendiente | `capturas/CAP-049-iniciar-otra-carga-conservando-borradores-previos.png` (pendiente; no existe) |
| [CAP-050](05-catalogo-capturas.md#cap-050) | Consultar historial de cargas y detalle de lote | [PAN-033](02-pantallas-y-permisos.md#pan-033) / [FUN-050](01-inventario-funcional.md#fun-050) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png](capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png) |
| [CAP-051](05-catalogo-capturas.md#cap-051) | Clasificar borrador | [PAN-016](02-pantallas-y-permisos.md#pan-016) / [FUN-051](01-inventario-funcional.md#fun-051) | Usuario; Técnico | pendiente | `capturas/CAP-051-clasificar-borrador.png` (pendiente; no existe) |
| [CAP-052](05-catalogo-capturas.md#cap-052) | Asociar vinculado a expediente | [PAN-016](02-pantallas-y-permisos.md#pan-016) / [FUN-052](01-inventario-funcional.md#fun-052) | Usuario; Técnico | pendiente | `capturas/CAP-052-asociar-vinculado-a-expediente.png` (pendiente; no existe) |
| [CAP-053](05-catalogo-capturas.md#cap-053) | Reasignar documento a otro expediente | [PAN-016](02-pantallas-y-permisos.md#pan-016) / [FUN-053](01-inventario-funcional.md#fun-053) | Usuario; Técnico | pendiente | `capturas/CAP-053-reasignar-documento-a-otro-expediente.png` (pendiente; no existe) |
| [CAP-054](05-catalogo-capturas.md#cap-054) | Cancelar carga propia | [PAN-034](02-pantallas-y-permisos.md#pan-034) / [FUN-054](01-inventario-funcional.md#fun-054) | Usuario; Técnico | pendiente | `capturas/CAP-054-cancelar-carga-propia.png` (pendiente; no existe) |
| [CAP-055](05-catalogo-capturas.md#cap-055) | Enviar a revisión y actualizar envío | [PAN-017](02-pantallas-y-permisos.md#pan-017) / [FUN-055](01-inventario-funcional.md#fun-055) | Usuario; Técnico | pendiente | `capturas/CAP-055-enviar-a-revision-y-actualizar-envio.png` (pendiente; no existe) |
| [CAP-056](05-catalogo-capturas.md#cap-056) | Consultar revisiones pendientes | [PAN-035](02-pantallas-y-permisos.md#pan-035) / [FUN-056](01-inventario-funcional.md#fun-056) | Usuario; Técnico | pendiente | `capturas/CAP-056-consultar-revisiones-pendientes.png` (pendiente; no existe) |
| [CAP-057](05-catalogo-capturas.md#cap-057) | Aprobar un documento | [PAN-017](02-pantallas-y-permisos.md#pan-017) / [FUN-057](01-inventario-funcional.md#fun-057) | Usuario; Técnico | pendiente | `capturas/CAP-057-aprobar-un-documento.png` (pendiente; no existe) |
| [CAP-058](05-catalogo-capturas.md#cap-058) | Rechazar un documento | [PAN-017](02-pantallas-y-permisos.md#pan-017) / [FUN-058](01-inventario-funcional.md#fun-058) | Usuario; Técnico | pendiente | `capturas/CAP-058-rechazar-un-documento.png` (pendiente; no existe) |
| [CAP-059](05-catalogo-capturas.md#cap-059) | Confirmar sin revisión | [PAN-017](02-pantallas-y-permisos.md#pan-017) / [FUN-059](01-inventario-funcional.md#fun-059) | Usuario; Técnico | pendiente | `capturas/CAP-059-confirmar-sin-revision.png` (pendiente; no existe) |
| [CAP-060](05-catalogo-capturas.md#cap-060) | Seguir y reintentar incorporación administrada | [PAN-017](02-pantallas-y-permisos.md#pan-017) / [FUN-060](01-inventario-funcional.md#fun-060) | Usuario; Técnico | pendiente | `capturas/CAP-060-seguir-y-reintentar-incorporacion-administrada.png` (pendiente; no existe) |
| [CAP-061](05-catalogo-capturas.md#cap-061) | Realizar búsqueda general | [PAN-038](02-pantallas-y-permisos.md#pan-038) / [FUN-061](01-inventario-funcional.md#fun-061) | Usuario; Técnico | pendiente | `capturas/CAP-061-realizar-busqueda-general.png` (pendiente; no existe) |
| [CAP-062](05-catalogo-capturas.md#cap-062) | Aplicar filtros documentales de búsqueda | [PAN-038](02-pantallas-y-permisos.md#pan-038) / [FUN-062](01-inventario-funcional.md#fun-062) | Usuario; Técnico | pendiente | `capturas/CAP-062-aplicar-filtros-documentales-de-busqueda.png` (pendiente; no existe) |
| [CAP-063](05-catalogo-capturas.md#cap-063) | Buscar por catálogo, ejercicio y expediente | [PAN-038](02-pantallas-y-permisos.md#pan-038) / [FUN-063](01-inventario-funcional.md#fun-063) | Usuario; Técnico | pendiente | `capturas/CAP-063-buscar-por-catalogo-ejercicio-y-expediente.png` (pendiente; no existe) |
| [CAP-064](05-catalogo-capturas.md#cap-064) | Navegar resultados agrupados y coincidencias | [PAN-038](02-pantallas-y-permisos.md#pan-038) / [FUN-064](01-inventario-funcional.md#fun-064) | Usuario; Técnico | pendiente | `capturas/CAP-064-navegar-resultados-agrupados-y-coincidencias.png` (pendiente; no existe) |
| [CAP-065](05-catalogo-capturas.md#cap-065) | Consultar y reconocer cambios documentales | [PAN-039](02-pantallas-y-permisos.md#pan-039) / [FUN-065](01-inventario-funcional.md#fun-065) | Usuario; Técnico | pendiente | `capturas/CAP-065-consultar-y-reconocer-cambios-documentales.png` (pendiente; no existe) |
| [CAP-066](05-catalogo-capturas.md#cap-066) | Consultar grupos de duplicados | [PAN-037](02-pantallas-y-permisos.md#pan-037) / [FUN-066](01-inventario-funcional.md#fun-066) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-066-consultar-grupos-de-duplicados.png](capturas/CAP-066-consultar-grupos-de-duplicados.png) |
| [CAP-067](05-catalogo-capturas.md#cap-067) | Consultar huella e historial documental | [PAN-018](02-pantallas-y-permisos.md#pan-018) / [FUN-067](01-inventario-funcional.md#fun-067) | Usuario; Técnico | pendiente | `capturas/CAP-067-consultar-huella-e-historial-documental.png` (pendiente; no existe) |
| [CAP-068](05-catalogo-capturas.md#cap-068) | Filtrar y ordenar auditoría global | [PAN-006](02-pantallas-y-permisos.md#pan-006) / [FUN-068](01-inventario-funcional.md#fun-068) | Usuario; Técnico | pendiente | `capturas/CAP-068-filtrar-y-ordenar-auditoria-global.png` (pendiente; no existe) |
| [CAP-069](05-catalogo-capturas.md#cap-069) | Leer detalle de evento y registro original | [PAN-007](02-pantallas-y-permisos.md#pan-007) / [FUN-069](01-inventario-funcional.md#fun-069) | Usuario; Técnico | pendiente | `capturas/CAP-069-leer-detalle-de-evento-y-registro-original.png` (pendiente; no existe) |
| [CAP-070](05-catalogo-capturas.md#cap-070) | Consultar auditoría de biblioteca | [PAN-040](02-pantallas-y-permisos.md#pan-040) / [FUN-070](01-inventario-funcional.md#fun-070) | Usuario; Técnico | pendiente | `capturas/CAP-070-consultar-auditoria-de-biblioteca.png` (pendiente; no existe) |
| [CAP-071](05-catalogo-capturas.md#cap-071) | Consultar registro exacto de búsquedas | [PAN-010](02-pantallas-y-permisos.md#pan-010) / [FUN-071](01-inventario-funcional.md#fun-071) | Usuario; Técnico | pendiente | `capturas/CAP-071-consultar-registro-exacto-de-busquedas.png` (pendiente; no existe) |
| [CAP-072](05-catalogo-capturas.md#cap-072) | Solicitar verificación manual de biblioteca | [PAN-010](02-pantallas-y-permisos.md#pan-010) / [FUN-072](01-inventario-funcional.md#fun-072) | Usuario; Técnico | pendiente | `capturas/CAP-072-solicitar-verificacion-manual-de-biblioteca.png` (pendiente; no existe) |
| [CAP-073](05-catalogo-capturas.md#cap-073) | Consultar resumen, progreso y tareas | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-073](01-inventario-funcional.md#fun-073) | Usuario; Técnico | pendiente | `capturas/CAP-073-consultar-resumen-progreso-y-tareas.png` (pendiente; no existe) |
| [CAP-074](05-catalogo-capturas.md#cap-074) | Cancelar recorrido y reanudar raíz | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-074](01-inventario-funcional.md#fun-074) | Usuario; Técnico | pendiente | `capturas/CAP-074-cancelar-recorrido-y-reanudar-raiz.png` (pendiente; no existe) |
| [CAP-075](05-catalogo-capturas.md#cap-075) | Cancelar extracción pendiente o activa | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-075](01-inventario-funcional.md#fun-075) | Usuario; Técnico | pendiente | `capturas/CAP-075-cancelar-extraccion-pendiente-o-activa.png` (pendiente; no existe) |
| [CAP-076](05-catalogo-capturas.md#cap-076) | Reintentar tarea fallida | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-076](01-inventario-funcional.md#fun-076) | Usuario; Técnico | pendiente | `capturas/CAP-076-reintentar-tarea-fallida.png` (pendiente; no existe) |
| [CAP-077](05-catalogo-capturas.md#cap-077) | Reintentar rutas con error y consultar omitidos | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-077](01-inventario-funcional.md#fun-077) | Usuario; Técnico | pendiente | `capturas/CAP-077-reintentar-rutas-con-error-y-consultar-omitidos.png` (pendiente; no existe) |
| [CAP-078](05-catalogo-capturas.md#cap-078) | Comprender descubrimiento, hash y cambios vinculados | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-078](01-inventario-funcional.md#fun-078) | Usuario; Técnico | pendiente | `capturas/CAP-078-comprender-descubrimiento-hash-y-cambios-vinculados.png` (pendiente; no existe) |
| [CAP-079](05-catalogo-capturas.md#cap-079) | Comprender vigilancia y pérdida de eventos | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-079](01-inventario-funcional.md#fun-079) | Usuario; Técnico | pendiente | `capturas/CAP-079-comprender-vigilancia-y-perdida-de-eventos.png` (pendiente; no existe) |
| [CAP-080](05-catalogo-capturas.md#cap-080) | Comprender extracción por formato | [PAN-013](02-pantallas-y-permisos.md#pan-013) / [FUN-080](01-inventario-funcional.md#fun-080) | Usuario; Técnico | pendiente | `capturas/CAP-080-comprender-extraccion-por-formato.png` (pendiente; no existe) |
| [CAP-081](05-catalogo-capturas.md#cap-081) | Comprender OCR de PDF | [PAN-013](02-pantallas-y-permisos.md#pan-013) / [FUN-081](01-inventario-funcional.md#fun-081) | Usuario; Técnico | pendiente | `capturas/CAP-081-comprender-ocr-de-pdf.png` (pendiente; no existe) |
| [CAP-082](05-catalogo-capturas.md#cap-082) | Comprender recuperación de cola tras reinicio | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-082](01-inventario-funcional.md#fun-082) | Usuario; Técnico | pendiente | `capturas/CAP-082-comprender-recuperacion-de-cola-tras-reinicio.png` (pendiente; no existe) |
| [CAP-083](05-catalogo-capturas.md#cap-083) | Verificar integridad administrada | [PAN-036](02-pantallas-y-permisos.md#pan-036) / [FUN-083](01-inventario-funcional.md#fun-083) | Usuario; Técnico | pendiente | `capturas/CAP-083-verificar-integridad-administrada.png` (pendiente; no existe) |
| [CAP-084](05-catalogo-capturas.md#cap-084) | Configurar reglas globales de archivos | [PAN-042](02-pantallas-y-permisos.md#pan-042) / [FUN-084](01-inventario-funcional.md#fun-084) | Usuario; Técnico | pendiente | `capturas/CAP-084-configurar-reglas-globales-de-archivos.png` (pendiente; no existe) |
| [CAP-085](05-catalogo-capturas.md#cap-085) | Configurar herencia de reglas de archivo | [PAN-026](02-pantallas-y-permisos.md#pan-026) / [FUN-085](01-inventario-funcional.md#fun-085) | Usuario; Técnico | pendiente | `capturas/CAP-085-configurar-herencia-de-reglas-de-archivo.png` (pendiente; no existe) |
| [CAP-086](05-catalogo-capturas.md#cap-086) | Interpretar rechazo de formato o contenido | [PAN-042](02-pantallas-y-permisos.md#pan-042) / [FUN-086](01-inventario-funcional.md#fun-086) | Usuario; Técnico | pendiente | `capturas/CAP-086-interpretar-rechazo-de-formato-o-contenido.png` (pendiente; no existe) |
| [CAP-087](05-catalogo-capturas.md#cap-087) | Configurar automatización y prioridad manual | [PAN-044](02-pantallas-y-permisos.md#pan-044) / [FUN-087](01-inventario-funcional.md#fun-087) | Usuario; Técnico | pendiente | `capturas/CAP-087-configurar-automatizacion-y-prioridad-manual.png` (pendiente; no existe) |
| [CAP-088](05-catalogo-capturas.md#cap-088) | Configurar parámetros OCR globales | [PAN-044](02-pantallas-y-permisos.md#pan-044) / [FUN-088](01-inventario-funcional.md#fun-088) | Usuario; Técnico | pendiente | `capturas/CAP-088-configurar-parametros-ocr-globales.png` (pendiente; no existe) |
| [CAP-089](05-catalogo-capturas.md#cap-089) | Configurar vigilancia y archivos ignorados | [PAN-044](02-pantallas-y-permisos.md#pan-044) / [FUN-089](01-inventario-funcional.md#fun-089) | Usuario; Técnico | pendiente | `capturas/CAP-089-configurar-vigilancia-y-archivos-ignorados.png` (pendiente; no existe) |
| [CAP-090](05-catalogo-capturas.md#cap-090) | Sobrescribir reglas avanzadas por biblioteca | [PAN-027](02-pantallas-y-permisos.md#pan-027) / [FUN-090](01-inventario-funcional.md#fun-090) | Usuario; Técnico | pendiente | `capturas/CAP-090-sobrescribir-reglas-avanzadas-por-biblioteca.png` (pendiente; no existe) |
| [CAP-091](05-catalogo-capturas.md#cap-091) | Consultar recursos y configurar concurrencia | [PAN-043](02-pantallas-y-permisos.md#pan-043) / [FUN-091](01-inventario-funcional.md#fun-091) | Usuario; Técnico | pendiente | `capturas/CAP-091-consultar-recursos-y-configurar-concurrencia.png` (pendiente; no existe) |
| [CAP-092](05-catalogo-capturas.md#cap-092) | Pausar y reanudar procesamiento general | [PAN-043](02-pantallas-y-permisos.md#pan-043) / [FUN-092](01-inventario-funcional.md#fun-092) | Usuario; Técnico | pendiente | `capturas/CAP-092-pausar-y-reanudar-procesamiento-general.png` (pendiente; no existe) |
| [CAP-093](05-catalogo-capturas.md#cap-093) | Configurar intentos y tiempos documentales | [PAN-043](02-pantallas-y-permisos.md#pan-043) / [FUN-093](01-inventario-funcional.md#fun-093) | Usuario; Técnico | pendiente | `capturas/CAP-093-configurar-intentos-y-tiempos-documentales.png` (pendiente; no existe) |
| [CAP-094](05-catalogo-capturas.md#cap-094) | Configurar previews, convertidor y caché | [PAN-045](02-pantallas-y-permisos.md#pan-045) / [FUN-094](01-inventario-funcional.md#fun-094) | Usuario; Técnico | pendiente | `capturas/CAP-094-configurar-previews-convertidor-y-cache.png` (pendiente; no existe) |
| [CAP-095](05-catalogo-capturas.md#cap-095) | Vaciar caché de vistas previas | [PAN-045](02-pantallas-y-permisos.md#pan-045) / [FUN-095](01-inventario-funcional.md#fun-095) | Usuario; Técnico | pendiente | `capturas/CAP-095-vaciar-cache-de-vistas-previas.png` (pendiente; no existe) |
| [CAP-096](05-catalogo-capturas.md#cap-096) | Habilitar acceso LAN o regresar a local | [PAN-046](02-pantallas-y-permisos.md#pan-046) / [FUN-096](01-inventario-funcional.md#fun-096) | Usuario; Técnico | pendiente | `capturas/CAP-096-habilitar-acceso-lan-o-regresar-a-local.png` (pendiente; no existe) |
| [CAP-097](05-catalogo-capturas.md#cap-097) | Consultar diagnóstico de acceso y firewall | [PAN-046](02-pantallas-y-permisos.md#pan-046) / [FUN-097](01-inventario-funcional.md#fun-097) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png](capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png) |
| [CAP-098](05-catalogo-capturas.md#cap-098) | Consultar estadísticas del Dashboard | [PAN-041](02-pantallas-y-permisos.md#pan-041) / [FUN-098](01-inventario-funcional.md#fun-098) | Usuario; Técnico | requiere reemplazo | [capturas/CAP-098-consultar-estadisticas-del-dashboard.png](capturas/CAP-098-consultar-estadisticas-del-dashboard.png) |
| [CAP-099](05-catalogo-capturas.md#cap-099) | Consultar gráficos, tablas accesibles y detalle por biblioteca | [PAN-041](02-pantallas-y-permisos.md#pan-041) / [FUN-099](01-inventario-funcional.md#fun-099) | Usuario; Técnico | pendiente | `capturas/CAP-099-consultar-graficos-tablas-accesibles-y-detalle-por-biblioteca.png` (pendiente; no existe) |
| [CAP-100](05-catalogo-capturas.md#cap-100) | Filtrar errores y consultar contexto | [PAN-047](02-pantallas-y-permisos.md#pan-047) / [FUN-100](01-inventario-funcional.md#fun-100) | Usuario; Técnico | pendiente | `capturas/CAP-100-filtrar-errores-y-consultar-contexto.png` (pendiente; no existe) |
| [CAP-101](05-catalogo-capturas.md#cap-101) | Marcar diagnóstico y copiar información de soporte | [PAN-047](02-pantallas-y-permisos.md#pan-047) / [FUN-101](01-inventario-funcional.md#fun-101) | Usuario; Técnico | pendiente | `capturas/CAP-101-marcar-diagnostico-y-copiar-informacion-de-soporte.png` (pendiente; no existe) |
| [CAP-102](05-catalogo-capturas.md#cap-102) | Configurar retención de diagnósticos | [PAN-047](02-pantallas-y-permisos.md#pan-047) / [FUN-102](01-inventario-funcional.md#fun-102) | Usuario; Técnico | pendiente | `capturas/CAP-102-configurar-retencion-de-diagnosticos.png` (pendiente; no existe) |
| [CAP-103](05-catalogo-capturas.md#cap-103) | Consultar estado, módulos y datos de licencia | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-103](01-inventario-funcional.md#fun-103) | Usuario; Técnico | pendiente | `capturas/CAP-103-consultar-estado-modulos-y-datos-de-licencia.png` (pendiente; no existe) |
| [CAP-104](05-catalogo-capturas.md#cap-104) | Activar licencia en línea | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-104](01-inventario-funcional.md#fun-104) | Usuario; Técnico | pendiente | `capturas/CAP-104-activar-licencia-en-linea.png` (pendiente; no existe) |
| [CAP-105](05-catalogo-capturas.md#cap-105) | Renovar estado y retomar solicitud de licencia | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-105](01-inventario-funcional.md#fun-105) | Usuario; Técnico | pendiente | `capturas/CAP-105-renovar-estado-y-retomar-solicitud-de-licencia.png` (pendiente; no existe) |
| [CAP-106](05-catalogo-capturas.md#cap-106) | Desactivar licencia en línea | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-106](01-inventario-funcional.md#fun-106) | Usuario; Técnico | pendiente | `capturas/CAP-106-desactivar-licencia-en-linea.png` (pendiente; no existe) |
| [CAP-107](05-catalogo-capturas.md#cap-107) | Generar solicitud de licencia sin conexión | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-107](01-inventario-funcional.md#fun-107) | Usuario; Técnico | pendiente | `capturas/CAP-107-generar-solicitud-de-licencia-sin-conexion.png` (pendiente; no existe) |
| [CAP-108](05-catalogo-capturas.md#cap-108) | Importar respuesta firmada y consultar artefactos | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-108](01-inventario-funcional.md#fun-108) | Usuario; Técnico | pendiente | `capturas/CAP-108-importar-respuesta-firmada-y-consultar-artefactos.png` (pendiente; no existe) |
| [CAP-109](05-catalogo-capturas.md#cap-109) | Interpretar solo lectura y avisos de reloj/identidad | [PAN-048](02-pantallas-y-permisos.md#pan-048) / [FUN-109](01-inventario-funcional.md#fun-109) | Usuario; Técnico | pendiente | `capturas/CAP-109-interpretar-solo-lectura-y-avisos-de-reloj-identidad.png` (pendiente; no existe) |
| [CAP-110](05-catalogo-capturas.md#cap-110) | Instalar en Windows | [PAN-049](02-pantallas-y-permisos.md#pan-049) / [FUN-110](01-inventario-funcional.md#fun-110) | Instalación; Técnico | pendiente | `capturas/CAP-110-instalar-en-windows.png` (pendiente; no existe) |
| [CAP-111](05-catalogo-capturas.md#cap-111) | Crear administrador e iniciar en Windows | [PAN-049](02-pantallas-y-permisos.md#pan-049) / [FUN-111](01-inventario-funcional.md#fun-111) | Instalación; Técnico | pendiente | `capturas/CAP-111-crear-administrador-e-iniciar-en-windows.png` (pendiente; no existe) |
| [CAP-112](05-catalogo-capturas.md#cap-112) | Abrir aplicación instalada desde acceso directo | [PAN-049](02-pantallas-y-permisos.md#pan-049) / [FUN-112](01-inventario-funcional.md#fun-112) | Instalación; Técnico | pendiente | `capturas/CAP-112-abrir-aplicacion-instalada-desde-acceso-directo.png` (pendiente; no existe) |
| [CAP-113](05-catalogo-capturas.md#cap-113) | Desinstalar en Windows conservando datos | [PAN-050](02-pantallas-y-permisos.md#pan-050) / [FUN-113](01-inventario-funcional.md#fun-113) | Instalación; Técnico | pendiente | `capturas/CAP-113-desinstalar-en-windows-conservando-datos.png` (pendiente; no existe) |
| [CAP-114](05-catalogo-capturas.md#cap-114) | Desinstalar y reiniciar datos internos en Windows | [PAN-050](02-pantallas-y-permisos.md#pan-050) / [FUN-114](01-inventario-funcional.md#fun-114) | Instalación; Técnico | pendiente | `capturas/CAP-114-desinstalar-y-reiniciar-datos-internos-en-windows.png` (pendiente; no existe) |
| [CAP-115](05-catalogo-capturas.md#cap-115) | Instalar en Ubuntu con paquete de arquitectura correcta | [PAN-051](02-pantallas-y-permisos.md#pan-051) / [FUN-115](01-inventario-funcional.md#fun-115) | Instalación; Técnico | pendiente | `capturas/CAP-115-instalar-en-ubuntu-con-paquete-de-arquitectura-correcta.png` (pendiente; no existe) |
| [CAP-116](05-catalogo-capturas.md#cap-116) | Crear administrador y arrancar en Ubuntu | [PAN-051](02-pantallas-y-permisos.md#pan-051) / [FUN-116](01-inventario-funcional.md#fun-116) | Instalación; Técnico | pendiente | `capturas/CAP-116-crear-administrador-y-arrancar-en-ubuntu.png` (pendiente; no existe) |
| [CAP-117](05-catalogo-capturas.md#cap-117) | Preparar e instalar bundle Ubuntu sin red | [PAN-051](02-pantallas-y-permisos.md#pan-051) / [FUN-117](01-inventario-funcional.md#fun-117) | Instalación; Técnico | pendiente | `capturas/CAP-117-preparar-e-instalar-bundle-ubuntu-sin-red.png` (pendiente; no existe) |
| [CAP-118](05-catalogo-capturas.md#cap-118) | Desinstalar en Ubuntu | [PAN-051](02-pantallas-y-permisos.md#pan-051) / [FUN-118](01-inventario-funcional.md#fun-118) | Instalación; Técnico | pendiente | `capturas/CAP-118-desinstalar-en-ubuntu.png` (pendiente; no existe) |
| [CAP-119](05-catalogo-capturas.md#cap-119) | Preparar dependencias e instalar en macOS | [PAN-052](02-pantallas-y-permisos.md#pan-052) / [FUN-119](01-inventario-funcional.md#fun-119) | Instalación; Técnico | pendiente | `capturas/CAP-119-preparar-dependencias-e-instalar-en-macos.png` (pendiente; no existe) |
| [CAP-120](05-catalogo-capturas.md#cap-120) | Configurar primer acceso y servicio en macOS | [PAN-052](02-pantallas-y-permisos.md#pan-052) / [FUN-120](01-inventario-funcional.md#fun-120) | Instalación; Técnico | pendiente | `capturas/CAP-120-configurar-primer-acceso-y-servicio-en-macos.png` (pendiente; no existe) |
| [CAP-121](05-catalogo-capturas.md#cap-121) | Desinstalar en macOS | [PAN-052](02-pantallas-y-permisos.md#pan-052) / [FUN-121](01-inventario-funcional.md#fun-121) | Instalación; Técnico | pendiente | `capturas/CAP-121-desinstalar-en-macos.png` (pendiente; no existe) |
| [CAP-122](05-catalogo-capturas.md#cap-122) | Consultar versión y diagnóstico local | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-122](01-inventario-funcional.md#fun-122) | Instalación; Técnico | pendiente | `capturas/CAP-122-consultar-version-y-diagnostico-local.png` (pendiente; no existe) |
| [CAP-123](05-catalogo-capturas.md#cap-123) | Recuperar administrador local | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-123](01-inventario-funcional.md#fun-123) | Instalación; Técnico | pendiente | `capturas/CAP-123-recuperar-administrador-local.png` (pendiente; no existe) |
| [CAP-124](05-catalogo-capturas.md#cap-124) | Recuperar identidad de licencia dañada | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-124](01-inventario-funcional.md#fun-124) | Instalación; Técnico | pendiente | `capturas/CAP-124-recuperar-identidad-de-licencia-danada.png` (pendiente; no existe) |
| [CAP-125](05-catalogo-capturas.md#cap-125) | Administrar servicio y logs por plataforma | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-125](01-inventario-funcional.md#fun-125) | Instalación; Técnico | pendiente | `capturas/CAP-125-administrar-servicio-y-logs-por-plataforma.png` (pendiente; no existe) |
| [CAP-126](05-catalogo-capturas.md#cap-126) | Actualizar instalación conservando estado | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-126](01-inventario-funcional.md#fun-126) | Instalación; Técnico | pendiente | `capturas/CAP-126-actualizar-instalacion-conservando-estado.png` (pendiente; no existe) |
| [CAP-127](05-catalogo-capturas.md#cap-127) | Comprender inicialización, migración y mantenimiento restringido | [PAN-053](02-pantallas-y-permisos.md#pan-053) / [FUN-127](01-inventario-funcional.md#fun-127) | Instalación; Técnico | pendiente | `capturas/CAP-127-comprender-inicializacion-migracion-y-mantenimiento-restringido.png` (pendiente; no existe) |

<a id="cap-001"></a>

### CAP-001 — Iniciar sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-001](02-pantallas-y-permisos.md#pan-001) |
| Funcionalidad / objetivo | [FUN-001](01-inventario-funcional.md#fun-001): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Marca; formulario; error; aviso de recuperación. Campos/controles: Usuario; Contraseña. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Cuenta creada; navegador admite cookies del origen. Permisos: Cuenta habilitada; sin rol previo. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-001-iniciar-sesion.png` |
| Estado | requiere reemplazo |
| Observaciones | Error de acceso con usuario sintético; sustituir por campos vacíos/estado editorial. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/login-mobile.png](../../web/test-results/login-mobile.png); copia byte a byte en capturas/CAP-001-iniciar-sesion.png. |

<a id="cap-002"></a>

### CAP-002 — Reconocer acceso incorrecto y limitación de intentos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-001](02-pantallas-y-permisos.md#pan-001) |
| Funcionalidad / objetivo | [FUN-002](01-inventario-funcional.md#fun-002): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Marca; formulario; error; aviso de recuperación. Campos/controles: Usuario; Contraseña. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Público. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-002-reconocer-acceso-incorrecto-y-limitacion-de-intentos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-003"></a>

### CAP-003 — Cerrar sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-002](02-pantallas-y-permisos.md#pan-002) |
| Funcionalidad / objetivo | [FUN-003](01-inventario-funcional.md#fun-003): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos de cuenta; formulario; Mis accesos. Campos/controles: Sin campos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Sesión activa. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-003-cerrar-sesion.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-004"></a>

### CAP-004 — Cambiar la contraseña propia

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-002](02-pantallas-y-permisos.md#pan-002) |
| Funcionalidad / objetivo | [FUN-004](01-inventario-funcional.md#fun-004): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos de cuenta; formulario; Mis accesos. Campos/controles: Contraseña actual; Nueva contraseña; Repetir nueva contraseña. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Sesión activa. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-004-cambiar-la-contrasena-propia.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-005"></a>

### CAP-005 — Consultar Mi cuenta y Mis accesos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-002](02-pantallas-y-permisos.md#pan-002) |
| Funcionalidad / objetivo | [FUN-005](01-inventario-funcional.md#fun-005): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos de cuenta; formulario; Mis accesos. Campos/controles: Paginación de accesos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Sesión propia. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-005-consultar-mi-cuenta-y-mis-accesos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-006"></a>

### CAP-006 — Comprender vencimiento y reemplazo de sesión

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-002](02-pantallas-y-permisos.md#pan-002) |
| Funcionalidad / objetivo | [FUN-006](01-inventario-funcional.md#fun-006): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos de cuenta; formulario; Mis accesos. Campos/controles: Sin formulario; tiempos en config.json. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Sesión activa. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-006-comprender-vencimiento-y-reemplazo-de-sesion.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-007"></a>

### CAP-007 — Crear usuario

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-004](02-pantallas-y-permisos.md#pan-004) |
| Funcionalidad / objetivo | [FUN-007](01-inventario-funcional.md#fun-007): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos; permisos globales; restablecimiento; deshabilitación. Campos/controles: Nombre visible; Nombre de usuario; Contraseña inicial. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: users.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-007-crear-usuario.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-008"></a>

### CAP-008 — Editar nombre y deshabilitar cuenta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-004](02-pantallas-y-permisos.md#pan-004) |
| Funcionalidad / objetivo | [FUN-008](01-inventario-funcional.md#fun-008): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos; permisos globales; restablecimiento; deshabilitación. Campos/controles: Nombre visible; Deshabilitar cuenta y cerrar sesiones. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: users.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-008-editar-nombre-y-deshabilitar-cuenta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-009"></a>

### CAP-009 — Asignar perfiles globales

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-004](02-pantallas-y-permisos.md#pan-004) |
| Funcionalidad / objetivo | [FUN-009](01-inventario-funcional.md#fun-009): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos; permisos globales; restablecimiento; deshabilitación. Campos/controles: Administrador de instalación; Auditor de accesos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: permissions.manage_global. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-009-asignar-perfiles-globales.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-010"></a>

### CAP-010 — Restablecer contraseña de otro usuario

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-004](02-pantallas-y-permisos.md#pan-004) |
| Funcionalidad / objetivo | [FUN-010](01-inventario-funcional.md#fun-010): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Datos; permisos globales; restablecimiento; deshabilitación. Campos/controles: Nueva contraseña. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: users.reset_password. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-010-restablecer-contrasena-de-otro-usuario.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-011"></a>

### CAP-011 — Revocar sesiones de una cuenta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Identidad; [PAN-003](02-pantallas-y-permisos.md#pan-003) |
| Funcionalidad / objetivo | [FUN-011](01-inventario-funcional.md#fun-011): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tabla; creación y edición en línea. Campos/controles: Confirmación del navegador. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: sessions.revoke. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-011-revocar-sesiones-de-una-cuenta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-012"></a>

### CAP-012 — Consultar sesiones e intentos de acceso

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Seguridad; [PAN-005](02-pantallas-y-permisos.md#pan-005) |
| Funcionalidad / objetivo | [FUN-012](01-inventario-funcional.md#fun-012): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Pestañas Sesiones / Intentos de acceso; tabla; paginación. Campos/controles: Desde; Hasta (sin incluir); resultado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: UI: sessions.read_all y authentication_attempts.read; backend permisos separados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-012-consultar-sesiones-e-intentos-de-acceso.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-013"></a>

### CAP-013 — Listar y abrir bibliotecas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-008](02-pantallas-y-permisos.md#pan-008) |
| Funcionalidad / objetivo | [FUN-013](01-inventario-funcional.md#fun-013): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tarjetas; creación; selección de biblioteca. Campos/controles: Sin formulario; selección de tarjeta. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Asignación a biblioteca o permissions.manage_global para administración. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-013-listar-y-abrir-bibliotecas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-014"></a>

### CAP-014 — Crear biblioteca vinculada

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-009](02-pantallas-y-permisos.md#pan-009) |
| Funcionalidad / objetivo | [FUN-014](01-inventario-funcional.md#fun-014): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; modalidad; asignación inicial. Campos/controles: Nombre; modalidad Vinculada; gestor inicial. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.create + permissions.manage_global en backend; módulo linked. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-014-crear-biblioteca-vinculada.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-015"></a>

### CAP-015 — Crear biblioteca administrada o híbrida

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-009](02-pantallas-y-permisos.md#pan-009) |
| Funcionalidad / objetivo | [FUN-015](01-inventario-funcional.md#fun-015): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; modalidad; asignación inicial. Campos/controles: Nombre; Administrada/Híbrida; gestor. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.create + permissions.manage_global; módulos correspondientes. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-015-crear-biblioteca-administrada-o-hibrida.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-016"></a>

### CAP-016 — Asignar perfiles por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Permisos; [PAN-023](02-pantallas-y-permisos.md#pan-023) |
| Funcionalidad / objetivo | [FUN-016](01-inventario-funcional.md#fun-016): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Búsqueda de candidatos; asignaciones; editor de perfiles. Campos/controles: Buscar persona; Lector; Colaborador; Revisor; Gestor; Auditor. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: permissions.manage_library o permissions.manage_global. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-016-asignar-perfiles-por-biblioteca.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-017"></a>

### CAP-017 — Cambiar nombre y ampliar modalidad

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-024](02-pantallas-y-permisos.md#pan-024) |
| Funcionalidad / objetivo | [FUN-017](01-inventario-funcional.md#fun-017): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | General; organización; destino; estructura; revisión; retención; reglas. Campos/controles: Nombre; modalidad. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-017-cambiar-nombre-y-ampliar-modalidad.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-018"></a>

### CAP-018 — Habilitar expedientes y ejercicio

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-024](02-pantallas-y-permisos.md#pan-024) |
| Funcionalidad / objetivo | [FUN-018](01-inventario-funcional.md#fun-018): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | General; organización; destino; estructura; revisión; retención; reglas. Campos/controles: Utilizar expedientes; etiqueta del identificador; ejercicio. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure; módulo expedientes; biblioteca no vinculada. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-018-habilitar-expedientes-y-ejercicio.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-019"></a>

### CAP-019 — Seleccionar destino y reglas de almacenamiento

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-024](02-pantallas-y-permisos.md#pan-024) |
| Funcionalidad / objetivo | [FUN-019](01-inventario-funcional.md#fun-019): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | General; organización; destino; estructura; revisión; retención; reglas. Campos/controles: Destino; estructura; formato de nombre; prefijo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure; destino administrado activo. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-019-seleccionar-destino-y-reglas-de-almacenamiento.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-020"></a>

### CAP-020 — Consultar ejemplo de nombre y ruta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-025](02-pantallas-y-permisos.md#pan-025) |
| Funcionalidad / objetivo | [FUN-020](01-inventario-funcional.md#fun-020): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Ejemplo de ruta/nombre conforme a valores guardados. Campos/controles: Identificador; Ejercicio; Categoría; Tipo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-020-consultar-ejemplo-de-nombre-y-ruta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-021"></a>

### CAP-021 — Configurar revisión y retención de cargas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-024](02-pantallas-y-permisos.md#pan-024) |
| Funcionalidad / objetivo | [FUN-021](01-inventario-funcional.md#fun-021): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | General; organización; destino; estructura; revisión; retención; reglas. Campos/controles: Revisión administrada; revisión inicial vinculada en híbridas; conservación días. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure; módulo revisión si se habilita. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-021-configurar-revision-y-retencion-de-cargas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-022"></a>

### CAP-022 — Inspeccionar una ruta de carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-020](02-pantallas-y-permisos.md#pan-020) |
| Funcionalidad / objetivo | [FUN-022](01-inventario-funcional.md#fun-022): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Raíces; estados; acordeones; alta; plan; retiro. Campos/controles: Origen de la carpeta; Ruta de la carpeta en el servidor. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-022-inspeccionar-una-ruta-de-carpeta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-023"></a>

### CAP-023 — Registrar raíz vinculada

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-021](02-pantallas-y-permisos.md#pan-021) |
| Funcionalidad / objetivo | [FUN-023](01-inventario-funcional.md#fun-023): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Igual / descendiente / antecesora / independiente; listado de raíces involucradas. Campos/controles: Plan y revisión; ruta. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots; módulo linked. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-023-registrar-raiz-vinculada.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-024"></a>

### CAP-024 — Registrar destino administrado

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-021](02-pantallas-y-permisos.md#pan-021) |
| Funcionalidad / objetivo | [FUN-024](01-inventario-funcional.md#fun-024): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Igual / descendiente / antecesora / independiente; listado de raíces involucradas. Campos/controles: Origen administrado; plan. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots; módulo managed. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-024-registrar-destino-administrado.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-025"></a>

### CAP-025 — Crear vista lógica dentro de una raíz

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-021](02-pantallas-y-permisos.md#pan-021) |
| Funcionalidad / objetivo | [FUN-025](01-inventario-funcional.md#fun-025): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Igual / descendiente / antecesora / independiente; listado de raíces involucradas. Campos/controles: Nombre de vista. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-025-crear-vista-logica-dentro-de-una-raiz.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-026"></a>

### CAP-026 — Consolidar raíces hijas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-021](02-pantallas-y-permisos.md#pan-021) |
| Funcionalidad / objetivo | [FUN-026](01-inventario-funcional.md#fun-026): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Igual / descendiente / antecesora / independiente; listado de raíces involucradas. Campos/controles: Plan antecesor; confirmación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-026-consolidar-raices-hijas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-027"></a>

### CAP-027 — Programar verificación periódica de carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-022](02-pantallas-y-permisos.md#pan-022) |
| Funcionalidad / objetivo | [FUN-027](01-inventario-funcional.md#fun-027): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Verificación periódica; plan de retiro con impacto. Campos/controles: Verificación habilitada; Comprobar cada; unidad. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-027-programar-verificacion-periodica-de-carpeta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-028"></a>

### CAP-028 — Dejar de supervisar una carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Almacenamiento; [PAN-022](02-pantallas-y-permisos.md#pan-022) |
| Funcionalidad / objetivo | [FUN-028](01-inventario-funcional.md#fun-028): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Verificación periódica; plan de retiro con impacto. Campos/controles: Política conservar índice o retirar referencias; confirmación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: storage.manage_roots. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-028-dejar-de-supervisar-una-carpeta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-029"></a>

### CAP-029 — Navegar carpetas y vistas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-011](02-pantallas-y-permisos.md#pan-011) |
| Funcionalidad / objetivo | [FUN-029](01-inventario-funcional.md#fun-029): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Filtros; búsqueda local; migas; listas/cuadrícula; carpetas; archivos; resumen duplicados. Campos/controles: Carpeta raíz; Vista lógica; migas. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-029-navegar-carpetas-y-vistas.png` |
| Estado | requiere reemplazo |
| Observaciones | Explorador con archivos ficticios; nombres de biblioteca de pruebas. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/h3-explorer.png](../../web/test-results/h3-explorer.png); copia byte a byte en capturas/CAP-029-navegar-carpetas-y-vistas.png. |

<a id="cap-030"></a>

### CAP-030 — Cambiar presentación y filtros del explorador

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-011](02-pantallas-y-permisos.md#pan-011) |
| Funcionalidad / objetivo | [FUN-030](01-inventario-funcional.md#fun-030): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Filtros; búsqueda local; migas; listas/cuadrícula; carpetas; archivos; resumen duplicados. Campos/controles: Formato; Disponibilidad; Vista de lista / Vista de cuadrícula. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-030-cambiar-presentacion-y-filtros-del-explorador.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-031"></a>

### CAP-031 — Buscar desde una carpeta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-011](02-pantallas-y-permisos.md#pan-011) |
| Funcionalidad / objetivo | [FUN-031](01-inventario-funcional.md#fun-031): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Filtros; búsqueda local; migas; listas/cuadrícula; carpetas; archivos; resumen duplicados. Campos/controles: Tipo de búsqueda; texto; alcance. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: search.execute en biblioteca. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-031-buscar-desde-una-carpeta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-032"></a>

### CAP-032 — Consultar ficha y original

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-012](02-pantallas-y-permisos.md#pan-012) |
| Funcionalidad / objetivo | [FUN-032](01-inventario-funcional.md#fun-032): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Modal; original; texto; procesamiento; clasificación; workflow; huella; retiro. Campos/controles: Sin edición inicial. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read y visibilidad. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-032-consultar-ficha-y-original.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-033"></a>

### CAP-033 — Descargar archivo original

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-012](02-pantallas-y-permisos.md#pan-012) |
| Funcionalidad / objetivo | [FUN-033](01-inventario-funcional.md#fun-033): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Modal; original; texto; procesamiento; clasificación; workflow; huella; retiro. Campos/controles: Sin formulario; botón Descargar FORMATO. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.download; documento accesible. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-033-descargar-archivo-original.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-034"></a>

### CAP-034 — Consultar texto por unidad y resumen de extracción

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Extracción; [PAN-013](02-pantallas-y-permisos.md#pan-013) |
| Funcionalidad / objetivo | [FUN-034](01-inventario-funcional.md#fun-034): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Texto nativo/OCR; contexto; unidad; resumen y advertencias. Campos/controles: Página o Unidad de texto. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-034-consultar-texto-por-unidad-y-resumen-de-extraccion.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-035"></a>

### CAP-035 — Solicitar y consultar preview de Word o Excel

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Previews; [PAN-014](02-pantallas-y-permisos.md#pan-014) |
| Funcionalidad / objetivo | [FUN-035](01-inventario-funcional.md#fun-035): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estados; conversión a PDF; mensaje si no hay convertidor. Campos/controles: Sin formulario; botones de estado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read; previews habilitadas; LibreOffice disponible. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png` |
| Estado | requiere reemplazo |
| Observaciones | Preview Word derivada real; framing de modal/documento requiere toma editorial. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/v2-phase4-docx.png](../../web/test-results/v2-phase4-docx.png); copia byte a byte en capturas/CAP-035-solicitar-y-consultar-preview-de-word-o-excel.png. |

<a id="cap-036"></a>

### CAP-036 — Reindexar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-015](02-pantallas-y-permisos.md#pan-015) |
| Funcionalidad / objetivo | [FUN-036](01-inventario-funcional.md#fun-036): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Botón; solicitud; progreso; aviso. Campos/controles: Idempotency-Key generado por UI. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.run; reglas de formato y licencia. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-036-reindexar-un-documento.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-037"></a>

### CAP-037 — Retirar documento del índice

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Documentos; [PAN-019](02-pantallas-y-permisos.md#pan-019) |
| Funcionalidad / objetivo | [FUN-037](01-inventario-funcional.md#fun-037): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeón; advertencia; confirmación. Campos/controles: Motivo; confirmaciones. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.remove_index. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-037-retirar-documento-del-indice.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-038"></a>

### CAP-038 — Crear, editar y archivar categorías

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-028](02-pantallas-y-permisos.md#pan-028) |
| Funcionalidad / objetivo | [FUN-038](01-inventario-funcional.md#fun-038): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Categorías; tipos; plantillas. Campos/controles: Nombre; archivado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read para consulta; catalogs.manage para cambios. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-038-crear-editar-y-archivar-categorias.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-039"></a>

### CAP-039 — Configurar tipos documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-028](02-pantallas-y-permisos.md#pan-028) |
| Funcionalidad / objetivo | [FUN-039](01-inventario-funcional.md#fun-039): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Categorías; tipos; plantillas. Campos/controles: Nombre; categoría; múltiples; título descriptivo; archivado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: catalogs.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-039-configurar-tipos-documentales.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-040"></a>

### CAP-040 — Crear plantilla y nuevas versiones

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-029](02-pantallas-y-permisos.md#pan-029) |
| Funcionalidad / objetivo | [FUN-040](01-inventario-funcional.md#fun-040): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeones de versiones; nuevo formulario. Campos/controles: Nombre; Incluir; cantidad; Obligatorio. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: templates.manage; expedientes habilitados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-040-crear-plantilla-y-nuevas-versiones.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-041"></a>

### CAP-041 — Listar y filtrar expedientes

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-030](02-pantallas-y-permisos.md#pan-030) |
| Funcionalidad / objetivo | [FUN-041](01-inventario-funcional.md#fun-041): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Filtro; lista; alta. Campos/controles: Consulta de identificador. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read; casos habilitados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-041-listar-y-filtrar-expedientes.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-042"></a>

### CAP-042 — Crear expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-030](02-pantallas-y-permisos.md#pan-030) |
| Funcionalidad / objetivo | [FUN-042](01-inventario-funcional.md#fun-042): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Filtro; lista; alta. Campos/controles: Identificador; Ejercicio; Plantilla inicial. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: cases.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-042-crear-expediente.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-043"></a>

### CAP-043 — Editar identificador y ejercicio

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-031](02-pantallas-y-permisos.md#pan-031) |
| Funcionalidad / objetivo | [FUN-043](01-inventario-funcional.md#fun-043): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Progreso; estados; requisitos; archivos; asociación vinculada. Campos/controles: Identificador; Ejercicio. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: cases.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-043-editar-identificador-y-ejercicio.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-044"></a>

### CAP-044 — Inicializar requisitos desde plantilla

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-031](02-pantallas-y-permisos.md#pan-031) |
| Funcionalidad / objetivo | [FUN-044](01-inventario-funcional.md#fun-044): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Progreso; estados; requisitos; archivos; asociación vinculada. Campos/controles: Versión de plantilla. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: requirements.edit; expediente sin requisitos inicializados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-044-inicializar-requisitos-desde-plantilla.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-045"></a>

### CAP-045 — Ajustar cantidades y obligatoriedad del expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-031](02-pantallas-y-permisos.md#pan-031) |
| Funcionalidad / objetivo | [FUN-045](01-inventario-funcional.md#fun-045): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Progreso; estados; requisitos; archivos; asociación vinculada. Campos/controles: Cantidad requerida; Obligatorio. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: requirements.edit. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-045-ajustar-cantidades-y-obligatoriedad-del-expediente.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-046"></a>

### CAP-046 — Interpretar avance y documentos de un expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-031](02-pantallas-y-permisos.md#pan-031) |
| Funcionalidad / objetivo | [FUN-046](01-inventario-funcional.md#fun-046): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Progreso; estados; requisitos; archivos; asociación vinculada. Campos/controles: Sin escritura; selección de documento. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png` |
| Estado | requiere reemplazo |
| Observaciones | Expediente sintético y estados reales; sustituir identificador de pruebas. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/h4-case.png](../../web/test-results/h4-case.png); copia byte a byte en capturas/CAP-046-interpretar-avance-y-documentos-de-un-expediente.png. |

<a id="cap-047"></a>

### CAP-047 — Preparar carga múltiple

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Cargas; [PAN-032](02-pantallas-y-permisos.md#pan-032) |
| Funcionalidad / objetivo | [FUN-047](01-inventario-funcional.md#fun-047): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Arrastrar/selector; lote; lista de envío; progreso individual. Campos/controles: Selector de archivos o arrastrar; lista de lote. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.upload; biblioteca administrada/híbrida. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-047-preparar-carga-multiple.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-048"></a>

### CAP-048 — Seguir carga y continuar tras error individual

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Cargas; [PAN-032](02-pantallas-y-permisos.md#pan-032) |
| Funcionalidad / objetivo | [FUN-048](01-inventario-funcional.md#fun-048): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Arrastrar/selector; lote; lista de envío; progreso individual. Campos/controles: Lista individual; porcentajes; reintento de lote. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.upload. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-048-seguir-carga-y-continuar-tras-error-individual.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-049"></a>

### CAP-049 — Iniciar otra carga conservando borradores previos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Cargas; [PAN-032](02-pantallas-y-permisos.md#pan-032) |
| Funcionalidad / objetivo | [FUN-049](01-inventario-funcional.md#fun-049): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Arrastrar/selector; lote; lista de envío; progreso individual. Campos/controles: Nuevo lote. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.upload. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-049-iniciar-otra-carga-conservando-borradores-previos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-050"></a>

### CAP-050 — Consultar historial de cargas y detalle de lote

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Cargas; [PAN-033](02-pantallas-y-permisos.md#pan-033) |
| Funcionalidad / objetivo | [FUN-050](01-inventario-funcional.md#fun-050): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tabla lotes; detalle de archivos; estados. Campos/controles: Selección de lote; Mostrar más. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Autor; revisores habilitados según alcance. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png` |
| Estado | requiere reemplazo |
| Observaciones | Historial de lote sintético multiformato; etiquetas de pruebas. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/v2-phase6-history-desktop.png](../../web/test-results/v2-phase6-history-desktop.png); copia byte a byte en capturas/CAP-050-consultar-historial-de-cargas-y-detalle-de-lote.png. |

<a id="cap-051"></a>

### CAP-051 — Clasificar borrador

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-016](02-pantallas-y-permisos.md#pan-016) |
| Funcionalidad / objetivo | [FUN-051](01-inventario-funcional.md#fun-051): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeón Clasificar o asociar a expediente; selector de expediente; categoría; tipo; Campos adicionales. Campos/controles: Expediente; Categoría; Tipo; Título; metadatos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.classify; visibilidad del autor/revisor. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-051-clasificar-borrador.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-052"></a>

### CAP-052 — Asociar vinculado a expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-016](02-pantallas-y-permisos.md#pan-016) |
| Funcionalidad / objetivo | [FUN-052](01-inventario-funcional.md#fun-052): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeón Clasificar o asociar a expediente; selector de expediente; categoría; tipo; Campos adicionales. Campos/controles: Expediente; Categoría; Tipo; Título. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.associate; biblioteca híbrida; documentos visibles. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-052-asociar-vinculado-a-expediente.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-053"></a>

### CAP-053 — Reasignar documento a otro expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Organización; [PAN-016](02-pantallas-y-permisos.md#pan-016) |
| Funcionalidad / objetivo | [FUN-053](01-inventario-funcional.md#fun-053): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeón Clasificar o asociar a expediente; selector de expediente; categoría; tipo; Campos adicionales. Campos/controles: Nuevo expediente; Motivo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.reassign. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-053-reasignar-documento-a-otro-expediente.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-054"></a>

### CAP-054 — Cancelar carga propia

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Cargas; [PAN-034](02-pantallas-y-permisos.md#pan-034) |
| Funcionalidad / objetivo | [FUN-054](01-inventario-funcional.md#fun-054): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Motivo; aviso de conservación y retención. Campos/controles: Motivo de cancelación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.cancel_own; autor. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-054-cancelar-carga-propia.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-055"></a>

### CAP-055 — Enviar a revisión y actualizar envío

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-017](02-pantallas-y-permisos.md#pan-017) |
| Funcionalidad / objetivo | [FUN-055](01-inventario-funcional.md#fun-055): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Selector de revisor; botones; razón; progreso de incorporación. Campos/controles: Revisor o Cola compartida de revisores. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.submit; revisión habilitada. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-055-enviar-a-revision-y-actualizar-envio.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-056"></a>

### CAP-056 — Consultar revisiones pendientes

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-035](02-pantallas-y-permisos.md#pan-035) |
| Funcionalidad / objetivo | [FUN-056](01-inventario-funcional.md#fun-056): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Pendientes; autor; fecha; asignación. Campos/controles: Sin campos específicos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.review y documents.read efectivo. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-056-consultar-revisiones-pendientes.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-057"></a>

### CAP-057 — Aprobar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-017](02-pantallas-y-permisos.md#pan-017) |
| Funcionalidad / objetivo | [FUN-057](01-inventario-funcional.md#fun-057): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Selector de revisor; botones; razón; progreso de incorporación. Campos/controles: Confirmación de acción. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.approve; revisor autorizado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-057-aprobar-un-documento.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-058"></a>

### CAP-058 — Rechazar un documento

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-017](02-pantallas-y-permisos.md#pan-017) |
| Funcionalidad / objetivo | [FUN-058](01-inventario-funcional.md#fun-058): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Selector de revisor; botones; razón; progreso de incorporación. Campos/controles: Motivo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.reject; revisor autorizado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-058-rechazar-un-documento.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-059"></a>

### CAP-059 — Confirmar sin revisión

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-017](02-pantallas-y-permisos.md#pan-017) |
| Funcionalidad / objetivo | [FUN-059](01-inventario-funcional.md#fun-059): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Selector de revisor; botones; razón; progreso de incorporación. Campos/controles: Clasificación ya guardada. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.finalize; revisión no requerida para origen. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-059-confirmar-sin-revision.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-060"></a>

### CAP-060 — Seguir y reintentar incorporación administrada

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Revisión; [PAN-017](02-pantallas-y-permisos.md#pan-017) |
| Funcionalidad / objetivo | [FUN-060](01-inventario-funcional.md#fun-060): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Selector de revisor; botones; razón; progreso de incorporación. Campos/controles: Motivo del reintento. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Autorización aprobador o indexing.retry según estado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-060-seguir-y-reintentar-incorporacion-administrada.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-061"></a>

### CAP-061 — Realizar búsqueda general

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Búsqueda; [PAN-038](02-pantallas-y-permisos.md#pan-038) |
| Funcionalidad / objetivo | [FUN-061](01-inventario-funcional.md#fun-061): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; filtros; resultados agrupados; avisos; auditoría de consultas si autorizada. Campos/controles: Texto; tipo; biblioteca. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: search.execute en cada biblioteca. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-061-realizar-busqueda-general.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-062"></a>

### CAP-062 — Aplicar filtros documentales de búsqueda

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Búsqueda; [PAN-038](02-pantallas-y-permisos.md#pan-038) |
| Funcionalidad / objetivo | [FUN-062](01-inventario-funcional.md#fun-062): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; filtros; resultados agrupados; avisos; auditoría de consultas si autorizada. Campos/controles: Formato; Disponibilidad; Origen; Estado; sin expediente. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: search.execute. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-062-aplicar-filtros-documentales-de-busqueda.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-063"></a>

### CAP-063 — Buscar por catálogo, ejercicio y expediente

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Búsqueda; [PAN-038](02-pantallas-y-permisos.md#pan-038) |
| Funcionalidad / objetivo | [FUN-063](01-inventario-funcional.md#fun-063): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; filtros; resultados agrupados; avisos; auditoría de consultas si autorizada. Campos/controles: Categoría; Tipo; Ejercicio; Localizar expediente; expediente. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: search.execute; biblioteca no vinculada con organización. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-063-buscar-por-catalogo-ejercicio-y-expediente.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-064"></a>

### CAP-064 — Navegar resultados agrupados y coincidencias

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Búsqueda; [PAN-038](02-pantallas-y-permisos.md#pan-038) |
| Funcionalidad / objetivo | [FUN-064](01-inventario-funcional.md#fun-064): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Formulario; filtros; resultados agrupados; avisos; auditoría de consultas si autorizada. Campos/controles: Grupo; tamaño de página 10/25/50; cursor. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: search.execute / documents.read. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-064-navegar-resultados-agrupados-y-coincidencias.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-065"></a>

### CAP-065 — Consultar y reconocer cambios documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Búsqueda; [PAN-039](02-pantallas-y-permisos.md#pan-039) |
| Funcionalidad / objetivo | [FUN-065](01-inventario-funcional.md#fun-065): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Hasta cien avisos; acción de lectura. Campos/controles: Aviso; acuse. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read; licencia permite lectura. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-065-consultar-y-reconocer-cambios-documentales.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-066"></a>

### CAP-066 — Consultar grupos de duplicados

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Duplicados; [PAN-037](02-pantallas-y-permisos.md#pan-037) |
| Funcionalidad / objetivo | [FUN-066](01-inventario-funcional.md#fun-066): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Grupos; recuentos; detalle; huella; documentos. Campos/controles: Selección de grupo; paginación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read; visibilidad por biblioteca. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-066-consultar-grupos-de-duplicados.png` |
| Estado | requiere reemplazo |
| Observaciones | Grupos y documentos sintéticos; rutas de entorno temporal que no se deben enseñar como instalación. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/r6-duplicates.png](../../web/test-results/r6-duplicates.png); copia byte a byte en capturas/CAP-066-consultar-grupos-de-duplicados.png. |

<a id="cap-067"></a>

### CAP-067 — Consultar huella e historial documental

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Auditoría; [PAN-018](02-pantallas-y-permisos.md#pan-018) |
| Funcionalidad / objetivo | [FUN-067](01-inventario-funcional.md#fun-067): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | SHA-256; historia paginada; modal de evento. Campos/controles: SHA-256; paginación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: documents.read; detalles según permisos. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-067-consultar-huella-e-historial-documental.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-068"></a>

### CAP-068 — Filtrar y ordenar auditoría global

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Auditoría; [PAN-006](02-pantallas-y-permisos.md#pan-006) |
| Funcionalidad / objetivo | [FUN-068](01-inventario-funcional.md#fun-068): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Visor de eventos; filtros; tabla ordenable. Campos/controles: Desde; Hasta; Tipo de evento; Usuario; Buscar; orden. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: audit.read_global. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-068-filtrar-y-ordenar-auditoria-global.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-069"></a>

### CAP-069 — Leer detalle de evento y registro original

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Auditoría; [PAN-007](02-pantallas-y-permisos.md#pan-007) |
| Funcionalidad / objetivo | [FUN-069](01-inventario-funcional.md#fun-069): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Modal; datos descriptivos; Información avanzada · registro original. Campos/controles: Sin edición. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Permiso del visor; restricciones de detalle. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-069-leer-detalle-de-evento-y-registro-original.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-070"></a>

### CAP-070 — Consultar auditoría de biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Auditoría; [PAN-040](02-pantallas-y-permisos.md#pan-040) |
| Funcionalidad / objetivo | [FUN-070](01-inventario-funcional.md#fun-070): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | AuditTable; filtros y diálogo. Campos/controles: Mismos filtros de AuditTable. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: audit.read_library. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-070-consultar-auditoria-de-biblioteca.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-071"></a>

### CAP-071 — Consultar registro exacto de búsquedas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Auditoría; [PAN-010](02-pantallas-y-permisos.md#pan-010) |
| Funcionalidad / objetivo | [FUN-071](01-inventario-funcional.md#fun-071): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Documentos; Duplicados; Cargas; Revisiones; Expedientes; Catálogos; Carpetas; Procesamiento; Personas; Configuración; Auditoría, condicionales; Consultas exactas bajo Documentos para audit.search_details. Campos/controles: Bibliotecas; consulta histórica. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: audit.search_details en ámbitos solicitados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-071-consultar-registro-exacto-de-busquedas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-072"></a>

### CAP-072 — Solicitar verificación manual de biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Bibliotecas; [PAN-010](02-pantallas-y-permisos.md#pan-010) |
| Funcionalidad / objetivo | [FUN-072](01-inventario-funcional.md#fun-072): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Documentos; Duplicados; Cargas; Revisiones; Expedientes; Catálogos; Carpetas; Procesamiento; Personas; Configuración; Auditoría, condicionales; Consultas exactas bajo Documentos para audit.search_details. Campos/controles: Sin campos; acción por biblioteca. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.run. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-072-solicitar-verificacion-manual-de-biblioteca.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-073"></a>

### CAP-073 — Consultar resumen, progreso y tareas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-073](01-inventario-funcional.md#fun-073): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Paginación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.run. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-073-consultar-resumen-progreso-y-tareas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-074"></a>

### CAP-074 — Cancelar recorrido y reanudar raíz

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-074](01-inventario-funcional.md#fun-074): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Raíz/trabajo; motivo interno de solicitud. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.run según cancelación; verificación manual para reanudación. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-074-cancelar-recorrido-y-reanudar-raiz.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-075"></a>

### CAP-075 — Cancelar extracción pendiente o activa

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-075](01-inventario-funcional.md#fun-075): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Trabajo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.run; tipos cancelables. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-075-cancelar-extraccion-pendiente-o-activa.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-076"></a>

### CAP-076 — Reintentar tarea fallida

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-076](01-inventario-funcional.md#fun-076): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Trabajo; motivo registrado por UI. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.retry. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-076-reintentar-tarea-fallida.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-077"></a>

### CAP-077 — Reintentar rutas con error y consultar omitidos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-077](01-inventario-funcional.md#fun-077): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Raíz y lista reciente. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: indexing.retry para rutas; indexing.run para vista. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-077-reintentar-rutas-con-error-y-consultar-omitidos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-078"></a>

### CAP-078 — Comprender descubrimiento, hash y cambios vinculados

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-078](01-inventario-funcional.md#fun-078): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: No formulario; políticas de raíz/archivo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Automático; módulo linked; raíz habilitada. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-078-comprender-descubrimiento-hash-y-cambios-vinculados.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-079"></a>

### CAP-079 — Comprender vigilancia y pérdida de eventos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-079](01-inventario-funcional.md#fun-079): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Intervalo raíz; opciones watcher. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Automático; watcher configurado; sistema permite vigilancia. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-079-comprender-vigilancia-y-perdida-de-eventos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-080"></a>

### CAP-080 — Comprender extracción por formato

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Extracción; [PAN-013](02-pantallas-y-permisos.md#pan-013) |
| Funcionalidad / objetivo | [FUN-080](01-inventario-funcional.md#fun-080): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Texto nativo/OCR; contexto; unidad; resumen y advertencias. Campos/controles: Política almacenar/indexar. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Automático; formato indexable; tamaño permitido. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-080-comprender-extraccion-por-formato.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-081"></a>

### CAP-081 — Comprender OCR de PDF

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Extracción; [PAN-013](02-pantallas-y-permisos.md#pan-013) |
| Funcionalidad / objetivo | [FUN-081](01-inventario-funcional.md#fun-081): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Texto nativo/OCR; contexto; unidad; resumen y advertencias. Campos/controles: Idioma; umbral; máximo páginas; timeout. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: OCR permitido por política y licencia. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-081-comprender-ocr-de-pdf.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-082"></a>

### CAP-082 — Comprender recuperación de cola tras reinicio

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-082](01-inventario-funcional.md#fun-082): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Sin formulario. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Automático; bloqueo exclusivo de estado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-082-comprender-recuperacion-de-cola-tras-reinicio.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-083"></a>

### CAP-083 — Verificar integridad administrada

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Procesamiento; [PAN-036](02-pantallas-y-permisos.md#pan-036) |
| Funcionalidad / objetivo | [FUN-083](01-inventario-funcional.md#fun-083): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Resumen; rendimiento; reconciliación; activos; tabla trabajos; errores/omitidos. Campos/controles: Raíz; sin modificaciones del original. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Automático; acceso físico al destino; permitido incluso solo lectura. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-083-verificar-integridad-administrada.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-084"></a>

### CAP-084 — Configurar reglas globales de archivos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-042](02-pantallas-y-permisos.md#pan-042) |
| Funcionalidad / objetivo | [FUN-084](01-inventario-funcional.md#fun-084): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Navegación de cinco secciones; reglas globales. Campos/controles: Almacenar; indexar; máximo archivo/indexación; desconocidos; no indexables. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-084-configurar-reglas-globales-de-archivos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-085"></a>

### CAP-085 — Configurar herencia de reglas de archivo

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-026](02-pantallas-y-permisos.md#pan-026) |
| Funcionalidad / objetivo | [FUN-085](01-inventario-funcional.md#fun-085): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Campos heredados/globales y efectivos; tabla formatos. Campos/controles: Heredar por campo; formatos; límites; políticas. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-085-configurar-herencia-de-reglas-de-archivo.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-086"></a>

### CAP-086 — Interpretar rechazo de formato o contenido

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-042](02-pantallas-y-permisos.md#pan-042) |
| Funcionalidad / objetivo | [FUN-086](01-inventario-funcional.md#fun-086): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Navegación de cinco secciones; reglas globales. Campos/controles: Nombre/bytes/tamaño del archivo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Reglas efectivas; sin permiso especial. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-086-interpretar-rechazo-de-formato-o-contenido.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-087"></a>

### CAP-087 — Configurar automatización y prioridad manual

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-044](02-pantallas-y-permisos.md#pan-044) |
| Funcionalidad / objetivo | [FUN-087](01-inventario-funcional.md#fun-087): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeones de automatización, OCR y vigilancia. Campos/controles: Procesamiento automático; Reprocesar documentos modificados; Priorizar trabajos manuales. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-087-configurar-automatizacion-y-prioridad-manual.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-088"></a>

### CAP-088 — Configurar parámetros OCR globales

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-044](02-pantallas-y-permisos.md#pan-044) |
| Funcionalidad / objetivo | [FUN-088](01-inventario-funcional.md#fun-088): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeones de automatización, OCR y vigilancia. Campos/controles: Habilitar OCR; OCR automático; Idioma; Umbral; Máximo páginas; Tiempo por operación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-088-configurar-parametros-ocr-globales.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-089"></a>

### CAP-089 — Configurar vigilancia y archivos ignorados

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-044](02-pantallas-y-permisos.md#pan-044) |
| Funcionalidad / objetivo | [FUN-089](01-inventario-funcional.md#fun-089): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Acordeones de automatización, OCR y vigilancia. Campos/controles: Vigilancia de cambios; Office temporales; ocultos; temporales; extensiones. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-089-configurar-vigilancia-y-archivos-ignorados.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-090"></a>

### CAP-090 — Sobrescribir reglas avanzadas por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-027](02-pantallas-y-permisos.md#pan-027) |
| Funcionalidad / objetivo | [FUN-090](01-inventario-funcional.md#fun-090): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Procesamiento automático; OCR; vigilancia; previews con herencia. Campos/controles: Heredar por parámetro; automatización/OCR/watcher/previews. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: libraries.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-090-sobrescribir-reglas-avanzadas-por-biblioteca.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-091"></a>

### CAP-091 — Consultar recursos y configurar concurrencia

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-043](02-pantallas-y-permisos.md#pan-043) |
| Funcionalidad / objetivo | [FUN-091](01-inventario-funcional.md#fun-091): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | CPU/memoria; concurrencia; política de pausa y reintentos. Campos/controles: Modo Automático/Manual; extracción directa; OCR; límite general. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-091-consultar-recursos-y-configurar-concurrencia.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-092"></a>

### CAP-092 — Pausar y reanudar procesamiento general

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-043](02-pantallas-y-permisos.md#pan-043) |
| Funcionalidad / objetivo | [FUN-092](01-inventario-funcional.md#fun-092): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | CPU/memoria; concurrencia; política de pausa y reintentos. Campos/controles: Pausar procesamiento / Reanudar procesamiento. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-092-pausar-y-reanudar-procesamiento-general.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-093"></a>

### CAP-093 — Configurar intentos y tiempos documentales

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-043](02-pantallas-y-permisos.md#pan-043) |
| Funcionalidad / objetivo | [FUN-093](01-inventario-funcional.md#fun-093): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | CPU/memoria; concurrencia; política de pausa y reintentos. Campos/controles: Máximo de intentos; Espera inicial entre intentos; Tiempo máximo por documento. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-093-configurar-intentos-y-tiempos-documentales.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-094"></a>

### CAP-094 — Configurar previews, convertidor y caché

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-045](02-pantallas-y-permisos.md#pan-045) |
| Funcionalidad / objetivo | [FUN-094](01-inventario-funcional.md#fun-094): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Política; estado LibreOffice; consumo; limpieza; confirmación. Campos/controles: Habilitación; límite caché; días; fuente; limpieza; LibreOffice. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-094-configurar-previews-convertidor-y-cache.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-095"></a>

### CAP-095 — Vaciar caché de vistas previas

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Configuración; [PAN-045](02-pantallas-y-permisos.md#pan-045) |
| Funcionalidad / objetivo | [FUN-095](01-inventario-funcional.md#fun-095): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Política; estado LibreOffice; consumo; limpieza; confirmación. Campos/controles: Confirmación de vaciado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-095-vaciar-cache-de-vistas-previas.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-096"></a>

### CAP-096 — Habilitar acceso LAN o regresar a local

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Red; [PAN-046](02-pantallas-y-permisos.md#pan-046) |
| Funcionalidad / objetivo | [FUN-096](01-inventario-funcional.md#fun-096): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Modo actual/configurado; cambio; puerto; URLs; diagnóstico firewall; Interfaces IPv4 disponibles. Campos/controles: Solo este equipo / Permitir acceso desde la red local. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-096-habilitar-acceso-lan-o-regresar-a-local.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-097"></a>

### CAP-097 — Consultar diagnóstico de acceso y firewall

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Red; [PAN-046](02-pantallas-y-permisos.md#pan-046) |
| Funcionalidad / objetivo | [FUN-097](01-inventario-funcional.md#fun-097): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Modo actual/configurado; cambio; puerto; URLs; diagnóstico firewall; Interfaces IPv4 disponibles. Campos/controles: Ejecutar diagnóstico; Interfaces IPv4. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png` |
| Estado | requiere reemplazo |
| Observaciones | Ajustes/red actuales con puerto del servidor de pruebas; sustituir por plataforma/puerto previstos. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/phase8-settings-desktop.png](../../web/test-results/phase8-settings-desktop.png); copia byte a byte en capturas/CAP-097-consultar-diagnostico-de-acceso-y-firewall.png. |

<a id="cap-098"></a>

### CAP-098 — Consultar estadísticas del Dashboard

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Observabilidad; [PAN-041](02-pantallas-y-permisos.md#pan-041) |
| Funcionalidad / objetivo | [FUN-098](01-inventario-funcional.md#fun-098): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | KPIs; gráficos; tablas accesibles; por biblioteca; duplicados; caché; errores; definiciones. Campos/controles: Sin filtro individual; actualización. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-098-consultar-estadisticas-del-dashboard.png` |
| Estado | requiere reemplazo |
| Observaciones | Dashboard real vacío; nombre de administradora de pruebas, framing fullpage y datos de demostración a sustituir. Imagen histórica reutilizada sin alterar; no es observación de UI en vivo ni captura aprobada para manual. |
| Origen real | [web/test-results/phase8-dashboard-desktop.png](../../web/test-results/phase8-dashboard-desktop.png); copia byte a byte en capturas/CAP-098-consultar-estadisticas-del-dashboard.png. |

<a id="cap-099"></a>

### CAP-099 — Consultar gráficos, tablas accesibles y detalle por biblioteca

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Observabilidad; [PAN-041](02-pantallas-y-permisos.md#pan-041) |
| Funcionalidad / objetivo | [FUN-099](01-inventario-funcional.md#fun-099): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | KPIs; gráficos; tablas accesibles; por biblioteca; duplicados; caché; errores; definiciones. Campos/controles: Ver datos de…; Ver definiciones; Mostrar más. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-099-consultar-graficos-tablas-accesibles-y-detalle-por-biblioteca.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-100"></a>

### CAP-100 — Filtrar errores y consultar contexto

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Observabilidad; [PAN-047](02-pantallas-y-permisos.md#pan-047) |
| Funcionalidad / objetivo | [FUN-100](01-inventario-funcional.md#fun-100): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tabla; filtros; Contexto; estado; retención; copia de soporte. Campos/controles: Desde; Hasta; Módulo; Gravedad; Estado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-100-filtrar-errores-y-consultar-contexto.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-101"></a>

### CAP-101 — Marcar diagnóstico y copiar información de soporte

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Observabilidad; [PAN-047](02-pantallas-y-permisos.md#pan-047) |
| Funcionalidad / objetivo | [FUN-101](01-inventario-funcional.md#fun-101): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tabla; filtros; Contexto; estado; retención; copia de soporte. Campos/controles: Estado revisado/pendiente; copia. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-101-marcar-diagnostico-y-copiar-informacion-de-soporte.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-102"></a>

### CAP-102 — Configurar retención de diagnósticos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Observabilidad; [PAN-047](02-pantallas-y-permisos.md#pan-047) |
| Funcionalidad / objetivo | [FUN-102](01-inventario-funcional.md#fun-102): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Tabla; filtros; Contexto; estado; retención; copia de soporte. Campos/controles: Días; máximo de eventos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: system.configure. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-102-configurar-retencion-de-diagnosticos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-103"></a>

### CAP-103 — Consultar estado, módulos y datos de licencia

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-103](01-inventario-funcional.md#fun-103): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Actualizar estado; identificación soporte. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-103-consultar-estado-modulos-y-datos-de-licencia.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-104"></a>

### CAP-104 — Activar licencia en línea

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-104](01-inventario-funcional.md#fun-104): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Operación activar; Clave de licencia en texto. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage; servidor y confianza configurados. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-104-activar-licencia-en-linea.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-105"></a>

### CAP-105 — Renovar estado y retomar solicitud de licencia

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-105](01-inventario-funcional.md#fun-105): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Renovar/validar; solicitud pendiente. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage; activación existente. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-105-renovar-estado-y-retomar-solicitud-de-licencia.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-106"></a>

### CAP-106 — Desactivar licencia en línea

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-106](01-inventario-funcional.md#fun-106): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Operación desactivar; confirmación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage; activación actual. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-106-desactivar-licencia-en-linea.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-107"></a>

### CAP-107 — Generar solicitud de licencia sin conexión

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-107](01-inventario-funcional.md#fun-107): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Activación/renovación/desactivación offline; requestID. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-107-generar-solicitud-de-licencia-sin-conexion.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-108"></a>

### CAP-108 — Importar respuesta firmada y consultar artefactos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-108](01-inventario-funcional.md#fun-108): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Archivo .lic; historial de solicitudes/respuestas. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: license.manage. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-108-importar-respuesta-firmada-y-consultar-artefactos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-109"></a>

### CAP-109 — Interpretar solo lectura y avisos de reloj/identidad

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Licenciamiento; [PAN-048](02-pantallas-y-permisos.md#pan-048) |
| Funcionalidad / objetivo | [FUN-109](01-inventario-funcional.md#fun-109): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Usuario; Técnico |
| Elementos | Estado; fechas; módulos; operación online; archivos offline; solicitudes; soporte. Campos/controles: Sin edición; diagnósticos. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Operación documental según estado; administración de seguridad sigue habilitada. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-109-interpretar-solo-lectura-y-avisos-de-reloj-identidad.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-110"></a>

### CAP-110 — Instalar en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-049](02-pantallas-y-permisos.md#pan-049) |
| Funcionalidad / objetivo | [FUN-110](01-inventario-funcional.md#fun-110): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Bienvenida; progreso; detalle de errores; final; accesos directos. Campos/controles: Instalador .exe amd64; aceptación UAC. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Paquete correcto; administrador; puerto18090 previsto; permisos y antivirus no bloquean. Permisos: Administrador del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-110-instalar-en-windows.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-111"></a>

### CAP-111 — Crear administrador e iniciar en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-049](02-pantallas-y-permisos.md#pan-049) |
| Funcionalidad / objetivo | [FUN-111](01-inventario-funcional.md#fun-111): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Bienvenida; progreso; detalle de errores; final; accesos directos. Campos/controles: Nombre de usuario; nombre visible; contraseña confirmada. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO; consola elevada. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-111-crear-administrador-e-iniciar-en-windows.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-112"></a>

### CAP-112 — Abrir aplicación instalada desde acceso directo

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-049](02-pantallas-y-permisos.md#pan-049) |
| Funcionalidad / objetivo | [FUN-112](01-inventario-funcional.md#fun-112): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Bienvenida; progreso; detalle de errores; final; accesos directos. Campos/controles: Sin campos; launcher.json contiene URL pública y nombre de servicio AIBID; admite la identidad anterior únicamente para actualización. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Usuario del SO; servicio ya configurado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-112-abrir-aplicacion-instalada-desde-acceso-directo.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-113"></a>

### CAP-113 — Desinstalar en Windows conservando datos

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-050](02-pantallas-y-permisos.md#pan-050) |
| Funcionalidad / objetivo | [FUN-113](01-inventario-funcional.md#fun-113): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Confirmación; checkbox datos; segundo aviso; resultado. Campos/controles: Checkbox de datos desmarcado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-113-desinstalar-en-windows-conservando-datos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-114"></a>

### CAP-114 — Desinstalar y reiniciar datos internos en Windows

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-050](02-pantallas-y-permisos.md#pan-050) |
| Funcionalidad / objetivo | [FUN-114](01-inventario-funcional.md#fun-114): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Confirmación; checkbox datos; segundo aviso; resultado. Campos/controles: Checkbox eliminar; segunda confirmación. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO; autorización explícita. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-114-desinstalar-y-reiniciar-datos-internos-en-windows.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-115"></a>

### CAP-115 — Instalar en Ubuntu con paquete de arquitectura correcta

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-051](02-pantallas-y-permisos.md#pan-051) |
| Funcionalidad / objetivo | [FUN-115](01-inventario-funcional.md#fun-115): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | APT; terminal bootstrap/doctor; systemd; prompts de conservación. Campos/controles: Paquete .deb amd64 o arm64. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: sudo/root. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-115-instalar-en-ubuntu-con-paquete-de-arquitectura-correcta.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-116"></a>

### CAP-116 — Crear administrador y arrancar en Ubuntu

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-051](02-pantallas-y-permisos.md#pan-051) |
| Funcionalidad / objetivo | [FUN-116](01-inventario-funcional.md#fun-116): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | APT; terminal bootstrap/doctor; systemd; prompts de conservación. Campos/controles: Nombre/contraseña bootstrap. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: sudo/root; wrapper corre backend como cuenta servicio. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-116-crear-administrador-y-arrancar-en-ubuntu.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-117"></a>

### CAP-117 — Preparar e instalar bundle Ubuntu sin red

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-051](02-pantallas-y-permisos.md#pan-051) |
| Funcionalidad / objetivo | [FUN-117](01-inventario-funcional.md#fun-117): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | APT; terminal bootstrap/doctor; systemd; prompts de conservación. Campos/controles: Paquete; carpeta nueva, admite espacios y Unicode. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: sudo/root; equipo preparador Ubuntu 24.04 misma arquitectura. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-117-preparar-e-instalar-bundle-ubuntu-sin-red.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-118"></a>

### CAP-118 — Desinstalar en Ubuntu

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-051](02-pantallas-y-permisos.md#pan-051) |
| Funcionalidad / objetivo | [FUN-118](01-inventario-funcional.md#fun-118): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | APT; terminal bootstrap/doctor; systemd; prompts de conservación. Campos/controles: Conservar por defecto; texto ELIMINAR si decide borrado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: sudo/root. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-118-desinstalar-en-ubuntu.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-119"></a>

### CAP-119 — Preparar dependencias e instalar en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-052](02-pantallas-y-permisos.md#pan-052) |
| Funcionalidad / objetivo | [FUN-119](01-inventario-funcional.md#fun-119): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Asistente Installer; terminal bootstrap; launchd; navegador. Campos/controles: Paquete .pkg arm64; dependencias externas. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-119-preparar-dependencias-e-instalar-en-macos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-120"></a>

### CAP-120 — Configurar primer acceso y servicio en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-052](02-pantallas-y-permisos.md#pan-052) |
| Funcionalidad / objetivo | [FUN-120](01-inventario-funcional.md#fun-120): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Asistente Installer; terminal bootstrap; launchd; navegador. Campos/controles: Bootstrap interactivo. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-120-configurar-primer-acceso-y-servicio-en-macos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-121"></a>

### CAP-121 — Desinstalar en macOS

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Instalación; [PAN-052](02-pantallas-y-permisos.md#pan-052) |
| Funcionalidad / objetivo | [FUN-121](01-inventario-funcional.md#fun-121): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | Asistente Installer; terminal bootstrap; launchd; navegador. Campos/controles: Conservar por defecto; ELIMINAR si borrado. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-121-desinstalar-en-macos.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-122"></a>

### CAP-122 — Consultar versión y diagnóstico local

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-122](01-inventario-funcional.md#fun-122): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: version; doctor; --config; --sample-directory. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Permisos del SO para configuración/estado. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-122-consultar-version-y-diagnostico-local.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-123"></a>

### CAP-123 — Recuperar administrador local

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-123](01-inventario-funcional.md#fun-123): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: recover-admin; --config; credencial interactiva. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Acceso administrativo SO; sin recuperación por correo. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-123-recuperar-administrador-local.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-124"></a>

### CAP-124 — Recuperar identidad de licencia dañada

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-124](01-inventario-funcional.md#fun-124): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: recover-license; --confirm-license-recovery; --reason. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Acceso local SO; confirmación explícita. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-124-recuperar-identidad-de-licencia-danada.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-125"></a>

### CAP-125 — Administrar servicio y logs por plataforma

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-125](01-inventario-funcional.md#fun-125): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: Get-Service/systemctl/launchctl; logs. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Cuenta autorizada del SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-125-administrar-servicio-y-logs-por-plataforma.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-126"></a>

### CAP-126 — Actualizar instalación conservando estado

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-126](01-inventario-funcional.md#fun-126): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: Instalador nuevo; marcadores existentes. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador SO; paquete compatible. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-126-actualizar-instalacion-conservando-estado.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |

<a id="cap-127"></a>

### CAP-127 — Comprender inicialización, migración y mantenimiento restringido

| Aspecto | Registro |
| --- | --- |
| Módulo / pantalla | Operación; [PAN-053](02-pantallas-y-permisos.md#pan-053) |
| Funcionalidad / objetivo | [FUN-127](01-inventario-funcional.md#fun-127): ilustrar controles, contexto y resultado del procedimiento. |
| Manuales | Instalación; Técnico |
| Elementos | CLI; doctor; bootstrap/recover-admin; recuperación identidad; mantenimiento. Campos/controles: init; configure-license; check-state; bootstrap-ready; migrate; rollback-empty; erase-internal-data. Resultado visible del procedimiento, sin secretos. |
| Condiciones necesarias | Sesión válida y acceso al módulo. En operaciones documentales, biblioteca activa, visibilidad del documento y licencia/módulos compatibles. Las herramientas locales requieren permisos del sistema operativo. Permisos: Administrador SO. Para ilustrar escrituras usar estado previamente preparado en demostración; no ejecutar el procedimiento en producción para tomar la imagen. |
| Archivo sugerido | `capturas/CAP-127-comprender-inicializacion-migracion-y-mantenimiento-restringido.png` |
| Estado | pendiente |
| Observaciones | Escena requerida; no capturada en sesión actual. Las acciones automáticas se ilustran mediante estado de Procesamiento/ficha, no con una UI inventada. |
| Origen real | Archivo no obtenido; nombre reservado para futura captura real. |
