# AIDEX 1.0.0 — Corrección de reinstalación en macOS ARM64

Fecha: 29 de octubre de 2026.

## Causa comprobada

El `postinstall` anterior comparaba literalmente la salida de:

```sh
dscl . -read /Users/_aidex NFSHomeDirectory
```

con `NFSHomeDirectory: /Library/Application Support/AIDEX/data`. En esta Mac,
Directory Services devuelve el atributo en dos líneas:

```text
NFSHomeDirectory:
 /Library/Application Support/AIDEX/data
```

La comparación era falsa y ejecutaba `exit 1` sin diagnóstico. Solo se ejecutaba
en la rama de cuenta existente, lo que explica que la primera instalación
funcionara y la siguiente fallara antes de migrar o iniciar el servicio.

Se comprobó mediante lecturas de la cuenta real, sin modificarla: comparación
anterior `False`; lectura de `dscl -plist` mediante `plutil` y comparación del
valor real `True`. `/var/log/install.log` confirma el éxito del 26 de octubre
a las 21:43:21 y los fallos sin salida de postinstall a las 21:46:02 y el
29 de octubre a las 13:00:05. No es un fallo de `launchctl` ni de firma.

## Cambios

- Lectura estructurada de atributos de la cuenta; conservación de UID/GID.
- Diagnóstico de scripts con etapa, archivo, línea y código de salida. No se
  registra la línea de comando, contraseñas, códigos comerciales ni configuración.
- Detección de errores del directorio local sin confundirlos con cuentas ausentes.
- Recuperación del grupo de una creación interrumpida solo si su identidad no
  está compartida. Identidades ambiguas o incompatibles se rechazan con diagnóstico.
- Configuración existente sin marcador de versión: se conserva y valida.
  Configuración ausente con archivos de datos presentes: se rechaza la
  inicialización, solicitando recuperar la configuración.
- La actualización no vuelve a ejecutar `configure-license` sobre la
  configuración existente. La migración usa los mecanismos habituales del backend.
- Parada idempotente y espera máxima de 60 segundos por el proceso anterior.
  Inicio según el estado registrado: `enable`, seguido de `bootstrap` si falta
  el trabajo o `kickstart` si está cargado.
- Comprobación HTTP en la dirección y puerto configurados, con 30 intentos y
  un máximo de 2 segundos por solicitud. Los fallos de puerto o arranque quedan
  identificados por etapa.
- La falta del primer administrador termina correctamente con las instrucciones
  de configuración inicial. Otros errores de `bootstrap-ready` ya no se presentan
  como si faltara el administrador.
- La app Desinstalar AIDEX conserva su comportamiento y sus confirmaciones.

## Verificaciones completadas

| Verificación | Resultado y alcance |
|---|---|
| Lectura nativa de `_aidex` | Reproduce la comparación fallida y verifica la corrección; solo lectura |
| `test_macos_install_scripts.py` | 14 casos con Bash/plutil reales y dobles de dscl/launchctl/sudo; rutas temporales |
| `test_macos_uninstaller.py` | 9 casos de conservación, borrado selectivo, instalaciones parciales y rutas seguras |
| Go: config, storage, CLI y maintenance | Cuatro paquetes de pruebas aprobados; límite de 120 segundos |
| `smoke_macos_reinstall.py` | Binarios reales extraídos del paquete del 26 de octubre y del nuevo; datos temporales |
| Migración/reinstalación nativa aislada | Dos migraciones consecutivas; conserva configuración, identidad local de licencia, usuarios e historial; SQLite íntegro |
| Arranque nativo aislado | Dos arranques y paradas; `/health/live` y la página principal responden en un puerto libre |
| Exclusión de mantenimiento | `check-state` rechaza el acceso mientras el servidor temporal está activo |
| Originales sintéticos | Documento y carga conservados byte por byte |
| Paquete | Manifiesto SHA-256 íntegro, scripts coincidentes con el código fuente y versión 1.0.0 |
| Firma local | Backend y app verifican sus firmas ad hoc |
| Permisos BOM | App y auxiliar root:wheel 0755; LaunchDaemon root:wheel 0644 |

La prueba nativa no contacta el servidor de licencias: utiliza un destino local
sin servidor y renovación automática deshabilitada en su configuración temporal.
No se instaló, detuvo, desinstaló, reinicializó ni borró la instalación real.
No se modificaron Windows, Ubuntu ni manuales de Word.

## Pendiente de validación manual

Ejecutar el PKG mediante Installer.app sobre la instalación existente y confirmar
que el LaunchDaemon real y la interfaz responden en el puerto configurado. La
prueba de `launchctl` cubre sus estados mediante dobles, sin intervenir el servicio
del equipo. La autorización gráfica del desinstalador y la desinstalación real
también quedan para una instalación de prueba.

El PKG no está firmado con Developer ID ni notarizado. Esa advertencia Code=101
es independiente de la comparación defectuosa que provocaba Code=112.

## Entrega

- Ruta: `/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-reinstalacion-20261029/AIDEX-1.0.0-macOS-arm64.pkg`
- Tamaño: **6,738,316 bytes**.
- SHA-256: `2bc8f65c834279fe491b0e12f91cc2ab46a85e4f625bde0d4279cbf4c332d5da`.
- Desinstalador: `/Applications/AIDEX/Desinstalar AIDEX.app`.
