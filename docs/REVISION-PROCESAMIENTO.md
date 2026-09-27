# Revisión de procesamiento y seguimiento (2026-09-26)

## Diagnóstico previo a cambios

Se revisaron runtime.go, scanner.go, extraction.go, documents.go, las migraciones,
las consultas de auditoría, las pantallas React y el empaquetado Windows.
No se dispone de la base ni de los 9,000 originales del Windows del propietario.

- Un worker predeterminado comparte recorridos y extracción: un scan largo impide
  consumir los extract ya encolados (intentos 0). No es prueba de intentos agotados.
- Cada scan relee todos los PDF para SHA-256 (el proyecto no utiliza SHA-1).
- Hay un límite de una hora por trabajo. El checkpoint guarda directorios pendientes,
  pero dentro de un directorio grande se repiten archivos al interrumpirse.
- Los errores marcan el scan como fallido; el siguiente intento empieza desde cero.
  Carpetas protegidas del sistema no tienen exclusiones.
- El scheduler vuelve a crear scans periódicos; borrar jobs no cancela esa política.
- SQLite ya usa WAL, cuatro lectores y un escritor serializado con transacciones.
  No hay E/S de PDF dentro de ingest, pero una transacción por archivo y consultas
  repetidas añaden coste; conservar FULL y límites de concurrencia.
- Ya existen fencing, exclusión por archivo, idempotencia, checkpoints, historial y
  recuperación al reiniciar. Se reutilizan, al igual que autorización y visor PDF.
- UI: tareas sin nombres/progreso ni refresco; duplicados limitados a 100 hashes;
  auditoría de biblioteca sin filtros; configuración concentrada en un formulario.

## Alcance de la corrección

Separar un worker de reconciliación del número configurado de workers de contenido;
cachear huellas con identidad, tamaño, mtime precisa y verificación periódica;
persistir progreso y errores por ruta; omitir exclusivamente carpetas conocidas del
sistema; continuar ramas legibles sin confirmar ausencias en recorridos parciales.
La cancelación será explícita y persistente. Las interfaces reutilizarán React,
componentes y autorización existentes. Migración aditiva; no editar migraciones previas.

Los resultados y límites de la validación se detallan al final.

## Implementación entregada

- Un worker dedicado a `scan`/`verify_managed`, más 1–4 workers de contenido
  (el valor existente `indexing.workers`). La cola, fencing y transacciones originales
  continúan controlando exclusión por archivo y publicación atómica.
- No hay límite global de una hora para el descubrimiento. Los límites de PDF/OCR
  se conservan. Cancelación cooperativa consultada cada 500 ms; los procesos hijos
  y copias privadas reciben el contexto de cancelación.
- Caché ligada a raíz/ruta, identidad del archivo, tamaño y mtime de nanosegundos.
  Se comprueba estabilidad antes de reutilizarla. Eventos de escritura invalidan
  la entrada; la huella se recalcula como máximo cada 24 horas en el siguiente recorrido.
  No sustituye las comprobaciones de integridad de documentos administrados ni la
  verificación de la copia que utiliza la extracción.
- Checkpoint de directorios y observaciones idempotentes por archivo, persistidos.
  Un directorio interrumpido se vuelve a enumerar, pero reutiliza los hashes ya
  confirmados. Reiniciar no descarta la cola ni las versiones existentes.
- Exclusiones específicas, insensibles a mayúsculas: `$RECYCLE.BIN` y
  `System Volume Information`. No se omiten arbitrariamente archivos ocultos.
- Los fallos de subcarpetas/PDF se registran por ruta y permiten continuar las demás
  ramas. `SCAN_PARTIAL` deja la raíz accesible y el recorrido incompleto; no confirma
  ausencias. El reintento de errores recorre hasta 1,000 rutas afectadas por petición.
- Cancelar reconciliación pausa también la programación automática de esa raíz;
  Verificar/Reanudar la reactiva. Cancelar extracción no vuelve a encolarla durante
  el siguiente scan sin cambios. Se conserva el historial de intentos y cancelación.
- Procesamiento con sondeo sin solicitudes superpuestas cada 2.5 segundos: tareas
  activas, nombres/rutas, operación, páginas, tiempos, estadísticas y errores.
  Descubrimiento usa progreso indeterminado; extracción muestra páginas completadas
  sobre páginas conocidas. El tiempo transcurrido incluye la interrupción si la hubo.
- Duplicados en pestaña propia, resumen de grupos/archivos, paginación de grupos y
  miembros, bibliotecas/rutas y visor autorizado. Incluye copias de otras bibliotecas
  que la persona pueda leer. No fusiona ni elimina archivos.
- Auditoría global y por biblioteca: filtros de fechas, evento, usuario y texto,
  ordenación por fecha/actor/evento, paginación de backend y modal de detalle.
  Consultas exactas mantienen su autorización especial y su interfaz independiente.
- Configuración distribuida en General, Identificadores, Almacenamiento, Estructura
  y Revisión/conservación. Las opciones administradas se ocultan en modo vinculado.
- Icono del isotipo existente en lanzador, backend, instalador y desinstalador Windows.
  ICO: 16, 24, 32, 40, 48, 64, 128 y 256 px; receta en `packaging/windows/ICON.md`.
  Las reinstalaciones conservan la cuenta configurada del servicio.

## Definiciones de contadores

