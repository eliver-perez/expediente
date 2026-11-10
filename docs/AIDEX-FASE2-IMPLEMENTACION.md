# AIDEX 1.0 — Fase 2: respaldo y restauración

## A. Arquitectura

La fuente de datos es SQLite en modo WAL y el esquema vigente tiene 15 migraciones. Los originales administrados residen en raíces físicas o en el almacenamiento privado de cargas retenidas; los vinculados permanecen en directorios externos. El formato `.aidexbak` versión 1 contiene ZIP con manifiesto, inventario de tablas, raíces, cantidades, tamaños y SHA-256. Se genera desde una transacción de lectura consistente. Para el respaldo completo, cada archivo se lee de forma confinada, se comprueba frente a la versión registrada y se rechaza si cambia durante la copia.

La restauración se inspecciona en una SQLite temporal, se programa y se aplica al siguiente inicio antes de los trabajadores. Primero prepara archivos en un directorio aislado, luego guarda un punto lógico de recuperación privado y reemplaza los datos permitidos dentro de una transacción. Las operaciones quedan registradas en un diario local para detectar reinicios y reintentar cuando requieren atención.

## B. Respaldo básico

Exporta por lista positiva usuarios y hashes de contraseña, roles, bibliotecas activas y archivadas, raíces y vistas, expedientes y catálogos, documentos y versiones, metadatos, extracción e índices, auditoría funcional, configuración y evidencia de la migración 0015. No incluye archivos físicos. Al restaurarlo, una ruta disponible recupera su identidad y cada archivo se marca disponible **solo** si su tamaño y SHA-256 coinciden con la versión respaldada. Si la ruta o el archivo faltan, se informan como inaccesibles o ausentes. Las cargas privadas que no están en el paquete pasan a estado fallido, sin fingir que pueden leerse.

## C. Respaldo completo

Añade documentos administrados publicados y cargas privadas en estado `staged` o `retained`, incluidas copias clasificadas que siguen siendo la única fuente física. No copia originales vinculados, previews, cachés ni transferencias incompletas. Una carga aún en recepción o una publicación física en curso impide crear un paquete potencialmente inconsistente. Cada copia se valida con SHA-256 y tamaño; el respaldo falla si un original cambia o falta.

## D. Restauración

La API exige `restore.manage`, sesión válida, CSRF en acciones mutables y licencia válida en el destino. La inspección verifica formato, versión, entradas duplicadas o maliciosas, límites de tamaño, hashes y referencias SQLite; la confirmación comprueba licencia, rutas, espacio y exclusión de directorios privados. SweetAlert2 muestra la sustitución del conjunto lógico. No hay fusión entre instalaciones.

Los documentos administrados del paquete completo se publican con nombres derivados de identificadores en un directorio nuevo. La escritura usa creación exclusiva y nunca sobreescribe un archivo existente; una colisión queda en atención y se puede reintentar. Para raíces vinculadas y respaldos básicos se puede indicar una ruta absoluta nueva por raíz; también se prueba la ruta original. El archivo solo se declara disponible tras una lectura confinada que rechaza enlaces y verifica su huella. Una ruta Windows o UNC en Linux requiere asignación explícita. Las bibliotecas archivadas mantienen ese estado.

El punto de recuperación es **lógico**, no una copia cruda de SQLite: usa la misma lista positiva y excluye licencia y secretos. Tras el cambio se invalidan todas las sesiones. Trabajos, recorridos y materializaciones inconclusos del origen quedan cancelados o fallidos para evitar que actúen sobre el destino. La evidencia de `entity_deletions`, `deletion_documents` y `root_retired_documents` se conserva; `deletion_cleanup` importado queda en atención con rutas e identidades físicas neutralizadas y su trabajador no lo ejecuta automáticamente.

## E. Seguridad

