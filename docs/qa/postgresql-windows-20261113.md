# AIDEX 1.0.0: PostgreSQL nativo en Windows

Validación del 13 de noviembre de 2026 en `feature/postgresql`. No se generó instalador ni se modificó la instalación AIDEX existente, que quedó en `Running`. Los documentos físicos originales no se modificaron.

## Entorno y aprovisionamiento

- Windows 10.0.26200, Intel Core i7-1355U, 32 GB RAM y NTFS. Go 1.27.1, Node 22.15 y PostgreSQL 18.6 nativo de Windows.
- Se partió de `origin/feature/postgresql` en `81a508e`. Los ensayos usaron clústeres PostgreSQL temporales en rutas con espacios y esquemas aislados; `scripts/test_postgresql_windows.ps1` los inicia con SCRAM, checksums y loopback, y los retira al terminar.
- Un binario AIDEX de canal `production`, compilado sin instalador y con su runtime PostgreSQL 18 adyacente, ejecutó `init` y `migrate` sin PostgreSQL global ni herramientas en `PATH`. Se verificaron el rol `aidex_app` sin `SUPERUSER`, `CREATEDB` ni `CREATEROLE`, los hashes SCRAM de ambos roles, la base propiedad de `aidex_app`, el rechazo de una contraseña incorrecta, las ACL privadas y el rechazo de un puerto ocupado. El estado de prueba tenía espacios en su ruta.
- El motor privado superó arranque, parada, recuperación tras caída y reconexión con pools abiertos. Se corrigió un bloqueo Windows: `pg_ctl start` ya había terminado, pero `CombinedOutput` esperaba al proceso PostgreSQL que conservaba la tubería heredada. Ahora `pg_ctl` redirige la salida a `os.DevNull` y PostgreSQL escribe en su archivo de registro. Las dos pruebas de ciclo y recuperación volvieron a pasar (20.41 s y 17.85 s).
- Un servicio temporal `AIDEX-PG-TEST`, separado del servicio instalado, arrancó dos veces mediante SCM con cuenta LocalSystem, respondió HTTP 200 en `/health/live` en ambos ciclos, se detuvo y se eliminó. Sus procesos AIDEX usaron 103/91 MB residentes y 247/253 handles. No se verificó todavía la cuenta virtual definitiva ni el firewall del instalador.

## Funcionalidad y rendimiento

| Prueba en PostgreSQL real | Resultado Windows |
| --- | --- |
| Almacenamiento, migraciones y recuperación | Suite completa aprobada; ciclo privado y recuperación repetidos después del último ajuste |
| HTTP, autenticación, permisos, auditoría y rutas | Suite `internal/httpapi` completa con `development`, elevada: aprobada en 45.28 s |
| Bibliotecas, OCR, filtros, cursores y búsqueda | Cinco pruebas dirigidas aprobadas en 17.13 s; OCR y extracción también pasaron en la batería previa |
| Interfaz Edge sobre clúster PostgreSQL temporal | 35 pruebas aprobadas en 5.2 min; 2 omisiones explícitas descritas abajo |
| Dashboard autenticado, 10,000 documentos, mientras había otras cargas | Resumen 167.57 ms; tarjetas 35.79 ms; bibliotecas 13.44 ms |
| 20,000 documentos, trabajos y extracciones; 200,000 páginas indexadas | Inserción 28.49 s; eliminación 13.03 s; 493 búsquedas con escritura de auditoría en otra biblioteca durante la eliminación, latencia máxima 186.55 ms; índices y páginas retirados, original de control intacto |
| Subbiblioteca física `OBRAS/2024/FEDERALES` | 1,116 archivos, unos 9.3 GB; 1,116 documentos y ubicaciones registrados, 1,117 trabajos; plan 40.93 ms y escaneo 2 min 9.34 s |
| `internal/windowsfolders` elevada | Suite completa aprobada, incluida la protección efectiva contra apertura de eliminación y renombrado |
| Compilación y análisis | Frontend, `go vet`, Windows AMD64 y compilación cruzada sin CGO para macOS ARM64 y Linux AMD64/ARM64 aprobados |

La búsqueda de frase sobre el corpus de 200,000 páginas tenía una página de diez resultados en 153 ms, pero el **conteo exacto agotaba los 10 s** de `Search`. El plan aplicaba `aidex_search_query` durante el escaneo GIN de muchas páginas. Al convertirla en subconsulta escalar no correlacionada, PostgreSQL construye el `tsquery` una vez: el conteo exacto de 20,000 documentos respondió en **184 ms** y la página en 127 ms. El resultado y los filtros no cambiaron.

