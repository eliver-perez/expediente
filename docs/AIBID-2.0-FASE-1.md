# AIBID 2.0 — fase 1: formatos, MIME y reglas de archivos

Entrega de pruebas: **2.0.0-alpha.1-f1**. Implementa exclusivamente la fase 1. La extracción DOCX/XLSX/TXT/CSV, las conversiones, el caché de previews y los demás cambios de las fases 2–8 quedan pendientes de autorización.

## Cambios funcionales

- PDF conserva su carga, extracción nativa, OCR, búsquedas y visor existentes.
- DOCX, XLSX, TXT y CSV pueden registrarse en bibliotecas vinculadas y cargarse en bibliotecas administradas/híbridas. Se pueden clasificar, asociar a expedientes, revisar, guardar definitivamente, consultar por nombre/metadatos y descargar. Todavía no se extrae ni se busca su contenido.
- El guardado definitivo conserva la extensión del documento y sus bytes. Los originales vinculados nunca se escriben ni se eliminan.
- La ficha muestra formato, MIME, discrepancia de extensión cuando corresponda y motivo de conservación sin indexación. Solo PDF se presenta en el visor; los demás originales se sirven como descarga con MIME detectado, `attachment` y `nosniff`.
- Procesamiento distingue documentos conservados sin indexación de trabajos pendientes. Las omisiones por formato se muestran por ruta, separadas de errores de lectura, sin generar trabajos de extracción.

## Configuración

En el menú administrativo, **Archivos y procesamiento** configura la instalación. En **Biblioteca → Configuración → Archivos y procesamiento**, cada lista, tamaño y política puede heredarse o sobrescribirse por separado.

| Regla | Valor inicial |
|---|---|
| Almacenar | PDF, DOCX, XLSX, TXT y CSV |
| Indexar contenido | PDF |
| Tamaño máximo de archivo | Límite anterior de `config.json`; 256 MiB en instalaciones nuevas |
| Tamaño máximo indexable | Igual al límite anterior; configurable por separado |
| Extensión desconocida | Rechazar |
| Almacenable sin indexación | Conservar original y metadatos |

Las listas heredadas de indexación se intersectan con los formatos que permite almacenar la biblioteca. El tamaño indexable heredado se limita al tamaño de almacenamiento. Una excepción explícita incompatible se rechaza; también se rechaza un cambio global que contradiga excepciones explícitas de bibliotecas. Guardar comprueba las revisiones global y local para evitar sobrescribir cambios concurrentes y registra auditoría.

Se permite configurar anticipadamente la indexación de DOCX/XLSX/TXT/CSV, pero la ficha informa **extracción aún no disponible** y no se encolan trabajos PDF para ellos. Seleccionar “Rechazar su incorporación” también rechaza un documento cuyo extractor aún no está disponible.

Los cambios afectan nuevas incorporaciones, siguientes recorridos y trabajos pendientes. Un trabajo PDF vuelve a comprobar la política antes de ejecutarse y antes de publicar. No se borran originales ni índices históricos por cambiar una regla. Los PDF conservados sin extracción pueden incorporarse a la cola en un recorrido posterior si sus reglas vuelven a permitirlo; no se reconstruyen índices existentes automáticamente.

## Detección y seguridad

`internal/documentformat` centraliza el catálogo, detección y política, sin nuevas dependencias externas:

- PDF: firma `%PDF-`; las cargas y el procesamiento conservan la validación Poppler existente. El reconocimiento durante el recorrido no ejecuta Poppler por cada PDF sin cambios.
- DOCX/XLSX: directorio ZIP acotado, nombres de partes seguros y únicos, tipos de contenido, relación al documento principal y XML principal con el espacio de nombres esperado. No se descomprime a disco ni se siguen referencias externas. Se rechazan paquetes con macros, ActiveX, objetos incrustados, ejecutables o rutas inseguras.
- TXT: contenido completo UTF-8 sin bytes binarios de control. CSV: además requiere registros delimitados consistentes; se prueban coma, punto y coma, tabulador y barra vertical. TXT/CSV no tienen firmas inequívocas: una extensión CSV solo se acepta si el contenido satisface esta validación.
- Se bloquean extensiones ejecutables, scripts, bibliotecas, paquetes, archivos comprimidos, formatos Office con macros y sus equivalentes contemplados en el catálogo de prohibiciones. También se comprueban firmas binarias y encabezados activos evidentes aunque se cambie la extensión.
- Una extensión conocida incompatible con el contenido se rechaza. Se conserva evidencia de la discrepancia en el diario de carga o diagnóstico de la ruta. Una extensión desconocida puede admitirse únicamente mediante la opción explícita de texto UTF-8 validado, con TXT permitido, y siempre sin indexar.
- Los formatos se validan al incorporar y al abrir el original. Los cambios externos en documentos administrados conservan evidencia histórica y requieren revisión; si sus bytes son rechazados no se encolan para extracción.

