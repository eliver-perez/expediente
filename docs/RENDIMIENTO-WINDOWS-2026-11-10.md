# AIDEX 1.0.0: rendimiento de la instalación Windows

Actualizado el 11 de noviembre de 2026. No se generó instalador.

## Prueba real

Las peticiones se autenticaron contra `http://127.0.0.1:18090/api/v1` en esta computadora. Se usó **la misma base de 14.5 GB y la misma biblioteca de 49,041 documentos** antes y después. El ejecutable original tenía SHA-256 `F13F6075…`; el instalado al terminar, `0E4867E1…`. El hash del ejecutable instalado coincide con el binario final compilado. El servicio quedó en `Running`. Los documentos físicos de origen no se modificaron.

La cuenta temporal de medición recibió permisos de administrador y gestor de biblioteca en la base de pruebas. El script la deshabilitó al terminar. Cada fase solicitó una nueva verificación de la misma raíz para medir durante el recorrido. En la última fase el contador del recorrido subió de 15,885 a 15,987 rutas entre las tres rondas. Las solicitudes de sesión permanecieron entre 9 y 28 ms.

| Ruta autenticada | Binario original, 3 rondas | Binario final, 3 rondas |
| --- | ---: | ---: |
| Dashboard | 3/3 timeouts a 50 s | primera 14.04 s; siguientes 18 y 15 ms |
| Tarjetas de bibliotecas del Dashboard | 11.90, 4.06, 4.43 s | primera 8.44 s; siguientes 15 y 18 ms |
| Procesamiento | 3/3 timeouts a 50 s | 1.17, 1.20, 1.20 s; HTTP 200 |
| Listado simple de bibliotecas | 245, 155, 24 ms | 21, 15, 19 ms |
| Carpetas en raíz | 7.60, 1.91, 2.14 s | 122, 113, 118 ms |
| Documentos en raíz, con total exacto | 6.22, 5.32, 2.69 s | 4.48, 4.41, 4.43 s |
| Documentos en raíz, vista de navegación | sin esta opción | 84, 98, 114 ms |
| Subcarpetas | 1.77, 1.87, 0.67 s | 1.24, 1.23, 1.18 s |
| Documentos en subcarpeta, con total exacto | 4.88, 5.72, 2.83 s | 1.32, 1.36, 1.41 s |
| Documentos en subcarpeta, vista de navegación | sin esta opción | 81, 81, 87 ms |

Los tiempos de Dashboard y sus tarjetas después de la primera carga usan instantáneas de hasta 30 s. El valor `generated_at` que ya muestra la interfaz indica cuándo se calcularon. La vista de navegación solicita `include_total=false`: conserva los mismos documentos y cursores, y devuelve `result_count=-1` en lugar de contar toda la biblioteca. La búsqueda general y `/explorer` sin esa opción mantienen el total exacto. El binario original ignoraba ese parámetro; por ello los tiempos de navegación rápida se muestran por separado de la comparación de la ruta exacta.

## Causas comprobadas

