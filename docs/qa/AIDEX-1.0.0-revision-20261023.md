# AIDEX 1.0.0 — revisión funcional e instalador Windows de pruebas internas

Fecha: 23 de octubre de 2026. Alcance: código corregido de AIDEX 1.0.0 y paquete Windows AMD64 generado **después** de las correcciones. No se reutilizó el instalador anterior ni se modificó el servidor de licencias o información de una instalación real.

## A. Correcciones funcionales

- **Reconciliaciones repetidas.** El programador calculaba el vencimiento principalmente desde el último recorrido completado. Un recorrido fallido o parcial dejaba esa fecha antigua y cada ciclo del programador podía volver a encolar otro trabajo; además, el disparo periódico no se distinguía de uno manual o del watcher. Ahora los trabajos periódicos conservan su origen en `payload_json`, la última programación se consulta de forma persistente y se verifica otra vez dentro de la transacción de encolado. El mismo origen respeta el intervalo incluso después de un reinicio o fallo, sin bloquear solicitudes manuales futuras. La prueba de intervalo diario y fallo repetido comprueba que no se produzcan cientos de encolados.
- **Eliminación definitiva.** La implementación anterior hacía la eliminación y su limpieza dentro del recorrido de la solicitud HTTP; con historial y temporales abundantes podía exceder el tiempo del cliente, dejando ambiguo si había terminado. Ahora la confirmación deja un trabajo persistente y responde `202`; el cliente consulta `pending`, `running`, `complete` o `failed`. Una restricción impide dos operaciones activas sobre la misma entidad. La eliminación de registros se realiza en transacción; la limpieza de temporales tiene diario y recuperación al reiniciar. Un recorrido, vista previa, carga o materialización pendiente bloquea el retiro para evitar carreras, y el watcher/programador no vuelve a encolar trabajo para una raíz cuyo retiro está pendiente. El manifiesto distingue temporales eliminables de documentos físicos originales y cargas clasificadas que deben preservarse. La prueba cubre error a mitad de la transacción, reintento, recuperación y conservación de originales. El timeout concreto observado en Windows no tenía una traza del servidor disponible; la ruta síncrona era el riesgo comprobable en el código.
- **Pausa y cancelación.** `Pausar` conserva el punto de avance y permite reanudar. `Cancelar definitivamente` termina el recorrido, cancela trabajos en cola del mismo recorrido, registra auditoría y libera la condición operativa después de detener el trabajo activo. No retira documentos ya indexados ni originales. Pruebas de reinicio confirman que el recorrido cancelado no reaparece como reanudable y que se puede archivar la biblioteca cuando no hay otras operaciones activas.

## B. Interfaz

- La superficie principal de la tarjeta de biblioteca activa abre la biblioteca mediante botón accesible, con foco y hover; un botón de tres puntos independiente contiene las acciones permitidas de archivar, restaurar y eliminar. Las archivadas no abren su contenido.
- `Eliminaciones y limpieza` usa el patrón de acordeón de trazabilidad. La cola, el historial y las omisiones presentan **10 registros iniciales**, con límite correspondiente en la consulta paginada del backend.
- Se ajustaron navegación y formularios de 360 px a escritorio, el texto de inicio de sesión y `Acerca de AIDEX`. El modal del contrato permite leer el texto completo y cerrarlo sin aceptar por accidente.
- La confirmación de reindexación se muestra en la bandeja global de notificaciones. Una prueba antigua esperaba esa confirmación dentro de la ficha; se corrigió la aserción y la operación pasó.
- Se corrigió el campo de clave de licencia: el límite HTML de 18 caracteres podía impedir que un código largo pegado sustituyera uno válido anterior, dejando habilitada la activación con la clave equivocada. El campo ahora deja pasar el texto al validador, que lo rechaza y muestra el error sin contactar al servidor.

## C. Estadísticas documentales

El apartado `Estadísticas` consulta expedientes, requisitos obligatorios vigentes, revisión/aprobación, disponibilidad e integridad de los documentos. Informa totales, completos, incompletos, sin documentos, sin plantilla evaluable, requisitos cumplidos/pendientes, documentos no disponibles, faltantes frecuentes y desgloses por plantilla y ejercicio. El porcentaje de expedientes completos es `completos / evaluables × 100`; el avance documental es `requisitos cumplidos / requisitos obligatorios esperados × 100`. Si el denominador es cero se muestra ausencia de dato, no un 100 % ficticio. Las versiones de una misma plantilla se agrupan bajo la plantilla. Se añadieron índices de consulta; las estadísticas se calculan a demanda y no crean otro programador periódico.

El ejercicio se valida como cuatro dígitos y la interfaz propone ejercicios existentes. Los filtros de la vista se aplican también a los totales, desgloses y faltantes.

## D. Contrato de licencia

