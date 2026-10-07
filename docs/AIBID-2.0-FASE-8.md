# AIBID 2.0 — fase 8: revisión final y estabilización

Entrega de pruebas **2.0.0-alpha.8-f8**, 6 de octubre de 2026. Continúa sobre
la fase 7. Conserva la corrección del instalador Windows f6r1 y el esquema **14**.
No requiere borrar datos ni volver a incorporar las bibliotecas.

## 1. Dashboard

Panel general pasa a llamarse **Dashboard**. Cuatro indicadores destacados muestran
documentos, índice vigente, incorporaciones de hoy y trabajos en curso o pendientes.
Las gráficas muestran incorporaciones de los últimos 30 días, formatos, estados de
procesamiento y origen vinculado/administrado. Se conservan duplicados, caché,
duraciones, colas, errores recientes y el desglose paginado por biblioteca.

Todos los valores proceden de la misma consulta transaccional de datos existentes.
Los días sin incorporaciones valen cero; una distribución vacía muestra un mensaje.
No se generan series de ejemplo ni se reconstruye una historia de procesamiento a
partir del estado actual. La serie diaria cuenta documentos vigentes según su fecha
original de incorporación, excluye retirados y usa días del calendario local del
servidor, incluidos cambios de horario. Reindexar no cuenta como incorporar.

Las gráficas usan **Chart.js 4.5.1**, empaquetado localmente y cargado al entrar al
Dashboard. No requieren CDN ni conexión a Internet. Incluyen valores accesibles,
leyendas, tablas desplegables y respeto a la preferencia de movimiento reducido.
El diseño usa los colores de AIBID y se adapta a escritorio y móvil.

## 2. Ajustes

El menú lateral reúne la configuración general bajo **Ajustes**, con estas secciones:

- Archivos.
- Procesamiento y recursos.
- Automatización, OCR y vigilancia.
- Vistas previas y caché.
- Acceso y red.

La configuración avanzada conserva sus controles dentro de Automatización, OCR y
vigilancia. Los enlaces anteriores siguen funcionando, incluida la navegación hacia
atrás y la recarga directa. El texto oscuro, el indicador de selección y el foco de
teclado resuelven el contraste insuficiente de las pestañas anteriores. La herencia
por biblioteca, los permisos y las validaciones del servidor se conservan.

## 3. Actualización y datos

No se añade una migración en esta fase: una instalación de fase 7 mantiene el esquema
14. Por ello esa actualización no necesita crear una instantánea por cambio de
esquema. Al actualizar desde fases anteriores se conserva el respaldo automático
previo a las migraciones pendientes. Los instaladores aceptan las versiones previas
del canal de pruebas y conservan configuración, usuarios, licencia y documentos.

Los originales vinculados y las muestras privadas no forman parte de los paquetes.
No se modifica el servicio instalado en esta Mac durante la construcción o validación.

## 4. Compatibilidad de formatos

| Formato | Extracción y búsqueda | Vista previa | Reindexación |
| --- | --- | --- | --- |
| PDF | Texto nativo; OCR según reglas y disponibilidad | PDF original | Conserva identidad e historial |
| DOCX | Párrafos, tablas y partes admitidas; contexto estructurado | PDF generado con LibreOffice | Publicación atómica del texto actual |
| XLSX | Hojas, celdas y valores guardados; no ejecuta fórmulas | PDF generado con LibreOffice | Publicación atómica del texto actual |
| TXT | Texto y codificación admitida | Texto en la ficha | Conserva identidad e historial |
| CSV | Registros/campos y contexto | Texto en la ficha | Conserva identidad e historial |

La matriz respeta las reglas globales y de cada biblioteca. No se introduce OCR para
imágenes incrustadas en Office ni soporte de macros. Las vistas de Office son
representaciones de consulta: fuentes, anchos de columna y paginación pueden variar.

## 5. Revisión del plan de fase 8

