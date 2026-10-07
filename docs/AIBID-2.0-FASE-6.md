# AIBID 2.0 — fase 6: configuración avanzada e historial de cargas

Entrega de pruebas **2.0.0-alpha.6-f6**, 5 de octubre de 2026. Continúa sobre
la fase 5; la fase 7 queda pendiente de autorización.

## 1. Implementación

La configuración administrativa reúne enlaces entre cuatro bloques: Archivos,
Procesamiento y recursos, Automatización/OCR/watcher y Vistas previas/caché.
Se conservan las pantallas y servicios existentes. En cada biblioteca,
Configuración permite heredar las reglas globales o sobrescribir campos concretos.

Los seis desplegables solicitados utilizan `Accordion` y `document-accordion`:
Editar identificador, Ajustar requisito, cada plantilla registrada, Clasificar o
asociar a expediente, Campos adicionales y Cancelar mi carga. Conservan los
formularios, permisos, validaciones y acciones anteriores.

**Historial de cargas** sustituye Mis lotes. Dos tablas muestran la referencia real,
fecha y cantidad de archivos; al seleccionar una carga, la segunda muestra nombre,
estado, formato detectado del contenido, tamaño y Ver detalles. La acción abre
`DocumentViewer`, incluida la clasificación y cancelación que correspondan. Se
conservan las transferencias fallidas y sus diagnósticos, aunque no exista ficha.
Las tablas se apilan en pantallas pequeñas; la selección usa `aria-pressed` y
el estilo de fila seleccionada existente. No se precarga todo el historial:
50 cargas y 25 archivos por página, con cursores independientes y acotados al
usuario y la carga. Las consultas anteriores continúan disponibles.

## 2. Archivos principales

- Backend nuevo: `internal/libraries/advanced_settings.go`,
  `watch_attributes_windows.go`, `watch_attributes_other.go`, `upload_history.go`;
  `internal/httpapi/advanced_settings.go`.
- Integración: `runtime.go`, `scanner.go`, `watcher.go`, `file_skips.go`,
  `extraction.go`, `processing.go`, `documents.go`, `reindex.go`, `job_control.go`,
  `service.go`, `uploads.go`, `previews.go`, `preview_worker.go`; extractores PDF.
- Interfaz nueva: `AdvancedSettings.tsx`, `SettingsNavigation.tsx`,
  `UploadHistory.tsx`. Integración en App, Libraries, ManagedSettings, FileSettings,
  ProcessingSettings, PreviewSettings, Uploads y tipos compartidos.
- Presentación: Accordion, Cases, Catalogs, ClassificationForm, Documents,
  ExtractionDetails, AuditTable y styles.css.
- Pruebas: advanced_settings_test, upload_history_test, pruebas PDF/migraciones,
  `web/tests/advanced-settings.spec.ts` y adaptación de selectores del historial.
- Distribución: versión del binario, constructor de instaladores, comprobación
  nativa, aceptación de actualización desde alpha.5, documentación y workflow CI.

## 3. Migración

**0013_advanced_settings** añade una tabla de configuración por ámbito
(`global` o `library:<id>`), JSON validado, revisión y fecha. Conserva como
excepciones los idiomas OCR anteriores distintos de español. Los demás valores
iniciales mantienen el comportamiento previo, salvo la omisión predeterminada de
temporales de Office y temporales comunes.

No renombra ni elimina documentos, cargas, expedientes, tablas de búsqueda o
historiales. Antes de migrar una instalación existente se crea una instantánea
SQLite consistente en `upgrade-backups`. La reversión vacía sigue restringida a
instalaciones nunca utilizadas. No se debe ejecutar un binario anterior sobre el
nuevo esquema; para volver a una entrega anterior se necesita su respaldo.

## 4. Parámetros

