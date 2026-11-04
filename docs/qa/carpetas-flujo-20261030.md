# AIDEX 1.0.0 — Carpetas macOS y flujo documental

Fecha: 30 de octubre de 2026. Versión comercial conservada: **1.0.0**.

## Cambios

### Carpetas y Finder

El diálogo ofrece **Autorizar carpeta**, **Comprobar acceso** y **Abrir ajustes de
privacidad**. En Configurar carpeta se puede reparar un destino administrado
existente para lectura desde Finder.

El PKG integra `/Applications/AIDEX/Autorizar carpeta AIDEX.app` con icono oficial,
esquema `aidex-auth` y confirmación nativa antes de solicitar privilegios. El
navegador entrega un ticket; no puede enviar comandos al auxiliar. La huella de
confirmación vincula ruta, origen y usuario local de Finder.

`bin/aidex-folder-helper` aplica ACL nativas por descriptor, sin chmod recursivo:

- Vinculadas: lectura y recorrido. Administradas: administración del árbol elegido.
- Superiores: únicamente recorrido, sin lectura recursiva ni herencia.
- Finder: lectura específica para el usuario local mostrado en la confirmación;
  no se autoriza a todos ni al grupo staff.
- Herencia para archivos/subdirectorios futuros; los modos POSIX privados 0600 y
  0700 no se convierten en permisos públicos.
- Inventario completo previo, comprobación de identidades al abrir y antes de
  aplicar, rechazo de raíces/home completos, estado privado, superposición de
  bibliotecas, enlaces, archivos especiales y montajes interiores.

Los tickets son locales, caducan y solo pueden utilizarse una vez. Se revalidan
sesión, permisos y licencia. Antes de cambiar ACL, el auxiliar exige que el puerto
coincida con la configuración fija y que su listener pertenezca al PID registrado
como `app.aidex`: otra aplicación local no puede inventar tickets privilegiados.

El backend verifica el acceso con su propia identidad; para administradas prueba
crear, escribir, sincronizar, publicar sin reemplazar y retirar una prueba de
nombre aleatorio. El auxiliar baja privilegios al usuario de Finder y comprueba
lectura. Un fallo no se presenta como éxito. Hay tiempos límite y watchdog.

No se modifica TCC ni se afirma que una ACL lo resuelva. Las instrucciones macOS
distinguen la privacidad del proceso del servicio de los permisos del navegador.

Las ACL documentales permanecen tras desinstalar. Al eliminar datos internos con
confirmación, los originales archivados en la ubicación fija `AIDEX/Documentos
conservados` reciben lectura para el usuario local y se verifica ese acceso antes
de anunciar éxito. No se siguen enlaces archivados. Ante un fallo se conservan
las herramientas de recuperación. Desinstalar no libera la activación comercial.

### Clasificación, correcciones y versiones

El Gestor dispone de **Exigir clasificación antes de incorporar documentos**.
Su ausencia en bibliotecas existentes significa **activada**: no cambian sus
reglas silenciosamente. Al desactivarla se admite clasificación parcial o vacía,
con revisión o confirmación según la biblioteca, y clasificación posterior.

La ficha y el explorador distinguen y filtran borrador temporal, incorporado sin
clasificar, clasificado y versión histórica. El filtro encuentra pendientes en
subcarpetas sin obligar a recorrerlas. Clasificación y aprobación son independientes.

Desactivar la regla no revela borradores privados. Sin revisión, el Colaborador
pulsa **Solicitar incorporación** para presentarlos explícitamente al Gestor;
no obtiene permiso para confirmar y otros colaboradores siguen sin verlos.
Con revisión se mantiene el envío y la decisión del Revisor.

El Gestor corrige metadatos y reasigna aprobados con motivo, sin mover archivos
ni retirar su aprobación física. **Exigir revisión para corregir metadatos de
documentos incorporados** conserva los valores vigentes hasta que otra persona
con permiso de aprobar autorice la propuesta. El Colaborador solo solicita
correcciones sobre sus propias cargas aprobadas, sin obtener edición implícita.

La sustitución usa otra carga y otro archivo. La anterior conserva vigencia hasta
que la nueva complete revisión/confirmación y guardado; luego queda histórica y
consultable. No se borra ni sobrescribe. Solo la nueva cuenta en el expediente,
incluso con tipos únicos. Cancelar una propuesta conserva la versión anterior.
Solicitantes, responsables, decisiones, fechas, motivos y valores anteriores/nuevos
quedan en los registros y auditoría. Los eventos que no cambian revisión no crean
instantáneas duplicadas.

Migración aditiva **0021**: presentación explícita de cargas, correcciones y
relaciones de versiones. Se mantienen licenciamiento, CLUF, permisos e historial.
Nuevos permisos de corrección/sustitución corresponden al Gestor; el Revisor
decide propuestas específicas y el Colaborador solicita.

