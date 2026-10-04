# AIBID 2.0 — fase 2: motor documental y extractores

Entrega de pruebas **2.0.0-alpha.2-f2**. Evoluciona la base multiformato de la fase 1
sobre los servicios existentes. Implementa exclusivamente la fase 2 autorizada.

## Comportamiento entregado

| Formato | Extractor y versión | Contenido y contexto |
|---|---|---|
| PDF | `poppler-tesseract`, 2 | Extracción nativa y OCR existentes, páginas y método de extracción |
| DOCX | `docx-openxml`, 1 | Párrafos, estilos de párrafo, secciones declaradas, tablas/fila/celda, encabezados, pies y notas vinculados al documento |
| XLSX | `xlsx-openxml`, 1 | Hojas y celdas, cadenas compartidas, texto enriquecido, valores guardados; ubicación como «Hoja Presupuesto 2026 · celda B14» |
| TXT | `text-unicode`, 1 | Texto agrupado en unidades con rango de líneas, codificación y caracteres sin saltos de línea |
| CSV | `csv-structured`, 1 | Registros, columnas y línea de origen, incluso campos entrecomillados con saltos de línea; delimitador y codificación |

Los resultados se publican en el índice existente, en una transacción, después de
comprobar versión, hash, permisos de procesamiento y reglas actuales. Cada intento
registra extractor, versión, formato, inicio/fin, resultado, código de error,
advertencias y resumen. La ficha muestra **Último procesamiento** y permite recorrer
las unidades de texto. La búsqueda muestra el contexto correspondiente en cada coincidencia.

Se mantienen las páginas físicas para PDF. En los demás formatos se presentan
**unidades de contenido**: no se inventan páginas de Word o Excel. La vista PDF
permanece disponible para PDF; los otros formatos conservan descarga y consulta de
texto. Los previews de otros formatos pertenecen a una fase posterior.

El registro central selecciona por el formato validado, no por condicionales de
extensión repartidos entre workers. PDF conserva la separación entre extracción
nativa y OCR; DOCX/XLSX/TXT/CSV no invocan Poppler, Tesseract, Microsoft Office ni
LibreOffice. Se procesa una copia privada verificada. Los originales permanecen intactos.

## Configuración y primera indexación

La actualización no cambia las listas que ya configuró el administrador. El valor
predeterminado de **Indexar contenido** sigue siendo PDF.

1. Abrir **Archivos y procesamiento** en el menú administrativo, o su sección dentro
   de **Biblioteca → Configuración**.
2. Permitir almacenar e indexar los formatos que se quieren consultar por contenido.
3. Guardar. Las bibliotecas heredan cada parámetro salvo su excepción explícita.

Al guardar, y al iniciar el servicio tras actualizar, se consideran los documentos
conservados sin primera extracción por política o por falta del extractor en la fase 1.
Si ahora son elegibles, se encolan. Se conservan las cancelaciones del usuario y los
errores terminales; estos mantienen sus mecanismos existentes de revisión y reintento.
No se reconstruyen índices existentes ni se reprocesan documentos solo por cambiar
una versión del extractor. La reindexación manual está reservada para su fase.

## Decisiones y límites

- **DOCX:** extrae texto estático por párrafo, incluyendo el de tablas. Mantiene
  estilo y parte de origen. Omite texto marcado como eliminado e instrucciones de
  campos; conserva el resultado visible guardado. Las referencias externas no se
  consultan. El conteo de secciones corresponde a propiedades de sección presentes,
  no a paginación calculada. No interpreta diseño, imágenes ni contenido `altChunk`.
- **XLSX:** lee celdas y nombres de hojas sin evaluar fórmulas. Usa exclusivamente
  resultados guardados y avisa si faltan o podrían estar desactualizados. No aplica
  formato visual: por ejemplo, una fecha numérica puede aparecer como número de serie.
  Las hojas sin celdas, como gráficos, se omiten con advertencia. Los nombres de hoja
  se incluyen como contexto del contenido.
- **TXT/CSV:** acepta UTF-8, con o sin BOM, y UTF-16LE/BE con BOM. Valida Unicode y
  rechaza controles binarios. No adivina Windows-1252 ni codificaciones de un byte.
  La cancelación conserva su diagnóstico y no se confunde con texto mal formado.
- **CSV:** prueba coma, punto y coma, tabulador y barra vertical con registros
  consistentes. La primera fila solo se interpreta como encabezado cuando presenta
  nombres únicos no numéricos y la siguiente aporta evidencia numérica. Se identifica
  explícitamente como inferencia y se conserva íntegra como contenido. En otros casos
  se usan columnas numeradas. No se pierde la primera fila ni se ejecutan fórmulas.

Los motores Office reutilizan la validación ZIP/XML de admisión: partes seguras,
límites de expansión, sin DTD ni referencias externas, sin descomprimir a disco.
Las partes adicionales leídas también se validan. Continúa el rechazo de macros,
ActiveX, objetos incrustados peligrosos y formatos prohibidos de la fase 1.
Esta validación no equivale a un antivirus ni a la validación completa del esquema OOXML.

