# AIBID 2.0 — fase 7: panel general y observabilidad

Entrega de pruebas **2.0.0-alpha.7-f7**, 6 de octubre de 2026. Continúa sobre
la fase 6 e incluye la corrección del instalador Windows f6r1. La fase 8 no se inicia.

## 1. Implementación

**Panel general** reúne incorporaciones de documentos, estado del índice, OCR,
origen, formatos, estado documental y de procesamiento, duplicados, colas, caché,
mediciones de duración y errores recientes. Utiliza indicadores y tablas, sin
gráficas que dupliquen la información. Los indicadores se consultan cada 15 segundos;
la tabla de bibliotecas tiene paginación y actualización manual.

**Sistema · Errores** muestra incidencias internas separadas de Auditoría. Permite
filtrar fecha de última aparición, módulo, severidad y estado, consultar contexto,
marcar revisado o pendiente y copiar información para soporte. La política de
retención se administra en esta misma sección. Ambas pantallas requieren el permiso
`system.configure`, también comprobado en cada endpoint.

## 2. Archivos principales

- Registro interno: `internal/diagnostics/diagnostics.go` y `messages.go`.
- Consultas y administración: `internal/libraries/dashboard.go`, `system_errors.go`.
- Captura: `runtime.go`, `scan_cache.go`, `watcher.go`, `preview_worker.go` y
  `internal/httpapi/server.go`; rutas en `internal/httpapi/observability.go`.
- Interfaz: `Dashboard.tsx`, `SystemErrors.tsx`, App, AuditTable y estilos compartidos.
- Pruebas: `observability_test.go` en libraries/httpapi, migración desde fase 6,
  `web/tests/observability.spec.ts`, verificación de paquetes y scripts Windows.
- Distribución: versión, constructor, preinstalación, smoke macOS y CI.

## 3. Migración y actualización

**0014_observability** añade `diagnostic_policy`, `diagnostic_events` e índices.
Es aditiva: conserva documentos, texto, usuarios, bibliotecas, licencias y Auditoría.
El mecanismo existente crea una instantánea SQLite consistente antes de actualizar
el esquema, en `upgrade-backups`. Los paquetes aceptan actualizaciones desde alpha.6.
No se debe ejecutar un binario anterior sobre el esquema nuevo; una reversión
requiere el respaldo compatible correspondiente.

El log comienza a recoger incidencias desde la actualización; no reconstruye
historiales anteriores. Los errores previos siguen en los trabajos y en Auditoría.

## 4. Parámetros y definición de indicadores

- Hoy, semana (desde el lunes) y mes: fecha original de incorporación, según la zona
  horaria del servidor indicada en pantalla. Reprocesar no crea una incorporación.
- Documentos: bibliotecas activas, documentos no retirados, incluidos borradores
  privados. Cada documento cuenta una vez aunque tenga varias ubicaciones.
  Son totales administrativos; no conceden acceso a los originales ni a su texto.
- Origen: vinculado o administrado por documento. Una biblioteca híbrida puede
  contribuir a ambos grupos. La tabla por biblioteca incluye bibliotecas vacías.
- Índice vigente: texto publicado para la versión actual. OCR: ese índice contiene
  al menos una unidad que pasó por OCR, incluso si el OCR devolvió texto vacío.
- Estado: deriva del último trabajo, reglas y publicación existentes. Un intento
  fallido de reindexar puede coexistir con un índice anterior vigente.
- Duplicados: SHA-256 de la versión actual; muestra grupos, documentos de esos
  grupos y copias adicionales. Conserva ubicaciones y registros independientes.
- Cola: trabajos pendientes, esperando reintento, pausados o ejecutándose y trabajos
  de vistas previas. Una regla o licencia puede mantener un trabajo en espera.
- Caché: bytes contabilizados por el servicio existente, presupuesto y vistas listas.
- Promedio: últimos intentos finales exitosos de extracción terminados en las
  últimas 24 horas, con inicio/fin válidos y generación de trabajo coincidente.
  Incluye lectura, espera interna de OCR y publicación; excluye espera inicial de
  cola e intentos fallidos. Si no hay muestras se indica sin inventar una duración.
- Retención inicial: **30 días y 5000 registros**. Rango: 1–365 días y 100–50000.
  La antigüedad se mide desde la última aparición. Se aplica al escribir, en la
  revisión periódica del servicio y al guardar una política nueva.

## 5. Componentes reutilizados

Se reutilizan `Accordion`, tablas, badges, métricas, `CursorControls`, `useCursorPage`,
`usePolling`, `Notice`, fechas, tamaños y nombres de estados. Se conservan sesiones,
CSRF, permisos, escrituras transaccionales, revisión optimista, colas y la Auditoría.
No se crea un segundo estado documental ni un sistema alternativo de procesamiento.

## 6. Componentes nuevos

El registro operacional tiene una tabla independiente, códigos de mensaje cerrados
y contexto estructurado acotado. Panel general resume datos existentes; Sistema ·
Errores administra exclusivamente ese registro. Las consultas usan páginas de
25 bibliotecas o 50 diagnósticos y cursores ligados al usuario y filtros.

