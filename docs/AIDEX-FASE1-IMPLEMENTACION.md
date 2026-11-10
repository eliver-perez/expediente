# AIDEX 1.0.0 — Fase 1: implementación y verificación

Entrega: 2026-10-21. Implementación terminada y verificaciones locales aprobadas. Esta fase no construye instaladores ni modifica la versión, el servidor de licencias o los inventarios/manuales anteriores. No incluye respaldos/restauración de Fase 2. Se utilizaron exclusivamente bases y archivos sintéticos para las pruebas destructivas. Los límites de validación nativa se detallan en G.

## A. Análisis realizado

- Bibliotecas: `libraries`, membresías, versiones de configuración y configuración por ámbito. `disabled_at` existe y las consultas documentales autorizadas lo excluyen; se utiliza como estado de archivado reversible.
- Raíces: vinculadas/administradas, planes, vistas lógicas, versiones, controles, recorridos, observaciones, caché de escaneo, omisiones y errores. Retirar conserva la raíz; faltaba la reanudación explícita que ahora se incorpora. Las superposiciones se comparan por ruta e identidad del SO.
- Documento → archivo físico → ubicaciones/versiones → extracciones/páginas/FTS. Las ubicaciones pueden compartir un archivo físico. El hash no identifica propiedad: contenidos idénticos pueden corresponder a documentos independientes. No se borran originales vinculados ni publicados en destinos administrados.
- Cargas: `upload_items` vincula biblioteca/lote/documento a un localizador privado. `receiving` es un journal de transferencia; `staged/retained` puede ser la única copia de un documento ya clasificado. No basta que esté dentro de uploads para declararlo descartable.
- Expedientes/catálogos: requisitos, plantillas/versiones, clasificaciones, revisión y materialización. Una materialización sin completar bloquea eliminación; un destino activo o predeterminado debe retirarse y dejar de usarse antes de eliminar su registro.
- Previews: caché por documento/hash/generador, no global por hash. Extractores y conversores crean directorios de trabajo exclusivos y los liberan al finalizar. No se eliminarán directorios de trabajo globalmente como consecuencia de una eliminación.
- Trabajos: leases y fencing en SQLite; cargas `receiving`, jobs `running`, previews `generating` y materializaciones pendientes prueban actividad. El archivado/eliminación se rechaza mientras existan operaciones que publican archivos; no se interrumpe una publicación a medias.
- Auditoría: append-only y sin FK hacia bibliotecas/documentos. Se conserva; la historia operativa retirada se resume en un manifiesto permanente y el evento de eliminación. La integridad referencial permanece habilitada.

### Decisiones de seguridad

El plan y la confirmación se autorizan por separado y se vuelven a verificar bajo el escritor SQLite. La eliminación de biblioteca requiere permiso exclusivo del administrador y nombre exacto. Los planes vencen y se invalidan cuando cambia su alcance. El manifiesto y el journal de limpieza sobreviven a los registros eliminados. La limpieza se hace exclusivamente por archivos individuales registrados, con identidad/hash, ubicación privada validada y comprobación de referencias ajenas; dudas o errores quedan pendientes y reintentables. Los documentos definitivos y las cargas clasificadas se conservan aunque estén en almacenamiento privado.

El archivado conserva configuración, registros, tareas e información de restauración. Los workers no reclaman trabajos archivados y el watcher se suspende; las tareas pendientes vuelven a ser elegibles al restaurar. No hay limpieza extraordinaria al archivar.

Los candidatos a limpieza se obtienen de relaciones documentales, no de una enumeración global de carpetas. Cada localizador debe corresponder al UUID y tipo interno esperados. Se comprueban contenedor autorizado, identidad, tamaño, SHA-256 y ausencia de referencias restantes. Antes de retirar relaciones se persisten el manifiesto y el journal. Los archivos se limpian después del commit, individualmente, mediante un nombre de cuarentena exclusivo y una segunda comprobación. Los planes vencen a los 15 minutos.

## B. Administración de carpetas

- **Reanudar supervisión** valida ruta, acceso de la cuenta del servicio, identidad, superposiciones, licencia, configuración y permisos. Recupera el estado operativo y programa una verificación.
- Se marcan los documentos retirados por la propia operación para reconstruir referencias e índice. No se revierten retiradas individuales. Para retiros anteriores a esta migración se exige evidencia del plan y la auditoría; los casos ambiguos permanecen retirados.
- **Inspeccionar carpeta** distingue inexistencia, permisos insuficientes, ruta de red no disponible y fallo de lectura. Informa si ya está registrada o retirada. Conserva validaciones de origen vinculado/administrado, seguridad, superposición y configuración obsoleta.
- **Eliminar carpeta de AIDEX** muestra alcance, ubicaciones y manifiesto. Retira su registro, vistas y dependencias exclusivas. Los originales físicos permanecen. Un archivo con otra ubicación registrada conserva su documento y caché.
- Un destino administrado debe retirarse y dejar de ser el predeterminado antes de eliminar su registro. Una materialización pendiente bloquea la eliminación para evitar interrumpir un guardado definitivo.
- El registro eliminado no puede reanudarse. Sus originales pueden registrarse después mediante el flujo ordinario de incorporación.

