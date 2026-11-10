# AIDEX 1.0.0 — Acceso efectivo Windows amd64

4 de noviembre de 2026. Entrega exclusivamente Windows.

## Diagnóstico y evidencia

El flujo anterior sí ejecutaba operaciones reales desde el proceso del servicio;
no era una evaluación indirecta de las ACL ni una prueba realizada como administrador.
Sin embargo, antes de leer el destino, `inspectDirectory` ejecutaba
`filepath.EvalSymlinks`: el Go utilizado hace `os.Lstat` sobre componentes y
`FindFirstFileW` para normalizar sus nombres. Esto puede exigir consultar atributos
o enumerar superiores donde solo corresponde recorrido. Una ACL RX/M del destino
no resuelve esa exigencia adicional. El mismo inspector intervenía al comprobar
y al planificar/confirmar el registro de la carpeta.

La evidencia es la implementación de Go 1.27.1 revisada en
`path/filepath/symlink.go` y `symlink_windows.go`. Referencias:
[FindFirstFileW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-findfirstfilew) y
[GetFinalPathNameByHandleW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfinalpathnamebyhandlew).

**No se puede atribuir con certeza el fallo del PC a una llamada concreta con el
registro recibido:** este había reducido el error del servicio a un código de
dominio. No se dispone aquí de Windows con AIDEX instalado. Se corrigió el requisito
adicional identificado en el código y se preparó su comprobación nativa; no se
declara confirmada la solución en esa instalación.

## Corrección

En Windows, el inspector abre directamente el destino con
`CreateFileW(FILE_GENERIC_READ)`, obtiene su ruta normalizada e identidad desde el
handle y enumera únicamente el destino. No ejecuta `EvalSymlinks`, búsquedas ni
aperturas de superiores en esta comprobación. Se conservan identidad física,
ruta canónica, límites de lectura y rechazo de puntos de reanálisis no resueltos.

Las operaciones se mantienen en el mismo hilo mientras se obtiene y utiliza el
token efectivo. Se registra si procede del hilo o del proceso, su SID y restricción;
en el servicio instalado también se compara el SID con `NT SERVICE\AIDEX`.
No se suplanta a un administrador ni se elevan permanentemente los privilegios.

La modalidad procede del parámetro `storage_source`. Una vinculada solo comprueba
lectura: el nombre físico de la carpeta no influye. Una administrada prueba creación,
escritura, sincronización, lectura, publicación sin reemplazo y retirada de una
prueba aleatoria, confinada al destino. No se modifican originales ni permisos.

Los errores ahora identifican operación, recurso, derechos, clase de acceso,
Win32/NTSTATUS original, SID efectivo, origen del token y PID. Ejemplos de operaciones:
`CreateFileW(selected_directory)`, `GetFinalPathNameByHandleW`,
`GetFileInformationByHandleEx(directory_entries)`, `WriteFile` y `NtSetInformationFile`.
Los detalles aparecen en `C:\ProgramData\AIDEX\logs\service.log` y los fallos se
replican en `folder-authorization.log`. El mensaje público sigue siendo breve.

No cambiaron las concesiones RX/Modify, herencia, UAC, watchdog ni controles de
seguridad del auxiliar. Tampoco se cambió su ejecutable GUI: se verificó idéntico
al instalador anterior. Una entrada ACL no se acepta como prueba de acceso.

## Archivos y aislamiento

- `internal/libraries/filesystem.go`: dos puntos mínimos de delegación a Windows,
  necesarios para que comprobación y registro utilicen el mismo inspector.
- `internal/libraries/folder_access_windows.go`: implementación y diagnóstico Windows.
- `internal/libraries/folder_access_windows_test.go`: pruebas nativas nuevas.

Solo Windows inicializa las delegaciones. Se comprobó que, al retirar esos puntos,
el cuerpo portátil coincide byte por byte con el anterior. Los nuevos archivos
están excluidos de las compilaciones macOS/Linux. Otros **211 archivos protegidos**
conservan sus hashes, incluidos macOS, Ubuntu, interfaz, licenciamiento y PKG/DEB.
No se editaron manuales ni se generaron paquetes de otras plataformas.

## Pruebas y límites

Ejecutadas: compilación Windows amd64 de bibliotecas/API y producto; `go vet` de
los paquetes afectados; siete pruebas de plantillas/parser; inspección del EXE,
versión 1.0.0, GUI, recursos, DLL y SHA de 1114 archivos. Todo PASS.

**Preparadas y compiladas, NO ejecutadas en Windows:** token restringido real con
superiores sin enumeración, carpeta vacía/con documentos en ambas modalidades,
rechazo del canonicalizador anterior, lectura nueva sin escritura vinculada,
prueba administrada sin residuos, conservación de originales/propietario y
registro del syscall denegado con Win32 5 y SID efectivo. La carpeta física se
llama “Administrada…” también al probar modalidad vinculada. No son simulaciones
de una respuesta exitosa, pero compilar no acredita su resultado nativo.

En Windows:

```powershell
go test -v ./internal/libraries -run NativeWindows
```

O ejecutar `build/aidex-windows-effective-libraries.test.exe -test.run NativeWindows -test.v`.
Estas fixtures no sustituyen la validación integrada del servicio instalado.

Pendiente en la instalación real: vacía vinculada, vinculada con documentos,
administrada con escritura, herencia y acceso del propietario, repetición sin
ACE duplicadas, cancelación UAC, denegaciones auténticas y registro posterior
desde AIDEX. La aplicación de ACL y sus pruebas anteriores no se alteraron.

Si persiste la denegación, la nueva traza mostrará cuál API falla y con qué token;
no se atribuirá automáticamente a falta de ACL ni al registro.

## Instalador

`dist/aidex-1.0.0-windows-acceso-efectivo-20261104/AIDEX-1.0.0-Windows-amd64.exe`

SHA-256: `20c188b79da2c50b0be8ec2d4de5589072841355877dbbe4903617dfeff20ba0`.

Evidencias: `evidencias/windows-acceso-efectivo-20261104/`.