| Apartado | Cobertura y comprobaciones |
| --- | --- |
| 8.1 Compatibilidad | Detección, extractores, búsqueda por formato/contexto, OCR PDF, descarga, reindexación y conversión Office real. Fixtures sintéticos y corpus privado separado. |
| 8.2 Seguridad | Rechazo de ejecutables/scripts/paquetes, tipo discordante, Office activo y archivos temporales; límites ZIP/XML; conversión desde copia saneada sin referencias externas; originales conservados y caché privada fuera de raíces. Permisos y CSRF en endpoints. |
| 8.3 Casos límite | Corrupción, tamaños, timeout, fallo de extractor/OCR/conversor, desaparición/cambio, duplicados, recuperación, reintentos y pausa/reanudación. Un trabajo fallido no bloquea el resto. |
| 8.4 Caché | Generación/reutilización, hash y generador, LRU, presupuesto, caducidad, limpieza y regeneración, cancelación y conflictos con limpieza. |
| 8.5 Configuración | Valores predeterminados, herencia global/biblioteca, reemplazos, persistencia, revisiones concurrentes y permisos. Navegación nueva conserva los controles existentes. |
| 8.6 Regresiones | Bibliotecas vinculadas/administradas/híbridas, reconciliación simultánea por raíces distintas, exclusión por raíz, watcher, expedientes, OCR, búsqueda, Auditoría, usuarios/sesiones, licencias online/offline y acceso local/LAN. Actualización de paquetes y protección de originales al desinstalar. |
| 8.7 Dependencias | Versiones fijadas, licencias incluidas, dependencias OCR empaquetadas en Windows, dependencias apt en Ubuntu y detección/ausencia explícita de LibreOffice. Comprobación del script final de Windows. |
| 8.8 Comentarios | Se documentan el calendario de la serie diaria, ciclo de vida y accesibilidad de gráficas, copia Office aislada y protección del BOM de PowerShell; se conservan los comentarios de seguridad, caché, colas y publicación. |

La evidencia ejecutada y los límites se registran en
`dist/aibid-2.0-fase-8/VALIDACION-FASE-8.json`. La cobertura automática no sustituye
las pruebas de instalación y permisos de los sistemas operativos destino.

## 6. Dependencias y distribución

- Chart.js **4.5.1** y su dependencia `@kurkle/color` **0.3.4**, licencias MIT,
  incluidas en `third-party/frontend` de la documentación distribuida. El bundle es
  local, con versión bloqueada en `web/package-lock.json`.
- LibreOffice es necesario **solo para generar vistas previas DOCX/XLSX**. La
  extracción, búsqueda y descarga de originales funcionan sin él. Su ausencia se
  explica en el instalador, en Ajustes y al solicitar la vista previa.
- Ubuntu: `libreoffice-writer` y `libreoffice-calc` son paquetes recomendados; una
  instalación normal con apt los incorpora. La preparación offline incluye los
  recomendados. Si el administrador los excluye, puede instalarlos posteriormente.
- Windows/macOS: instalar LibreOffice en el servidor desde
  <https://www.libreoffice.org/download/>. AIBID busca la instalación habitual y
  permite indicar la ruta en Ajustes → Vistas previas y caché. No se incluye una
  copia de LibreOffice en esos instaladores. La comprobación nativa de esta entrega
  usa **26.2.6 para macOS ARM**, en un directorio temporal.
- Se mantienen Poppler/Tesseract e idiomas español/inglés. Windows lleva su runtime
  y manifiesto; Ubuntu declara paquetes; macOS conserva la comprobación al configurar.

Consultar `THIRD-PARTY.md` para licencias y distribución. La auditoría npm de
dependencias de producción se registra como una comprobación puntual, no como una
garantía frente a vulnerabilidades futuras o de herramientas externas.

## 7. Archivos principales

Interfaz: `Dashboard.tsx`, `DashboardChart.tsx`, `Settings.tsx`,
`SettingsNavigation.tsx`, `App.tsx`, pantallas de configuración y `styles.css`.
Servidor: `internal/libraries/dashboard.go` y ruta SPA en `internal/httpapi/server.go`.
Regresión: pruebas de observabilidad, `dashboard-settings.spec.ts`, navegación y
pruebas de conversión real. Distribución: constructor, plantillas, CI y documentos.

## 8. Validación