## 7. Decisiones de funcionamiento y privacidad

- Se capturan fallos y reintentos de trabajos, lectura de carpetas, degradación del
  watcher/pérdida de eventos, generación de vistas previas, fallos HTTP 5xx,
  recuperación de trabajos tras reinicio e inicio del servicio.
- Las severidades corresponden a códigos conocidos: información, advertencia,
  error y crítico. No se guardan textos arbitrarios de excepciones.
- El contexto admite únicamente UUID de biblioteca/raíz/trabajo/documento/petición,
  operación conocida, intento y estado HTTP. Se rechazan valores arbitrarios tanto
  al registrar como al devolver datos. Mensajes desconocidos pasan a INTERNAL_ERROR.
- El diagnóstico copiable añade versión con revisión, SO y arquitectura del servidor.
  No contiene rutas, nombres de archivos, consultas, contenido, contraseñas, tokens
  ni datos de autenticación. Si el navegador no permite portapapeles (por ejemplo
  acceso HTTP por LAN), se ofrece un campo seleccionable para copiar manualmente.
- Repeticiones del mismo problema y destino se agrupan, conservando primera/última
  fecha y contador. Petición e intento describen la última aparición.
- Marcar revisado no resuelve ni reintenta el problema. Una aparición nueva vuelve
  a dejar el diagnóstico pendiente. Los cambios usan revisión optimista y Auditoría.
- Limpiar diagnósticos nunca elimina Auditoría. Un límite reducido elimina también
  registros pendientes antiguos; la pantalla explica el efecto antes de guardar.
- La retención es un límite lógico de filas y antigüedad; SQLite puede conservar
  páginas libres reutilizables y el WAL según su mecanismo habitual.

## 8. Pruebas

La evidencia ejecutada se registra en `dist/aibid-2.0-fase-7/VALIDACION-FASE-7.json`.
Las pruebas cubren límites de día/semana/mes, agregados, duplicados, retiro del índice,
muestras de duración, estado vacío, paginación, permisos, CSRF, datos sensibles,
revisión obsoleta, reapertura, retención, persistencia y ocurrencias concurrentes.

Chrome provoca un error real de extracción con un PDF inválido, comprueba el panel,
filtra y revisa el diagnóstico, obtiene la información de soporte, cambia la retención,
recarga y verifica su persistencia. Se comprueban pantallas de 1800 y 390 píxeles,
además de los flujos existentes. La actualización del paquete macOS usa datos
provisionales de fase 6 y comprueba respaldo, integridad y versión.

## 9. Resultados y distribución

Paquetes en `dist/aibid-2.0-fase-7/`: Windows amd64, macOS Apple Silicon y Ubuntu
24.04 amd64/arm64. `SHA256SUMS` identifica los artefactos de esta entrega.
El script Windows generado debe superar el analizador PowerShell, la comprobación
de un único BOM y de su bloque de parámetros antes de construir el instalador.
La validación detallada identifica qué pruebas se ejecutaron y sus límites.

## 10. Riesgos y límites

Los paquetes son de pruebas, sin firma comercial. Windows y Ubuntu se compilan y
se inspeccionan desde esta Mac; la instalación/servicio y los permisos reales SMB
requieren verificación en los equipos destino. No se instala ni reinicia el servicio
real de esta Mac. Se preservan las carpetas y documentos proporcionados como muestras.

El log no sustituye los registros del sistema operativo: un fallo antes de abrir la
base de datos no puede almacenarse en ella. Los fallos de escritura del propio log
no impiden responder al cliente ni se registran recursivamente. Los códigos seguros
aportan diagnóstico, sin trazas crudas potencialmente sensibles. No se importa el
historial de errores del servidor comercial de licencias; se mantienen sus mecanismos
existentes. Los indicadores son una instantánea consultada, no telemetría por segundo.

## 11. Prueba manual sugerida

1. Actualiza conservando los datos y comprueba `2.0.0-alpha.7-f7`, las bibliotecas,
   los usuarios y la instantánea previa de `upgrade-backups`.
2. Abre Panel general; carga documentos de prueba y compara fechas, formatos,
   origen y progreso. Comprueba una biblioteca vinculada y otra administrada.
3. Procesa un PDF escaneado y dos copias idénticas; verifica OCR y duplicados.
4. En una biblioteca de prueba procesa un archivo corrupto: debe aparecer el
   diagnóstico en Sistema · Errores y entre los errores recientes del panel.
5. Filtra, copia el diagnóstico, márcalo revisado y reintenta el trabajo. Una nueva
   aparición debe volver a pendiente. Verifica que el texto copiado no incluya rutas.
6. Cambia la retención, recarga y confirma persistencia; revisa que la acción permanezca
   en Auditoría. Los diagnósticos fuera del límite se eliminan de forma definitiva.
7. Comprueba desde otra cuenta sin permisos administrativos que no pueda consultar
   estas pantallas ni sus APIs, y prueba la presentación desde una pantalla pequeña.

**La fase 8 queda pendiente de autorización.**