Las tablas `license_*`, la identidad privada de instalación, `config.json`, sesiones, tokens y archivos secretos no entran en los paquetes. Los eventos de auditoría `license.*` tampoco se exportan. La licencia y el KID propios del destino no se sustituyen. Un paquete cifrado usa Argon2id y XChaCha20-Poly1305 autenticado por bloques; la contraseña no se guarda en diario, manifiesto ni logs. Un respaldo sin contraseña sigue siendo confidencial y la interfaz lo advierte. Los SHA-256 prueban integridad de contenido, **no autenticidad de origen**.

El importador limita tamaño comprimido y expandido, cantidad de entradas y tamaño de filas; rechaza rutas absolutas, `..`, duplicados, enlaces, archivos especiales, esquemas incompatibles y componentes fuera de la lista positiva. Ningún elemento del paquete se ejecuta. Los archivos de respaldo y los puntos de recuperación viven bajo el estado privado, nunca en el directorio web.

## F. Interfaz

La sección **Respaldos y restauración** permite generar, consultar estado, descargar, inspeccionar, asignar rutas, programar y descartar. Muestra etapas reales sin porcentajes inventados, Toast para eventos y SweetAlert2 para confirmar la restauración. La pantalla separa generación y restauración según los permisos del usuario.

## G. Cambios técnicos

Se añadió `internal/backups` y sus pruebas; se conectó a `cmd/gestor-documental`, `internal/httpapi` y al servidor sintético E2E. El cliente React incorpora `web/src/Backups.tsx` y la ruta administrativa. Los endpoints son `GET/POST /api/v1/backups`, `GET /api/v1/backups/{id}`, descarga/cancelación, y `POST /api/v1/restores/inspect`, confirmación, reintento y descarte. Los permisos específicos se conceden al administrador de instalación. No se agregó una migración de Fase 2, dependencia, instalador ni incremento de versión.

## H. Pruebas realizadas

- `go test -tags development ./...`: pasó al permitir sockets locales para los tests ya existentes de red y licencias. Un primer intento en el sandbox falló exclusivamente por `bind: operation not permitted`.
- Pruebas dirigidas posteriores: básico y completo entre instalaciones sintéticas, licencia perpetua firmada y válida sin servidor, exclusión de activaciones, ruta original, mapeo a otra ruta, archivo alterado, biblioteca archivada, evidencia 0015, neutralización de limpieza, trabajos heredados, contraseña incorrecta, paquete truncado o manipulado, esquema incompatible, colisión y reintento, autorización y CSRF: pasaron.
- `npm --prefix web run build`: pasó. Playwright enfocado en respaldo: **1/1**; Dashboard y ciclo de vida: **3/3**.
- Compilación Go: macOS arm64, Windows amd64, Linux amd64 y Linux arm64: pasaron. No se generaron instaladores.
- El recorrido E2E completo se detuvo al límite de 180 s tras un caso largo preexistente de Fase 6. No se declara aprobada esa suite completa; los casos enfocados anteriores sí se ejecutaron y pasaron.

## I. Riesgos y pruebas pendientes

No hubo ejecución nativa de la restauración en Windows ni Ubuntu, ni simulación controlada de disco lleno o prueba de estrés con un corpus documental grande en esta fase. La validación final de rutas UNC, permisos de servicio, volumen alto y capacidad de disco debe hacerse en los sistemas destino antes de la distribución. Un corte durante la publicación puede dejar un archivo parcial **solo** en la nueva carpeta aislada; la operación queda en atención y no borra originales. En un equipo con poco espacio, también debe reservarse capacidad para el paquete importado, la validación temporal y el punto lógico de recuperación.

Para el Manual de usuario: explicar diferencias entre básico y completo, custodia de contraseña, asignación de rutas al cambiar de equipo, licencia previa a restaurar, reinicio requerido, revisión de operaciones en atención y retiro manual de archivos huérfanos únicamente dentro de la carpeta aislada de restauración tras verificar el diagnóstico.
