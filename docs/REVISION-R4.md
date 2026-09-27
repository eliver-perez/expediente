# Revisión r4 — diagnóstico y decisiones (2026-09-27)

Antes de modificar se revisaron extracción, runtime, cola y reintentos, SQLite,
buscador FTS, explorador, visor y contrato/cliente de licencias V1.

## Hallazgos

- r3 ya separa reconciliación de contenido; el contenido aún usa un worker por
  defecto. Dentro de cada PDF se invoca pdftotext por página y después OCR en la
  misma ejecución. Crear procesos y volver a abrir el PDF por página añade coste.
- La copia privada con hash verifica la versión y protege frente a modificaciones
  del original; se conserva. Poppler/Tesseract ya son procesos externos cancelables
  y Tesseract usa un hilo. No es necesario crear otro servicio de colas.
- SQLite mantiene WAL, lectores separados y un escritor; las transacciones de
  publicación son atómicas. La búsqueda actual retiene el escritor mientras lee
  y arma fragmentos; se separarán lectura consistente y auditoría breve.
- La búsqueda ya ofrece filtros, FTS, permisos, totales y cursores. Se reutilizará
  para grupos por biblioteca y búsqueda interna. El visor es un iframe autorizado;
  ampliar con CSS permite conservar la misma instancia y su página/zoom.
- El licenciamiento ya implementa activación, renovación, desactivación y archivos
  offline, Ed25519/JWS, módulos y tolerancia. Falta provisionar el origen y la
  clave pública proporcionados, preservando configuración y claves existentes.

## Diseño de procesamiento

Extracción nativa por lotes de páginas, grupo de extracción y grupo de OCR con
límites separados y un límite global de operaciones intensivas. Transferencia
acotada de documentos preparados entre ambos grupos; una sola tarea/intento y
fence conservados hasta publicar. Las instantáneas permanecen privadas, no se
vuelve a leer por red al pasar a OCR. Tras un cierre, la recuperación existente
reintenta solamente las tareas interrumpidas (máximo cinco intentos).

Configuración automática conservadora según CPU disponible, núcleos físicos
cuando se conocen y memoria; configuración manual administrativa persistente.
No se promete un porcentaje exacto de CPU: el límite controla concurrencia.
No se harán pruebas de carga masivas; se verificarán escenarios funcionales.

La conexión HTTPS por sí sola no certifica activación ni estado de la clave.
Las pruebas reales usarán una instalación temporal y la licencia facilitada,
sin claves comerciales o privadas en código, documentación ni paquetes.

## Resultado de implementación

- `pdftotext` procesa lotes de hasta 32 páginas por proceso; conserva límites de
  salida, páginas y timeout. Separadores de página preservan páginas vacías. El
  criterio existente de menos de 32 caracteres alfanuméricos para OCR no cambia.
- Grupos Go de extracción y OCR ejecutan los programas externos existentes. La
  cola de transferencia admite dos documentos por worker OCR configurado al
  iniciar. El límite global restringe operaciones de contenido activas; existe
  además un worker de descubrimiento/verificación. El OCR reserva capacidad para
  no quedar bloqueado por nuevas extracciones. La cola acotada aplica contrapresión.
- Un trabajo mantiene el mismo intento y fence al pasar a OCR. El timeout de una
  hora se aplica a cada etapa activa; la espera entre etapas no consume ese tiempo.
  Cancelación, cinco intentos, publicación atómica y recuperación existentes se
  conservan. Al reiniciar se eliminan solo instantáneas privadas huérfanas; las
  tareas interrumpidas se reintentan, no se descarta el índice ni la cola persistida.
- Recursos: núcleos físicos cuando se conocen, CPU lógicas/afinidad, memoria total
  y disponible; límites cgroup v2 en Linux y Job Object accesible en Windows.
  macOS usa sysctl/vm_stat. La lectura se realiza al iniciar. Si faltan datos se
  eligen límites conservadores. Se reservan aproximadamente un tercio de los
  núcleos y la mitad de la memoria disponible; máximo automático 6 operaciones de
  contenido y 3 OCR. No equivale a una cuota exacta de CPU o RAM.
- **Procesamiento** en el menú administrativo: automático o manual (extracción
  1–8, OCR 1–4, general 1–8). Cambios persistentes y auditados, revisión optimista,
  aplicación a nuevos trabajos en unos segundos sin interrumpir los activos.
- Estadísticas por biblioteca: workers activos, preparados para OCR, cola, media
  de documentos/minuto y duración media de trabajos terminados durante los últimos
  cinco minutos. La duración incluye espera; la tasa divide siempre por cinco.
  Los agregados se calculan en SQLite, sin cargar todos los documentos en Go.
- Búsqueda usa lectores SQLite y mantiene autorización, FTS, filtros, fragmentos,
  orden por ID, totales reales y cursores ligados a la consulta. La auditoría usa
  una transacción breve posterior. No se anidan conexiones del pool de lectura.
  `summary_only` obtiene únicamente grupos/totales sin cargar documentos.
- Resultados por biblioteca con paginación/scroll independientes, 10/25/50,
  colapso sin perder estado; búsqueda interna por nombre, expediente y contenido
  con filtros y alcance de carpeta. Explorador alterno con iconos, breadcrumbs y
  carpeta superior; conserva la ubicación al cambiar de vista y no mueve archivos.