Verificado: retiro/reanudación, retiros antiguos, inexistencia, permisos, duplicados, superposición, rechazo de reanudación de registros eliminados, aislamiento por raíz, referencias compartidas y conservación de originales/auditoría.

## C. Administración de bibliotecas

**Archivar biblioteca** la oculta del listado activo y permite encontrarla entre las archivadas. Conserva archivos, registros, configuraciones, relaciones y colas. Bloquea las escrituras operativas, la selección de nuevos trabajos/previews y la vigilancia. No dispara limpieza extraordinaria; sus cargas también quedan excluidas de la retención ordinaria mientras está archivada.

**Restaurar biblioteca** revisa licencia, configuración y raíces operativas. Si una ubicación indispensable no está disponible, cambió de identidad o tiene conflictos, se rechaza toda la restauración y se informa la causa. No se declara operativa una restauración parcial. Al restaurar, las tareas vuelven a ser elegibles y se programan verificaciones.

**Eliminar biblioteca definitivamente** exige Administrador de instalación en backend, advertencia irreversible en SweetAlert2 y nombre exacto. El botón permanece deshabilitado hasta que coincida. Muestra conteos de registros afectados, documentos conservados, cargas, candidatos temporales y ubicaciones. Ofrece un manifiesto JSON Lines antes y después de eliminar.

Se retiran las relaciones operativas correspondientes: documentos exclusivos, ubicaciones registradas, índices y extracciones, tareas/historial operativo, previews, cargas/lotes, raíces/vistas, expedientes/requisitos, plantillas/versiones, catálogos, membresías y configuraciones. Se conservan eventos de auditoría y evidencia independiente de eliminación. Los originales y copias clasificadas privadas no se borran.

**Eliminaciones y limpieza** conserva el resultado tras recargar o reiniciar. Distingue registros eliminados con limpieza pendiente de una operación terminada, muestra referencia de soporte y diagnósticos, permite descargar el manifiesto y reintentar de forma segura.

La limpieza inmediata está limitada a 2 segundos/25 archivos; el reintento solicitado, a 5 segundos/100 archivos; el mantenimiento, a 30 segundos/50 archivos por ronda. El journal mantiene lo pendiente. Estos límites acotan el trabajo cooperativo; un bloqueo de E/S del sistema operativo tiene los límites indicados en G.

## D. Interfaz y notificaciones

- SweetAlert2 para eliminar, archivar, restaurar, retirar/reanudar y errores que requieren intervención, con estilos de AIDEX. Nombres y rutas se insertan como texto, sin interpretar HTML del usuario.
- Toast reutilizable fijo en la esquina superior derecha, con éxito/error/advertencia/información, cierre manual, duración automática y anuncios accesibles. Mantiene hasta tres notificaciones recientes, pausa su expiración al enfocarlas y se oculta durante los diálogos para no taparlos.
- Se trasladaron los éxitos de Usuarios, permisos, Bibliotecas, Carpetas, configuración, Cargas, clasificación, revisión, Procesamiento, red y licenciamiento. Se conservan errores junto a los campos. Una solicitud de licencia pendiente se presenta como información, no como éxito.
- Las acciones de ciclo de vida muestran actividad inmediata y evitan duplicación. Los mensajes distinguen verificación programada, restauración terminada y limpieza pendiente.
- Omisiones e historial usan paginación del servidor: búsqueda, filtros, orden estable y 10/25/50/100 registros. Las rutas se truncan y se pueden expandir. Se conservan estados, tipos, prioridades, reintentos y cancelaciones de tareas.
- Las omisiones muestran todos los registros actualmente conservados, no un historial permanente inventado. Se actualizan con los recorridos. Las búsquedas con `%`, `_` y barras se interpretan literalmente.
- Se mantienen trabajos activos y barras de progreso. Dashboard conserva cálculos y gráficas; sus tablas desplegables añaden orden, paginación y **Ocultar días sin actividad**, activado inicialmente.
- El nombre visible es **Registro de errores**; se conservan las rutas internas.

Capturas: [Toast en móvil](qa/aidex-fase1-20261021/toast-mobile.png) y [Procesamiento](qa/aidex-fase1-20261021/processing.png).

## E. Cambios técnicos

