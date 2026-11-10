# AIBID 2.0 — fase 3: robustez del procesamiento

Entrega de pruebas **2.0.0-alpha.3-f3**, del 4 de octubre de 2026. Incluye las
correcciones de Word, Excel e iconos solicitadas tras la fase 2. La fase 4
(previews y caché) queda pendiente de autorización.

## Correcciones comprobadas con los originales

**Word:** el documento de prueba tenía párrafos vacíos antes de su texto, incluido
el área inicial del logotipo. El visor abría la primera unidad y su mensaje podía
interpretarse como ausencia de texto en todo el documento. Ahora abre la primera
unidad con texto, también en extracciones antiguas, y ofrece Anterior/Siguiente y
el total de unidades. El extractor `docx-openxml` versión 2 omite párrafos vacíos
como unidades nuevas, conservando sus números originales y el conteo del documento.
No hace falta reindexar los Word ya procesados para corregir la apertura del visor.

**Excel:** ambos libros contienen partes binarias de configuración de impresión
legítimas. La prohibición general de binarios dentro de ZIP las confundía con
contenido peligroso. Se aceptan exclusivamente `xl/printerSettings/printerSettingsN.bin`
y su equivalente de Word, con el tipo de contenido OOXML correspondiente, límites
existentes y rechazo de cabeceras ejecutables. No se interpretan ni ejecutan esas
partes. Se mantiene el rechazo de macros, ActiveX, objetos incrustados y otros binarios.
Microsoft documenta estas partes en
[WordprocessingML](https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.wordprocessing.printersettingsreference)
y en las [estructuras de impresión de Excel](https://learn.microsoft.com/nl-nl/openspecs/office_file_formats/MS-XLSB/1cdc4cb9-836d-41d6-a5b5-9ac0428f491c).

Prueba local optativa con los tres archivos suministrados, sin modificar los originales:

| Archivo | Resultado |
|---|---|
| Cotizacion Famisa.docx | 40 párrafos, 29 unidades con texto y una tabla |
| Cotizacion Eliver.xlsx | 122 celdas en dos hojas; 44 fórmulas con resultados guardados |
| Cotizacion Saul.xlsx | 51 celdas en una hoja; 14 fórmulas, nueve sin resultado guardado |

Los libros se leen sin recalcular fórmulas. Se advierte cuando faltan sus resultados
o pueden estar desactualizados. Para disponer de valores ausentes, recalcular y
guardar el libro en Excel antes de volver a verificar la biblioteca.
Los documentos privados permanecen en la carpeta proporcionada: no se copian al
repositorio, fixtures ni instaladores. Las pruebas repetibles usan documentos sintéticos.

**Iconos:** SVG locales por formato (PDF, Word, Excel, texto y CSV), con tamaño fijo,
sustituyen las letras que se partían verticalmente. Son gráficos propios del proyecto;
no se incorpora un paquete de marcas ni se consulta un servicio externo.

## Cola y límites

En **Administración → Procesamiento → Cola y límites de procesamiento**:

| Parámetro | Predeterminado | Rango |
|---|---:|---:|
| Máximo de intentos, incluido el inicial | 5 | 1–5 |
| Espera inicial entre intentos | 2 segundos | 1–3600 segundos |
| Tiempo máximo por documento e intento | 3600 segundos | 10–3600 segundos |

Los valores quedan fijados al primer intento de cada trabajo. Cambiarlos afecta a
trabajos aún sin comenzar y a nuevos reintentos manuales; no altera el presupuesto
de un trabajo que ya empezó. La espera se duplica entre intentos, con tope de una hora.
Solo se reintentan automáticamente fallos transitorios, como tiempo agotado,
archivo cambiando o almacenamiento temporalmente no disponible. Un documento
inválido o un formato no admitido no consume reintentos automáticos inútiles.
Al agotar el límite queda el error y se conserva el reintento manual con trazabilidad.

**Pausar procesamiento** impide tomar nuevos trabajos en todas las colas. Los
trabajos ya tomados pueden terminar, incluida su etapa OCR. Se mantienen búsquedas,
consultas, cargas autorizadas y los trabajos pendientes. La pausa persiste al
reiniciar. **Reanudar procesamiento** permite continuar la cola sin reconstruirla.
La pausa y la toma de trabajos comparten una transacción de escritura, evitando
que se inicie otro trabajo después de confirmar la pausa. Configurar estos valores
requiere `system.configure`, control de revisión y auditoría.

La pausa global no interrumpe un recorrido de carpeta que ya comenzó. Continúan
disponibles los controles específicos de pausa/cancelación de recorridos y trabajos.
Los recorridos de carpetas usan sus puntos de reanudación existentes; el tiempo
por documento no limita la duración de una biblioteca completa.

El tiempo del documento comprende lectura de la copia privada, validación,
extracción, espera de OCR y publicación. Los programas PDF/OCR conservan además sus
límites por página y la terminación de procesos. Los lectores internos comprueban
cancelación y límites durante su avance. Un fallo recuperable del código de un
trabajo se registra de forma genérica, sin exponer contenido ni detener los demás.

Se conservan los límites independientes de almacenamiento/indexación, tamaño,
páginas OCR, expansión ZIP/XML, unidades y texto de la fase 2. Un exceso no publica
texto parcial ni elimina un original que esté permitido almacenar.

## Estados, versiones y errores

La pantalla de procesamiento presenta Pendiente, Analizando archivo, Validando tipo,
Extrayendo texto, Procesando OCR, Esperando OCR, Generando metadatos, Indexando,
Completado, Completado con advertencias, No soportado, Error, Reintentando, Pausado
y Cancelado. Los textos españoles se mantienen separados de los identificadores.
Los estados se derivan de la cola y de los resultados de extracción existentes;
no se añade otro ciclo documental mutable que pueda quedar desincronizado.

Los trabajos muestran intentos reales y su máximo, último diagnóstico, clase de
error y fecha del siguiente intento. Se distinguen tiempo agotado, documento
inválido/dañado, formato o herramienta no disponible, límite excedido, fallo del
extractor, interrupción y fallo del sistema/almacenamiento. No se afirma corrupción
cuando el programa solo informa que un PDF podría estar dañado o cifrado.

Se conserva SHA-256 y la comprobación de versión antes de publicar: un recorrido
sin cambios no repite la extracción vigente. Un original vinculado modificado
conserva su documento e historial, genera su nueva versión y sustituye el texto
publicado según las reglas actuales. Resultados de una versión anterior no pueden
publicarse sobre la actual. No se introduce reindexación general ni caché de previews.

## Migración y compatibilidad

Migración aditiva **`0011_processing_policy`**: tabla de configuración global y
presupuesto de espera/tiempo en cada trabajo. La cola existente conserva los
presupuestos compatibles (cinco intentos, dos segundos iniciales y una hora) y la
pausa viene deshabilitada. No cambian usuarios, licencia, identidades, documentos,
hashes, FTS, expedientes ni originales.

Antes de actualizar una base de esquema 8, 9 o 10 se crea una instantánea SQLite
consistente en `state/upgrade-backups/before-processing-*.db`. Incluye WAL confirmado
y no se repite tras actualizar. El respaldo de SQLite no sustituye el de configuración,
identidad y archivos administrados. No utilizar binarios anteriores con la base migrada.
La reversión automática de esquema solo admite instalaciones vacías.

Los instaladores admiten `0.7.0-test.1`, `2.0.0-alpha.1`, `2.0.0-alpha.2` y
reinstalación de esta versión. Conservan el canal **AIBID Pruebas**, los datos y el
puerto configurado; migran antes de marcar la nueva versión e iniciar el servicio.
No se crea una segunda instalación. No hay nuevas dependencias de ejecución.
Los paquetes se generan juntos en `dist/aibid-2.0-fase-3/`; se conservan los anteriores.

## Archivos de esta fase

| Área | Archivos principales |
|---|---|
| Correcciones Office | `internal/documentformat/office.go`, `internal/extraction/{docx,document}.go`, `internal/libraries/{documents,search}.go` |
| Política y cola | `internal/libraries/{processing_policy,processing_state,runtime,ocr_workers,processing,extraction}.go` |
| Programas externos | `internal/extraction/pdf.go` |
| API | `internal/httpapi/{processing_settings,server}.go` |
| Migración | `db/migrations/0011_processing_policy.{up,down}.sql`, `internal/storage/{database,upgrade_backup}.go` |
| Interfaz | `web/src/{ProcessingPolicy,ProcessingSettings,Processing,Documents,ExplorerEntries,FileIcon,Libraries,Search,AuditTable}.tsx`, `libraryTypes.ts`, `styles.css` |
| Pruebas | `internal/documentformat/printer_settings_test.go`, `internal/extraction/{document,local_corpus}_test.go`, `internal/libraries/{processing_policy,document_extractors}_test.go`, `internal/storage/{database,document_formats_upgrade}_test.go`, `web/tests/processing-phase3.spec.ts` |
| Fixtures | `scripts/generate_multiformat_fixtures.py`, `testdata/documents/{blank-leading.docx,printer-settings.xlsx}` |
| Distribución | `internal/buildinfo/version.go`, `scripts/{build_installers,smoke_macos_installer}.py`, preinstalación macOS/Ubuntu, `packaging/windows/manage.ps1`, `.github/workflows/h7-installers.yml` |

La base multiformato de la fase 2 y sus archivos se describen en
[el informe previo](AIBID-2.0-FASE-2.md).

## Validación

El resultado final se registra en `dist/aibid-2.0-fase-3/VALIDACION-FASE-3.json`
y los hashes en `SHA256SUMS`. Las pruebas siguientes quedaron aprobadas el
4 de octubre de 2026:

- Suites Go completas de producción y desarrollo, `go vet`, compilación TypeScript/Vite.
- Detector de carreras en formato, extracción, bibliotecas, almacenamiento y HTTP.
- Presupuestos y espera persistentes, máximo de intentos, nuevo presupuesto tras
  reintento manual, control de revisiones y permisos.
- Pausa persistente, bloqueo de nuevas tomas y publicación/búsqueda de un trabajo
  que ya estaba activo.
- Extractor externo bloqueado: vence a los diez segundos y otro documento se
  procesa sin quedar bloqueado por él; fallo interno contenido por trabajo.
- Word con párrafos iniciales vacíos y extracciones antiguas; Excel con impresión
  válida, rechazo de cabecera ejecutable/tipo falso/binarios ajenos.
- Tres originales locales, sin publicación de su contenido; pruebas sintéticas
  para regresión y navegador.
- Migraciones desde 8/9/10: conserva IDs, hashes, FTS e historial; verifica copia
  consistente y evita repetirla tras reiniciar.
- Navegador: catorce escenarios, incluidos pausa, recuperación, Word, Excel,
  iconos, formato móvil y regresiones de PDF/OCR, usuarios, licencia y bibliotecas.
- Paquetes macOS ARM, Windows amd64, Ubuntu amd64 y ARM64. macOS Intel se excluye
  conforme a lo solicitado.

La prueba macOS ejecuta el binario extraído en datos desechables y actualiza desde
el paquete de fase 2; no instala un servicio real. Comprueba PDF/OCR, bootstrap,
HTTP, acceso local/LAN, reinicio, esquema 10→11, respaldo, permisos y limpieza.
Windows se comprueba mediante compilación, NSIS, iconos, inventario PDF/OCR y
dependencias PE. Ubuntu se comprueba mediante estructura, arquitectura ELF,
propietarios, dependencias y orden de actualización. La ejecución nativa de los
instaladores/servicios en Windows y Ubuntu sigue pendiente en los equipos destino.

Para repetir la prueba privada, establecer `AIBID_LOCAL_DOCUMENTS` a la ruta absoluta
de una carpeta de prueba y ejecutar `go test ./internal/extraction -run TestLocalDocumentCorpus -v`.
Sin esa variable la prueba se omite. Solo informa conteos y advertencias.

## Comprobación en Windows y Ubuntu

1. Respaldar los datos de prueba e instalar el paquete de esta fase sobre la fase 2.
   No es necesario borrar la base ni desactivar la licencia para actualizar.
2. Abrir el Word ya procesado: debe mostrar la primera unidad con texto. Probar
   Anterior/Siguiente, búsqueda y descarga del original.
3. Confirmar que XLSX esté permitido para indexación y pulsar **Verificar biblioteca
   ahora**. Los dos libros antes omitidos deben incorporarse y procesarse. Revisar
   la advertencia de fórmulas sin resultado guardado en el libro correspondiente.
4. En Administración → Procesamiento, pausar y añadir documentos. Deben permanecer
   pendientes; los ya iniciados pueden concluir. Buscar/consultar documentos,
   reiniciar AIBID y comprobar que la pausa sigue activa.
5. Reanudar y comprobar que avanzan. En datos desechables, ajustar el máximo de
   intentos y el tiempo, comprobar un fallo y su reintento manual tras corregirlo.
6. Verificar una carpeta sin cambios y luego editar un archivo de prueba: no debe
   reprocesar el primero; debe actualizar la versión/texto del segundo sin duplicarlo.

## Límites pendientes

- Las operaciones de disco/red que el sistema operativo mantenga bloqueadas dentro
  de una llamada no se pueden interrumpir de forma portátil con un contexto Go;
  se aplican sus propios tiempos de espera de almacenamiento/SMB. El tiempo del
  documento cubre los lectores cooperativos y la terminación de programas externos.
- No se calculan fórmulas de Excel ni se aplica OCR a imágenes dentro de Word.
- La fase 4 queda sin iniciar: Word/Excel ofrecen texto y descarga, todavía sin
  vista previa de su diseño original.
- No se modificó la instalación real ni se contactó al servidor comercial de licencias.
