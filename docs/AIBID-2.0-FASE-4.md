# AIBID 2.0 — fase 4: vistas previas y caché

Entrega de pruebas **2.0.0-alpha.4-f4**, del 4 de octubre de 2026. Continúa sobre
la fase 3 existente. La fase 5 no está iniciada.

## Comportamiento

Al abrir un DOCX o XLSX se solicita su representación PDF. Mientras se genera,
el usuario puede consultar el texto extraído. Las siguientes aperturas reutilizan
la misma vista. El visor permite ampliarla y reintentar una conversión fallida.
PDF conserva el visor del original; TXT y CSV mantienen su consulta de texto.

La vista es una caché privada y descartable: no crea documentos, ubicaciones,
versiones de contenido, expedientes ni extracciones. Sus archivos se guardan en
`<StateDirectory>/previews/<id>.preview`, fuera de las carpetas vigiladas; los
trabajos temporales están en `<StateDirectory>/preview-work/`. Se mantiene el
rechazo de raíces que se solapen con el estado privado de AIBID.

Cada entrada relaciona documento, versión de contenido, SHA-256 del original,
versión del generador, fecha de generación y último acceso. El generador incluye
la revisión del adaptador y una huella de la ruta, tamaño y fecha de modificación
del ejecutable LibreOffice. Cambiar el hash o actualizar/configurar el conversor
invalida las representaciones anteriores. El contenido se comprueba otra vez
antes de publicar y servir una vista. Cambios externos aún no detectados por el
recorrido requieren **Verificar biblioteca ahora**, como el resto del índice.

Estados: No generada, Pendiente de generación, Generando, Disponible, Obsoleta,
Error y No disponible. Los errores no cambian la validez del documento ni el
resultado de extracción; permiten consultar el texto y descargar el original
cuando el usuario tiene permiso.

## Dependencia de conversión

**Instalar LibreOffice en el equipo servidor de AIBID para generar las vistas de
Word y Excel.** Es opcional para el resto del sistema y no se incluye en estos
instaladores. No se necesitan Microsoft Office, servicios externos ni conexión a
un conversor remoto. Se conserva el extractor estático de texto existente.

Se eligió PDF mediante LibreOffice para mostrar tablas, imágenes y disposición
de página de ambos formatos con un mismo visor. Una vista HTML de solo lectura
habría evitado la dependencia, pero no habría conservado esos elementos de diseño.
La representación puede variar respecto de Microsoft Office por las fuentes
instaladas, los saltos de página y las opciones de impresión del libro.

| Sistema | Preparación y detección |
|---|---|
| Windows | Instalar LibreOffice para todos los usuarios. Se busca `LibreOffice\program\soffice.com` en Program Files. Si está en otra ruta, indicarla en Vistas previas. La cuenta del servicio necesita ejecutar el programa. |
| macOS ARM | Instalar LibreOffice para Apple Silicon en `/Applications/LibreOffice.app`. Se detecta `Contents/MacOS/soffice`. |
| Ubuntu 24.04 amd64/ARM | `sudo apt update` y `sudo apt install libreoffice-writer libreoffice-calc`; se busca `libreoffice` en PATH. |