| Área | Archivos principales |
| --- | --- |
| Ciclo de vida, permisos y restauración | `internal/libraries/lifecycle.go`, `service.go`, `documents.go`, `roots.go`, `filesystem.go`, `license.go` |
| Eliminación y limpieza | `internal/libraries/deletion.go`, `deletion_cleanup.go`; integración en `runtime.go`, `preview_worker.go`, `managed_integrity.go`, `extraction_admission.go` |
| Tablas | `internal/libraries/processing_tables.go`, `internal/httpapi/processing_tables.go` |
| API/diagnóstico | `internal/httpapi/lifecycle.go`, `library_handlers.go`, `server.go`, `internal/diagnostics/{diagnostics,messages}.go` |
| Migración | `db/migrations/0015_library_lifecycle.{up,down}.sql`, `internal/storage/database.go`, `upgrade_backup.go` |
| UI reutilizable | `web/src/{feedback,Lifecycle,CompactTables}.tsx`, `components.tsx`, `main.tsx`, `styles.css` |
| Pantallas | Bibliotecas/configuración, Dashboard, Registro de errores y formularios principales con mensajes de éxito |
| Pruebas | `internal/libraries/lifecycle_test.go`, ajustes de retiro/migración, actualización desde esquema 14, `web/tests/lifecycle.spec.ts` y expectativas de Dashboard/errores |

La migración añade `root_retired_documents`, `entity_deletions`, `deletion_documents`, `deletion_cleanup`, índices y el permiso `libraries.delete` exclusivamente al rol de instalación. La evidencia permanece después de eliminar la entidad. La copia preventiva existente se ejecuta también al pasar del esquema 14 al 15; esto integra la migración, no incorpora respaldos de Fase 2. La reversión de una base vacía reconoce el esquema 15 y rechaza descartar evidencia de ciclo de vida.

Dependencia añadida: **SweetAlert2 11.26.25**, fijada en `package.json` y lockfile. Se conservan las tecnologías existentes.

### API añadida

Todos los endpoints conservan protección de sesión; las mutaciones usan CSRF y autorización de backend.

| Método y ruta | Propósito |
| --- | --- |
| `GET /api/v1/libraries?archived=true` | Consultar archivadas |
| `POST /api/v1/libraries/{library}/archive` y `/restore` | Archivar/restaurar con revisión esperada |
| `POST /api/v1/roots/{root}/resume` | Reanudar con revisión esperada |
| `POST /api/v1/libraries/{library}/deletion-plans` | Preparar eliminación de biblioteca |
| `POST /api/v1/roots/{root}/deletion-plans` | Preparar eliminación de raíz |
| `GET /api/v1/deletions` y `/{operation}` | Historial paginado y estado autorizado |
| `POST /api/v1/deletions/{operation}/confirm` y `/retry` | Confirmar o reintentar limpieza |
| `GET /api/v1/deletions/{operation}/manifest` | Descargar manifiesto |
| `GET /api/v1/libraries/{library}/file-skips` y `/job-history` | Filtrar, ordenar y paginar registros conservados |

## F. Pruebas y resultados

Entorno: macOS ARM, Go 1.27.1 y Chrome/Playwright. Datos sintéticos aislados. Los comandos largos tuvieron límite externo y terminaron; no quedaron verificaciones ejecutándose.

| Comprobación | Resultado |
| --- | --- |
| Regresión `go test -tags development ./...` | Ejecutada completa. Detectó ajustes de superposición de origen, adaptación de reanudación y migración 15; se corrigieron. Los demás paquetes pasaron. |
| Regresión final completa de bibliotecas y almacenamiento | **Aprobada**, 91.984 s y 3.314 s. Cubre los paquetes afectados por las correcciones; no se repitieron innecesariamente los demás. |
| `go test -race ... -run '^TestLifecycle'` | **Aprobada**, 115.267 s; incluye concurrencia, rollback e interrupciones. |
| Preparación con traversal, symlink, sustitución y contenedor redirigido | **Aprobada**, 2.561 s. Prueba adicional posterior a race; los archivos no verificados se conservaron con diagnóstico. |
| Actualización desde esquema 14 de producción | **Aprobada**, 0.483 s; conserva documentos, hash, expediente, índice, configuración y copia previa. Reabrir no repite la copia innecesariamente. |
| Navegador: módulos afectados | **13/13 aprobados**, 1.4 min: ciclo de vida, Dashboard, usuarios/sesiones, permisos, bibliotecas/búsqueda, licencia con simulador, LAN/local, errores, expedientes/cargas, procesamiento y revisión. |
| TypeScript/build frontend | **Aprobados**. Advertencia no bloqueante: bundle principal 528.38 kB minificado / 147.97 kB gzip. |
| `go vet -tags development ./...`; `git diff --check` | **Aprobados**. |
| Backend sin tag de desarrollo | **Compila** para macOS ARM64, Windows AMD64, Linux AMD64 y Linux ARM64. Binarios temporales de comprobación, no instaladores. |