Los nuevos extractores limitan el texto a **32 MiB por documento**, **4 MiB por
unidad** y **50 000 unidades**. TXT limita cada línea a aproximadamente 4 MiB y
agrupa hasta 200 líneas por unidad. CSV mantiene el límite aproximado de 8 MiB por
registro y hasta 16 384 columnas. XLSX admite hasta 1 024 hojas y limita la tabla de
cadenas compartidas a 200 000 entradas/32 MiB. Se mantienen los límites ZIP de la
fase 1 y XML de 32 MiB por parte/profundidad 128. Un exceso genera error y no publica
una extracción parcial. PDF conserva sus límites existentes.

Estas decisiones pueden rechazar archivos legítimos muy grandes. Los documentos
con fórmulas, diseños complejos, imágenes o texto en codificaciones no admitidas
requieren comprobar manualmente la utilidad del resultado. El original no se convierte
ni se reemplaza para eludir una limitación.

Referencias de estructura utilizadas: [WordprocessingML y sus párrafos](https://learn.microsoft.com/en-us/office/open-xml/word/how-to-open-and-add-text-to-a-word-processing-document),
[celdas de SpreadsheetML](https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.spreadsheet.cell)
y [cadenas compartidas](https://learn.microsoft.com/en-us/office/open-xml/spreadsheet/working-with-the-shared-string-table).
No se incorpora el SDK de Microsoft como dependencia.

## Migración y compatibilidad

Migración aditiva **`0010_document_extractors`**:

- `extraction_runs`: identificador/versión de extractor, formato, inicio, resultado,
  cantidad de unidades, resumen y advertencias JSON.
- `extraction_pages`: tipo de unidad, etiqueta y contexto JSON. `page_number` sigue
  siendo el ordinal interno compatible con los endpoints e índices existentes.
- Los registros PDF anteriores conservan su revisión y reciben metadatos derivados
  únicamente de los datos existentes; no se inventa una versión desconocida.

No cambian IDs, hashes, originales, FTS, expedientes, usuarios, auditoría o licencia.
Se crea una instantánea SQLite consistente antes de pasar de esquema 8 o 9 a 10,
incluyendo WAL confirmado, en `state/upgrade-backups/before-processing-*.db`.
No se repite al reiniciar una base ya actualizada. El respaldo no sustituye el de
configuración, identidad y archivos administrados.

La reversión de esquema está protegida para instalaciones vacías; con extracciones
requiere recuperar el respaldo y su binario compatible. No ejecutar el binario
de fase 1 contra la base ya migrada.

Los instaladores admiten actualización desde `0.7.0-test.1` y `2.0.0-alpha.1`, o
reinstalación de `2.0.0-alpha.2`. Migran antes de cambiar el marcador e iniciar el
servicio. Se conserva la identidad del canal **AIBID Pruebas**, sus datos y el
puerto configurado; estos paquetes no crean automáticamente una segunda instalación.

## Archivos modificados

| Área | Archivos |
|---|---|
| Registro y extractores | `internal/extraction/{document,office,docx,xlsx,text,pdf}.go` |
| Lectura segura y catálogo | `internal/documentformat/{format,office,text}.go` |
| Ejecución y primera indexación | `internal/libraries/{extraction,extraction_admission,file_settings,runtime}.go` |
| Consulta y resultados | `internal/libraries/{documents,search,extraction_report}.go` |
| Migración y respaldo | `db/migrations/0010_document_extractors.{up,down}.sql`, `internal/storage/{database,upgrade_backup}.go` |
| Interfaz | `web/src/{Documents,ExtractionDetails,FileSettings,Processing,Search,Uploads}.tsx`, `web/src/libraryTypes.ts` |
| Pruebas Go | `internal/documentformat/{format,text}_test.go`, `internal/extraction/document_test.go`, `internal/libraries/{document_extractors,service}_test.go`, `internal/storage/{database,document_formats_upgrade}_test.go` |
| Prueba de navegador | `web/tests/document-extractors.spec.ts` |
| Paquetes | `internal/buildinfo/version.go`, `scripts/{build_installers,smoke_macos_installer}.py`, `packaging/macos/preinstall`, `packaging/ubuntu/preinst`, `packaging/windows/manage.ps1` |
| CI y documentación | `.github/workflows/h7-installers.yml`, `README.md`, `docs/{MILESTONES,AIBID-2.0-FASE-2}.md` |

No hay nuevas dependencias Go, frontend ni herramientas de ejecución. Se utilizan
las bibliotecas estándar de Go. PDF/OCR conserva Poppler/Tesseract y el inventario
fijado del runtime Windows; Linux y macOS mantienen sus dependencias declaradas.

## Validación automatizada

- Compilación TypeScript/Vite; suites Go completas de producción y desarrollo;
  `go vet ./...`: aprobados.
- Detector de carreras en `documentformat`, `extraction`, `libraries`, `storage`
  y `httpapi`: aprobado.
- **13 escenarios Chrome aprobados**: los cuatro nuevos formatos y su búsqueda,
  navegación por celdas y ficha del extractor, además de regresiones PDF/OCR,
  reglas, expedientes, usuarios, licencias, LAN y vista móvil.
- Fixtures DOCX con párrafos/tabla/encabezado/campos, XLSX con cadenas compartidas y
  fórmulas con/sin resultado, UTF-16 en ambos órdenes, CSV entrecomillado multilínea.
- Rechazo de XML con DTD, partes Office inválidas, rutas de relaciones inseguras,
  binarios/scripts disfrazados y límites de texto. La falla no publica FTS parcial.
- Carga, primera indexación al habilitar reglas, búsqueda con contexto, descarga
  idéntica y ausencia de invocación PDF/OCR en formatos nuevos.
- Documento vinculado modificado: conserva identidad, cambia hash y sustituye texto
  publicado; documento eliminado: conserva evidencia/texto según reglas existentes.
- Actualización desde esquemas 8 y 9: conserva IDs, hashes, FTS y expediente; verifica
  respaldo previo y evita repetirlo al siguiente arranque.
- Formato Go, sintaxis de scripts, `git diff --check` y 14 comprobaciones de diseño
  y enlaces locales: aprobados.

La evidencia de paquetes y pruebas nativas se registra en
`dist/aibid-2.0-fase-2/VALIDACION-FASE-2.json`. La compilación cruzada no sustituye
la ejecución de instaladores en Windows y Ubuntu, pendiente en los equipos destino.

Comprobaciones de empaquetado realizadas:

- **macOS ARM:** manifiesto y firma ad hoc del binario extraído; diagnóstico real
  PDF/OCR español, administrador con contraseña de seis caracteres, login HTTP,
  acceso local/LAN con reinicio, bloqueo de estado, integridad SQLite y limpieza
  explícita sobre datos desechables que conserva originales. Actualización desde
  el binario de fase 1: esquema 9 a 10, respaldo consistente, usuarios y auditoría
  preservados, sin repetir el respaldo al reiniciar la migración.
- **Ubuntu amd64/ARM:** estructura ar/tar, arquitectura ELF, versión/dependencias,
  propietarios del payload, rutas seguras y migración antes del marcador de versión.
- **Windows amd64:** compilación de backend y lanzador, iconos, inventario y hashes
  fijados de PDF/OCR, dependencias PE y generación NSIS. NSIS 3.13 es una herramienta
  temporal de construcción; no se incorpora como dependencia del usuario final.

No se modificó la instalación real ni se contactó al servidor comercial de licencias.
La prueba macOS usa datos temporales. La validación nativa de servicios/instaladores
Windows, Ubuntu y launchd, y el ensayo offline en una VM limpia, siguen pendientes.

## Instaladores y pruebas manuales

Los cuatro objetivos son macOS Apple Silicon, Windows amd64 y Ubuntu 24.04 amd64/ARM.
No se genera macOS Intel. Salida: `dist/aibid-2.0-fase-2/`. Se conservan los paquetes
de la fase 1 y r7. Se incluyen fixtures en `LEEME/formatos/` y la guía de instalación
existente; sus nombres de versiones anteriores son referencias históricas.

En Ubuntu ARM usar **`aibid-test_2.0.0~alpha.2_arm64.deb`**. Los `.deb` siguen
requiriendo las dependencias PDF/OCR declaradas o su preparación offline existente.

1. Actualizar una instalación de pruebas con datos de fase 1. Confirmar administrador,
   licencia, reglas, expedientes, PDF/OCR, búsquedas e instantánea anterior al esquema 10.
2. Habilitar DOCX/XLSX/TXT/CSV en **Indexar contenido**, globalmente o para una biblioteca.
   Confirmar que documentos de esos formatos almacenados sin extracción entran a la cola.
3. Cargar documentos propios: Word con párrafos y tablas, Excel con varias hojas y
   fórmulas guardadas, TXT Unicode y CSV con el delimitador utilizado habitualmente.
   Buscar una frase interna y comprobar el contexto y **Último procesamiento**.
4. Probar la carpeta local y la UNC con estos formatos. Cambiar una copia de prueba
   externamente y verificar que se actualiza el índice sin cambiar su identidad.
5. Descargar originales y comparar SHA-256 antes/después. Abrir los archivos en
   sus aplicaciones habituales y confirmar que AIBID no alteró sus bytes.
6. Comprobar que una biblioteca limitada a PDF mantiene su restricción y que almacenar
   sin indexar sigue permitiendo conservar y descargar según la política elegida.
7. Verificar instalación/actualización, servicio y permisos en Windows y Ubuntu.
   macOS se comprueba con binarios extraídos y datos temporales; ello no certifica
   launchd, la cuenta de servicio ni la instalación con privilegios del paquete.

Para repetir la comprobación macOS de actualización desde la fase 1:

```sh
python3 scripts/smoke_macos_installer.py \
  dist/aibid-2.0-fase-2/AIBID-Pruebas-2.0.0-alpha.2-macOS-arm64.pkg \
  --previous-package dist/aibid-2.0-fase-1/AIBID-Pruebas-2.0.0-alpha.1-macOS-arm64.pkg
```

**Fase 2 implementada. La fase 3 requiere autorización expresa.**