Se ejecutan compilación TypeScript/Vite, suites Go normal y de desarrollo, detector
de carreras en bibliotecas/HTTP/almacenamiento/extracción/previews/formatos/mantenimiento,
análisis estático, validaciones SQL y las pruebas del constructor PowerShell.
Chrome cubre 19 escenarios, incluidos Dashboard, Ajustes y los flujos anteriores.

Las verificaciones visuales usan 1800 y 390 píxeles, foco de teclado, navegación
directa, historial, ausencia de desbordamiento horizontal y contraste mínimo 4.5:1
en los enlaces de Ajustes. Las gráficas tienen tablas equivalentes y no solicitan
recursos externos. Se comprueba también el agrupamiento por días con cambio horario.

La prueba nativa usa el binario extraído del paquete macOS con datos temporales,
inicia el servicio, comprueba PDF/OCR y actualiza datos creados por el paquete de
fase 7. Windows y Ubuntu se inspeccionan por arquitectura, dependencias, manifiesto,
recursos y frontend embebido. Las dos copias de `manage.ps1` del EXE final deben
tener un único BOM y superar el analizador PowerShell y su contrato de parámetros.

La revisión corrigió también una prueba nativa que exigía 20 bytes de texto incluso
a hojas sintéticas de dos celdas. Ahora comprueba texto no vacío y la conservación
del rótulo y la celda numérica conocidos, sin rechazar una vista corta válida.

## 9. Instaladores

En `dist/aibid-2.0-fase-8/`:

- `AIBID-Pruebas-2.0.0-alpha.8-f8-Windows-amd64.exe`.
- `AIBID-Pruebas-2.0.0-alpha.8-macOS-arm64.pkg`.
- `aibid-test_2.0.0~alpha.8_amd64.deb`.
- `aibid-test_2.0.0~alpha.8_arm64.deb` (Ubuntu ARM en la VM de Apple Silicon).

`SHA256SUMS` y `VALIDACION-FASE-8.json` identifican los artefactos comprobados.
Se conservan los paquetes anteriores. No se genera macOS Intel.

## 10. Límites de la entrega

Esta es una entrega alpha de pruebas, sin firma comercial ni notarización. La
ejecución del instalador/servicio en Windows y Ubuntu, los permisos reales de SMB y
el instalador del sistema macOS con launchd quedan para los equipos destino.
Las licencias se prueban con el simulador y las claves de fixtures; esta fase no
activa ni desactiva la licencia comercial del usuario.

Los indicadores se actualizan por consulta cada 15 segundos; no representan consumo
de CPU ni telemetría continua. La serie diaria usa incorporaciones originales de
documentos aún vigentes; no es un registro inmutable de altas/bajas. Los permisos
administrativos del Dashboard no conceden acceso al contenido documental.

## 11. Comprobación en los equipos destino

1. Actualizar a alpha.8-f8 conservando los datos. Comprobar usuarios, licencia,
   documentos y bibliotecas existentes; no desinstalar ni borrar el índice.
2. Abrir Dashboard y contrastar las gráficas/tablas con una biblioteca pequeña.
   Incorporar un documento y comprobar su aparición al actualizar.
3. Recorrer Ajustes y guardar una opción de prueba; recargar y comprobar persistencia.
   Verificar herencia/reemplazo en una biblioteca sin cambiar las demás.
4. Procesar PDF, DOCX, XLSX, TXT y CSV; buscar texto conocido, abrir, descargar y
   reindexar. Comprobar Office con LibreOffice detectado y su aviso si no está instalado.
5. Reconciliar dos carpetas distintas a la vez y volver a solicitar la misma raíz;
   debe mantenerse un único recorrido simultáneo por raíz. Probar SMB con la cuenta
   real del servicio en Windows.
6. Revisar un archivo corrupto de prueba y el diagnóstico correspondiente; confirmar
   que continúan los trabajos sanos, las búsquedas y las vistas previas disponibles.
7. Probar limpieza/regeneración de caché, pausa/reanudación y reinicio del servicio.
   Confirmar que los originales no cambian. Comprobar acceso desde la LAN autorizada.