### Integridad y aislamiento

1. Tres bibliotecas híbridas con originales y cargas propias: eliminar B conserva sus originales y su carga clasificada, limpia su carga no incorporada y conserva temporales/registros de A y C.
2. Varias raíces: eliminar una limpia solamente su preview exclusivo; conserva previews de las demás, una carga sin pertenencia a esa raíz y todos los originales. Las raíces restantes siguen siendo verificables.
3. Archivo con varias ubicaciones: se conserva documento/caché al eliminar una raíz y se redirige la ubicación primaria registrada.
4. Documento administrado publicado y asociado a expediente: conservación byte por byte; materialización pendiente bloquea la eliminación.
5. Interrupciones después del commit, del renombrado y del unlink: recuperación idempotente sin eliminación ajena.
6. Permisos, cambio de archivo/contenedor, symlinks y referencias compartidas: archivo conservado y diagnóstico pendiente. Restituir permisos permite reintentar cuando la identidad ya estaba comprobada.
7. Fallo de transacción inyectado: conserva registros y archivos. Dos confirmaciones concurrentes producen una única eliminación auditada.
8. Plan obsoleto, nombre incorrecto y gestor sin permiso global: rechazados; se revalidan sesión y permisos.
9. `PRAGMA foreign_key_check` sin referencias rotas y `PRAGMA integrity_check = ok` en los escenarios de eliminación.
10. 137 omisiones en backend y 115 archivos omitidos en navegador: páginas más allá de los primeros 100, filtros literales, orden, estado vacío y ruta expandible.
11. Toast visible con desplazamiento, móvil de 390 px, cancelación por teclado, nombre exacto, manifiesto y doble clic con un solo POST. Sin errores JavaScript en el nuevo escenario de bibliotecas.

La primera regresión detectó un rechazo de superposición que debía mantenerse para orígenes distintos, una prueba que aún utilizaba el antiguo atajo de reactivación y dos expectativas/mecanismos de migración pendientes de adaptar. Se corrigieron y pasaron las regresiones indicadas. La prueba nueva de navegador también detectó que la cola de Toast podía ocultar una notificación reciente; ahora se muestran las tres más recientes. La protección contra planes obsoletos funcionó: la prueba espera que termine la verificación programada antes de preparar la eliminación.

Evidencia y comandos: [directorio QA](qa/aidex-fase1-20261021/README.md). Se conserva la primera corrida fallida junto a las comprobaciones posteriores.

## G. Pendientes y límites

- Falta validación nativa bajo la cuenta real del servicio en Windows: UNC/SMB, ACL NTFS y junctions; también ejecución de esta fase en Ubuntu AMD64/ARM64. Compilar no sustituye estas pruebas. Los instaladores están diferidos hasta después de Fase 2.
- Las interrupciones se simularon mediante estados persistidos y fallos de transacción; no mediante un corte eléctrico real. Incluir reinicio forzado en aceptación de los próximos paquetes.
- Las E/S de red pueden esperar al sistema operativo. Los contextos acotan el trabajo cooperativo; no se incorporó un nuevo aislamiento de E/S.
- Un temporal cuya identidad/propiedad no pudo demostrarse se conserva con journal y referencia de soporte. El reintento no fuerza el borrado de archivos cambiados. Hace falta revisar la incidencia, no borrar directorios internos manualmente.
- La preparación calcula hashes de candidatos. Puede tardar con grandes volúmenes de temporales; la interfaz muestra actividad y una cancelación/fallo anterior al commit conserva los registros. No se midió con millones de archivos ni con grandes volúmenes de red productivos.
- Los planes no confirmados y manifiestos se conservan. El vencimiento bloquea confirmaciones antiguas; no se añadió una política nueva de purga de esta evidencia.
- `npm audit` detectó **dos avisos altos preexistentes de herramientas de desarrollo**: `vite` y `source-map-js`. El backend distribuido no ejecuta un servidor Vite. Se recomienda actualizar y validar esas herramientas antes de preparar los paquetes oficiales; el informe se conserva, sin incorporar actualizaciones ajenas a esta fase.
- El aviso de tamaño del bundle no bloqueó compilación ni ejecución. Dividirlo adicionalmente sería una optimización separada.
- Para el futuro Manual de usuario: retirar/reanudar frente a eliminar; archivadas/restauración; nombre exacto y permisos; limpieza pendiente; descarga/custodia del manifiesto; conservación de originales; filtros y paginación. Los manuales anteriores no se modificaron.

No se identificó pérdida de originales ni cruce de temporales entre entidades en los escenarios probados. Fase 1 queda entregada con estos límites explícitos. La Fase 2 requiere autorización posterior.
