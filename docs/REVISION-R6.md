# AIBID — revisión r6

Solicitud del 27 de septiembre de 2026. Implementación y comprobaciones locales del 28 de septiembre.

## Hallazgos y cambios

### Reconciliación y watcher

La reconciliación recorre la raíz, registra ubicaciones y versiones, compara metadatos y
huellas cuando corresponde y encola la extracción de contenido nuevo o modificado. Solo un
recorrido completo sin incidencias puede confirmar que un original desapareció. Conserva
el índice histórico ante carpetas inaccesibles. La verificación administrada comprueba
originales ya publicados y no importa cualquier archivo encontrado en el destino.

Había cuatro causas distintas del comportamiento observado:

- El intervalo inicial era 900 segundos. Ahora las raíces nuevas reciben 86400 segundos.
  La migración no cambia los intervalos existentes. El formulario admite segundos, minutos,
  horas y días, de 10 segundos a 30 días, conservando internamente segundos.
- Los eventos del watcher provocaban recorridos completos, incluso ante cambios de atributos.
  Ahora las creaciones/modificaciones se agrupan en conjuntos acotados de rutas; los cambios
  de atributos y directorios reservados se omiten. Renombres, eliminaciones, desbordamientos
  y el vencimiento periódico requieren un recorrido completo. Hay un breve tiempo de espera
  para que termine la escritura. Un watcher operativo sigue detectando cambios entre revisiones.
- `WATCH_EVENTS_LOST` quedaba retenido en memoria y en la base. Ahora se registran generaciones
  de pérdida y recuperación. Un recorrido completo sin errores reconoce únicamente las
  pérdidas anteriores a su inicio. Una pérdida posterior o un recorrido parcial no pueden
  borrar el aviso. Los fallos reales de registro del watcher mantienen la vigilancia periódica.
- El mismo recorrido se mostraba en dos secciones de Procesamiento. Ahora su progreso y botón
  de cancelación aparecen una vez; la sección de trabajos activos muestra el procesamiento
  de contenido. Además, un recorrido completo con incidencias respeta su intervalo y no crea
  un ciclo inmediato de nuevas verificaciones. Las incidencias siguen visibles.

El diagnóstico de las raíces se refresca automáticamente sin descartar cambios del formulario.
Las raíces muestran fecha de comprobación, estado del watcher y último resultado de recorrido,
con fecha de finalización y contador de altas o cambios de contenido. Ese contador no pretende
contar eliminaciones, renombres ni eventos del sistema. Los recorridos parciales del watcher
no aplazan la revisión completa periódica. Al iniciar se reconstruye la cobertura de vigilancia;
esto puede requerir un recorrido adicional aunque el intervalo no haya vencido.

La migración 0008 unifica solicitudes heredadas redundantes y establece una restricción SQLite:
solo una verificación en cola, activa, en reintento o pausada por raíz. Se reutiliza el trabajo
existente ante solicitudes manuales redundantes. Los eventos recibidos durante un recorrido
se conservan para comprobarlos después. La recuperación tras reinicio conserva checkpoints y
utiliza nuevos tokens de ejecución; el lock del estado evita dos procesos propietarios.
Las reglas de raíces anidadas, vistas lógicas y consolidación se mantienen.

### Desinstalación y reinstalación

Windows r6 incluye la casilla **Eliminar los datos de AIBID al desinstalar**, desmarcada por
defecto, y una segunda confirmación. Sin marcarla se conservan configuración, identidad local,
base, índices, texto, auditoría y documentos. Reinstalar la misma versión permite recuperar
esos datos. La cuenta de servicio personalizada no se cambia al actualizar una instalación.

Con la opción marcada, el servicio se detiene y la herramienta de mantenimiento toma el lock
exclusivo. Se validan todas las rutas antes de comenzar. Se eliminan solamente:

- `documental.db`, WAL y SHM; incluyen usuarios, bibliotecas, índice FTS, texto/OCR y auditoría.
- Respaldos SQLite internos `upgrade-backups/before-processing-*.db`, comprobando su cabecera.
- Identidad y diario de recuperación de licencia, con sus archivos históricos reservados.
- Configuración privada y sus respaldos de provisionamiento, marcadores de versión e inicio.

