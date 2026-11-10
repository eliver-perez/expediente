# AIDEX 1.0.0 — Permisos nativos macOS ARM64

Fecha: 30 de octubre de 2026. Alcance: auxiliar Autorizar carpeta, pruebas y diagnósticos. Modal existente conservado.

## Diagnóstico y límites de la evidencia

El comando manual modifica únicamente la carpeta elegida. La implementación anterior, después de inventariar el árbol, ejecutaba `allow(..., SEARCH)` y finalmente `acl_set_fd_np` sobre **cada** superior, incluso si ya permitía recorrido. El orden era Documentos → directorio personal → `/Users`; solo después autorizaba el destino seleccionado.

En esta Mac se consultaron, sin modificarlos, propietario, modos, atributos y ACL de `/Users`, `/Users/eliver`, Documentos y `aidexprueba3`. `/Users` es de root y tiene `sunlnk`; los superiores personales ya tenían recorrido para `_aidex` y el usuario local. La carpeta reportada mantenía el permiso administrado añadido manualmente. El código sí intentaba escrituras adicionales innecesarias en los superiores; el comando manual exitoso no las hace.

**El lugar exacto del EPERM de la ejecución anterior no quedó registrado y no se pudo reproducir aquí con elevación.** La evidencia apunta a la modificación de un superior (especialmente `/Users`), pero no permite presentar como confirmada una denegación concreta de `acl_set_fd_np` sobre esa ruta ni atribuirla a privacidad. El defecto comprobado es la escritura incondicional en superiores que ya permiten recorrido y la ausencia de diagnósticos que distinguieran las llamadas nativas.

La prueba con la API nativa sobre una carpeta nueva en Documentos sí aplicó correctamente los permisos administrados para `_aidex`, sin `chmod` previo. Se verificaron propietario, modos y herencia. Esa prueba corrió como propietario, no como auxiliar elevado ni como `_aidex`.

## Corrección

- Los superiores se comprueban desde fuera hacia dentro con un subproceso del auxiliar que utiliza la cuenta, grupo principal y grupos complementarios reales del destinatario. El hijo rechaza root y comprueba recorrido con la identidad efectiva correspondiente.
- Si el recorrido ya está disponible, el superior no se abre para modificar permisos. Solo si falta se abre por descriptor, se revalida su identidad, se añade exclusivamente recorrido y se comprueba otra vez. Un fallo de comprobación no se trata como permiso concedido.
- Se conservan los alcances existentes: lectura/recorrido para vinculadas; lectura, escritura, creación y administración documental para administradas; lectura heredable para el usuario local de Finder. No se modifican propietarios, modos POSIX ni permisos ajenos ni se eliminan ACL anteriores.
- La construcción de ACL valida cada llamada nativa y registra la instrucción exacta que devuelve el error. No continúa tras un fallo de construcción.
- Se mantienen tickets locales de un solo uso, confirmación de huella, comprobación de puerto/PID/ejecutable/cuenta del servicio y protección ante enlaces, cambios de identidad y rutas inseguras.
- El backend sigue comprobando acceso con su propia identidad antes de anunciar éxito: en administradas crea un directorio temporal exclusivo, escribe, sincroniza, lee y elimina una prueba. La lectura de Finder también sigue verificándose con su identidad.
- No se rediseñó el modal ni se añadieron controles. Los errores de aplicación muestran un mensaje breve y remiten al registro interno; no exponen el volcado técnico ni afirman que un EPERM sea necesariamente privacidad.

## Diagnósticos

Registro privado: `/Library/Application Support/AIDEX/data/logs/folder-authorization.log`, modo 0600 y propietario del proceso elevado. Incluye operación, ruta, errno, UID real/efectivo, GID efectivo, destinatario, propietario/modos/atributos/inodo/dispositivo y comprobación del backend. No registra tickets, contraseñas, sesiones ni licencias.

El archivo es acotado; rechaza enlaces, hardlinks y archivos inseguros. Si no puede abrirse, se registra en el sistema con la etiqueta `AIDEX-folder-authorization`. Nunca se crean directorios internos para escribir el registro, también cuando el auxiliar es invocado para conservar originales durante una desinstalación.

