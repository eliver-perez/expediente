# AIBID 2.0 — fase 5: búsqueda, reindexación y reconciliaciones simultáneas

Entrega de pruebas **2.0.0-alpha.5-f5**, del 5 de octubre de 2026. Continúa sobre
la fase 4 y añade la petición de reconciliar carpetas distintas simultáneamente.
La fase 6 no está iniciada.

## Búsqueda por formato

**Formato de archivo** está disponible en Buscar documentos y en Documentos de
cada biblioteca, tanto al explorar como al buscar contenido. Su valor inicial
es **Mostrar todo**; permite seleccionar PDF, Word (DOCX), Excel (XLSX), Texto
(TXT) y CSV. Se combina con biblioteca, raíz, vista lógica, carpeta, disponibilidad
y los filtros documentales existentes. Los resultados, totales, agrupaciones y
paginación utilizan el mismo filtro; un cursor no puede reutilizarse con otro
formato. Las búsquedas auditadas conservan también este criterio.

Se utiliza el formato detectado del contenido. Los resultados muestran una
etiqueta explícita de formato junto al icono SVG. El contexto sigue las unidades
reales de extracción: página PDF, párrafo/tabla/parte DOCX, hoja y celda XLSX,
y fila/columnas CSV. No se inventan páginas o secciones de Word. La ficha mantiene
formato detectado, extractor, versión, fecha, resultado, advertencias y contadores.

Las carpetas de navegación permanecen visibles aunque no contengan resultados
del formato seleccionado. Los permisos y la privacidad de las cargas temporales
se aplican antes de contar, agrupar o devolver documentos.

## Reindexar documento

La ficha incorpora **Reindexar documento** para usuarios con lectura e
`indexing.run` en esa biblioteca. Requiere una licencia que permita modificar e
indexar documentos. No admite documentos cancelados, retirados ni en proceso de
guardado definitivo; las cargas privadas mantienen sus restricciones de acceso.

1. Comprueba acceso al original, tipo detectado, tamaño, identidad física y SHA-256.
   La lectura del archivo se realiza fuera de la transacción de escritura de SQLite.
2. Revalida usuario, permisos, reglas de archivo, revisión del documento y raíz.
   Si la configuración ya no permite indexar el formato/tamaño, informa del motivo.
3. Encola extracción usando los extractores y parámetros vigentes. Solicitudes
   simultáneas sobre la misma versión reutilizan el trabajo pendiente o activo;
   repetir una solicitud con su clave de idempotencia no genera otro trabajo.
4. El trabajador vuelve a comprobar el original mediante una copia privada y publica
   el nuevo índice de forma atómica. Mientras trabaja, y si falla, conserva el texto
   anterior. Mantiene las extracciones históricas.
5. La ficha muestra el estado y actualiza texto e información de procesamiento al
   terminar. Se puede cerrar y consultar el trabajo en Procesamiento.

Si el contenido cambió pero sigue siendo el mismo archivo físico, registra una
nueva versión dentro del mismo documento, sin cambiar sus identificadores ni
perder el expediente. Un original administrado modificado queda **Requiere revisión**
si estaba aprobado y conserva el aviso de integridad. Si el sistema operativo
indica que el original fue reemplazado por otro archivo, solicita verificar la
biblioteca primero. Una carga temporal alterada tampoco se acepta silenciosamente.

La acción registra **Reindexación solicitada** con responsable, documento, trabajo,
versión y huella comprobada. No modifica los bytes originales, no crea copias
documentales y no cambia los permisos. Una reindexación del mismo hash no invalida
la vista previa PDF; si cambia el contenido, la caché de fase 4 reconoce el nuevo hash.

## Reconciliaciones simultáneas

La espera anterior se debía a un único trabajador de recorridos. Ahora hay un
grupo independiente con **hasta cuatro reconciliaciones de raíces distintas** a la
vez. Al añadir otra carpeta, puede comenzar sin esperar al recorrido anterior,
si queda un turno libre. Las demás esperan en cola. El límite incluye la revisión
de raíces administradas y es independiente del límite de extracción/OCR.

Se conservan la exclusión por raíz y la coalescencia existentes en la base de datos:
una raíz no acumula dos trabajos pendientes/activos por pulsar repetidamente
Verificar biblioteca. Los eventos nuevos ocurridos durante un recorrido pueden
solicitar una comprobación posterior, nunca otra simultánea sobre esa raíz.
Los límites globales, pausa, cancelación, recuperación tras reinicio y protección
frente a trabajadores antiguos siguen aplicándose. La configuración del equipo
explica ahora este límite de cuatro carpetas.

