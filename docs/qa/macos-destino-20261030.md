# AIDEX 1.0.0 — Autorización macOS y destino predeterminado

Fecha: 30 de octubre de 2026. Alcance exclusivo: incidencias del documento de autorización de carpetas y destino predeterminado. Versión comercial 1.0.0.

## Causa y corrección

`probeManagedRoot` convertía el fallo de creación de un archivo en un error genérico `INVALID_REQUEST`. El formulario y el endpoint del auxiliar solo reconocían `ROOT_PERMISSION_DENIED` y `ROOT_PRIVACY_DENIED`. Por eso una carpeta legible pero no escribible no ofrecía autorización.

La inspección, confirmación y elección del destino comparten ahora `CheckAuthorizedDirectory`: se ejecuta con la identidad del backend, crea un directorio exclusivo de prueba, escribe y sincroniza un archivo, verifica sus bytes leyéndolo, prueba publicación mediante enlace sin reemplazo y elimina archivo/enlace/directorio. Conserva los códigos de permisos y privacidad. No toca documentos existentes.

El diálogo macOS ofrece **Autorizar carpeta** para ambos orígenes y conserva **Abrir ajustes de privacidad** y **Comprobar acceso**, con instrucciones simplificadas. Se reutilizan el ticket local, confirmación administrativa y auxiliar nativo existentes: vinculada = lectura/recorrido; administrada = administración del árbol elegido, recorrido mínimo en superiores y lectura heredable para el usuario local de Finder mostrado en la confirmación. No se conceden permisos públicos ni se cambia TCC.

Cambiar ruta, origen o modalidad descarta resultados, errores y enlaces anteriores. Un contador de inspecciones descarta también respuestas tardías y evita abrir una autorización que ya no corresponde a la selección actual.

## Destino predeterminado

Se conserva el campo existente `default_managed_root_id`; no hay migración ni nueva estructura de datos.

- La confirmación de la primera carpeta administrada asigna el destino solo después de la prueba efectiva de escritura. No se elige entre carpetas administradas preexistentes.
- Las tarjetas muestran **Destino predeterminado** y **Establecer como destino**. La selección usa una transacción, permisos `storage.manage_roots`, licencia, revisión optimista de biblioteca, pertenencia, origen, estado activo, identidad física y escritura efectiva.
- La selección desde Configuración conserva compatibilidad y valida la escritura cuando cambia el destino.
- Se registra `storage.destination_changed` con responsable, fecha, destino anterior/nuevo y versiones de configuración.
- Desactivar o retirar el destino elimina la selección sin elegir otra carpeta; se avisa que debe seleccionarse un destino antes de nuevas incorporaciones.
- Las incorporaciones futuras verifican el destino; las operaciones ya planificadas conservan su ruta. Los controles de revisión/estado de raíz impiden confirmar sobre un destino desactivado durante la operación. Los documentos anteriores mantienen bytes, ruta e historial.

## Validación automatizada: PASS

- `go test -tags development ./internal/libraries ./internal/httpapi ./internal/macosfolders ./cmd/macos-folder-helper -count=1 -timeout=180s`: todos los paquetes pasan (bibliotecas 109 s; API 14 s).
- Cuatro pruebas nuevas de destino: asignación inicial, segunda carpeta, bibliotecas existentes sin selección, permisos, origen vinculado/ajeno, revisión obsoleta, auditoría, prueba real de solo lectura, ubicación antes/después del cambio, desactivación/retiro, conservación y desactivación durante publicación.
- La prueba de ubicaciones se repitió ampliada para cambiar el destino antes de ejecutar un trabajo ya planificado: PASS; mantiene la ruta congelada y las incorporaciones posteriores usan el nuevo destino.
- ACL nativas sobre archivos propios temporales: herencia para archivos 0600/directorios 0700, lectura Finder específica, idempotencia, ausencia de permisos públicos y protección de enlaces. Política: rechazo de discos/home completos, estado privado y rutas no admitidas. Tickets: sesión, uso único y comprobación efectiva del backend.
- `go vet` de bibliotecas, API, ACL y auxiliar: PASS.
- TypeScript y compilación Vite: PASS; advertencia ya existente de bundle mayor de 500 kB.
- Chrome/Playwright: dos pruebas nuevas pasan (tarjetas, cambio de destino, desactivación, limpieza por ruta/origen, botón macOS administrado y respuesta tardía). El error HTTP del diálogo es simulado; la falta de escritura real se prueba aparte en Go.
- Chrome/Playwright: dos pruebas de regresión de incorporación, corrección, sustitución y privacidad del Colaborador: PASS.
- Scripts de instalación macOS: 22 pruebas pasan; desinstalador: 11 pruebas pasan.
- PKG extraído: 41 archivos verificados por SHA, ARM64/1.0.0, scripts válidos, cuatro `ProgramArguments`, auxiliar y dos apps con firmas ad hoc e iconos oficiales: PASS.
- Ciclo del PKG con scripts/binario reales y cuentas/launchd/recibos simulados: instalación limpia, administrador/login/HTTP reales en puerto temporal, actualización, conservar/reinstalar con usuario y licencia local intactos, dos ciclos de eliminación/reinstalación, recuperación de residuos antiguos y rechazo de datos recuperables sin configuración: PASS. Originales sintéticos conservados.
- `git diff --check` en archivos de esta incidencia: PASS.

Todas las pruebas usan datos temporales. No se modificó el servicio de desarrollo ni se realizaron operaciones con licencias reales. No se generaron paquetes Windows/Ubuntu ni se modificaron manuales Word.

## Pendiente de validación nativa interactiva

No se ejecutó Installer.app ni una elevación administrativa real del auxiliar, ni se modificó TCC. Queda comprobar en una instalación de pruebas las carpetas vinculada/administrada sin permisos usando la cuenta real `_aidex`, la confirmación del sistema y la lectura desde Finder de documentos creados por ese servicio. Las pruebas ACL son nativas pero no suplantan esa validación integral. El paquete sigue sin firma Developer ID ni notarización; las firmas internas verificadas son ad hoc.

## Archivos de esta incidencia

- `internal/libraries/managed_destination.go` y `managed_destination_test.go`.
- `internal/libraries/roots.go`, `filesystem.go`, `settings.go`, `service.go`, `documents.go`, `materialization.go`.
- `internal/httpapi/library_handlers.go`.
- `web/src/LibrarySettings.tsx`, `web/src/AuditTable.tsx`, `web/tests/managed-destination.spec.ts`.
- Este informe y `evidencias/macos-destino-20261030/paquete-verificado.json`.

El auxiliar nativo y los scripts de empaquetado existentes ya admitían los dos alcances; se reutilizaron sin cambiar su arquitectura.

## Entregable

Ruta absoluta:
`/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-destino-20261030/AIDEX-1.0.0-macOS-arm64.pkg`

Tamaño: **10,038,213 bytes**.

SHA-256:
`21e1bd63906e190a5324c5dc6e776a15de46e3ba19e0e08e58872be0b34d66a0`.

Para actualizar, abrir este PKG directamente, sin desinstalar. Después, en Carpetas, inspeccionar el origen elegido, usar Autorizar carpeta si corresponde y Comprobar acceso antes de registrar. La primera administrada queda seleccionada automáticamente; para las siguientes usar Establecer como destino. Comprobar que un documento incorporado antes del cambio mantiene su ubicación y que el siguiente utiliza el nuevo destino.
