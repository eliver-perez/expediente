# AIDEX 1.0.0: investigación de la detección de Defender en Windows

13 de noviembre de 2026; rama `feature/postgresql`. Investigación estática y de registros. No se modificó el producto, la protección de Defender ni el instalador.

## Detección observada

El registro `Microsoft-Windows-Windows Defender/Operational` contiene eventos **1116** de detección y **1117** de cuarentena para `Trojan:Win32/Bearfoos.A!ml` (ID `2147731250`). La primera detección registrada fue a las **21:33:57** sobre `C:\Program Files\AIDEX\aidex-server.exe`; otras señalaron el servicio `AIDEX`, un `a.out.exe` generado por Go a las 21:38 y `aidex-repair.exe` extraído por NSIS a las 21:57. Los eventos indican protección en tiempo real, tipo `FastPath`, motor `1.1.26100.4` e inteligencia `1.463.290.0`. La inteligencia había pasado de `1.463.281.0` a `1.463.290.0` a las **21:12:41**. La secuencia temporal no demuestra que esa actualización causara la detección. Los eventos identifican archivos y procesos que los accedieron; **no describen una acción maliciosa ejecutada por AIDEX**. [Microsoft explica el significado de los eventos 1116 y 1117](https://learn.microsoft.com/en-us/defender-endpoint/troubleshoot-microsoft-defender-antivirus).

Se extrajeron en memoria las dos entradas del instalador ya generado, sin escribir el ejecutable detectado en disco. `aidex-server.exe` y `$PLUGINSDIR\aidex-repair.exe` son **idénticos**, de 17,196,032 bytes y SHA-256 `2b5b2226a073b4631c2e42d699b766225eed851b2d370584867f5f11774367e0`. El hash coincide con `package-files.json` dentro del mismo instalador y con el registrado durante la instalación. El script NSIS asigna deliberadamente el segundo nombre al mismo binario para `check-state` antes de sustituir archivos. El hash del instalador es `14846a8d96c196c97daac719304c846d8e6c288e94d05e73ca600a3e102f7aaf`.

## Comparación e integridad

| Propiedad | Ejecutable anterior conservado | Ejecutable detectado |
| --- | --- | --- |
| SHA-256 | `f13f6075700a6d3f221dba79a6c92382b93775e7325fd9482fd0f1641f9cb2ca` | `2b5b2226a073b4631c2e42d699b766225eed851b2d370584867f5f11774367e0` |
| Tamaño | 16,986,624 bytes | 17,196,032 bytes |
| Motor y formato | Go 1.27.1, Windows AMD64, `CGO_ENABLED=0`, `-trimpath`, sin firma Authenticode | Igual; ambos PE importan únicamente `kernel32.dll` |
| Base de datos | `modernc.org/sqlite` y sus dependencias | `github.com/jackc/pgx/v5` y PostgreSQL privado |

El binario señalado contiene metadatos Go de la revisión `e45c31c` con `vcs.modified=true`, canal de producción y diez dependencias cuyos hashes `h1` coinciden **10/10** con `go.sum`. El indicador `dirty` impide atribuir sus bytes únicamente al commit; en esa compilación había cambios del empaquetador sin confirmar. El constructor usa `-s -w` tanto en la versión antigua como en la nueva. La migración añade arranque y supervisión de `initdb`/`pg_ctl`, credenciales privadas y conexión PostgreSQL a `127.0.0.1`; en producción solo busca esas herramientas junto a AIDEX. La revisión dirigida del código nuevo de almacenamiento, rutas NTFS y scripts de instalación no encontró descarga de código, persistencia ajena al servicio previsto ni envío de datos a destinos nuevos. La detección del `a.out.exe` compilado por Go muestra que NSIS no es condición necesaria para la alerta.

Existe otra compilación PostgreSQL anterior (`720bd1d`, SHA-256 `7362bc1fce02b089e2721174ddcabd0373e1df20e4d2e4b90d654d1727e96738`, 17,190,400 bytes) creada antes de la actualización de inteligencia citada. No aparece un evento de detección asociado a ella; esto no demuestra que pasaría el análisis con la inteligencia `1.463.290.0`.

El ZIP local de Go 1.27.1 coincide con el [SHA-256 publicado por Go](https://go.dev/dl/). El archivo PostgreSQL 18.6 coincide con el hash fijado en `packaging/windows/postgresql.lock.json`; `postgres.exe`, `pg_ctl.exe` y `unaccent.dll` empaquetados son idénticos a sus entradas en ese archivo y al manifiesto del paquete. El constructor verificó las dependencias OCR contra `ocr.lock.json` y el cierre de DLL. [PostgreSQL identifica a EDB como distribuidor de los ZIP binarios de Windows](https://www.postgresql.org/download/windows/). No se obtuvo una firma o un hash del ZIP PostgreSQL publicados independientemente por EDB; el hash del lock solo prueba consistencia con el archivo usado. PostgreSQL se distribuye como proceso y DLL externos: el ejecutable Go no enlaza esas DLL (`CGO_ENABLED=0`).

## Conclusión y siguiente paso

**No se encontraron indicios verificables de código malicioso ni una alteración comprobada de la cadena de compilación.** Tampoco hay evidencia suficiente para declarar falso positivo: falta la regla o explicación técnica de Microsoft y una reconstrucción independiente byte a byte. El nombre de detección tiene una [descripción pública sin detalles técnicos del comportamiento](https://www.microsoft.com/en-us/wdsi/threats/malware-encyclopedia-description?name=Trojan%3AWin32%2FBearfoos.A%21ml&threatid=2147731250). La coincidencia de la detección con la nueva firma, el nuevo contenido PostgreSQL y la falta de firma del EXE son hipótesis; ninguna quedó aislada como causa.

La muestra exacta se envió a Microsoft con ID `5aae197c-a30f-403d-b6b8-843ffb5968dd`; último estado conocido: **Submitted / Pending**. [Microsoft indica que `Submitted` solo confirma la recepción y que debe esperarse la determinación final](https://learn.microsoft.com/en-us/defender-xdr/submission-guide). Mantener suspendida la distribución y la reinstalación de este artefacto hasta conocerla. Si Microsoft lo declara limpio, volver a comprobar el mismo hash con inteligencia actualizada y repetir únicamente la reinstalación pendiente. Si confirma malware, detener la distribución e investigar con mayor profundidad el entorno de compilación y los artefactos antes de publicar otro paquete. Registrar una compilación reproducible y firma de código para futuras entregas mejora la trazabilidad, aunque no explica por sí misma esta alerta.
