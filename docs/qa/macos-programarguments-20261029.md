# AIDEX 1.0.0 — Corrección de ProgramArguments en macOS ARM64

Fecha: 29 de octubre de 2026.

## Causa reproducida

En un plist temporal con cuatro argumentos, el comando anterior:

```sh
plutil -replace ProgramArguments.3 -string "$CONFIG" "$PLIST"
```

produjo cinco elementos en esta Mac: insertó la configuración sin retirar la
entrada anterior. La generación inicial del paquete ya contenía cuatro entradas
correctas; el defecto se introducía en `postinstall`.

La prueba anterior de empaquetado solo inspeccionaba la posición 3 y no detectaba
el elemento sobrante. La prueba nativa construía el comando del servidor de forma
independiente, por lo que tampoco ejercitaba los argumentos finales del plist.
Ambas comprobaciones se corrigieron.

## Corrección

`postinstall` llama a `aidex_rebuild_program_arguments`, que sustituye la propiedad
completa mediante un arreglo XML. Las rutas se escapan como texto XML. No se
modifican posiciones del arreglo ni se reutilizan argumentos anteriores.

El resultado es siempre:

```text
/Library/Application Support/AIDEX/bin/aidex-server
serve
--config
/Library/Application Support/AIDEX/data/config.json
```

Esto repara automáticamente las definiciones duplicadas al ejecutar el nuevo
PKG sobre la instalación existente. El manifiesto SHA-256 se actualiza después
de reconstruir el plist, manteniendo la desinstalación segura.

El proceso conserva la parada acotada antes de la migración. Si el trabajo está
cargado al registrar el servicio, se retira primero y luego se registra el plist
corregido: `kickstart` por sí solo podría reutilizar los argumentos que launchd
tenía almacenados. Solo se declara el arranque correcto después de obtener la
respuesta de AIDEX en el puerto configurado. La ausencia del primer administrador
sigue siendo un estado inicial válido, con instrucciones de configuración.

## Archivos modificados en esta corrección

- `packaging/macos/installer-common.sh`: reconstrucción completa del arreglo.
- `packaging/macos/postinstall`: uso del auxiliar y registro de la definición corregida.
- `scripts/test_macos_install_scripts.py`: igualdad exacta de los cuatro elementos,
  recuperación de duplicados, escape de rutas y uso de los argumentos por la simulación.
- `scripts/smoke_macos_reinstall.py`: arranque real desde el plist reparado,
  instalación limpia y conservación de una biblioteca sintética.
- `scripts/check_production_packages.py`: validación del arreglo completo y del
  auxiliar real del paquete, sustituyendo la comprobación insuficiente por índice.
- `docs/qa/macos-programarguments-20261029.md`: este informe.

No se modificaron Windows, Ubuntu ni manuales de Word. No se modificó la
instalación real de desarrollo durante las pruebas.

## Pruebas automatizadas y aisladas

| Escenario | Resultado y alcance |
|---|---|
| Instalación limpia | PASS; scripts con dobles del SO y binario real con datos temporales; primer administrador pendiente aceptado |
| Actualización funcional | PASS; scripts conservan datos y arrancan usando el arreglo reconstruido |
| Reinstalación 1.0.0 consecutiva | PASS; cuatro elementos exactos después de cada ejecución |
| LaunchDaemon cargado con configuración duplicada | PASS; definición retirada, reparada y registrada; manifiesto actualizado |
| LaunchDaemon sin registrar con argumentos sobrantes | PASS; elimina todos los argumentos adicionales |
| Servicio deshabilitado o detenido | PASS; habilitación y registro sin bootstrap sobre un trabajo cargado |
| Rutas con espacios, Unicode, ampersand y signos XML | PASS; tres reconstrucciones consecutivas conservan los valores exactos |
| Suite de scripts de instalación | 17 pruebas aprobadas; Bash/plutil reales y dobles de dscl/launchctl/sudo |
| Suite de desinstalación | 9 pruebas aprobadas; conservación predeterminada, originales y rutas seguras |
| Binario real del paquete | PASS; reparación deliberada del plist seguida de dos arranques HTTP desde sus argumentos |
| Conservación de datos reales de prueba | PASS; dos migraciones conservan usuarios, auditoría, biblioteca, configuración e identidad local de licencia; SQLite íntegro |
| Originales y cargas sintéticos | PASS; contenidos idénticos después de las pruebas |
| Paquete final | PASS; manifiesto íntegro, scripts coincidentes con el código fuente, cuatro argumentos exactos y versión 1.0.0 |
| App del desinstalador | PASS estructural; bundle, icono y firma ad hoc; auxiliar y ejecutable root:wheel 0755 |
| LaunchDaemon | PASS estructural; root:wheel 0644 |

La prueba nativa extrae el paquete del 26 de octubre y el nuevo sin instalarlos,
crea datos desechables y usa un puerto libre. La configuración temporal de
licencias apunta a un destino local sin servidor, con renovación deshabilitada.

Comandos principales:

```sh
python3 scripts/test_macos_install_scripts.py -v
python3 scripts/test_macos_uninstaller.py -v
python3 scripts/smoke_macos_reinstall.py dist/aidex-1.0.0-macos-argumentos-20261029/AIDEX-1.0.0-macOS-arm64.pkg --previous-package dist/aidex-1.0.0-etapa3-20261026-final/AIDEX-1.0.0-macOS-arm64.pkg
```

También se verificaron la sintaxis Bash/Python, los permisos del BOM y la
coincidencia del SHA-256 con `SHA256SUMS`.

## Validación manual pendiente

No se ejecutó Installer.app ni se registró un LaunchDaemon real durante estas
pruebas. Queda comprobar la actualización mediante doble clic sobre una
instalación de prueba, la respuesta del servicio real y la autorización gráfica
del desinstalador. El PKG está preparado para actualizar directamente; no requiere
desinstalar, borrar datos ni ejecutar comandos manuales para reparar los argumentos.

El PKG sigue sin firma Developer ID ni notarización. Las firmas ad hoc del
binario y la app se verificaron, y son distintas de la firma del paquete.

## Artefacto final

- Ruta: `/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-argumentos-20261029/AIDEX-1.0.0-macOS-arm64.pkg`
- Tamaño: **6,738,643 bytes**.
- SHA-256: `464aaafe8f7a7bf5c7f99203068d912f9aae94407a45a99cfbbe40a0bbca66b5`.
- Desinstalador integrado: `/Applications/AIDEX/Desinstalar AIDEX.app`.