Los paquetes Ubuntu declaran Writer y Calc como `Suggests`; no se instalan
obligatoriamente al actualizar AIBID. En equipos sin Internet, preparar también
estos paquetes y sus dependencias para la arquitectura destino; el bundle OCR
anterior no contiene LibreOffice. En macOS/Windows obtener su instalador oficial
para la arquitectura del servidor. Descargar desde
[LibreOffice](https://www.libreoffice.org/download/download-libreoffice/).
Los paquetes [Writer](https://packages.ubuntu.com/noble/libreoffice-writer) y
[Calc](https://packages.ubuntu.com/noble/libreoffice-calc) están disponibles para
las dos arquitecturas de Ubuntu utilizadas por AIBID.

En **Administración → Vistas previas** se muestra si está detectado. El campo de
ruta opcional acepta la ruta absoluta al ejecutable, sin argumentos. No hace falta
reiniciar AIBID después de configurar la ruta. Si faltaba el conversor y ya se
instaló, volver a abrir el documento o pulsar **Reintentar vista previa**.

## Límites y limpieza

| Ajuste | Predeterminado | Rango |
|---|---:|---:|
| Vistas previas habilitadas | Sí | Sí/No |
| Tamaño máximo de caché | 512 MB | 1–20480 MB |
| Antigüedad máxima sin uso | 30 días | 1–365 días |
| Tamaño máximo del original | 64 MB | 1–512 MB |
| Limpieza automática por antigüedad | Sí | Sí/No |

Los tamaños de la configuración se calculan con 1024×1024 bytes por MB. El límite
se refiere a PDF guardados; el trabajo temporal de la conversión se contabiliza
aparte. Se admite una conversión simultánea, hasta 32 solicitudes pendientes o
en curso, 120 segundos por conversión y un PDF de hasta 64 MB. La validación de
OOXML conserva los límites de expansión y XML existentes; la copia de conversión
limita cada parte a 32 MB.

LRU utiliza el **último acceso efectivo al PDF**, no su fecha de generación.
Al necesitar espacio se retiran primero las vistas menos recientemente usadas.
La limpieza por antigüedad se comprueba al iniciar, cada hora, al guardar ajustes
y al publicar nuevas vistas. Deshabilitar esa limpieza conserva la obligación de
respetar el tamaño máximo total. La pantalla muestra bytes usados y vistas listas.

**Vaciar caché de vistas previas** requiere confirmación y permiso de configuración.
Elimina vistas y solicitudes, incluidas las que ya se están convirtiendo: una
conversión activa no puede volver a publicar una entrada retirada. No borra
originales, texto indexado, expedientes ni auditoría. Si Windows mantiene un PDF
abierto y no permite retirarlo, se informa del fallo; cerrar la vista y reintentar.
Una interrupción no publica resultados parciales; al reiniciar se limpia trabajo
temporal propio y se recuperan entradas o archivos de caché huérfanos.

## Aislamiento y permisos

Hay un trabajador independiente de las colas de extracción y OCR. La pausa global
impide tomar nuevas conversiones y deja terminar las que ya comenzaron. El proceso
se cancela al cerrar el servicio; dispone de límite de tiempo y vigilancia del
tamaño de salida. Se finaliza su grupo de procesos en macOS/Linux y su árbol en
Windows. Nunca se ejecuta una línea de shell construida con nombres de documentos.

La conversión recibe una copia privada verificada por SHA-256 y un perfil temporal
exclusivo. Las macros, contenido activo, OLE y actualización de enlaces/campos están
deshabilitados. Además se retiran de la copia las relaciones externas, instrucciones
de campos, referencias activas y fórmulas, preservando sus resultados guardados.
No se recalculan fórmulas; si el Excel no contiene el valor guardado, puede verse
vacío. Para actualizar esos valores, guardar el libro recalculado en Excel y
verificar la biblioteca. La copia y el perfil se eliminan al terminar.

El aislamiento es por proceso y directorio privado bajo la cuenta del servicio;
no es un contenedor con bloqueo de red ni un límite duro de memoria del sistema
operativo. El tiempo y los límites de entrada/salida reducen el alcance de una
conversión compleja, pero su consumo puede variar. Mantener LibreOffice actualizado.
No se registra el contenido ni la salida de diagnóstico del conversor.

Cada consulta y entrega valida sesión, permiso de lectura, visibilidad de cargas
privadas, usuario habilitado, licencia de lectura, documento y hash vigente. El
identificador de una vista no concede acceso por sí mismo. Crear una vista usa
CSRF y lectura documental; no modifica el expediente y funciona bajo la licencia
en modo de consulta. La configuración requiere `system.configure`, revisión
optimista y auditoría. El PDF se entrega con `no-store`, MIME fijo, `nosniff` y CSP
sandbox, sin exponer una carpeta estática pública de caché.

Los parámetros de LibreOffice siguen su
[documentación de ejecución](https://help.libreoffice.org/latest/en-GB/text/shared/guide/start_parameters.html)
y los esquemas oficiales de
[seguridad Common](https://raw.githubusercontent.com/LibreOffice/core/master/officecfg/registry/schema/org/openoffice/Office/Common.xcs)
y [actualización Writer](https://raw.githubusercontent.com/LibreOffice/core/master/officecfg/registry/schema/org/openoffice/Office/Writer.xcs).
Sus licencias son independientes de AIBID; ver [avisos de LibreOffice](https://www.libreoffice.org/licenses/)
y `THIRD-PARTY.md` en los paquetes.

## Migración y distribución

Migración aditiva **0012_preview_cache**: tablas de configuración y caché. Conserva
identidades, usuarios, licencia, originales, FTS, extracción, expedientes y auditoría.
Se crea una instantánea SQLite consistente anterior a la migración en
`<StateDirectory>/upgrade-backups/before-processing-*.db`; incluye WAL confirmado.
Las pruebas cubren actualizaciones desde esquemas 8, 9, 10 y 11 y verifican sus
asociaciones, hashes e índice. No usar un binario anterior con la base migrada;
el rollback automático sigue limitado a instalaciones vacías.

Los instaladores admiten la versión de pruebas 0.7.0 y las fases 1, 2 y 3, además
de reinstalar esta fase. Mantienen el canal **AIBID Pruebas**, el estado y el puerto
configurado, y migran antes de marcar la nueva versión y arrancar el servicio.
Se generan juntos en `dist/aibid-2.0-fase-4/`: macOS ARM, Windows amd64, Ubuntu
amd64 y Ubuntu ARM. Se conservan los paquetes anteriores. No se crea instalador
macOS Intel. Los paquetes no incluyen documentos privados suministrados por el usuario.

## Archivos de esta fase

| Área | Archivos principales |
|---|---|
| Conversión aislada | `internal/previews/{converter,sanitize,process_unix,process_windows}.go` |
| Caché y trabajador | `internal/libraries/{previews,preview_settings,preview_worker,runtime,service}.go` |
| API | `internal/httpapi/{previews,server}.go` |
| Migración | `db/migrations/0012_preview_cache.{up,down}.sql`, `internal/storage/{database,upgrade_backup}.go` |
| Interfaz | `web/src/{DocumentPreview,PreviewSettings,Documents,App,AuditTable}.tsx`, `styles.css` |
| Pruebas | `internal/previews/converter_test.go`, `internal/libraries/previews_test.go`, `internal/httpapi/previews_test.go`, `internal/storage/{database,document_formats_upgrade}_test.go`, `web/tests/{previews,document-extractors}.spec.ts` |
| Distribución | `internal/buildinfo/version.go`, `scripts/{build_installers,smoke_macos_installer}.py`, preinstalación macOS/Ubuntu, `packaging/windows/manage.ps1`, `.github/workflows/h7-installers.yml`, `packaging/THIRD-PARTY.md` |

## Validación

Las pruebas finales y la evidencia de los cuatro paquetes se registran en
`dist/aibid-2.0-fase-4/VALIDACION-FASE-4.json` junto a `SHA256SUMS`.

Se verificaron los tres archivos reales con LibreOffice **26.2.6 macOS ARM**,
ejecutado temporalmente fuera de Aplicaciones: Word de Famisa y ambos Excel
generan PDF válido con texto legible. Sus hashes originales permanecen idénticos.
El instalador oficial utilizado se verificó por SHA-256:
`94bb3248df074c225490a8a6d1d9dc87c7d6783dbb7a8e9f0d0c3d94348552af`.
La validación local del conversor se puede repetir opcionalmente con
`AIBID_TEST_LIBREOFFICE` y `AIBID_LOCAL_DOCUMENTS`; estos documentos no se incorporan
a fixtures ni se publican en el informe.

Pruebas de servicio: demanda/deduplicación, LRU con acceso reciente, caducidad,
recuperación de archivos ausentes, limpieza durante generación, límites de entrada
y salida, invalidación por hash/generador, pausa, permisos y cargas privadas,
independencia respecto del índice, HTTP/CSRF y lectura PDF por rangos.
Pruebas del conversor: eliminación de enlaces externos/instrucciones/fórmulas,
preservación del texto guardado, rechazo XML inseguro, original intacto y cancelación.

## Comprobación en los equipos destino

1. Actualizar AIBID conservando datos y configuración. Instalar LibreOffice para
   la arquitectura del equipo servidor y comprobar **Vistas previas → LibreOffice detectado**.
2. Abrir un DOCX y un XLSX conocidos. Ver Generando y luego el PDF; comprobar logo,
   tablas y páginas. Ampliar la vista; cerrar y abrir para verificar reutilización.
3. Confirmar que el documento sigue siendo uno solo en el expediente y que texto,
   descarga original y búsquedas permanecen disponibles.
4. Cambiar un original vinculado, verificar su biblioteca y reabrirlo. La vista
   debe regenerarse. Si Excel no guardó resultados de fórmulas, recalcularlo primero.
5. Reducir el máximo de caché para comprobar LRU y uso mostrado. Vaciar la caché
   y confirmar que abrir un documento vuelve a generar su vista.
6. Deshabilitar vistas o configurar temporalmente una ruta de conversor inexistente.
   El diagnóstico debe ser claro y la extracción/búsqueda debe seguir funcionando.
7. Probar con un lector sin acceso a una carga privada: ni ficha ni vista deben
   permitir consultar su contenido.

La instalación del servicio nativo en Windows y Ubuntu, y la del paquete completo
mediante el instalador de macOS/launchd, requieren comprobación en esos entornos.
La prueba local del binario macOS empaquetado no equivale a instalar un servicio.

**Fin de fase 4. Fase 5 pendiente de autorización.**
