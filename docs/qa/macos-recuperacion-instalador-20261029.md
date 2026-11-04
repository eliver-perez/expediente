# AIDEX 1.0.0 — Recuperación directa de residuos desde el instalador

Fecha: 29 de octubre de 2026. Objetivo exclusivo: macOS ARM64.

## Causa y corrección

El guard de `postinstall` usaba `find` para bloquear cualquier archivo o enlace
cuando no existía `config.json`. No distinguía `logs/service.log` o
`state/service.lock` de una base o licencia recuperable. La recuperación añadida
anteriormente al desinstalador no participaba en la instalación del nuevo PKG.

Ahora `postinstall` invoca una operación específica de recuperación, independiente
de la eliminación completa del desinstalador. Inspecciona primero todo `data`, sin
seguir enlaces, y solo permite registros de servicio conocidos, bloqueos vacíos y
marcadores reconocidos de versión/servicio. Los directorios vacíos conocidos se
conservan. Antes de retirar archivos exige el bloqueo exclusivo del estado y
vuelve a comprobar el inventario. No recorre rutas documentales externas.

SQLite, WAL, SHM, licencias, respaldos de configuración, originales, enlaces,
tipos especiales y elementos desconocidos impiden la recuperación. Se informa
el tipo/ruta relativa encontrado y no se inicializa ni borra ese inventario.
Los mensajes de recuperación enumeran los residuos retirados y el resultado;
los errores de los comandos conservan etapa, archivo, línea y código en
`/var/log/install.log`, sin imprimir configuración, claves ni contenidos.

El código 113 se comprobó mediante una consulta de solo lectura a un servicio
inexistente de launchd: significa que el servicio no está registrado. La captura
de salida heredaba el trap ERR de Bash y generaba un ERROR engañoso. Se trata
ahora explícitamente: servicio ausente es una condición normal; otros errores de
consulta/bootout detienen la instalación. Un 113 en bootout requiere confirmar
que el trabajo realmente desapareció. La espera de terminación continúa acotada.
La misma captura corregida evita un falso ERROR cuando falta el primer
administrador; los errores reales de bootstrap siguen siendo fatales.

La reconstrucción determinista de los cuatro ProgramArguments permanece intacta.
El desinstalador gráfico y su auxiliar no requirieron modificaciones.

## Archivos modificados en esta incidencia

- `packaging/macos/postinstall`
- `packaging/macos/installer-common.sh` (también utilizado por `preinstall`)
- `internal/maintenance/macos_install_recovery.go`
- `internal/maintenance/macos_install_recovery_test.go`
- `cmd/gestor-documental/main.go` (comando de mantenimiento limitado a macOS)
- `scripts/test_macos_install_scripts.py`
- `scripts/smoke_macos_lifecycle.py`
- Este informe de QA.

`preinstall` fue revisado; su lógica no necesita otra modificación aparte del
auxiliar compartido. No se modificaron Windows, Ubuntu, manuales ni la instalación
real de desarrollo. No se contactó al servidor de licencias.

## Validaciones efectuadas

1. Sintaxis Bash de preinstall, postinstall y auxiliar; compilación sintáctica de
   los dos scripts Python: correctas. El primer intento de py_compile no pudo
   escribir en la caché global del sandbox; se repitió con caché en `/private/tmp`.
2. `go test ./internal/maintenance -run TestMacOSInstallerRecovery -count=1
   -timeout=45s`: correcto. Cinco pruebas principales con subcasos para residuos,
   idempotencia, bases/WAL/SHM, licencia, respaldo, originales, desconocidos,
   enlaces, configuración existente y estado ocupado. Se canonizaron las rutas
   temporales del test para no confundir el enlace de macOS `/var` con la fixture.
3. Suite de scripts: 22 casos. La primera pasada cubrió correctamente 21 y
   detectó el falso ERROR al faltar administrador. Tras corregirlo se repitieron
   los tres casos afectados (residuos, instalación limpia y error real de
   bootstrap), todos correctos. Incluye servicio inexistente/113, errores reales,
   carrera en bootout, espera acotada, actualización, reinstalación consecutiva,
   reparación de argumentos duplicados y conservación de datos.
4. PKG generado mediante el empaquetador existente, versión comercial 1.0.0.
   Los cuatro scripts extraídos coinciden byte por byte con sus fuentes
   sustituyendo únicamente `@VERSION@`. El plist contiene exactamente ejecutable,
   `serve`, `--config` y ruta. Firmas ad hoc del servidor y aplicación verificadas.
5. `smoke_macos_lifecycle.py` contra el **PKG entregable**: PASS. Ejecuta sus
   preinstall/postinstall y servidor real, con rutas temporales y dobles de las
   herramientas administrativas del SO. Resultados:
   - Instalación limpia: éxito y primer administrador pendiente como corresponde.
   - Creación del administrador, arranque HTTP y acceso real: correctos.
   - Actualización directa: conserva el acceso del administrador anterior.
   - Desinstalación conservando datos y reinstalación: configuración, identidad
     local de licencia, usuario/hash y biblioteca sintética intactos; login válido.
   - Falta de config con base/licencia: postinstall rechaza la inicialización;
     todos los archivos previos permanecen byte por byte.
   - Eliminación completa y reinstalación limpia: dos ciclos correctos; SQLite
     íntegro, sin usuarios/bibliotecas anteriores y con creación de administrador.
   - Reproducción del borrado antiguo: sin config ni SQLite, con logs y
     service.lock, servicio detenido y cuenta retenida. **Reinstalación directa
     mediante postinstall del nuevo PKG, sin ejecutar antes el desinstalador**:
     éxito; se comprueba el retiro de ambos residuos y ausencia de falsos ERROR.
     Se crea el administrador nuevo y se verifica login HTTP con el servidor real.
   - Originales externos y cargas/administrados internos conservados en sus rutas
     externas o en el área de documentos conservados del desinstalador.
   - Cada postinstall exitoso comprueba los cuatro argumentos exactos.
6. Aplicación gráfica integrada: bundle y firma válidos, auxiliar idéntico al
   existente; AppleScript compilado comprobado por decompilación: conservación
   predeterminada, confirmación ELIMINAR y solicitud de privilegios administrativos.

Las acciones administrativas de launchd, dscl, sudo y recibos están simuladas en
la prueba aislada. Los scripts, operaciones de archivos, SQLite, bootstrap,
servidor y solicitudes HTTP son reales. Cada subproceso tiene timeout (incluidos
35 s por script, 30 s para bootstrap y 10 s para HTTP); el proceso temporal se
detiene al finalizar. No se ejecutó Installer.app ni la autorización gráfica
contra el sistema anfitrión. Esa instalación nativa y el diálogo de elevación
requieren validación manual en un equipo de pruebas. El PKG no está firmado con
Developer ID Installer ni notarizado; las firmas locales ad hoc no sustituyen
esa firma de distribución.

## Entregable

Ruta absoluta:

`/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-recuperacion-20261029/AIDEX-1.0.0-macOS-arm64.pkg`

Tamaño: **6,759,750 bytes**.

SHA-256:

`374d39fe461065bcf2211fe812331e3cf5612a759ab4ee1d4c21b039f88bbced`

Este paquete se instala directamente sobre el estado residual descrito; no
requiere ejecutar primero el desinstalador. Si encuentra datos recuperables sin
configuración, su rechazo es intencional: restaurar/revisar la configuración sin
eliminar esos datos. En una instalación nueva, usar Configurar AIDEX.command
para crear el primer administrador y arrancar el servicio.
