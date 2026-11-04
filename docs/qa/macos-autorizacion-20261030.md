# AIDEX 1.0.0 — Autorización de carpetas macOS ARM64

Fecha: 30 de octubre de 2026. Alcance: auxiliar de autorización y su diálogo; versión 1.0.0.

## Causa confirmada nativamente

El auxiliar ejecutaba `lsof -nP -iTCP:<puerto> -sTCP:LISTEN -Fp` y exigía que todas las líneas fueran `p<PID>`. En macOS 4.91, `-Fp` también devuelve el descriptor del socket (`f4`, por ejemplo). El descriptor se interpretaba erróneamente como un proceso ajeno.

La prueba creó un listener TCP real y reprodujo estas salidas del sistema, sin mocks:

```text
-Fp: "p95807\nf4\n"
-t:  "95807\n"
```

La validación anterior rechazaba la primera salida aunque el PID coincidía. La nueva usa `-t`, que devuelve solo identificadores; no filtra por PID, para poder detectar otros procesos que ocupen el mismo puerto. Una segunda prueba abrió un listener IPv6 del mismo puerto en otro proceso y obtuvo `95807\n95825\n`: el conflicto auténtico fue rechazado.

## Corrección y seguridad

- Se mantiene la comparación del puerto del enlace con el `listen_address` de la configuración fija de AIDEX, abierta sin seguir enlaces y comprobando la identidad del archivo.
- El PID debe pertenecer a `system/app.aidex`. Mediante `proc_pidpath` y `proc_pidinfo`, el kernel confirma la ruta exacta `/Library/Application Support/AIDEX/bin/aidex-server`, la cuenta `_aidex` y el momento de inicio.
- Se verifica que el ejecutable y sus directorios sean propiedad de root, no sean enlaces y no sean escribibles por grupo/otros.
- Se vuelve a consultar el registro del servicio y su identidad antes de continuar para detectar un reinicio o reutilización del PID.
- Fallos de consulta, listener vacío/ajeno, ejecutable o cuenta diferentes continúan rechazándose antes de modificar permisos.
- Se conserva el mecanismo existente: ticket local de un solo uso, confirmación con huella de carpeta/uso/usuario, autorización administrativa y permisos delimitados mediante API nativa. El navegador no envía comandos.
- Vinculadas: lectura y recorrido. Administradas: lectura, escritura, creación de archivos/subdirectorios y administración documental. En superiores solo recorrido. Se conservan las autorizaciones existentes y la lectura heredable de Finder.
- El resultado sigue dependiendo de `/finish`: el backend comprueba con su propia identidad el acceso, y en destinos administrados crea/escribe/lee/elimina una prueba exclusiva sin tocar documentos reales. No se anuncia éxito si la comprobación falla.

## Interfaz

El modal de macOS muestra únicamente el nombre de carpeta en el mensaje principal. Las tres acciones están agrupadas y son compactas: Autorizar carpeta (principal, escudo verificado), Comprobar acceso (actualización) y Abrir ajustes de privacidad (engranaje). Se usan SVG inline reales y el componente/estilos de botones existentes.

Las instrucciones de privacidad aparecen al recibir `ROOT_PRIVACY_DENIED` e indican explícitamente habilitar **aidex-server** en Acceso total al disco. Se eliminó la terminología técnica de la confirmación nativa y los mensajes de permisos; esta conserva la ubicación completa para distinguir carpetas homónimas y las advertencias sobre el alcance, el acceso de Finder y la ausencia de permisos públicos. Cancelar sigue siendo la opción predeterminada.

## Pruebas efectuadas

