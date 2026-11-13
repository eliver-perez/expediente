# AIDEX 1.0.0: eliminación masiva en Windows

Investigación del 12 de noviembre de 2026. Instalación de prueba con una biblioteca de 49,041 documentos y una base SQLite de 14.5 GB. Los originales físicos no se han modificado. No se generó instalador.

## Causa reproducida en la instalación

La operación de biblioteca `4fc85534-ddc6-4321-aa2c-fbb238e3ebec` falló dos veces con `PROCESS_TIMEOUT`. El trabajador tenía un plazo global fijo de 10 minutos y ejecutaba todos los borrados en una sola transacción. Las trazas del servicio muestran que `job_progress` terminó en 0.7–3.7 s y `job_attempts` en 1.4–6.6 s; el siguiente paso, `DELETE FROM jobs`, no terminó antes del plazo. Faltaba un índice sobre la clave foránea autorreferente `jobs.retry_of_job_id`, que SQLite debe comprobar por cada tarea retirada.

También faltaban índices para comprobar referencias a extracciones, páginas y ubicaciones durante el resto de la eliminación. La preparación del plan real requirió 16.5 s de huella y 15.6 s de manifiesto: superaba el plazo HTTP de escritura de 30 s y cerraba la conexión antes de mostrar la confirmación. El plazo se amplió solo en las rutas de plan y confirmación; la ejecución confirmada ya ocurre en segundo plano.

## Cambios