Componentes principales: `internal/macosfolders`, `cmd/macos-folder-helper`,
aplicación/auxiliar/empaquetado macOS, autorización HTTP, servicios de bibliotecas,
migración 0021 y configuración, ficha, explorador, correcciones y revisión web.
No se modificaron la instalación real de desarrollo ni manuales de Word. No se
operó con licencias reales ni se modificó su servidor. No se publicaron etiquetas.

## Pruebas

- Compilación TypeScript/Vite y compilación de los cuatro objetivos Go: PASS.
- Regresión completa de bibliotecas y API: PASS. Tras el ajuste final de privacidad
  se repitieron los escenarios afectados de presentación, roles, clasificación,
  revisión, solicitudes, corrección y sustitución: PASS.
- Pruebas nuevas de Colaborador/Revisor/Gestor con y sin revisión: PASS;
  aprobación física conservada, tipo único, cancelación y original anterior intacto.
- Navegador real: regresión anterior de revisión, incorporación sin clasificar,
  corrección y sustitución: PASS. Dos sesiones verifican privacidad del borrador,
  presentación explícita y confirmación por Gestor: PASS.
- ACL nativas en archivos temporales propios: herencia sobre archivos 0600 y
  subdirectorios 0700, idempotencia, bytes/modos conservados y rechazo previo de
  enlaces simbólicos/duros: PASS. No se modificaron superiores del sistema.
- Política de rutas, URL/tickets, huella/UID, proceso/listener, claim/finish de
  un uso, sesión cerrada y ausencia de residuos de prueba: PASS.
- Scripts macOS: 22 pruebas; desinstalación aislada: 11 pruebas, PASS.
- Migraciones: usuarios y auditoría conservados, reversión de base poblada
  rechazada: PASS. Se adaptó una expectativa antigua de 20 migraciones al número
  real, sin cambiar el comportamiento de reversión.
- Ciclo aislado del PKG final: instalación limpia, administrador/login HTTP,
  actualización, conservar/reinstalar con usuario y licencia anteriores,
  eliminar/reinstalar dos veces, originales intactos, recuperación directa de
  residuos antiguos y rechazo de bases sin configuración: PASS.
- Paquetes: arquitecturas PE/ELF/Mach-O, versión 1.0.0, scripts/BOM PowerShell,
  DLL, 1092 archivos OCR Windows, manifiestos SHA, iconos, esquema URL y firmas
  ad hoc: comprobados por `check_production_packages.py`.

Las fallas iniciales de navegador fueron de preparación de datos (revisión
obsoleta y campo root_id); se corrigieron y se repitieron solo esos escenarios.
Se corrigió un defecto real de auditoría al crear solicitudes sin incrementar
revisión: no deben insertar otra instantánea de la misma revisión.

PowerShell necesitó la ubicación del runtime .NET portátil existente; se restauraron
recursos vacíos de NSIS 3.13 desde su distribución oficial HTTPS. No se modificó
por ello el instalador Windows ni el sistema anfitrión. Vite conserva su advertencia
de tamaño del bundle. Solo se reconstruyeron paquetes cuando debían incluir código
corregido; los ya completados se conservaron durante los ajustes de herramientas.

## Validación nativa pendiente

En el ciclo aislado son reales scripts, servidor, SQLite, archivos, bootstrap y
HTTP; cuentas, launchd, recibos y elevación están simulados. Las pruebas ACL usan
la API real sin root sobre archivos propios. No se ejecutó la autorización
privilegiada contra las rutas reales del equipo de desarrollo.

Requieren validación manual: Installer.app, apertura inicial del esquema URL,
diálogo administrativo, acceso efectivo de `_aidex` instalado y usuario de Finder,
TCC/Acceso total al disco y desinstalación gráfica. También instalación/ejecución
nativas en Windows y Ubuntu de ambas arquitecturas. El PKG no cuenta con firma
comercial ni notarización; las firmas ad hoc no las sustituyen.

No se declara una instalación nativa completa de esos sistemas. No quedan comandos
de compilación o prueba deliberadamente ejecutándose tras la entrega.

## Paquetes y evidencias

Directorio: `/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-carpetas-flujo-20261030/`.

Rutas, tamaños y SHA-256 definitivos: `paquetes-verificados.json` y `SHA256SUMS`.
Resultados de inspección y ciclo: `evidencias/carpetas-flujo-20261030/`.

Referencias: [ACL de Apple](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemDetails/FileSystemDetails.html),
[esquemas URL de Apple](https://developer.apple.com/documentation/appkit/nsapplicationdelegate/application(_:open:)),
[NSIS 3.13 oficial](https://sourceforge.net/projects/nsis/files/NSIS%203/3.13/).
