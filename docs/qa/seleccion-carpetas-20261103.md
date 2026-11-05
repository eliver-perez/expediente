# AIDEX 1.0.0 — Selección de carpetas y autorización Windows

Fecha: 3 de noviembre de 2026. Versión comercial conservada: **1.0.0**.

## Resultado y alcance

Implementados el botón **Seleccionar carpeta**, los auxiliares nativos y el
explorador restringido del servidor. Windows utiliza autorización nativa con UAC,
supervisión y diagnóstico. Se conserva la autorización de macOS: no se modificaron
`internal/macosfolders`, el auxiliar de ACL ni su aplicación AppleScript.

Se generaron instaladores Windows amd64, macOS ARM64 y Ubuntu amd64/ARM64 con el
código compartido actualizado. No se instaló ningún paquete en el sistema real.
No se modificaron manuales de Word ni la lógica de licenciamiento.

**La implementación no constituye una confirmación del fallo nativo de Windows.**
No se dispone de un Windows para reproducir los diez minutos de espera, comprobar
UAC ni ejecutar las nuevas pruebas ACL allí. Tampoco se completó una prueba del
selector gráfico en los escritorios de las tres plataformas. La matriz inferior
separa las comprobaciones realizadas de las pendientes.

## Causas encontradas en el código anterior

1. El protocolo abría `powershell.exe` elevado con `SW_SHOWNORMAL`: la consola
   podía verse vacía durante toda la ejecución.
2. Antes del primer mensaje de resultado se realizaban consultas CIM, validaciones,
   carga de WinForms, confirmación y operaciones ACL. Las consultas y los procesos
   `icacls` no tenían un supervisor con tiempo máximo.
3. La salida de `icacls` se descartaba con `Out-Null`. Aunque se comprobaba su código
   al terminar, no había progreso ni registro suficiente para localizar una espera.
4. Se reescribían ACL de superiores sin comprobar primero si el servicio podía
   atravesarlos. Escribir una ACL con entradas heredables puede propagar esas
   entradas a descendientes, incluso si el nuevo permiso añadido es solo recorrido.
   El nuevo flujo evita esa propagación en los superiores.
5. El endpoint admitía únicamente bibliotecas vinculadas; los destinos administrados
   quedaban excluidos de la autorización automática de Windows.

Estos son defectos verificables por revisión del flujo anterior. No se atribuye
la espera concreta observada a CIM, herencia o una confirmación oculta sin una
traza nativa del equipo afectado.