- Ampliación del PDF mediante CSS sobre el mismo iframe: no cambia URL ni crea
  otro visor, conserva su estado y mantiene los endpoints y permisos originales.

## Licenciamiento real y actualización

Origen centralizado: `license.server_url`. Nuevas instalaciones usan
`https://aibid.adariel.com`, refresh de 24 horas y `lic-prod-2026-a` con la clave
pública Ed25519 proporcionada. No se consultan endpoints de claves ni se confía
en claves incluidas en una licencia. Solo se admiten claves registradas localmente.

El comando de mantenimiento `configure-license --config RUTA` agrega esta clave
sin eliminar otras, detecta colisiones de `kid`, rellena el origen si estaba vacío,
deshabilita bypass y valida la configuración completa. Conserva un origen ya
configurado (permite cambiar de dominio centralmente). Respalda el JSON antes de
reemplazarlo de forma atómica, bajo el mismo bloqueo exclusivo del servicio.
La presencia de CA/claves de desarrollo requiere corregir la configuración;
el binario de producción no las acepta. La instalación Windows r4 ejecuta este
paso conservando la cuenta del servicio y los datos. Los instaladores nuevos ya
no compilan con `-tags development`; no contienen claves comerciales ni privadas.

La nueva licencia de prueba pasó con un cliente **sin build development**:
activación, desafío/prueba Ed25519, JWS firmado con la clave configurada, vínculo
con instalación/huella, modalidad perpetua y los cinco módulos; renovación,
reimportación de `.lic`, generación de `.licreq` firmada y desactivación confirmada.
La plaza quedó libre. Esto prueba que el servidor emite firmas válidas con la clave;
no es una inspección del campo `staged/active` en su panel administrativo.

La primera clave recibida devolvió `LICENSE_NOT_FOUND`; no se alteró su texto ni
se intentó adivinar variantes. Ninguna clave comercial se registra en el proyecto.

Migración aditiva 0007: ajustes e índice de tiempos. Se crea una copia SQLite antes
de actualizar una base existente. Migraciones 0001–0006 y contrato V1 conservados.
Windows r4 se instala sobre r3 sin desinstalar; requiere activar la licencia desde
la interfaz. No volver a ejecutar r3 contra el esquema nuevo. La base de desarrollo
del propietario (puerto 8090) no se modifica.

## Validación y límites

- PDF nativo y OCR español reales; 34 páginas generadas con páginas vacías y cruce
  del lote de 32: se conservan texto, numeración y selección de OCR.
- Seis documentos reales: cuatro nativos terminan mientras un OCR está bloqueado
  controladamente y otro espera; reinicio recupera ambos OCR sin duplicación.
- Detección real de recursos en macOS y selección con recursos desconocidos,
  poca memoria y CPU limitada; configuración persistente tras reinicio.
- Búsqueda agrupada, totales, filtros combinados, aislamiento de cursores y
  permisos, búsquedas simultáneas, integridad SQLite y fencing.
- Navegador: bibliotecas con 13 documentos cada una; paginación/scroll independientes,
  colapso, filtros internos, cambio de vista, mismo iframe ampliado y configuración.
- Suite Go completa en compilaciones de producción y desarrollo aprobada. Detector
  de carreras aprobado en extracción, bibliotecas, HTTP, almacenamiento y configuración.
- Nueve pruebas Chrome aprobadas en una ejecución completa (1,1 minutos), incluidas
  sesiones, organización documental, OCR, licencias, recuperación de solo lectura,
  navegación y procesamiento. Dos esperas interrumpidas de una ejecución previa
  no se reprodujeron; las trazas se conservaron para diagnóstico.
- TypeScript/Vite, formato Go, `go vet` y los 14 controles de diseño aprobados.

No se ejecutan cargas de miles de documentos ni se comparan tiempos con PHP/SIBAD.
No se dispone de ejecución Windows/Ubuntu ni de las carpetas SMB reales. Se compilan
los objetivos Windows/Linux; la detección de sus recursos y ACL requieren validación
nativa. Cgroup v1 o Job Objects anidados pueden imponer límites adicionales no
observables. La cancelación de una E/S bloqueada depende del sistema operativo.

Falta completar el circuito **offline en el servidor PHP**: importar una `.licreq`
de activación en su panel, asignar licencia y devolver la `.lic` a esa misma
instalación. Se verifican localmente firma, importación, renovación y política
offline mediante vectores/simulador, pero no se afirma esa emisión manual real.
Suscripción, vencimiento, 15 días de tolerancia, firmas inválidas y rechazo de
claves de desarrollo se comprueban con los tests del contrato, sin alterar la
licencia perpetua real del propietario ni aceptar nuevas autoridades de confianza.

Referencias de plataforma: [topología Windows](https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/nf-sysinfoapi-getlogicalprocessorinformationex),
[memoria Windows](https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/nf-sysinfoapi-globalmemorystatusex),
[Job Objects](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-queryinformationjobobject),
[cgroup v2](https://docs.kernel.org/admin-guide/cgroup-v2.html).