Límites defensivos independientes de la política: 4096 MiB por archivo, 10 000 entradas ZIP, 4 MiB de directorio central, 64 MiB por parte expandida, 512 MiB expandidos en total, relación de compresión máxima 200:1, XML principal de hasta 32 MiB y profundidad de 128 elementos. Los registros CSV están acotados aproximadamente a 8 MiB. ZIP64 y paquetes multipartes/ambiguos se rechazan. Estos límites pueden rechazar documentos Office legítimos muy grandes o con objetos incrustados; no se cambia silenciosamente el original para aceptarlos.

Esto es detección y admisión documental, no un antivirus ni una validación completa de todos los esquemas de Office. El texto puede contener ejemplos de código; AIBID nunca lo ejecuta. TXT/CSV en otras codificaciones y archivos Office cifrados no están admitidos en esta fase.

## Migración y compatibilidad

Migración aditiva `0009_document_formats`:

1. Añade formato, MIME, extensión, discrepancia y motivo de no indexación a `content_versions`.
2. Añade evidencia de detección al diario `upload_items` y a la caché de observaciones `file_scan_cache`.
3. Crea `document_file_settings` para reglas globales/excepciones y `document_file_skips` para omisiones vinculadas.

Las versiones anteriores reciben metadatos PDF compatibles. No se modifican IDs, bytes, hashes, FTS, extracciones, expedientes, permisos ni licencias. No se reconstruye información. Antes de actualizar una base existente se crea una instantánea SQLite consistente en `state/upgrade-backups/before-processing-*.db`; incluye WAL confirmado. No se repite en cada arranque.

El descenso de esquema en una instalación vacía sigue disponible. Si ya hay formatos nuevos, la migración inversa lo impide; para recuperar una instalación anterior se necesita su respaldo consistente. No se debe intentar usar el binario anterior con la base migrada.

Los instaladores admiten el paso desde el canal de pruebas anterior `0.7.0-test.1` (incluida r7) y la reinstalación de esta fase. Rechazan otras versiones. Ejecutan la migración con el binario nuevo antes de escribir el marcador de versión e iniciar el servicio. El respaldo automático cubre SQLite; no sustituye un respaldo integral de configuración, identidad y documentos. No se implementa un actualizador general ni una reversión automática de binarios en esta fase.

## Archivos principales

| Área | Archivos |
|---|---|
| Catálogo, detección y política | `internal/documentformat/{format,office,policy}.go` |
| Reglas, auditoría y omisiones | `internal/libraries/{file_settings,file_skips}.go`, `internal/httpapi/file_settings.go` |
| Incorporación y workers | `internal/libraries/{uploads,scanner,scan_cache,watcher,extraction,runtime,managed_integrity,materialization,settings}.go` |
| Consulta y descarga | `internal/libraries/{search,documents,processing}.go`, `internal/httpapi/{library_handlers,organization_handlers,server}.go` |
| Migración y respaldo | `db/migrations/0009_document_formats.*.sql`, `internal/storage/{database,upgrade_backup}.go` |
| Interfaz | `web/src/{FileSettings,App,Libraries,Documents,ExplorerEntries,Processing,Uploads,api,libraryTypes,AuditTable}.tsx/ts` |
| Pruebas | `internal/documentformat/format_test.go`, `internal/libraries/document_formats_test.go`, `internal/storage/document_formats_upgrade_test.go`, `web/tests/document-formats.spec.ts` y ajustes a pruebas anteriores |
| Datos sintéticos | `testdata/documents/sample.{docx,xlsx,txt,csv}`, reproducibles con `scripts/generate_multiformat_fixtures.py` |
| Compilación y transición desde r7 | `internal/buildinfo/version.go`, `scripts/build_installers.py`, `scripts/smoke_macos_installer.py`, `packaging/macos/{preinstall,postinstall}`, `packaging/ubuntu/{preinst,postinst,install-offline.sh}`, `packaging/windows/manage.ps1` |
| Documentación y CI | `README.md`, `docs/{MILESTONES,AIBID-2.0-FASE-1}.md`, `.github/workflows/h7-installers.yml` |

## Pruebas manuales recomendadas

