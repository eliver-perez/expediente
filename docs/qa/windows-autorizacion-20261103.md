# AIDEX 1.0.0 — Autorización Windows amd64

3 de noviembre de 2026. Corrección exclusiva de Windows.

## Diagnóstico

Se localizó un defecto en **`CreateFileW` al abrir directorios superiores con
`MAXIMUM_ALLOWED`**. Esa máscara puede solicitar derechos de datos y DELETE;
choca con lectores que no compartan borrado, produciendo
`ERROR_SHARING_VIOLATION` (32) antes de autorizar una raíz, aunque esté vacía.
El `os.Open` del Go utilizado comparte lectura/escritura, pero no borrado.

El log no es el origen directo del error propagado: `folder authorization failed`
se emitía al devolver error `windowsfolders.Authorize`. Los errores al abrir el
registro no se devolvían desde esa función. Supervisor y worker abren el log en
append, comparten lectura/escritura y no solicitan DELETE: sus aperturas son
compatibles entre sí. No hay archivos temporales de intercambio en este tramo;
se comunican mediante tickets HTTP locales.

**El registro aportado no identifica la llamada ni el directorio exactos que
fallaron en aquel PC.** Se corrigió el defecto concreto localizado en el código,
pero no se afirma haber reproducido ni confirmado la causa de esa ejecución.
El nuevo registro permitirá distinguir apertura, lectura/modificación de ACL y
comprobación del servicio, con el recurso concreto.

Referencias: [CreateFileW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew),
[propagación de SetSecurityInfo](https://learn.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-setsecurityinfo),
[NtSetSecurityObject](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntsetsecurityobject).

## Corrección

- Superiores abiertos con `READ_CONTROL`; solo se reabren con
  `READ_CONTROL | WRITE_DAC` cuando es necesario conceder recorrido.
- Se conserva la ausencia de `FILE_SHARE_DELETE`, el rechazo de reparse points y
  la protección contra sustituciones. Al reabrir se compara volumen e identidad.
- La raíz conserva permisos heredables RX/Modify para `NT SERVICE\AIDEX` mediante
  `SetSecurityInfo`. En superiores se añade únicamente recorrido mediante
  `NtSetSecurityObject`, por descriptor, sin volver a propagar ACL a árboles ajenos.
- Se conservan propietarios, permisos de otros usuarios, denegaciones, UAC,
  verificaciones de instalación/listener y límite de 90 segundos. No se habilita
  toma de propiedad ni se concede Control total al servicio.
- Diagnóstico con operación, recurso, código Win32, acceso solicitado, compartición,
  PID, rol y fase; también NTSTATUS cuando corresponde. Se verifica y registra la
  elevación del worker. Los tickets no se escriben en el registro.
- Se registran errores anteriores a reclamar el ticket y se evita repetir su cierre
  cuando ya se intentó confirmar el resultado.

El modal y sus mensajes permanecen intactos. El servicio sigue comprobando acceso
real antes de anunciar éxito; no se sustituye esa prueba por permisos del administrador.

## Archivos

Producción, todos específicos de Windows:

- `internal/windowsfolders/acl_windows.go`
- `internal/windowsfolders/diagnostics_windows.go` — nuevo
- `cmd/aidex-launcher/folder_windows.go`

Pruebas nuevas: `internal/windowsfolders/sharing_windows_test.go` y
`cmd/aidex-launcher/folder_windows_test.go`.

**344 hashes protegidos sin cambios** respecto del inicio de esta tarea: macOS,
Ubuntu, PKG/DEB existentes, interfaz y código compartido. No se editaron
licenciamiento, gestión documental, manuales ni scripts de empaquetado.

## Verificaciones

Ejecutadas:

- Compilación Windows amd64 del GUI y de ambos paquetes de pruebas: PASS.
- Análisis `go vet` para ambos paquetes Windows: PASS.
- Siete pruebas de plantillas/parser con PowerShell real: PASS.
- Instalador: PASS en versión 1.0.0, amd64/GUI, iconos, SHA de 1114 archivos,
  scripts y DLL. Runtime OCR sin cambios. Sin invocaciones a PowerShell/icacls en el launcher.
- Integridad de los 344 archivos protegidos: PASS.

**Las pruebas nativas están compiladas, pero NO ejecutadas.** Cubren el conflicto
antiguo, handles mínimos, rechazo de DELETE concurrente, vacías/con documentos,
RX/Modify, conservación de ACE/propietario/contenido, herencia, repetición y ausencia
de propagación a una hermana. Otra prueba escribe 200 registros desde dos procesos.

En Windows:

```powershell
go test -v ./internal/windowsfolders ./cmd/aidex-launcher
```

Alternativa: ejecutar con `-test.v` los binarios de `build/`:
`aidex-windows-sharing-folders.test.exe` y `aidex-windows-sharing-launcher.test.exe`.
Solo usan fixtures temporales y un SID sintético; no sustituyen la prueba del servicio.

Validación pendiente en la instalación Windows real, con carpetas de prueba:

1. Vacía sin permiso previo del servicio: autorizar y comprobar acceso.
2. Vinculada con documentos: consultar/indexar y conservar hashes originales.
3. Administrada: comprobar escritura y creación/consulta documental.
4. Nuevos archivos/subcarpetas: herencia y acceso legítimo del usuario.
5. Repetición: sin ACE duplicadas ni permisos adicionales.
6. Cancelar UAC, denegaciones y conflictos auténticos: sin éxito falso.
7. Sin PowerShell visible; comprobar cierre normal y watchdog.
8. Inspeccionar y registrar la carpeta desde AIDEX tras autorizarla.

Si vuelve a fallar, `C:\ProgramData\AIDEX\logs\folder-authorization.log` identificará
operación, recurso y código. No se atribuirá automáticamente otro error 32 al log.

## Entrega

`dist/aidex-1.0.0-windows-autorizacion-20261103/AIDEX-1.0.0-Windows-amd64.exe`

SHA-256: `970517b4f76ffe38ddf6967c46cd93c5b5152cf1c6ba0198986df4db54b53546`.

Evidencias: `evidencias/windows-autorizacion-20261103/`.
