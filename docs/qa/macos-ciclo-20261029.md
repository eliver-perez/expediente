# AIDEX 1.0.0 — Instalación, desinstalación y reinstalación en macOS

Fecha: 29 de octubre de 2026.

## Causa

El desinstalador llamaba a `erase-internal-data`, diseñado para borrar una lista
limitada de metadatos conservando cualquier otro archivo. Retiraba `config.json`
y SQLite, pero dejaba `state/service.lock`, registros, cachés y temporales.
El instalador encontraba archivos sin configuración y detenía correctamente la
inicialización. La prueba anterior verificaba solo la creación de otra base, sin
ejecutar de nuevo el `postinstall` real y su protección ante estos residuos.

## Corrección

Se añadió una operación completa específica para la desinstalación de macOS:
`erase-macos-installation`, con confirmación explícita. Los comandos anteriores
conservan su comportamiento. No se generaron ni modificaron paquetes Windows o
Ubuntu ni manuales de Word.

- Verifica la configuración, el área fija `AIDEX/data`, las rutas documentales
  registradas y el acceso exclusivo al estado antes de limpiar.
- Elimina archivos reservados de configuración, SQLite, índices/usuarios/auditoría
  contenidos en SQLite, licencia local, registros, cachés, temporales y respaldos
  internos reconocidos.
- Conserva los originales externos sin moverlos ni recorrer sus rutas.
- Los originales y archivos no identificados que estaban dentro de `data` se
  trasladan a `/Library/Application Support/AIDEX/Documentos conservados/<UUID>/`,
  manteniendo su estructura y contenido. El resultado indica la ubicación. Es
  una ubicación privada que puede requerir acceso administrativo.
- Los enlaces se conservan como enlaces, sin recorrer ni eliminar sus destinos.
  Un enlace en la propia ruta de configuración o estado causa un rechazo seguro.
- Borra archivos identificados individualmente y retira después directorios
  vacíos; no utiliza una eliminación recursiva indiscriminada del estado.
- Comprueba que `data` haya desaparecido. El auxiliar vuelve a verificarlo y no
  retira las herramientas ni el recibo del paquete si la limpieza falló o dejó
  residuos. Los errores se muestran como desinstalación incompleta.
- Espera la terminación del servicio con un límite de 60 segundos. Conserva
  `_aidex` y su identidad para respetar la propiedad de los documentos retenidos.
  Retira el LaunchDaemon y olvida exclusivamente el recibo `app.aidex`.
- El modo conservar mantiene la configuración, la base, la biblioteca y el estado
  local de licencia. La posterior instalación usa la cuenta administrativa anterior.
- La app mantiene conservación predeterminada, autorización nativa y confirmación
  adicional al eliminar. La advertencia explica la conservación de originales.
- Ninguno de los modos desactiva la licencia en el servidor.

El instalador conserva íntegramente la protección ante datos sin configuración.
La recuperación de residuos antiguos solo puede solicitarse explícitamente desde
el desinstalador actualizado: si faltan configuración y SQLite, limpia los restos
reconocidos y conserva los documentos. Si existe una base, WAL o SHM potencialmente
recuperable sin configuración, se detiene y conserva los datos. Un estado configurado
fuera de `AIDEX/data/state` requiere revisión y no se recorre para borrarlo.

## Pruebas completadas

| Escenario | Resultado |
|---|---|
| Instalación limpia | PASS con scripts y binario reales en una fixture; primer administrador pendiente aceptado |
| Actualización directa | PASS con servidor temporal activo y datos sintéticos |
| Conservar → reinstalar | PASS; inicio de sesión HTTP con el administrador y contraseña anteriores |
| Configuración, usuario, biblioteca y licencia local | PASS; datos conservados tras reinstalar |
| Eliminar → reinstalar limpio | PASS; SQLite nuevo sin usuarios ni bibliotecas; nueva cuenta administrativa creada e inicio de sesión correcto |
| Repetición de ciclos | PASS; dos reinstalaciones limpias consecutivas y ausencia de `data` tras cada eliminación |
| Residuos del desinstalador anterior | PASS; se ejecutó el borrado limitado anterior, se comprobaron logs sin config y luego se recuperó con el nuevo auxiliar |
| Originales externos vinculados, administrados e híbridos | PASS; contenido sintético conservado byte por byte |
| Originales internos y cargas | PASS; conservación en la carpeta separada, sin residuos en el área que debe reinicializarse |
| Enlaces, estado ocupado y base corrupta | PASS; protección de destinos y rechazo seguro antes de borrar metadatos |
| Bases recuperables sin config | PASS; conservadas, sin inicialización silenciosa |
| Auxiliar que comunica éxito sin limpiar | PASS negativo; el desinstalador detecta residuos y aborta conservando herramientas/recibo |
| Scripts de instalación | 17 casos aprobados |
| Scripts de desinstalación | 11 casos aprobados |
| Go | Pruebas de maintenance/CLI/config/storage aprobadas, incluidos seis casos nuevos de limpieza completa |
| Paquete final | Manifiesto íntegro, scripts coincidentes, cuatro ProgramArguments y versión 1.0.0 |
| App/binario | Firmas ad hoc válidas; auxiliar y ejecutable root:wheel 0755, LaunchDaemon root:wheel 0644 |

`smoke_macos_lifecycle.py` ejecuta preinstall, postinstall, el auxiliar de
desinstalación y el binario extraído del paquete. Usa SQLite, HTTP y procesos
reales en rutas temporales. Las cuentas del SO, launchctl y pkgutil se simulan;
la prueba comprueba sus transiciones y no modifica los recursos del sistema.
La licencia temporal apunta a un destino local sin servidor y sin renovación.

Comandos principales:

```sh
go test -timeout 120s ./internal/maintenance ./cmd/gestor-documental ./internal/config ./internal/storage
python3 scripts/test_macos_install_scripts.py -v
python3 scripts/test_macos_uninstaller.py -v
python3 scripts/smoke_macos_lifecycle.py dist/aidex-1.0.0-macos-ciclo-20261029/AIDEX-1.0.0-macOS-arm64.pkg
```

## Alcance de la validación nativa

No se utilizó Installer.app ni el LaunchDaemon real, y no se ejecutó la app gráfica
contra la instalación de desarrollo. Queda la validación manual de Installer.app,
la autorización gráfica y las cuentas, servicios y recibos reales en una instalación
de prueba. El PKG sigue sin firma Developer ID ni notarización.

## Archivos de esta corrección

- `cmd/gestor-documental/main.go`.
- `internal/maintenance/macos_erase.go` y `macos_erase_test.go`.
- `packaging/macos/uninstall-helper.sh` y `uninstall.applescript`.
- `scripts/test_macos_uninstaller.py` y `test_macos_install_scripts.py`.
- `scripts/smoke_macos_lifecycle.py` y `smoke_macos_reinstall.py`.
- Este informe técnico.

## Entrega

- PKG: `/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-macos-ciclo-20261029/AIDEX-1.0.0-macOS-arm64.pkg`.
- Tamaño: **6,753,727 bytes**.
- SHA-256: `7cc963633c81ff11a27a23e7184eecf6415b469ebf7bfd404657a2f2c4875311`.
- App: `/Applications/AIDEX/Desinstalar AIDEX.app`.