1. En una instalación de prueba con PDF ya indexados, actualizar y comprobar búsquedas, OCR, expedientes, permisos e historial. Confirmar el respaldo previo a la migración.
2. Cargar un ejemplar propio de DOCX, XLSX, TXT UTF-8 y CSV. Clasificar, guardar en un expediente y descargar; comparar SHA-256 del origen y descarga. No debe aparecer extracción pendiente indefinidamente para estos formatos.
3. Registrar una carpeta local y una UNC con los cinco formatos. Comprobar que los nuevos se registran, los PDF se indexan y los originales permanecen intactos.
4. Probar una biblioteca que hereda todo y otra limitada a PDF. Cambiar solo el tamaño o una lista y verificar las otras herencias al recargar.
5. Probar tamaño máximo de archivo y tamaño indexable por separado; alternar conservar/rechazar archivos sin indexación.
6. Intentar un ejecutable renombrado a PDF/TXT y un ZIP renombrado a DOCX. Deben rechazarse en cargas y aparecer como omitidos en vinculadas; no deben ejecutarse ni desaparecer del disco.
7. Confirmar el servicio en Windows y Ubuntu amd64/ARM. La compilación cruzada no sustituye esas pruebas nativas.

Paquetes de esta fase: `dist/aibid-2.0-fase-1/`. Son instaladores de prueba, con la misma identidad de servicio y directorios AIBID-Test; no aíslan automáticamente los datos de una instalación de pruebas anterior. Para empezar desde cero, usar el procedimiento de reinicialización ya existente sobre datos descartables. Los instaladores r7 anteriores se conservan en `dist/installers/`.

Se incluyen archivos sintéticos en `LEEME/formatos/` para probar cargas sin utilizar documentos privados. En Ubuntu ARM se utiliza `aibid-test_2.0.0~alpha.1_arm64.deb`; `amd64` corresponde a equipos Intel/AMD. Los paquetes Ubuntu requieren las dependencias declaradas de PDF/OCR o el bundle offline preparado con el procedimiento existente.

## Validación automatizada de la fase

- `npm --prefix web run build`: TypeScript y Vite aprobados.
- `go test ./...` y `go test -tags development ./...`: suites completas aprobadas.
- `go test -race -tags development ./internal/documentformat ./internal/libraries ./internal/storage ./internal/httpapi`: aprobado.
- `go vet ./...`: aprobado.
- `npm --prefix web run test:e2e`: **12 escenarios Chrome aprobados**, incluyendo reglas globales/herencia, carga/descarga DOCX, rechazo de ejecutable disfrazado y vista móvil, además de regresiones PDF/OCR, expedientes, usuarios, licencia y LAN.
- Pruebas específicas de migración desde esquema 8: preservación de IDs, hashes, FTS, expedientes, instantánea consistente y arranque repetido sin nuevo respaldo. Pruebas de DOCX/XLSX/TXT/CSV: bytes originales conservados en carga, descarga y guardado definitivo.
- `scripts/check_format.py`, `scripts/check_design.py` (14 comprobaciones), sintaxis de scripts y `git diff --check`: aprobados.

No se añadieron dependencias Go ni frontend. La validación usa bibliotecas estándar de Go; PDF/OCR mantiene Poppler/Tesseract. Windows conserva las versiones y hashes fijados de su runtime anterior.

## Paquetes verificados

Se generaron macOS ARM, Windows amd64 y Ubuntu amd64/ARM. `SHA256SUMS` verifica todos los archivos de la entrega; `VALIDACION-FASE-1.json` identifica los paquetes y las comprobaciones realizadas.

- macOS: ejecución del binario extraído del `.pkg` con datos desechables, firma ad hoc, manifiesto, diagnóstico PDF/OCR real, administrador con contraseña de seis caracteres, login HTTP, local/LAN con reinicio, bloqueo de estado, integridad SQLite y limpieza confirmada preservando originales. También se creó una instalación temporal con el binario r7 y se actualizó con el de esta fase: respaldo del esquema 8 y conservación de usuarios/auditoría aprobados.
- Ubuntu: ambos `.deb` comprobados en formato ar/tar, arquitectura ELF amd64/arm64, control, rutas seguras, propietarios del payload y orden de migración antes del marcador. No se ejecutaron en Ubuntu.
- Windows: backend y lanzador compilados, iconos, inventario fijado y hashes PDF/OCR, cierre de dependencias PE amd64 y paquete NSIS comprobados. La herramienta de empaquetado usada localmente fue **NSIS 3.13**, descargada del bottle oficial de Homebrew y verificada por SHA-256. Solo se usa para construir; no añade una dependencia de ejecución. CI conserva su versión fijada anterior y no se ejecutó remotamente.

La prueba macOS ejecuta binarios en carpetas temporales: no certifica el instalador del sistema, launchd ni su cuenta de servicio. Windows y Ubuntu requieren las pruebas nativas indicadas arriba; también queda pendiente el ensayo offline en una VM limpia. No se modificaron la instalación real ni sus datos.

Para repetir la prueba local de actualización:

```sh
python3 scripts/smoke_macos_installer.py \
  dist/aibid-2.0-fase-1/AIBID-Pruebas-2.0.0-alpha.1-macOS-arm64.pkg \
  --previous-package dist/installers/AIBID-Pruebas-0.7.0-test.1-macOS-arm64.pkg
```

**Fase 1 terminada. Fase 2 no iniciada.**