## Pruebas realmente ejecutadas

- `AIDEX_MACOS_DOCUMENTS_QA=1 go test ./cmd/macos-folder-helper ./internal/macosfolders -count=1 -timeout=60s`: PASS.
- Nueva carpeta sintética dentro de `~/Documents`, sin comandos manuales de permisos: aplicación nativa de ACL para `_aidex` y el usuario local, repetición sin duplicados, subcarpeta y documento nuevos con herencia correcta, lectura/creación/eliminación bajo el propietario de la fixture y conservación de propietario/modos. Fixture retirada; no quedaron carpetas `AIDEX-QA-permisos-*`.
- Árbol administrado real temporal: todos los superiores accesibles conservan exactamente sus ACL tras autorizaciones repetidas; originales intactos y herencia verificada.
- Árbol vinculado real temporal: solo lectura en las entradas añadidas; sin escritura, borrado o ejecución de archivos; repetición idempotente.
- Subproceso de recorrido real: acceso permitido, denegación efectiva y rechazo cuando no puede adoptarse la identidad solicitada. La prueba no privilegiada no suplanta a `_aidex`.
- Fallo nativo provocado en descriptor cerrado: identifica `acl_get_fd_np`, errno 9, ruta e identidades/atributos. El error del usuario no contiene el detalle técnico. Rechazo de comprobación fallida sin cambiar permisos.
- Registro privado: permisos, límite de tamaño y protección ante enlaces; original externo intacto.
- Regresiones: política de rutas, enlaces, herencia, conservación de ACL anteriores, PID/ejecutable/cuenta, `lsof` real y conflicto real de puerto: PASS.
- Pruebas focalizadas de API/bibliotecas: tickets de un solo uso, sesión, verificación efectiva del backend de prueba, destino sin escritura y protección de estado/bibliotecas ajenas: PASS.
- `go vet` del auxiliar y módulo nativo; `git diff --check`: PASS.
- PKG final expandido: 41 archivos verificados por SHA, backend/auxiliar ARM64, versión 1.0.0, cuatro ProgramArguments, apps/firma ad hoc, scripts válidos e importaciones nativas de permisos, identidad y registro: PASS.
- Auxiliar extraído del propio PKG: `check-traverse` con acceso real permitido y denegado; retorna 3 sin volcado técnico ante denegación: PASS, ejecutado como usuario de la fixture.

## Pendiente: no se declara cerrada la autorización completa

No se ejecutó el ciclo gráfico con autorización administrativa ni se autorizó/incorporó un documento como identidad efectiva `_aidex`. `sudo -n` requiere contraseña; no se solicitó ni almacenó ninguna. Tampoco se instaló el paquete sobre el servicio actual.

Queda validar: botón Autorizar carpeta en una carpeta nueva de Documentos, registro como destino, incorporación y lectura desde AIDEX/Finder de un documento creado por el servicio, incluidos sus subdirectorios. Ante un nuevo error, el registro permite distinguir exactamente operación, ruta, identidad y errno; no debe inferirse privacidad a partir del código 1.

No se modificaron carpetas existentes, usuarios, licencias, datos del servicio, manuales ni módulos ajenos. No se generaron paquetes Windows/Ubuntu. Las firmas verificadas son ad hoc; el PKG no tiene firma Developer ID ni notarización.

## Archivos de esta incidencia

- `cmd/macos-folder-helper/main.go`, `traversal_darwin.go`, `traversal_darwin_test.go`, `log_darwin.go`.
- `internal/macosfolders/acl_darwin.go`, `diagnostic_darwin.go`, `ancestor_darwin_test.go`; adaptación de la firma en `acl_darwin_test.go`.
- Este informe y `evidencias/macos-permisos-20261030/paquete-verificado.json`.

## Instalador

`/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-permisos-20261030/AIDEX-1.0.0-macOS-arm64.pkg`

Tamaño: **10,109,351 bytes**.

SHA-256: `c8454f72419e421d7e8bd622906cc2ffef02befd1cdb97505dc200d05d0ef23a`.

Actualizar abriendo el PKG directamente, sin desinstalar ni borrar datos. Versión comercial conservada: **1.0.0**.