## Actualización y paquetes

Se mantiene el **esquema 12** de la fase 4; esta fase no necesita migración ni
reconstrucción de la base de datos. Usuarios, licencia, originales, expedientes,
texto e historial se conservan. Las versiones anteriores siguen las migraciones
aditivas existentes con respaldo previo cuando corresponde. Actualizar desde
fase 4 no crea una migración ni un respaldo de migración ficticio.

Paquetes en `dist/aibid-2.0-fase-5/`, canal AIBID Pruebas:

- Windows amd64: `AIBID-Pruebas-2.0.0-alpha.5-f5-Windows-amd64.exe`.
- macOS Apple Silicon: `AIBID-Pruebas-2.0.0-alpha.5-macOS-arm64.pkg`.
- Ubuntu amd64: `aibid-test_2.0.0~alpha.5_amd64.deb`.
- Ubuntu ARM: `aibid-test_2.0.0~alpha.5_arm64.deb`.

No se genera macOS Intel. Se conservan las entregas anteriores. Los documentos
privados del usuario no se incluyen en los paquetes. Las vistas previas de Word y
Excel mantienen su dependencia opcional de LibreOffice; consultar
[AIBID-2.0-FASE-4.md](AIBID-2.0-FASE-4.md) para su instalación.

## Archivos principales

| Área | Archivos |
|---|---|
| Filtros y permisos | `internal/libraries/{search,documents}.go`, `internal/httpapi/library_handlers.go` |
| Reindexación | `internal/libraries/reindex.go`, reutiliza observación y extracción existentes |
| Concurrencia | `internal/libraries/runtime.go`, `reconciliation_test.go` |
| Interfaz | `web/src/{FormatOptions,ReindexDocument,Search,Libraries,Documents,ProcessingSettings,AuditTable}.tsx`, `libraryTypes.ts` |
| Pruebas | `internal/libraries/{reindex,search_formats}_test.go`, `internal/httpapi/reindex_test.go`, `web/tests/search-reindex.spec.ts` |
| Distribución | `internal/buildinfo/version.go`, `scripts/{build_installers,smoke_macos_installer}.py`, preinstalación por plataforma y flujo CI |

## Validación

Las pruebas específicas verifican solicitudes simultáneas, idempotencia, permisos,
licencia de solo lectura, originales ausentes, cambio de contenido administrado,
conservación de expediente e historial, reemplazo de FTS, cancelación con texto
retenido, contexto de resultados, privacidad, totales y cursores filtrados.

La prueba del runtime inicia el grupo de producción y mantiene sus recorridos
ocupados de forma controlada: comprueba cuatro raíces reclamadas, una quinta en
cola y una sola solicitud sobre la primera raíz. Al liberarlas, verifica recorrido
y extracción de las cinco. No depende de crear una carpeta artificialmente enorme
ni de que la red sea lenta.

La prueba de navegador cubre las dos pantallas de búsqueda, formatos, contexto
XLSX, reindexación completa, actualización de ficha, auditoría y vista móvil.
Resultados finales y comprobación de paquetes:
`dist/aibid-2.0-fase-5/VALIDACION-FASE-5.json` y `SHA256SUMS`.

La instalación de servicios nativos Windows y Ubuntu debe comprobarse en esos
equipos. La prueba local del binario macOS empaquetado verifica inicialización,
actualización y arranque, y no equivale a instalar el servicio completo con launchd.

## Comprobación en los equipos destino

1. Actualizar sobre fase 4 conservando los datos. Confirmar usuarios, licencia,
   bibliotecas y documentos anteriores.
2. Seleccionar cada formato en Buscar documentos y en una biblioteca. Combinarlo
   con búsqueda por contenido y disponibilidad; revisar totales y contexto.
3. Abrir un documento ya extraído y pulsar Reindexar documento. Confirmar nuevo
   procesamiento, mismo documento, expediente e historial, y original intacto.
4. Probar con un original temporalmente inaccesible: debe informar del error y
   seguir permitiendo consultar el texto retenido.
5. Mientras se recorre una carpeta grande, añadir otra raíz distinta. Comprobar
   que ambas avanzan en Procesamiento. Pulsar varias veces Verificar sobre la
   primera: no debe iniciar recorridos simultáneos duplicados de esa carpeta.
6. Añadir más de cuatro raíces: comprobar que el resto espera su turno y continúa
   al liberarse capacidad. Verificar pausa, reanudación y reinicio del servicio.

**Fin de fase 5. Fase 6 pendiente de autorización.**