- Migración 0025: índices de claves foráneas para tareas reintentadas, páginas por extracción, extracción vigente y observaciones por ubicación.
- El trabajador de eliminación ya no impone un plazo global de 10 minutos. Si se interrumpe el servicio, SQLite revierte la transacción y el mismo ID vuelve a la cola; el proceso no deja registros parcialmente retirados.
- Estado de ejecución consultable por API con fase, filas procesadas, total y unidad. La API y el trabajador comparten el avance aun cuando usan instancias distintas del servicio. Los contadores provienen de filas realmente retiradas. La interfaz muestra actividad sin porcentaje cuando la fase no permite calcularlo, etapas, resultado, manifiesto y reintento. El seguimiento permanece en **Eliminaciones y limpieza** al navegar o cerrar la confirmación.
- `DELETE FROM jobs` completó sobre la instalación real: 49,047 tareas en unos 12 s. Se observaron 49,041 documentos retirados en la transacción, con consultas de sesión de 15–41 ms mientras trabajaba.
- La eliminación individual de 2.39 millones de páginas todavía resultó demasiado lenta. Se añadió una ruta para retiradas grandes que reconstruye el índice FTS desde las páginas restantes cuando la operación elimina al menos 100,000 páginas y la mitad del total. SQLite documenta que `rebuild` recrea un índice FTS5 de contenido externo desde su tabla de contenido: [documentación oficial](https://www.sqlite.org/fts5.html#the_rebuild_command). El cambio de disparador y la reconstrucción ocurren dentro de la misma transacción. Las páginas de extracción se retiran por lotes de 512 extracciones, lo que permite actualizar el avance entre lotes.
- Al terminar correctamente se limpia el código de error de una interrupción anterior. La API también omite diagnósticos históricos en operaciones completadas.
- Los planes caducados muestran una explicación y requieren crear un plan nuevo; no se ofrece repetir indefinidamente el mismo plan desactualizado.

## Pruebas realizadas

| Prueba | Resultado |
| --- | --- |
| Reintento del plan antiguo de 49,041 documentos | `DELETION_PLAN_STALE` en 23 s; sin cambios en los registros; 15 originales conservan su SHA-256 |
| Plan nuevo sobre la base real | 49,041 documentos; 16.5 s de huella + 15.6 s de manifiesto; confirmación y ejecución en segundo plano |
| Eliminación de tareas en la base real | 49,047 tareas retiradas en ~12 s, frente a 2 timeouts previos de 10 minutos |
| Avance y lecturas simultáneas | Fases y cantidades reales visibles; 218 respuestas autenticadas de estado durante la fase activa en 16.6–385 ms; 22 consultas de sesión en 12.1–81.6 ms |
| Reinicio durante eliminación | Estado `running` → `pending` → `running` con el mismo ID; la transacción se revirtió |
| Carpeta grande en prueba de código Windows | 10,000 documentos, 1,000 tareas y 2,000 páginas extraídas e indexadas; terminó en 6.75 s; FTS y archivo original de control intactos |
| FTS con otra biblioteca presente | Prueba de reconstrucción e integridad FTS5 correcta; disparador posterior restaurado |
| Compatibilidad | Frontend compilado; servidor Go compilado para Windows, macOS y Linux; pruebas dirigidas de eliminación y rollback aprobadas |

## Resultado final en la biblioteca real

La operación nueva `37421a3f-89c8-4986-89bb-eb6a12e036af` terminó como `complete`. Tras reinicios controlados que comprobaron la recuperación, el último recorrido continuo tardó **487.9 s (8 min 8 s)** desde la primera muestra hasta `cleanup_pending`; la limpieza llegó después a `complete`. Frente a ello, el binario original falló **dos veces** por `PROCESS_TIMEOUT` al llegar a 10 minutos, sin retirar registros.

La traza autenticada siguió los estados `queued`, `validating`, `preparing_manifest`, `stopping_jobs`, `removing_documents`, `removing_index`, `removing_extraction`, `rebuilding_search`, `finalizing` y `cleanup`. Se observaron 49,041 de 49,041 documentos y el contador de páginas creció hasta 2,383,882 de 2,392,692 antes de cambiar de etapa. El muestreo cada dos segundos puede no capturar el último lote; no se infiere un porcentaje para las etapas sin total. Durante el recorrido, el máximo observado fue **315 handles, 185.7 MB de memoria residente y 220.3 MB privada**. Estas cifras corresponden al proceso durante esta ejecución, no a una carga simultánea de OCR sostenido.

La verificación elevada de SQLite tras finalizar encontró **cero** bibliotecas, raíces, documentos, archivos físicos registrados, tareas, expedientes, versiones, extracciones, páginas indexadas y páginas de extracción de esa biblioteca. Permanecieron las **49,041 entradas del manifiesto**. `PRAGMA foreign_key_check` devolvió cero incidencias; la prueba de integridad FTS5 pasó y sus filas coinciden con las páginas indexadas restantes (cero en esta base de una biblioteca). Los **15 archivos originales** muestreados existen y conservan su SHA-256. El servicio quedó en `Running` y `/health/live` devolvió HTTP 200.

Se instaló la última compilación de producción con SHA-256 `6F64AD587E3863932236EFE88A9BEFE953606728A1BC0ED34663C94B2F65D0D7`, idéntico en `build/aidex-server.exe` y `C:\Program Files\AIDEX\aidex-server.exe`. Con ese binario, la consulta autenticada final `GET /api/v1/deletions/37421a3f-89c8-4986-89bb-eb6a12e036af` respondió **HTTP 200 en 40.4 ms**, con `state=complete`, cero limpiezas pendientes y ningún diagnóstico. El registro anterior conservaba internamente un código `INTERRUPTED` de un reinicio previo; ya no se expone al usuario tras completar y la nueva ruta de éxito lo limpia en futuras operaciones.

Los datos crudos están en `.tools/real-benchmark/deletion-progress.jsonl`, `deletion-result.json`, `deletion-verified.json` y `deletion-final-api.json`. Los scripts reproducibles son `scripts/benchmark-deletion-windows.ps1`, `verify-deletion-windows.ps1`, `verify-deletion-api-windows.ps1` e `install-deletion-windows.ps1`. Las cuentas temporales de medición se deshabilitaron. No se generó instalador.

## Límite de la comprobación

La biblioteca real de 49,041 documentos y 2.39 millones de páginas terminó correctamente. Una carpeta grande se comprobó en Windows con 10,000 documentos y 2,000 páginas en una base temporal; no se midió una carpeta real de 49,000 documentos porque la única biblioteca de esta instalación se retiró en la prueba principal. La operación conserva una sola transacción SQLite durante el borrado, por lo que otras escrituras esperan hasta su finalización; las lecturas de estado y sesión siguieron disponibles durante la prueba. Una biblioteca con cientos de miles de documentos aún no se ha medido en esta instalación.

## Archivos principales

- `internal/libraries/deletion.go`, `deletion_progress.go` y `runtime.go`: ejecución, índices FTS, avance, recuperación y exclusión de tareas relacionadas.
- `internal/httpapi/lifecycle.go`: plazo de preparación y confirmación del plan.
- `db/migrations/0025_bulk_deletion_fk_indexes.*.sql`: índices de claves foráneas.
- `web/src/Lifecycle.tsx`, `DeletionTracker.tsx` y `styles.css`: seguimiento entre pantallas y diagnóstico.
- `internal/libraries/bulk_deletion_test.go` y `deletion_progress_test.go`: carpeta grande, original intacto, FTS y progreso compartido.
- `scripts/benchmark-deletion-windows.ps1`, `inspect-deletion-windows.ps1`, `inspect-startup-deletion-windows.ps1`, `verify-deletion-windows.ps1`, `verify-deletion-api-windows.ps1` e `install-deletion-windows.ps1`: medición, integridad y aplicación local en Windows.