1. Dashboard repetía varias veces una CTE de 49,041 archivos. Con `files AS MATERIALIZED` el cálculo terminó, pero las trazas reales aún mostraron 4.7–12.2 s en el resumen durante carga y 0.8–3.7 s en el promedio de trabajos. Un índice que cubre las columnas de intentos redujo este último tramo. La primera consulta del Dashboard sigue siendo costosa; la instantánea mantiene disponibles las consultas posteriores mientras se recalcula en segundo plano.
2. Procesamiento calculaba agregados costosos antes de devolver el recorrido. Sus lecturas compartían cuatro conexiones con otras operaciones. Se separaron grupos de conexiones para control, avance y análisis; los conteos tienen presupuesto. En la prueba final `stats_available=false`, pero se devolvieron el recorrido y los trabajos activos en ~1.2 s. El avance siguió aumentando.
3. El proceso original mantenía unos 32,800 handles por vigilancia de carpetas NTFS. El watcher nuevo limita las carpetas vigiladas a 1,024 y utiliza la reconciliación periódica cuando llega a ese límite. También se evita registrar y escribir el mismo fallo por cada carpeta.
4. La consulta original de carpetas raíz unía todos los documentos antes de deduplicar nombres. En la base real, una consulta de nombres sobre el índice de rutas con validación de un documento vigente por carpeta devolvió exactamente los mismos nombres en **24 ms frente a 1,192 ms** en SQL directo. Instalada, la ruta respondió en 94–122 ms.
5. El plan real del explorador ordenaba los documentos en una tabla temporal y escaneaba `document_replacements` en una subconsulta por documento. Índices específicos hicieron que cargar la página bajara a 13–17 ms; el conteo exacto seguía costando varios segundos. La interfaz ahora pide solo la página y su cursor. Para subcarpetas, el conteo exacto comienza por el rango `(root_id, relative_path)` y bajó bajo esta carga a alrededor de 1.1–1.4 s.
6. El binario recompilado inicialmente no arrancó: las 21 migraciones instaladas tenían hashes LF y el checkout Windows incorporó los mismos SQL con CRLF. La normalización de saltos de línea hizo coincidir **21/21** huellas sin alterar el esquema. El contrato aceptado presentaba el mismo problema: su hash LF coincidía con la base, pero el archivo CRLF del checkout devolvía HTTP 403. También se normalizó esa huella. No se aceptó otro contrato ni se alteró la aceptación existente.
7. Los 49,040 documentos indexados de la base tenían `ocr_pages` en `extraction_runs.summary_json`; **0** valores faltantes y **0** discrepancias frente a las 1,196,346 filas de `extraction_pages`. Dashboard y Procesamiento usan ese resumen con respaldo para datos históricos que carezcan de él.

La reconciliación también dejó de actualizar todos los archivos físicos cuando solo cambiaban ubicaciones primarias retiradas. Se eliminó un segundo escaneo de estadísticas de expedientes. Estas mejoras se verificaron con pruebas de código y datos sintéticos; no se les atribuye una mejora aislada en las cifras HTTP reales de la tabla.

## Recursos y límites pendientes

- Binario original, ventana de 304 s: máximo de 32,786 handles, 1.76 GB residentes, 3.76 GB privados y 231 s de CPU acumulada (~0.76 núcleos medios).
- Binario final, ventana de 48 s durante el recorrido: máximo de 462 handles, 0.19 GB residentes, 0.23 GB privados y ~1.38 núcleos medios. En una ejecución corregida anterior, el mismo diseño de watcher alcanzó 930 handles y ~0.21 GB residentes con el servicio aún activo. Las ventanas y edades de proceso difieren; la comparación de RAM y CPU no demuestra por sí sola el consumo estable de una indexación completa. El límite de vigilancia sí evita repetir los ~32,800 handles anteriores.
- La primera carga sin instantánea del Dashboard todavía tardó 14.04 s con esta carga; las tarjetas, 8.44 s. Se entregan correctamente y luego responden en milisegundos. El conteo exacto de `/explorer` en raíz todavía tarda ~4.4 s; la vista normal no lo solicita. El listado de subcarpetas sigue alrededor de 1.2 s.
- Los conteos detallados de Procesamiento permanecieron temporalmente indisponibles en las tres rondas (`stats_available=false`). El avance de descubrimiento, los trabajos y el estado sí se consultaron; sigue pendiente acelerar esos agregados si se necesita que aparezcan durante toda la carga.
- El recorrido de 51,000 archivos no llegó a terminar durante esta ventana de medición. Debe vigilarse el consumo de memoria y handles hasta completar una carga entera.

## Validación y archivos

Compilaron el frontend con Node 22.15 y el servidor de producción con Go 1.27.1. Pasaron las pruebas dirigidas de Dashboard, búsqueda, carpetas, Procesamiento y compatibilidad de migraciones. La base instalada aplicó las migraciones 0022–0024 y el servicio reinició correctamente. No se generó instalador.

Los datos crudos están en `.tools/real-benchmark/measurements.json`, `resources-before.csv`, `resources-after.csv`, `metadata.json`, `startup.json` y las capturas de fases intermedias. El script reproducible es `scripts/benchmark-installed-windows.ps1`; `scripts/apply-performance-windows.ps1` sustituye el ejecutable desde PowerShell elevado. Las mediciones contienen identificadores técnicos y contadores, no credenciales ni texto de documentos.