En Windows, `GetFinalPathNameByHandleW` con nombre DOS devolvía acceso denegado cuando los ancestros permitían solo recorrido. La variante NT sí obtenía la ruta; se añadió un mapeo de unidad obtenido antes de suplantar un token restringido. Pasaron las cuatro variantes de la prueba elevada para carpetas vinculadas y administradas. Otra prueba sobre NTFS comprobó que un handle abierto solo con `READ_CONTROL` **no impedía** obtener otro handle de eliminación ni renombrar la carpeta; añadir `FILE_TRAVERSE` al handle de fijación bloqueó ambas operaciones y la suite nativa pasó.

## Biblioteca física grande y límites

Se intentó un recorrido de la raíz real completa sin modificarla. Tras unos 26 minutos había 9,182 documentos registrados y el proceso había leído 93.5 GB. El inventario de `OBRAS` suma aproximadamente 407 GB; la ventana de 30 minutos no bastaba para hashear toda esa fuente. Se canceló ese clúster de prueba y se completó la subbiblioteca real indicada arriba. Las muestras históricas de 15 originales conservaron sus hashes SHA-256 después de los ensayos. Durante 27 s del recorrido grande el proceso Go alcanzó 89.3 MB residentes, 81.8 MB privados y 216 handles, con un promedio de 0.26 núcleos de CPU. Las sesiones PostgreSQL observadas alternaban consultas cortas de progreso y registro, sin espera de bloqueo persistente; el volumen leído explica la duración del recorrido.

La primera ejecución de navegador falló porque el arnés ubicaba las carpetas de prueba bajo el árbol del programa: AIDEX rechazó correctamente esas raíces. El arnés de navegador usa ahora el directorio temporal normal de Windows. Las pruebas se adaptaron a separadores NTFS y al visor PDF de Edge, que no expone `loadProgress_` de la extensión de Chrome; en Windows se verifica la navegación del iframe y la respuesta PDF autenticada. El flujo de privacidad exclusivo de macOS se omite en Windows. El flujo LAN se omite porque el servidor temporal no tiene la regla de firewall que prepara el instalador; la política LAN del producto no se cambió.

La ejecución completa previa de `internal/libraries` en Windows dejó cuatro fallos de fixtures que simulan permisos con `chmod` o enlaces simbólicos sin token elevado. Los subcasos de enlaces pasaron después con UAC y la nueva prueba de recorrido restringido pasó. Siguen pendientes los fixtures basados en `chmod`, que en NTFS no reproducen una denegación de ACL. No se declara aprobada la batería completa de bibliotecas por esa razón. Tampoco se completó el hash de la raíz física entera ni se validó una cuenta virtual de servicio o la exposición LAN con una regla real del instalador. macOS y Linux compilaron; no se ejecutaron nativamente después de estos cambios.

## Evidencia y archivos

Los registros crudos quedan en `.tools/postgresql-windows/`: `large-search-probes.log`, `large-search-once.log`, `real-linked-subtree.log`, `real-scan-resources.csv`, `browser-final.log`, `httpapi-full-elevated.log`, `windowsfolders-fixed.log`, `storage-regression.log`, `dashboard-10k.log`, `service-result.json` y los registros de compilación. Son artefactos locales ignorados por Git. Los clústeres temporales terminaron y se retiraron.

Cambios del producto: `internal/storage/postgres_service.go`, `internal/libraries/search.go`, `internal/libraries/folder_access_windows.go` e `internal/windowsfolders/acl_windows.go`. Pruebas y arnés: `internal/windowsfolders/sharing_windows_test.go`, `internal/libraries/postgresql_windows_real_test.go`, `scripts/test_postgresql_windows.ps1` y las pruebas de navegador de carpetas, PDF y red. No se cambió la versión 1.0.0, SQLite no se reintrodujo, y no se modificaron licenciamiento, servidor de licencias ni instaladores.

Commits de implementación: `8f4f86c` (arranque Windows del motor), `720bd1d` (arnés SCRAM y CLUF), `331840f` (búsqueda y NTFS) y `2b884a3` (pruebas reales y Edge). Este registro se publica en un commit documental separado sobre la misma rama.