| Bloque | Global | Por biblioteca |
| --- | --- | --- |
| Formatos almacenables/indexables, tamaños y políticas | Servicio existente | Herencia/excepciones existentes |
| Procesos, concurrencia, pausa, intentos, espera y timeout total | Servicios existentes | Compartidos; no se duplican presupuestos |
| Procesamiento automático, reprocesar cambios y prioridad manual | Nuevo | Herencia/excepción por campo |
| OCR habilitado, automático e idioma | Nuevo bloque; idioma existente integrado | Herencia/excepción por campo |
| Umbral nativo, páginas PDF y segundos por operación | Umbral antes fijo y límites antes en config.json | Herencia/excepción por campo |
| Vigilancia, temporales Office, ocultos, temporales y extensiones | Nuevo | Herencia/excepción por campo |
| Vistas previas habilitadas y límite por archivo | Servicio existente | Nuevas excepciones |
| Caché, caducidad sin uso, limpieza, uso y vaciado | Servicio existente | Un único presupuesto compartido |

Valores iniciales: automatización, reprocesamiento, prioridad manual, OCR,
OCR automático y vigilancia activados; español; umbral de 32 letras/números;
páginas y timeout PDF/OCR tomados del config.json existente (normalmente 1000 y
120 segundos). Límites administrables: 1–10000 caracteres, 1–10000 páginas,
5–300 segundos por operación PDF/OCR y 1–512 MiB por original de vista previa.

Office (`~$...`, `.~lock....`), temporales `.tmp`, `.temp`, `.swp` y nombres
terminados en `~` se omiten por defecto. Ocultos es opcional y comprende nombres
con punto inicial y el atributo Hidden de Windows; Windows también respeta
Temporary. Las carpetas protegidas y archivos con atributo System se excluyen
siempre. Hasta 32 extensiones adicionales, sin rutas ni comodines.

## 5. Componentes reutilizados

`Accordion`, tablas y badges existentes, `CursorControls`/`useCursorPage`,
`FileIcon`, `formatSize`, `formatDate`, `Notice`, `DocumentViewer`, el mapa de estados
de procesamiento, los extractores y colas anteriores, y las pantallas de archivos,
recursos, reintentos y caché. No hay un segundo visor ni nuevos estados documentales.

## 6. Componentes nuevos

`AdvancedSettings` edita únicamente reglas que faltaban y excepciones de vistas
previas. `SettingsNavigation` enlaza los bloques existentes. `UploadHistoryFiles`
consulta y presenta una página de archivos de la carga seleccionada. Los registros
internos siguen llamándose batches; no hay identificadores nuevos.

## 7. Decisiones de funcionamiento

- Desactivar automatización conserva las tareas pendientes; Verificar biblioteca,
  Reindexar documento y los reintentos explícitos solicitan trabajo manual. Una
  verificación manual transmite su origen a las extracciones que descubre.
- La selección de tareas filtra las reglas **antes** del límite de candidatos:
  una cola suspendida no debe impedir que otra biblioteca procese sus documentos.
  Los trabajos manuales prioritarios se eligen antes de los automáticos; no se
  interrumpe una tarea en curso. La pausa general tiene prioridad sobre ambas.
- Desactivar reprocesamiento impide extracciones automáticas de documentos que
  ya tienen texto indexado. La detección de cambios y su trazabilidad continúan;
  reindexar manualmente permite actualizar su texto.
- La verificación de integridad de originales administrados y el guardado
  definitivo continúan; desactivar automatización no anula esas protecciones.
- Desactivar OCR automático permite OCR en solicitudes manuales. Desactivar OCR
  por completo conserva solo el texto nativo y muestra una advertencia. DOCX,
  XLSX, TXT y CSV no se envían al OCR PDF.
- Los nuevos parámetros se leen al comenzar el trabajo. Cambiarlos no inicia una
  reindexación masiva ni borra el texto guardado; la invalidación por cambio de
  idioma mantiene la comprobación de publicación anterior.
- Desactivar vigilancia conserva los recorridos periódicos configurados en
  Carpetas. Las exclusiones se aplican tanto a eventos como a recorridos completos
  o parciales. Un subárbol omitido no se toma como evidencia de archivos ausentes.