El resumen cuenta documentos no retirados y visibles, uno por archivo físico, nunca
por cada ubicación, versión histórica o intento. Vinculados + administrados = total.
Procesados = contenido indexado vigente; errores = última extracción de la versión
actual fallida; pendientes = resto (incluye pausas y cancelaciones pendientes de
reanudación). Procesados = solo texto directo + con OCR; un PDF mixto o vacío tras
OCR cuenta una sola vez en con OCR. Errores de recorrido tienen contador separado.
El recorrido cuenta rutas PDF observadas; varios enlaces físicos pueden representar
un único archivo en el resumen. Una huella común no reduce el número de copias físicas.

## Migración y actualización Windows

Migración aditiva `0006_processing_visibility`: caché, progreso, errores, controles
persistentes e índices. Las migraciones 0001–0005 y `LICENSE_CONTRACT.md` no cambian.
Al abrir una DB anterior, se crea primero una copia SQLite consistente con `VACUUM INTO`
en `<state_directory>/upgrade-backups/before-processing-<fecha>.db`, con acceso privado.
Incluye el WAL confirmado. No es un respaldo completo de originales/configuración ni
sustituye el sistema general de backup/restore pendiente de H7.

Instalador de esta entrega: `AIBID-Pruebas-0.7.0-test.1-r3-Windows-amd64.exe`.
Instalar encima de r2 sin desinstalar. No modificar `jobs` con el servicio activo;
la UI ofrece cancelación, reanudación y reintentos. Tras migrar, el binario r2 ya no
reconoce el esquema nuevo: no volver a instalarlo sobre esa base.
Esta revisión no reemplaza los paquetes macOS/Ubuntu ya entregados.

## API añadida / ampliada

- `GET /libraries/{id}/processing`: resumen, recorridos, trabajos activos y hasta
  100 errores recientes (permiso `indexing.run`). Los paths absolutos respetan
  `storage.view_paths`; los trabajos de temporales conservan su visibilidad privada.
- `POST /jobs/{id}/cancel`: cancelación de scan/extract y evento auditado.
- `POST /roots/{id}/retry-errors`: reintenta rutas afectadas (`indexing.retry`).
- `GET /libraries/{id}/duplicates`: `items`, `next_cursor`, `groups`, `files`.
- `GET /libraries/{id}/duplicates/{sha256}`: miembros paginados y autorizados.
- `GET /audit-options` y `GET /libraries/{id}/audit-options`: tipos/actores del ámbito.
- Auditoría admite `from`, `to` (límite superior exclusivo), `event_type`,
  `actor_user_id`, `q`, `sort=date|actor|event`, `direction=asc|desc`, `limit`, `cursor`.
  El cursor está ligado a usuario, ámbito, orden y filtros.

## Evidencia y límites

- Prueba optativa `AIBID_LARGE_TEST=1`: 9,000 archivos sintéticos pequeños con
  cabecera PDF, SQLite real, interrupción a los 211 archivos, cierre/reapertura,
  recuperación sin duplicados y `integrity_check=ok`. Primer recorrido con recuperación:
  15.67 s; siguiente: 4.46 s; 9,000 reutilizaciones de caché y cero hashes nuevos.
  No mide OCR masivo ni rendimiento de SMB o de los originales reales del propietario.
- Prueba con PDF nativo real: extracción/publicación mientras otro recorrido se
  mantiene detenido de forma controlada; no requiere terminar la reconciliación.
- Pruebas específicas: errores sin falsas desapariciones, reintento limitado,
  exclusiones, cancelación persistente, permisos de duplicados entre bibliotecas,
  filtros/orden/cursor de auditoría y snapshot previo a migración.
- Sin ejecución nativa del nuevo paquete en Windows/SMB ni Ubuntu en este equipo.
  La compilación cruzada y recursos PE no certifican el comportamiento del instalador,
  ACL, dominio ni rendimiento de la red real.
- Una edición que conserve tamaño y mtime y cuyo evento nativo se pierda puede
  detectarse al siguiente rehash periódico (24 h), no necesariamente de inmediato.
  Cancelar espera a que termine una llamada de E/S que el sistema operativo no pueda
  interrumpir. La recuperación mantiene el límite existente de cinco intentos.

Comprobaciones ejecutadas en esta entrega:

- `go test ./...`: correcto (compilación normal).
- `go test -tags development ./...`: correcto. El simulador HTTPS requirió permiso
  de escucha local; la primera ejecución restringida falló únicamente al abrir su puerto.
- `go test -race -tags development ./internal/libraries ./internal/httpapi ./internal/storage ./internal/extraction`:
  correcto; pruebas adicionales de cancelación, duplicados y migración también con race.
- `AIBID_LARGE_TEST=1 go test -tags development ./internal/libraries -run '^TestLargeLibrary9000$' -v`: correcto.
- `npm --prefix web run build`: correcto.
- Playwright/Chrome: ocho escenarios completos correctos; el nuevo escenario también
  pasó después del ajuste final de legibilidad y de los nombres en auditoría.
- `go vet ./...`, formato Go y 14 comprobaciones de diseño: correctos.
- Tests de bibliotecas compilados para Windows amd64; no ejecutados en Windows.
- Instalador NSIS compilado; verificaciones de arquitectura, dependencias DLL, los
  ocho tamaños del icono en ambos ejecutables y subsistema GUI del lanzador correctas.