1. `go test -v ./cmd/macos-folder-helper ./internal/macosfolders -count=1 -timeout=45s`: PASS. Reproducción nativa de `lsof`, conflicto real con otro proceso, consulta nativa de ruta/cuenta/inicio, rechazo de identidad ajena y proceso inexistente, URL/ticket, política de rutas, herencia e idempotencia de permisos.
2. Nueva prueba `TestNativeManagedACLRetainsExistingPermissionsAndInheritance`: PASS, sobre archivos propios temporales. Conserva una entrada previa, no acumula entradas al repetir, hereda lectura/escritura sin ejecución de archivos ni permisos públicos, crea/lee/elimina contenido sintético. No suplanta a `_aidex`.
3. Consulta de protección del ejecutable instalado: PASS, solo lectura. Se verificaron los propietarios/permisos actuales, sin cambiarlos.
4. Pruebas focalizadas de API y bibliotecas: `TestMacFolderTicketsSingleUseEffectiveProbeAndSession`, `TestManagedDestinationPermissionFailureUsesAuthorizationCode`, `TestFolderAuthorizationScopeProtectsLibrariesAndPrivateState`: PASS. Bases/carpetas temporales; el backend de prueba corre con la cuenta de la prueba.
5. `go vet ./cmd/macos-folder-helper ./internal/macosfolders`: PASS.
6. TypeScript + Vite: PASS. Se mantiene la advertencia existente por tamaño del bundle.
7. Chrome/Playwright, dos pruebas en `managed-destination.spec.ts`: PASS. Interfaz real, errores de permisos/privacidad simulados; SVG visibles de 16×16, nombres/ruta, instrucciones condicionales, limpieza de respuestas tardías y regresión de tarjetas de destino. Capturas reales a 1920×1080 revisadas visualmente.
8. PKG expandido: PASS. Los 41 archivos coinciden con su manifiesto, backend y auxiliar son ARM64, versión 1.0.0, cuatro `ProgramArguments`, importaciones de libproc presentes, AppleScript nativo compilado con confirmación administrativa, aplicaciones/iconos y firmas ad hoc válidos. Scripts extraídos pasan `bash -n`.
9. `git diff --check` de los archivos de interfaz: PASS.

No se modificó la instalación real, ni el estado del servicio, ni sus permisos. No se ejecutaron licencias reales ni se generaron paquetes Windows/Ubuntu. No se modificaron manuales.

## Validación nativa pendiente

El falso rechazo de `lsof` y el rechazo de un conflicto auténtico se comprobaron con procesos/sockets reales de macOS; no son simulaciones.

**La autorización integral desde la aplicación con elevación administrativa y escritura efectiva como `_aidex` todavía no está comprobada.** No había autorización sudo no interactiva disponible (`sudo -n` requiere contraseña). No se instaló el nuevo paquete sobre el desarrollo ni se cambiaron los permisos de carpetas reales para facilitar las pruebas.

En una instalación de pruebas queda ejecutar Autorizar carpeta para una vinculada y una administrada, confirmar la elevación, comprobar el acceso efectivo del servicio y la lectura desde Finder, incluyendo un documento nuevo y una ubicación protegida por privacidad. Las comprobaciones nativas de permisos sobre fixtures y el modal con respuestas simuladas no sustituyen esa prueba completa. No se declara cerrada la autorización integral hasta verificarla.

El PKG no tiene firma Developer ID ni notarización; las firmas internas comprobadas son ad hoc.

## Archivos de esta incidencia

- `cmd/macos-folder-helper/main.go`, `main_test.go`, `process_darwin.go`, `process_darwin_test.go`.
- `internal/macosfolders/acl_darwin.go` (solo mensaje de error) y `acl_darwin_test.go`.
- `packaging/macos/authorize-folder.applescript`.
- `web/src/LibrarySettings.tsx`, `ActionButtons.tsx`, `styles.css` y `web/tests/managed-destination.spec.ts`.
- Este informe y evidencias en `evidencias/macos-autorizacion-20261030/`.

## Entregable

`/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-autorizacion-20261030/AIDEX-1.0.0-macOS-arm64.pkg`

Tamaño: **10,041,562 bytes**.

SHA-256: `8b94435cce583f01bb4e29410c35abdac05ae7f2172868b3cf85fbd8f5471a88`.

Abrir el PKG directamente para actualizar, sin desinstalar ni borrar datos. El informe de inspección del archivo final está en `evidencias/macos-autorizacion-20261030/paquete-verificado.json`.

Capturas de interfaz (1920×1080):

- `evidencias/macos-autorizacion-20261030/modal-administrada.png`.
- `evidencias/macos-autorizacion-20261030/modal-vinculada.png`.
- `evidencias/macos-autorizacion-20261030/modal-privacidad.png`.