- Las excepciones de vistas previas nunca saltan la desactivación global. Caché,
  conversor y presupuesto de recursos siguen perteneciendo a la instalación.
- Duplicados conserva el comportamiento seguro anterior: registros y ubicaciones
  independientes, revisión por contenido, sin fusionar ni borrar originales. No
  se añadió una opción destructiva que altere esas reglas.
- No se incorpora un porcentaje artificial de CPU ni un horario intensivo nuevo:
  se utilizan procesos concurrentes, colas y pausa, que ya son controles reales.
- Los cambios de configuración llevan revisión optimista, autorización y auditoría.
  Un guardado obsoleto solicita actualizar en lugar de sobrescribir otro cambio.

## 8. Pruebas

La validación comprende Go normal/desarrollo, detector de carreras, vet, formato,
TypeScript/Vite, checks SQL, Chrome y actualización de un paquete macOS de fase 5.
Los escenarios nuevos cubren herencia, excepciones, persistencia al recrear el
servicio, conflictos de revisión, permisos, automatización y trabajos manuales,
omisiones sin falsos desaparecidos, OCR desactivado/umbral y vistas por biblioteca.

El historial se prueba con 51 cargas y 26 archivos, cursores incompatibles,
transferencias sin documento, privacidad, formatos detectados, estados pendientes
y completados. Chrome abre/cierra los seis acordeones, guarda identificador,
requisito y clasificación con campos adicionales, conserva cancelación disponible,
cambia de carga, verifica conteos y estados vacíos y compara disposición a 1800 y
390 píxeles. Se ejecutan también los flujos previos de bibliotecas administradas.

## 9. Resultados

La ejecución final y la verificación de paquetes se registran en
`dist/aibid-2.0-fase-6/VALIDACION-FASE-6.json` y `SHA256SUMS`.

## 10. Riesgos y límites

Los instaladores son de pruebas, sin firma de distribución. La ejecución nativa
Windows y Ubuntu y sus permisos reales de carpetas/SMB deben validarse en tus equipos.
La compilación para Windows comprueba también el soporte de atributos del sistema;
no equivale a ejecutar esas pruebas en Windows. La cuenta del servicio sigue
necesitando acceso a los originales. LibreOffice continúa siendo opcional para
vistas previas DOCX/XLSX. Los idiomas OCR seleccionados deben estar instalados.
No se instala ni reinicia el servicio real de esta Mac durante la validación.

## 11. Prueba manual sugerida

1. Actualiza desde fase 5 conservando datos; verifica versión alpha.6-f6, acceso,
   bibliotecas y la instantánea en `upgrade-backups`.
2. Abre Configuración avanzada; cambia un valor global y verifica que la biblioteca
   lo herede. Define una excepción y vuelve a marcar Heredar. Reinicia el servicio.
3. Desactiva procesamiento automático en una biblioteca, carga un archivo y solicita
   Reindexar documento. Comprueba que otras bibliotecas sigan procesando.
4. Cambia un original con reprocesamiento desactivado; verifica que conserva el texto
   anterior hasta reindexarlo manualmente.
5. Prueba un PDF escaneado con OCR habilitado/deshabilitado y con OCR automático
   desactivado. Reindexa explícitamente para comparar resultados.
6. En una carpeta de pruebas agrega `~$prueba.docx`, `.tmp` y archivos ocultos; verifica
   sus omisiones. Comprueba que la exclusión no marque como ausentes originales ya
   registrados. En Windows prueba también Hidden, System y Temporary.
7. Desactiva vistas previas solo para una biblioteca; comprueba que la otra conserve
   la función y que la desactivación global tenga prioridad.
8. Revisa Historial de cargas vacío y con varias cargas. Cambia de selección, abre
   detalles, clasifica, prueba los seis acordeones y realiza una cancelación de una
   carga de prueba. Comprueba permisos con otra cuenta y la vista en pantalla pequeña.

**La fase 7 no está iniciada.**
