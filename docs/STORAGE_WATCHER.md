# Rutas, almacenamiento y vigilancia

Propuesta de H1 sujeta a D-02/D-03. Los adaptadores se implementan y verifican en
Windows, Linux y macOS; una prueba macOS no demuestra comportamiento en NTFS/SMB.

## Zonas de almacenamiento

| Zona | Ubicación | Acceso | Exposición |
| --- | --- | --- | --- |
| Estado/DB/clave | Disco local de servidor | Cuenta de servicio y recuperación administrativa | Ninguna ruta por API ordinaria |
| Temporal | Local, privada, configurable, fuera de raíz pública | Servicio; preview por ID autorizado | Nunca la ruta física, incluso al rechazar/fallar |
| Final managed | Raíz de biblioteca en disco o montaje con escritura | Servicio y ACL acordada con cliente | Ruta definitiva solo con `storage.view_paths` |
| Linked | Raíz existente del servidor o recurso montado | Solo lectura del servicio | Ruta original solo con `storage.view_paths` |

Temporal, estado, instalación y raíces documentales no se superponen. Raíces linked
y managed tampoco pueden contenerse mutuamente; carpetas hermanas sí. No escribir
archivos de prueba en linked: comprobar lectura/enumeración con operaciones de
lectura. En managed/temporal la prueba de creación/sync/eliminación de un archivo
aleatorio requiere gestión de rutas, deja evento y limpia únicamente su propio archivo.

## Canonicalización y alta de raíces

1. Recibir ruta solo de administrador autorizado; la ruta corresponde al **servidor**.
   Un navegador remoto nunca envía su carpeta local como raíz del servicio.
2. Resolver ruta absoluta y volumen, normalizar separadores por SO, distinguir UNC,
   letras de unidad, case sensitivity real y forma Unicode pertinente al volumen.
   No convertir todas las rutas a minúsculas ni comparar prefijos de cadenas.
3. Resolver enlaces de la raíz elegida hasta su destino real y comprobar identidad
   de directorio/volumen. En V1 propuesta se omiten enlaces/junctions internos con
   aviso. Detectar ciclos/alias por identidad. Montajes se identifican; no seguir
   un cambio de volumen/identidad inesperado durante el recorrido sin verificarlo.
4. Comparar componentes canónicos e identidades contra **toda** la instalación,
   incluidas ubicaciones retenidas de raíces retiradas. Las letras de unidad de un
   usuario interactivo no necesariamente existen en una cuenta de servicio.
5. Preparar resultado y plan con revisión de configuración; al confirmar, bajo
   exclusión de cambios de raíces, volver a validar filesystem y DB. Responder
   `ROOT_PLAN_STALE` si cambió algo. La previsualización no es autorización permanente.

| Relación | Resultado |
| --- | --- |
| Igual a raíz existente | `ROOT_ALREADY_REGISTERED`; mostrar existente solo si puede verla |
| Subcarpeta | `ROOT_IS_DESCENDANT`; ofrecer vista lógica en biblioteca propietaria; sin watcher/OCR nuevos |
| Padre de una o más raíces propias | Plan explícito de consolidación; conservar IDs, OCR, clasificación y permisos |
| Padre/alias con otras bibliotecas | Conflicto; reorganización explícita exige permisos en todas y mapa de autorización |
| Sin superposición | Registrar raíz y encolar escaneo incremental propio |
| Linked contiene managed o viceversa | `ROOT_STORAGE_OVERLAP`, bloqueo sin excepción automática |

Una vista lógica guarda prefijo relativo por componentes y raíz propietaria;
consultas con límites de componente (`carpeta/`, no `carpeta-otra`). No crea
documentos, archivo físico ni watcher. El plan de reorganización puede ser un
diagnóstico sin operación ejecutable hasta contar con sus pruebas completas.

## Identidad, renombrados y consolidación

Identidad fuerte: volumen + identificador del SO con señales de generación/creación
cuando existan. Validar que no sea un inode/file ID reutilizado. Renombrados se
correlacionan con eventos, ausencia/presencia y observaciones; hash ayuda pero no
prueba por sí solo que dos copias sean la misma. Caso ambiguo: no fusionar, registrar
diagnóstico y mantener evidencia. Distintas rutas al mismo hard link se representan
como alias de una identidad, nunca documentos paralelos entre bibliotecas.