Se integró íntegramente el texto contractual de `AIDEX_CLUF_v1.0_BORRADOR.md` en `internal/httpapi/eula-draft.txt`, excluyendo solo la nota de elaboración que el propio archivo indica no mostrar al usuario. Su identificador es `1.0-borrador-2026-10-22`; la huella del texto, versión, fecha, instalación y usuario se conservan en el historial de aceptación. Aceptar el borrador anterior no satisface este contrato. El uso documental queda bloqueado hasta la aceptación vigente; `Acerca de AIDEX` permite consultarlo también sin licencia activa. Siguen pendientes de revisión jurídica la razón social, el domicilio y el contacto del licenciante. El instalador **no** presenta una página legal propia: la aceptación obligatoria ocurre en el primer acceso al sistema.

## E. Verificaciones

- `npm run build`: TypeScript y Vite correctos. Vite informa un aviso de tamaño de un chunk, sin fallo de compilación.
- `go test ./internal/... -count=1 -timeout=180s`: pasó en modo normal; incluye exigencia del CLUF.
- `go test -tags development ./internal/... -count=1 -timeout=180s`: pasó. Las dos pruebas de bloqueo contractual se ejecutan en modo normal porque el modo de desarrollo omite ese bloqueo deliberadamente.
- `AIDEX_LARGE_TEST=1 go test -tags development ./internal/libraries -run ^TestLargeLibrary9000$`: pasó con 9 000 archivos sintéticos, interrupción/reanudación y una segunda pasada incremental; no ejercita OCR de 9 000 archivos.
- `go vet ./internal/...`, `git diff --check` y validación de PowerShell de transición Windows: pasaron.
- Navegador: **27 de 27 casos pasaron** en la corrida final, incluidos licencia simulada, CLUF, eliminación, estadísticas, red, filtros, reindexación y diseño adaptable. Durante las corridas previas se corrigieron la aserción antigua de notificaciones y el rechazo de códigos de licencia largos pegados.

Se usaron datos sintéticos y servidores locales efímeros. No se hicieron activaciones comerciales reales.

## F. Instalador Windows AMD64

- Archivo: `dist/aidex-1.0.0-revision-20261023-r2-windows/AIDEX-1.0.0-Windows-amd64.exe`
- Tamaño: **40 394 204 bytes**.
- SHA-256: `fbddfee7a2d2df04dcf1aa93bb0b53070ef3d8bcd35642b3b76d3c9eedb8443b`.
- `SHA256SUMS` verificó todos los archivos de salida. NSIS generó un EXE Unicode en español. Se comprobaron recursos de icono de ocho tamaños tanto en servidor como en lanzador, arquitectura PE AMD64, cierre de dependencias DLL, OCR español/inglés y sintaxis/codificación de scripts PowerShell. La compilación usa el canal normal de AIDEX y la clave pública de verificación; no empaqueta claves privadas ni códigos comerciales.
- El instalador detecta los servicios anteriores `AIBID`, `AIBIDTest` y `AIBID-Test` únicamente para **detener una transición potencialmente destructiva**. Estas referencias son deliberadas y necesarias para compatibilidad; el servicio, rutas y marca nuevos son `AIDEX`. No elimina servicios o datos anteriores automáticamente.
- La actualización de una instalación AIDEX compatible conserva `%ProgramData%\AIDEX`. La desinstalación conserva datos por defecto; borrar datos internos exige casilla y confirmación adicional. Los originales físicos no forman parte de esa limpieza.
- Guía de instalación y prueba: `dist/aidex-1.0.0-revision-20261023-r2-windows/PRUEBAS-WINDOWS-1.0.0.md`. La carpeta `revision-20261023-windows` es un artefacto intermedio anterior a la corrección del campo de licencia y **no debe instalarse**.

**Límite:** en macOS se verificó la compilación cruzada y la estructura del paquete, pero no se ejecutó el EXE. Falta comprobar en Windows 11 real instalación, actualización sobre la instalación actual, inicio con la cuenta de servicio, ACL, firewall/LAN, OCR, aceptación del CLUF y desinstalación conservando datos.

## G. Cambios técnicos y siguiente revisión documental

Se añadió la migración `0017_deletion_execution` para estados de ejecución y consultas de estadísticas. Los endpoints de eliminación devuelven estado consultable tras la confirmación; se añadieron endpoints de estadísticas y ejercicios. Los componentes principales de esta revisión son `Lifecycle`, `CaseStatistics`, `CompactTables`, `About` y `Eula`. El Manual de usuario debe reflejar la separación entre pausar/cancelar, el retiro asíncrono, el cálculo de ambos porcentajes, las acciones de las tarjetas, el nuevo CLUF y los pasos de actualización en Windows.

El artefacto está destinado **solo a pruebas internas**. Antes de considerar una distribución comercial faltan la revisión jurídica del CLUF, firma del instalador y validación nativa en Windows 11. No se generaron paquetes Ubuntu/macOS ni etiquetas de lanzamiento.
