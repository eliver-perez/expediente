# Estadísticas de Procesamiento — AIDEX 1.0.0

Fecha: 2026-11-16. Rama: `feature/postgresql`. Motor: PostgreSQL 18.

## Causa

El informe combinado calculaba los conteos antes del progreso, usando el pool
general y un plazo de 750 ms. Una espera de conexión o un conteo que superara
ese plazo devolvía `stats_available=false`. El frontend interpretaba esa bandera
como una orden de desmontar los indicadores; tampoco conservaba un snapshot
al salir de la sección. Cada nueva petición podía volver a sufrir la misma
cancelación, sin completar un conteo. La consulta repetía expresiones de estado
del último trabajo y realizaba búsquedas históricas incluso para contenido vigente.

## Corrección

- Consultas independientes: actividad cada 2,5 s después de terminar su petición;
  estadísticas cada 10 s después de terminar la suya, sin peticiones superpuestas.
- Conteo en `AnalyticsReader`, acotado a 5 s. El cliente también cancela su espera
  a los 8 s. El progreso conserva su ruta y el pool de estado independientes.
- Agregado PostgreSQL con `FILTER`, estados materializados una sola vez y
  consultas de trabajos/páginas condicionadas a la frescura del contenido.
  Conserva la definición de cada indicador y los filtros de autorización.
- Respuesta de estadísticas con fecha del servidor; ante timeout omite conteos
  y fecha, e informa el motivo. La ruta combinada sigue disponible por compatibilidad.
- Snapshot completo en memoria, limitado a 32 bibliotecas y separado por sesión.
  Se recupera al regresar a Procesamiento. Una respuesta parcial, pendiente o un
  error temporal conserva los valores y su fecha. Sin snapshot se muestra carga
  o el motivo de indisponibilidad; los ceros solo proceden de respuestas completas.
- React conserva los nodos de los indicadores y cambia sus textos. El bloque
  memorizado tampoco se renderiza de nuevo si los valores no cambiaron.
- Aviso discreto al actualizar, fecha de última obtención y aviso explícito de
  datos desactualizados ante un fallo o más de 60 s sin snapshot reciente.

## Validación dirigida

Todas las comprobaciones siguientes terminaron correctamente:

- `npm --prefix web run build`: TypeScript y compilación web.
- `TestProcessingStatisticsSnapshots`: biblioteca vacía, conteos completos,
  nuevas cifras, autorización y `EXPLAIN (ANALYZE, BUFFERS)` real. El plan del
  fixture utiliza `jobs_file_version` y ejecuta una búsqueda por archivo pendiente;
  tiempo de ejecución observado: 0,340 ms para dos archivos (no es un benchmark
  de bibliotecas grandes).
- `TestProcessingStatisticsTimeoutDoesNotBlockActivity`: pool de analítica
  agotado, respuesta sin ceros ni conteos parciales al vencer el plazo,
  recuperación y consulta independiente de recorridos/trabajos activos.
- `TestProcessingStatisticsContentClassifications`: pendientes, conservados,
  texto directo, OCR por resumen y por páginas, y exclusión de contenido antiguo.
  Usa TXT y metadatos de prueba; no ejecuta un corpus OCR.
- `TestProcessingStatisticsPostgresWaitBudget`: bloqueo real en PostgreSQL;
  una espera de 1,1 s completa el conteo, una espera prolongada vence a los 5 s,
  la actividad responde durante la espera y el conteo se recupera al liberar el bloqueo.
- Regresiones existentes `TestScanCachePartialErrorsAndTargetedRetry` y
  `TestProgressAndDashboardWithGeneralReadersExhausted` (la comprobación de
  progreso ahora invoca la vista de actividad).
- `processing-statistics.spec.ts`, Playwright con Edge y servidor PostgreSQL:
  endpoints reales, carga inicial, primer timeout sin snapshot, respuestas
  incompletas, `stats_available=false` con ceros, HTTP 503, fallo de red, timeout
  del cliente, recuperación, navegación de ida/vuelta, separación entre bibliotecas,
  actualización del recorrido y las unidades de trabajos activos. Comprueba la
  identidad de los nodos DOM y la fecha retenida, además de los nueve valores.
- `git diff --check`.

Sin cambios de versión, Dashboard, mecanismos de indexación, OCR o reconciliación.
No se generaron instaladores ni se repitió la batería masiva.