Consolidar padre: calcular plan de hijos, rutas y bibliotecas afectadas; pausar solo
sus watchers, esperar/cancelar leases afectados, volver a comprobar identidades,
crear raíz padre, cerrar ubicaciones históricas y crear actuales con mismo
`physical_file_id`. Actualizar vistas por nuevo prefijo. Cambiar ownership entre
bibliotecas requiere mapear documentos, expedientes, catálogos y permisos sin
silenciar conflictos; no aceptar un cambio parcial que eluda FK o autorización.
Usar transacción de metadatos y journal de operación recuperable para lotes grandes;
durante consolidación mantener una revisión de raíz activa para servir lecturas.
Marcar raíces sustituidas, reiniciar vigilancia y escanear solo identidades nuevas.
Conservar aprobación, hash, OCR y toda evidencia de ruta previa.

Deshabilitar pausa exploración automática; no borra datos. Retirar pide elegir
`retain_index` o `remove_index_references`, presenta cantidad y expedientes
afectados y exige `documents.remove_index` para lo segundo. Nunca llama a delete
sobre originales. Si se retienen referencias, reañadir una raíz debe reconciliar
esas identidades, no duplicarlas. Las rutas retiradas no son un escape al control
de colisiones.

## Watcher recursivo y reconciliación

Adaptador nativo por SO y registro de directorios vigilados. Al iniciar una raíz,
vigilar directorios y escanear con barrera final para cubrir carreras entre ambos;
al aparecer un subdirectorio, registrarlo y enumerar su contenido inmediatamente.
Retirar watches de directorios desaparecidos. Gestionar cierre, overflow, permisos
y límites de descriptores; diagnóstico visible y modo polling si no hay eventos
fiables. No afirmar que una API no recursiva vigila todos los descendientes.
[fsnotify documenta esos límites](https://github.com/fsnotify/fsnotify).

Cada evento marca un candidato «sucio»; agrupar por identidad/ruta y observar tamaño,
mtime y posibilidad de lectura durante una ventana configurable. Propuesta inicial:
debounce 750 ms y estabilidad 2 s, sin tratar estos tiempos como prueba absoluta.
Comparar metadatos antes/después de leer; si cambia, descartar resultado y reagendar.
Reintentar archivos ocupados/compartidos con límites. No indexar simultáneamente el
mismo archivo. No omitir automáticamente `chmod`: puede indicar permiso perdido
o preceder eliminación según SO.

Escaneo/reconciliación guarda generación, revisión de raíz, cola/cursor de directorios,
observaciones y errores por subárbol. Propuesta inicial de reconciliación cada
15 minutos, ajustable al corpus/montaje; «Verificar biblioteca ahora» encola el mismo
proceso idempotente. Recorrer por lotes y ceder recursos a HTTP/cargas.

Solo un recorrido completo, con raíz e identidad de montaje comprobadas y sin error
del subárbol relevante, puede confirmar ausencias. Un permiso denegado, volumen
desmontado o interrupción vuelve `can_confirm_absence=false`; no hacer un UPDATE
masivo a missing. Al volver la conexión, reconciliar desde checkpoint compatible
o iniciar nueva generación sin perder índice anterior. El cursor no presupone que
los directorios permanecieron estáticos: repetir un tramo es seguro por idempotencia.

## Cambios, avisos y texto retenido

Persistir una observación de cambio con clave propia, no solo `(archivo, hash)`:
A→B→A son dos cambios. Agrupar ráfaga de eventos para una escritura estabilizada;
registrar nuevo hash/metadatos, evento y notificación una vez. Visitas posteriores
leen la misma notificación; acuse por usuario en `notification_receipts`. Si hubo
varias escrituras entre observaciones no prometer conocer estados intermedios.

Modificar linked conserva clasificación/aprobación y texto anterior hasta nueva
extracción completa. Si OCR falla, `stale` y diagnóstico. Eliminar linked conserva
metadatos, ruta previa y páginas/FTS; abrir/descargar devuelve `DOCUMENT_UNAVAILABLE`.
Reaparición reconocida restaura disponibilidad; no genera otro documento.

Integridad managed se verifica con vigilancia y reconciliación distinta: comparar
bytes contra versión aprobada, no adoptar una edición externa como versión aprobada.
Una escritura de la app lleva operación esperada, ruta, hash y estado; correlacionar
ese resultado, nunca ignorar indiscriminadamente todos los eventos por tiempo.
Al faltar, alerta de restauración; al reaparecer, verificar contra hash aprobado o
enviar nueva revisión. Preservar OCR histórico aunque el PDF previo ya no exista.

## Materialización managed recuperable

1. Con permiso/capacidad y revisión esperada, reservar destino único y revisión de
   nomenclatura; persistir operación `planned` y estado documental `materializing`.
2. Copiar temporal a parcial privado dentro del volumen destino, sin sobrescribir
   archivos existentes. Limitar E/S, verificar tamaño y SHA-256 desde destino.
3. Sincronizar archivo y directorio según SO, publicar con operación sin reemplazo
   dentro del destino; verificar garantías del montaje. Estado `published`.
4. En transacción corta, revalidar documento, licencia y decisión; registrar ruta
   exacta, contenido aprobado, historial y auditoría. `approved` solo tras commit.
5. Limpiar temporal solo tras releer operación committed y comprobar destino íntegro;
   marcar `cleaned`. Temporales rechazados/cancelados usan política de retención.

| Punto de fallo | Recuperación |
| --- | --- |
| Antes/durante copia | Temporal intacto; reanudar o recomenzar parcial de esa operación |
| Hash/tamaño distinto | No publicar ni aprobar; registrar error de almacenamiento |
| Destino ocupado | Nuevo nombre reservado; no sobrescribir; actualizar plan auditado |
| Publicado, sin commit DB | Reconciliar usando journal/ID/hash; completar si válido o retener para recuperación |
| Commit, sin borrar temporal | Detectar éxito; limpiar idempotentemente |
| Licencia/permisos/versión cambian | Pausar decisión final; conservar evidencia y temporal, no aprobar con contexto obsoleto |

Una revisión posterior de managed ya definitivo no vuelve a aplicar la nomenclatura
como si fuera una nueva carga: verifica el contenido y decide sobre esa versión.
La UI nunca muestra como aprobada una operación meramente aceptada para procesamiento.

## Estructura y nombres

Selector por biblioteca: `{ejercicio}/{identificador}/{categoria}`,
`{identificador}/{categoria}`, `{categoria}/{identificador}`, `{identificador}`;
opciones sin expediente según campos disponibles, por ejemplo `{categoria}` o
raíz. No admitir token ejercicio si está deshabilitado, ni identificador sin
expediente cuando el patrón lo requiere. Mostrar vista previa antes de guardar.

Tokens de nombre: `{prefijo}`, `{identificador}`, `{ejercicio}`, `{tipo_documento}`,
`{consecutivo}` o nombre original saneado. Lista permitida de patrones, sin código
ejecutable. Sanear traversal, controles, separadores, nombres reservados Windows,
puntos/espacios finales, longitud y normalización; comparar con semántica del volumen.
Consecutivo reservado en transacción; sufijo estable por identidad si hay colisión.
Guardar nombre visible/original separado de ruta física. Cambiar categoría, etiqueta,
plantilla o configuración solo afecta destinos futuros, nunca reorganiza disco.

## Verificación administrada H5

La reconciliación managed consulta únicamente ubicaciones definitivas registradas,
por lotes de 100; no importa otros archivos ni procesa cargas temporales. Se puede
solicitar manualmente y usa el intervalo de verificación de cada raíz. Una raíz
inaccesible cambia la disponibilidad efectiva sin declarar ausencias masivas.
Hash/identidad nuevos crean versión/aviso; aprobado pasa a `needs_review`. Una
escritura propia ya registrada por la materialización no genera alerta externa.
La política linked aprobada permanece sin cambios. [Detalle H5](H5.md).