Nunca se recorre una raíz de biblioteca ni se borra recursivamente el directorio de estado.
Se conservan en su ubicación las bibliotecas vinculadas, los originales administrados, las
cargas privadas en `state/uploads`, instantáneas PDF de extracción y archivos no reconocidos.
Se rechazan enlaces en rutas de metadatos. La limpieza del programa Windows usa un manifiesto
de archivos distribuidos y verifica sus hashes; los archivos ajenos o modificados se conservan.
Pueden quedar carpetas vacías y el archivo de lock: no impiden iniciar una base nueva.

Tras borrar los datos se debe crear el administrador, activar y registrar nuevamente las
bibliotecas. Los archivos definitivos siguen en sus rutas. Las cargas privadas conservadas
pierden su relación con la base eliminada; no se reasignan ni se importan automáticamente.
La limpieza local no libera una activación en el proveedor: si se desea liberarla, usar
**Desactivar licencia** antes de desinstalar. El borrado de metadatos no contacta al proveedor.

El código de desinstalación macOS ofrece la elección con respuesta predeterminada No y una
segunda confirmación escrita. Ubuntu incorpora `sudo aibid-test-uninstall` con la misma
confirmación; `apt remove` y `apt purge` conservan datos. Los paquetes macOS Apple Silicon y Ubuntu incluyen también estos scripts; sus servicios y
las opciones de desinstalación requieren verificación nativa en cada sistema. macOS Intel queda excluido.

### Qué hace retirar una carpeta

La opción se llama ahora **Dejar de supervisar esta carpeta…**. Al pulsarla se prepara un plan
con revisión, conteos e impacto en expedientes, válido durante 15 minutos. Todavía no desactiva
la raíz ni cambia documentos. La confirmación detiene la supervisión y permite elegir:

| Opción | Efecto interno |
| --- | --- |
| Conservar texto y resultados de búsqueda — predeterminada | La raíz queda retirada y pausada; se conservan documentos, texto y resultados. |
| Retirar sus documentos del índice | Además, marca esos documentos como retirados y elimina sus páginas del índice de búsqueda. Los conteos de expedientes dejan de incluirlos. |

Ambas conservan archivos físicos, relaciones históricas, versiones, extracciones y auditoría.
El watcher deja de seguir esa raíz. La verificación puede habilitarse nuevamente desde su
configuración. Esto no restaura documentos que se retiraron expresamente del índice.
No se cambió la semántica de retiro; se aclararon nombre, alcance y confirmación.

### Qué hace retirar un documento del índice

No es un borrado del archivo ni únicamente una eliminación del resultado de búsqueda:

- Marca `documents.deleted_at`; el registro principal sigue existiendo.
- Elimina `indexed_pages` y sus entradas FTS. Deja de aparecer en búsquedas y listados normales.
- Conserva las relaciones con biblioteca y expediente, versiones y extracciones históricas
  de texto/OCR, historial y auditoría. Los conteos activos excluyen el documento retirado.
- Los registros históricos conservados no equivalen a que la ficha retirada siga accesible
  en la navegación normal.
- Un nuevo recorrido del mismo archivo físico no revierte la baja. No se añadió una función
  de restauración de documentos retirados.
- No elimina, mueve ni modifica el original. La operación no se permite durante materialización.

Se mantiene **Retirar del índice**, con explicación visible y confirmación obligatoria; se
conserva además la confirmación del impacto en expedientes cuando corresponde.

### Duplicados, explorador e historial

Se reutilizan los servicios de permisos, el visor PDF/texto y los componentes de auditoría.
No hay eliminación automática ni un segundo visor.

- Duplicados: tabla de grupos con hash abreviado, tamaño, archivos y bibliotecas; selección
  y paginación de 25 grupos. Panel de archivos con ruta, biblioteca, tamaño, fecha disponible
  y disponibilidad; páginas independientes de 25 archivos y desplazamiento acotado. La huella
  completa se consulta una vez por grupo. Las consultas conservan su filtrado por permisos.
