# AIDEX 1.0.0 — PostgreSQL en macOS, 2026-11-13

## Entorno y alcance

- macOS ARM64; Go 1.27.1; PostgreSQL 18.6 (Homebrew) como herramienta de desarrollo.
- Rama `feature/postgresql`, basada en `5e3c44f`, con los cambios de rendimiento
  Windows `220c711` y eliminación masiva `5e3c44f` conservados.
- PostgreSQL real en clústeres temporales propios. Las fixtures de servicios usan
  esquemas aislados; el aprovisionamiento usa un clúster independiente con SCRAM
  y rol `aidex_app` sin privilegios administrativos.
- Versión 1.0.0. No se modificó el servidor de licencias ni la interfaz del producto.
  No se generaron ni modificaron instaladores; no se hizo merge a `main`.

## Pruebas de servicios e integridad

| Verificación | Resultado |
| --- | --- |
| Producción, `go test -race ./...` | Aprobada: 199 casos/subcasos; 4 omisiones explícitas de fixtures opcionales |
| Almacenamiento, detector de carreras | Aprobada: lectores protegidos, escritores simultáneos, aislamiento, rollback, FK diferidas, checksum y bloqueo |
| Motor privado | Aprobada: inicialización SCRAM/checksums, rol sin SUPERUSER/CREATEDB/CREATEROLE, arranque/parada y recuperación tras parada abrupta |
| Recuperación sin cerrar pools | Aprobada: supervisor reinicia el motor, reconecta pools, conserva commits y revierte la transacción interrumpida |
| Diagnóstico conectado al motor activo | Aprobada: el cliente no migra ni detiene el motor al cerrar |
| Respaldos básicos y completos, detector de carreras | Aprobada: inspección aislada, restauración, colisiones, conservación de originales y licencia de destino |
| Páginas indexadas en respaldos | Aprobada: exportación sin columnas generadas, reconstrucción GIN, secuencias IDENTITY y nuevas inserciones después de restaurar |
| CLUF y trazabilidad | Aprobada: aceptación local conservada, auditoría inmutable y correcciones/sustituciones incluidas en la lista portable |
| Identidad, HTTP y mantenimiento, detector de carreras | Aprobada: sesión única concurrente, rollback ante fallo de auditoría, autorización, limpieza y protección de originales |
| Bibliotecas, detector de carreras | Aprobada: 143 casos/subcasos; 4 cargas opcionales omitidas aquí y ejecutadas por separado; duración 780,73 s |
| Búsqueda | Aprobada: términos, frases, acentos, palabras comunes, filtros, cursores, permisos, agrupación, formatos y fragmentos; cirílico/árabe/japonés y puntuación |

Las omisiones de las suites normales corresponden a corpus externos u operaciones
de escala que requieren variables explícitas; las cargas descritas debajo sí se
activaron y ejecutaron. Una ejecución inicial de la suite de bibliotecas con
`-race` alcanzó su plazo de 7 minutos; se repitió separadamente con `-timeout 20m`.
La compilación y las pruebas SQL aisladas no se usan como sustituto de la batería
funcional sobre PostgreSQL.

## Cargas y concurrencia

| Corpus | Resultado medido |
| --- | --- |
| Navegación de 50 000 archivos | Inserción 21,99 s; primera página de carpetas 37,43 ms |
| 100 000 documentos en tres bibliotecas | Aprobada; Dashboard 616 ms, tarjetas 161 ms, procesamiento 184 ms, estadísticas 71 ms y búsqueda por nombre 50 ms |
| Eliminación de biblioteca con 80 000 documentos | 11,74 s; 115 lecturas concurrentes, máxima latencia 9,34 ms; original de control intacto |
| 20 000 documentos, trabajos y extracciones; 200 000 páginas indexadas | Inserción 14,00 s; búsqueda de frase con 20 000 resultados correcta |
| Eliminación del corpus de 200 000 páginas | 9,38 s; 391 búsquedas concurrentes con escritura de auditoría en otra biblioteca, máxima latencia 135,63 ms; original de control intacto |
| 9 000 archivos físicos de prueba | Interrupción/reapertura/reanudación 26,87 s; escaneo incremental 12,94 s; 9 000 tareas únicas; originales conservados |

Las cifras son mediciones locales bajo carga de pruebas; no comparan Windows con
macOS ni garantizan un SLA. El corpus de 9 000 archivos contiene cabeceras PDF
pequeñas para carga del escáner y no ejecuta OCR. Las pruebas funcionales separadas
sí procesan documentos PDF nativos/escaneados, Word, Excel, TXT y CSV.

## Interfaz y compilación

- TypeScript y Vite: aprobados. No se modificó `web/src` ni el diseño de la interfaz.
- Navegador: **37/37** flujos aprobados en una ejecución conjunta (3,4 minutos).
  Se corrigieron fixtures que consultaban antes de confirmar una escritura y
  selectores que dependían de que otras pruebas no hubieran creado datos.
- Motor aprovisionado por AIDEX: **5/5** flujos adicionales de navegador aprobados
  (47,5 s), sin `AIDEX_TEST_POSTGRES_DSN`: respaldos, Dashboard, extractores,
  biblioteca vinculada/OCR y publicación/revisión con otra cuenta. Se utilizó el
  rol técnico real de la aplicación, sin acceso de superusuario.
- Compilación sin CGO: macOS ARM64, Windows AMD64, Linux AMD64 y Linux ARM64 aprobadas.
- `go vet` de producción/desarrollo, formato Go y comprobación de diferencias: aprobados.
- `scripts/test_postgresql.sh`: probado con creación y retirada de su clúster.

## Límites de esta evidencia

Las pruebas nativas se realizaron en macOS. La compilación cruzada no acredita
servicios, permisos ni runtimes nativos de Windows o Ubuntu AMD64/ARM64. Esas
verificaciones y el empaquetado por plataforma quedan para las etapas autorizadas
correspondientes. El workflow de pruebas fue adaptado para PostgreSQL 18; no se
presenta su ejecución remota como evidencia local.

Los binarios de Homebrew se usaron únicamente para desarrollo. El futuro paquete
debe incluir un runtime PostgreSQL completo y redistribuible (ejecutables,
bibliotecas y datos compartidos). El contrato está en
[POSTGRESQL.md](../POSTGRESQL.md); no se copió un runtime parcial a los instaladores.


## Commits

- `485e453`: preparación e inventario de la migración.
- `a2ce193`: persistencia PostgreSQL, esquema, búsqueda, respaldos, mantenimiento,
  motor privado y pruebas de integración/escala.
- `78488d5`: arnés reproducible, diagnóstico conectado al motor activo, CI,
  documentación y sincronización/aislamiento de fixtures del navegador.
- El commit que incorpora este registro cierra la evidencia de macOS y retira
  las últimas reglas de limpieza exclusivas del motor anterior.

Los logs brutos de las pruebas se produjeron en `/private/tmp/aidex-postgresql-*.log`
y `.jsonl`; los clústeres y datos sintéticos se retiran tras terminar la sesión.
Las capturas históricas que las pruebas sobrescribieron se restauran: no forman
parte de este cambio ni representan un rediseño.
