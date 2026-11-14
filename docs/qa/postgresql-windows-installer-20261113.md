# AIDEX 1.0.0: instalador PostgreSQL para Windows

Validación nativa del 13 de noviembre de 2026, rama `feature/postgresql`. Se obtuvo el último estado de `origin/feature/postgresql` antes de editar. No se generaron paquetes macOS ni Linux y no se tocó ningún documento original.

## Construcción

El instalador NSIS existente incluye ahora el runtime privado PostgreSQL 18.6 de EnterpriseDB: `bin`, `lib` y `share`, incluida la extensión `unaccent`. El ZIP de origen está fijado por SHA-256 en `packaging/windows/postgresql.lock.json`; el paquete incorpora su licencia y avisos de terceros. Las DLL de Visual C++ necesarias se copian desde el runtime OCR de conda-forge ya fijado por el proyecto. El instalador también conserva las herramientas PDF/OCR anteriores.

El constructor acepta `--windows-postgres-archive`, verifica el ZIP antes de extraerlo y comprueba los archivos esenciales y el cierre de dependencias PE AMD64. En Windows, NSIS necesitaba rutas de staging con barras invertidas; el constructor genera el separador adecuado para el host. Se añadió BOM UTF-8 al script NSIS para que el compilador lea correctamente el texto español.

Artefacto: `dist/postgresql-windows/AIDEX-1.0.0-Windows-amd64.exe`, **71,131,381 bytes**, SHA-256 `14846a8d96c196c97daac719304c846d8e6c288e94d05e73ca600a3e102f7aaf`. El mismo hash figura en `SHA256SUMS`. Se usaron Go 1.27.1, Node 22.15, Python portable 3.13.17 y NSIS 3.13. El frontend embebido se recompiló. Pasaron el análisis sintáctico de PowerShell 5.1, `scripts/test_windows_transition.ps1` y la compilación Python de los scripts de paquete. No se ejecutaron las baterías funcionales masivas ya aprobadas.

## Prueba de instalación real

Se ejecutó el EXE sobre esta computadora con `PATH` limitado a Windows, sin PostgreSQL global. Antes de instalar no existía servicio AIDEX, `config.json` ni base SQLite. El instalador silencioso tomó el puerto **18123** preparado en la opción existente del registro. Terminó con código 0 y creó:

- `C:\Program Files\AIDEX\tools\postgresql` con PostgreSQL y `unaccent`, además del runtime PDF/OCR.
- Clúster PostgreSQL privado con `PG_VERSION=18` en `C:\ProgramData\AIDEX\state\postgresql`.
- Servicio AIDEX con cuenta `NT SERVICE\AIDEX` y regla de firewall propia para el puerto 18123, limitada a la subred local.

El auxiliar de la prueba creó un administrador de prueba mediante las API internas. Su primera compilación omitía `buildinfo.ServiceName=AIDEX` y por ello volvió a proteger el directorio `state` sin el SID del servicio; el primer intento de arranque terminó con `Acceso denegado`. Se recompiló el auxiliar con la identidad del servicio, se restauraron las ACL ejecutando el binario de producción y el arranque pasó. Esta interferencia fue del auxiliar, posterior a la instalación, no del aprovisionamiento NSIS.

Con el servicio en ejecución, `/health/live` respondió HTTP 200; el inicio de sesión y `/api/v1/auth/session` respondieron correctamente contra PostgreSQL. El comando existente `manage.ps1 -Action Port` cambió el puerto a **18124**; HTTP y la regla de firewall quedaron actualizados. La primera desinstalación silenciosa terminó con código 0: retiró servicio, firewall y runtime empaquetado; conservó `config.json`, el clúster PostgreSQL y el usuario de prueba.

## Detección de Defender y límite de la reinstalación

La reinstalación sobre los datos conservados llegó a ejecutar `check-state`, pero Windows Security bloqueó la ejecución de `aidex-server.exe` al configurar: evento Defender 1116, detección **`Trojan:Win32/Bearfoos.A!ml`** con inteligencia `1.463.290.0`, y código 2 del instalador. El archivo detectado tenía el SHA-256 `2b5b2226a073b4631c2e42d699b766225eed851b2d370584867f5f11774367e0` según el manifiesto del paquete. Una recompilación idéntica también fue bloqueada por Defender. No hay dictamen que permita calificar la detección como falso positivo.

Se extrajo **en memoria** el binario exacto desde el instalador y se verificó su SHA-256 antes y después de incluirlo en `.tools/packaging/aidex-server-defender-sample.zip`, ZIP AES protegido con la contraseña indicada por Microsoft para muestras sospechosas. Tras la autorización y el inicio de sesión del usuario, [Microsoft Security Intelligence](https://www.microsoft.com/en-us/wdsi/filesubmission) confirmó la recepción el 13 de noviembre de 2026 a las 21:51, con ID de envío `5aae197c-a30f-403d-b6b8-843ffb5968dd`. El estado inicial es **Submitted** y el dictamen **Pending**. No se añadieron exclusiones de antivirus.

La instalación parcial bloqueada se retiró con un script que verificó la ruta, el servicio, el estado detenido de PostgreSQL y cada hash del manifiesto antes de borrar 2,702 archivos empaquetados. Quedaron conservados `config.json`, el clúster PostgreSQL y cualquier archivo ajeno al paquete. No queda servicio AIDEX registrado. La validación de **reinstalación completa** sigue pendiente hasta que Defender permita ejecutar el binario después de la revisión.

La investigación posterior de eventos, hashes, código y componentes está en [defender-windows-20261113.md](defender-windows-20261113.md).

## Evidencia

Registros locales ignorados por Git: `.tools/packaging/build.log`, `smoke-result.json`, `continue-result.json`, `service-diagnostic.txt`, `cleanup-result.json`, `ms-submit.log`, `ms-submission-result.txt` y `aidex-server-defender-sample.zip`. Los JSON de resultados no contienen la contraseña temporal; su archivo se borró tras la prueba. `scripts/build_installers.py`, `packaging/windows/installer.nsi`, `packaging/windows/postgresql.lock.json`, `packaging/windows/manage.ps1` y `scripts/check_production_packages.py` son los archivos de producto y paquete modificados.