Referencia para la propagación de ACL y el uso de `MAXIMUM_ALLOWED` en superiores:
[Microsoft: SetSecurityInfo](https://learn.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-setsecurityinfo).

## Selección segura

- El navegador recibe un ticket aleatorio, temporal y vinculado a sesión/biblioteca.
  La utilidad nativa reclama y devuelve la selección mediante HTTP de loopback.
  No se usa `input webkitdirectory`, `C:\fakepath` ni un listado de archivos del cliente.
- Antes de mostrar el selector, la utilidad comprueba que el puerto pertenece al
  servicio instalado en su propio equipo: PID/SCM en Windows, launchd/PID en macOS
  y UID dedicado del listener en Linux. Una conexión que parece local por un túnel
  no autoriza enviar rutas del cliente. Si la comprobación falla, ofrece la
  alternativa del explorador habilitado o la entrada manual.
- Windows usa el selector del sistema; macOS, `choose folder`; Ubuntu, Zenity o
  KDialog desde una sesión de escritorio. Los auxiliares se ejecutan en la sesión
  del usuario, separados del servicio. Se registró `aidex-pick` en los instaladores.
- El ticket vence a los tres minutos. El auxiliar dispone de dos minutos; la
  verificación local y cada llamada HTTP tienen límites propios. La interfaz
  abandona la espera y explica las alternativas si no recibe un resultado.
- Al seleccionar, se completa el campo y se ejecuta la inspección existente con
  la identidad del servicio. Se mantienen modalidades, planes, confirmación,
  conflictos y comprobaciones de permisos. Una denegación abre la autorización.
- Un cliente remoto explora únicamente raíces registradas de su biblioteca y las
  ubicaciones que el administrador haya habilitado explícitamente. No se descubren
  discos, hogares ni directorios globales. Se exigen permisos para administrar
  raíces y ver rutas. Se excluyen estado privado y cargas privadas de AIDEX.
- El explorador usa límites de lectura y paginación. Abre y verifica cada componente
  de la frontera por descriptor; rechaza enlaces, escapes y cambios de identidad.
  Las rutas relativas del protocolo usan `/`, también al navegar en Windows.

Para habilitar ubicaciones iniciales adicionales, el administrador puede añadir
al `config.json` del servidor una lista por biblioteca y reiniciar el servicio:

```json
{
  "folder_browse_roots": [
    {"library_id": "ID_REAL_DE_LA_BIBLIOTECA", "path": "/srv/documentos/Archivo"}
  ]
}
```

El ejemplo es un fragmento: se conserva el resto de la configuración. En Windows,
se utiliza una ruta como `"D:\\Documentos\\Archivo"`. Las rutas deben ser absolutas
y canónicas. No se crea una carpeta ni se conceden permisos por configurarla.
Sin ubicaciones adicionales ni raíces registradas, se informa al usuario y se
mantiene disponible la entrada manual. En un Ubuntu sin escritorio no hace falta
instalar herramientas gráficas para utilizar AIDEX.

## Autorización y modal Windows

El protocolo eleva mediante UAC el ejecutable GUI instalado `AIDEX.exe`. No abre
PowerShell, CIM, `icacls` ni una consola. El auxiliar elevado supervisa un proceso
separado para las ACL, con un límite de **90 segundos** y espera de cierre acotada.
No vuelve a consultar SCM después de un timeout. Se registra la etapa y el error en:

`%ProgramData%\AIDEX\logs\folder-authorization.log`

La instalación y la cuenta real del servicio deben coincidir con
`NT SERVICE\AIDEX`. Se usan API de ACL por descriptor y se conservan propietarios,
entradas de otros usuarios, denegaciones y protección de herencia:

- Vinculadas: lectura y recorrido (`RX`).
- Administradas: lectura, escritura, creación y eliminación de documentos (`Modify`);
  no se concede a la cuenta del servicio administración de ACL ni propiedad.
- Herencia únicamente dentro del árbol seleccionado. No hay un recorrido recursivo
  explícito de documentos. Windows puede propagar la herencia necesaria dentro de
  ese árbol; esa operación también queda dentro del tiempo máximo del auxiliar.
- Primero se comprueba si basta con autorizar la raíz. Solo si sigue siendo
  necesario se añade recorrido en superiores, sin herencia y sin propagar sus
  entradas existentes a árboles ajenos, mediante handles `MAXIMUM_ALLOWED`.
- Se rechazan UNC para autorización automática, unidades de red mapeadas, reparse
  points, nombres abreviados que eludan la ruta canónica y ubicaciones protegidas.
  Los conflictos con otras bibliotecas y el estado privado se validan en el backend.

El servicio verifica lectura efectiva. Para destinos administrados también crea,
escribe, sincroniza, lee, publica sin reemplazo y retira una prueba aleatoria.
La respuesta indica que la comprobación cubre la **raíz**; no promete acceso a
cada documento con ACL explícitas o protegidas. No se eliminan originales ni se
retiran denegaciones para forzar un resultado positivo.

Los archivos administrados se crean en el destino con su herencia existente;
no se sustituyen sus permisos por una DACL privada de staging. Se añadieron pruebas
nativas Windows para conservar acceso del usuario original y verificar la herencia
mínima en documentos nuevos. **Esas pruebas están compiladas, pendientes de ejecución
en Windows.** Ejecutarlas allí con `go test ./internal/windowsfolders -v`, o con
`build/aidex-windows-folders.test.exe -test.v`. Usan directorios temporales y un SID
sintético; no sustituyen la prueba integrada del servicio instalado.

El modal presenta el nombre de carpeta, la finalidad de la autorización y los
botones **Autorizar carpeta** y **Comprobar acceso**, con SVG reales a la izquierda
y el mismo grupo de acciones que macOS. Muestra la operación y permite comprobar
el acceso o cerrar. La espera automática de Windows se limita a 115 segundos;
una cancelación de UAC también informa mediante la utilidad local. No se comunica
éxito sin la comprobación del servicio. Tras un resultado positivo se repite la
inspección para registrar la carpeta mediante el flujo existente.

## Pruebas ejecutadas

- TypeScript/Vite: PASS. Permanece el aviso existente de tamaño de bundle.
- `go test ./...`: PASS con sockets locales permitidos. Incluye pruebas ACL reales
  de macOS sobre archivos propios temporales; no sobre el usuario `_aidex` instalado.
- Pruebas de tickets, sesión revocada, caducidad, repetición, cancelación, entrada
  relativa inválida, exploración por biblioteca, estado privado, enlaces y escapes: PASS.
- Navegador: cinco escenarios distintos aprobados; tres repetidos tras los cambios
  finales de interfaz. Explorador remoto con API real, autocompletado/inspección,
  cancelación, modal Windows, destinos administrados y regresión del modal macOS.
  La respuesta del auxiliar y el sistema Windows se simulan en las pruebas web;
  esas pruebas no acreditan un selector nativo ni UAC.
- Tabla TCP real de macOS: socket temporal propio identificado correctamente y
  rechazo de otro PID. Analizadores Windows/Linux probados con tablas sintéticas;
  no se confunden con ejecución nativa en esos sistemas.
- 44 pruebas Python de empaquetado/perfiles/conservación: PASS. Se completaron dos
  fixtures antiguas que no incluían `listen_address` ni simulaban los sondeos de
  salud. Se verificó también la retirada de la nueva utilidad por manifiesto.
- Binario y pruebas Windows compilados; auxiliares Linux de ambas arquitecturas
  y macOS ARM64 compilados. Verificación de paquetes: PASS para los cuatro destinos,
  incluidos ejecutables, versiones, manifiestos SHA-256, recursos, firmas ad hoc,
  protocolos, scripts y cierre de DLL del runtime OCR reutilizado y verificado.
- Ciclo aislado del PKG: PASS para instalación, actualización, conservación,
  reinstalación y recuperación. Scripts, servidor, SQLite y documentos son reales;
  cuentas, launchd y recibos se simulan. Los originales externos y cargas se conservan.

En el sandbox se detectaron restricciones de puertos, consultas nativas y
compilación AppleScript. Las comprobaciones que lo requerían se ejecutaron con
acceso al sistema, manteniendo datos y servicios dentro de fixtures temporales.
No se relajó el código del producto para superar esas restricciones.

## Validación nativa pendiente

| Plataforma | Comprobado aquí | Pendiente en el sistema de destino |
| --- | --- | --- |
| Windows amd64 | Compilación, GUI subsystem, scripts, recursos, manifiesto, DLL, modal web | Selector, asociación URL, UAC/cancelación, watchdog real, cuenta `NT SERVICE\AIDEX`, vinculada/administrada, registro, creación/consulta y permisos de otros usuarios |
| macOS ARM64 | PKG, firmas ad hoc, ACL nativas en fixture, listener nativo, modal y ciclo aislado | Selector gráfico, asociación URL y flujo con `_aidex`/Finder/TCC reales |
| Ubuntu amd64 y ARM64 | DEB, ELF del servidor/selector, scripts y registro MIME | Escritorio GNOME/KDE, selección/cancelación, servicio instalado, permisos e instalación nativa |

No se declara reproducido ni resuelto nativamente el bloqueo de Windows solo por
compilación o simulación. Las firmas ad hoc del PKG no equivalen a firma comercial
ni notarización. Antes de distribución a usuarios debe completarse la matriz
nativa, especialmente los documentos nuevos y las ACL protegidas de Windows.

## Entrega

Paquetes: `dist/aidex-1.0.0-seleccion-carpetas-20261103/`.

- `AIDEX-1.0.0-Windows-amd64.exe`
- `AIDEX-1.0.0-macOS-arm64.pkg`
- `aidex_1.0.0_amd64.deb`
- `aidex_1.0.0_arm64.deb`

Hashes: `SHA256SUMS`. Inventario final: `paquetes-verificados.json`.
Evidencias: `evidencias/seleccion-carpetas-20261103/`.