- Explorador: lista de detalles y cuadrícula muestran el contenido directo de la misma carpeta.
  Alternar la vista conserva carpeta y página. La preferencia de vista se guarda en la sesión.
  Carpetas y documentos tienen páginas independientes de 50 elementos; desaparece “Mostrar
  más carpetas”. La búsqueda de toda la biblioteca o de una carpeta sigue disponible.
- Colores: fondo claro y texto oscuro al pasar el cursor, selección distinguible y foco visible;
  se conserva la paleta de AIBID.
- Historial: tabla con fecha, evento comprensible, responsable real o Sistema, descripción y
  detalles. Se cubren los eventos existentes de documentos, incluidos clasificación, asociación,
  reasignación, revisión, publicación, cambios/disponibilidad, cancelación y retiro. Los
  identificadores técnicos y JSON quedan en la sección avanzada del detalle reutilizado.
- Huella y trazabilidad y Retirar del índice son desplegables independientes; ambos pueden
  permanecer abiertos. La transición respeta la preferencia de movimiento reducido.

## Validación y límites

Pruebas funcionales pequeñas, sin carga masiva ni cambios en bibliotecas reales. El escenario
nuevo de navegador usa 51 PDFs pequeños para comprobar el límite de una página de 50 y dos
contenidos distintos para los grupos de duplicados.

- Suite Go con `development`: aprobada, incluyendo recuperación del watcher, coordinación,
  reanudación, cadencia ante errores, retiro/conservación y las suites existentes H3–H6.
- Limpieza de metadatos: documentos vinculados, administrados, cargas privadas, instantáneas
  y archivos desconocidos permanecen idénticos; una base nueva se inicializa vacía.
  Se comprueba rechazo de enlaces antes de borrar datos.
- Migración: se comprueba coalescencia de solicitudes heredadas y conservación de 900 segundos.
- TypeScript y Vite: aprobados. Diez escenarios Chrome comprobados: nueve aprobaron en la
  primera ejecución; H5 se adaptó a la navegación por carpetas y aprobó junto con los dos
  escenarios de procesamiento/r6 en una ejecución final de tres pruebas (24,1 segundos).
- Detector de carreras: bibliotecas, almacenamiento y mantenimiento aprobados.
- Empaquetado Windows: compilación y verificaciones del constructor; la instalación nativa
  y las dos opciones de desinstalación deben comprobarse en Windows con este artefacto.

No se certifican aquí ACL, UNC/SMB, elevación, desinstalación ni reinstalación nativas de Windows.
Esas comprobaciones corresponden al equipo del propietario. No se ejecutaron cambios contra
su carpeta compartida ni contra su activación real. H7 sigue teniendo pendientes de distribución,
firmas y respaldo/restauración integral que esta revisión no sustituye.

## Cierre local

- Compilaciones de Windows amd64, macOS Apple Silicon y Ubuntu amd64 terminadas.
  NSIS generó Windows r6; sus iconos y dependencias PE pasaron la comprobación del constructor.
- Ejecutable extraído del paquete macOS: manifiesto por hashes, firma ad hoc, diagnóstico
  PDF/OCR, administrador de seis caracteres, login, reinicio, lock, permisos y SQLite íntegra.
  La CLI rechazó borrar sin confirmación; con confirmación eliminó los metadatos temporales,
  conservó el PDF privado y permitió inicializar una base nueva sin administrador anterior.
  Esto se hizo en una carpeta temporal, sin instalar launchd ni modificar bibliotecas reales.
- Ubuntu: estructura AR/tar, propietario root del payload, ELF amd64 y comando de
  desinstalación con permisos ejecutables verificados.
- Suite Go de producción, `go vet`, formato y `git diff --check`: aprobados.
  Comprobaciones dirigidas finales de vigilancia, formulario y limpieza también aprobadas.
- Todos los archivos de distribución coinciden con `dist/installers/SHA256SUMS`.

Los informes de esta sección se añaden después de construir los paquetes; la copia de este
informe en `dist/installers/LEEME` contiene el cierre más reciente.
