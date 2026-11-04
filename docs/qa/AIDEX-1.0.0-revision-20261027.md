# AIDEX 1.0.0 — revisión de incidencias (27 de octubre de 2026)

Esta revisión conserva la versión comercial 1.0.0. Los paquetes están destinados a validación interna; la inspección de su estructura y compilación no equivale a instalar y ejecutar el servicio en Windows o Ubuntu.

## Incidencias y correcciones

| Incidencia | Diagnóstico y corrección | Validación y límite |
| --- | --- | --- |
| Segundo documento sin acción | La acción dependía de la clasificación, origen y política efectiva de revisión; la interfaz omitía toda la sección en un borrador vinculado sin categoría. Ahora la consulta del flujo exige clasificación completa para ofrecer la transición y explica qué perfil debe continuar si el usuario no puede enviarlo o confirmarlo. El formulario obtiene la biblioteca por ID, sin depender de la primera página de bibliotecas. | Nueva prueba con dos cargas del mismo Colaborador, un Revisor independiente, cola asignada y compartida, y resolución de ambas. Pasa. |
| Confirmación directa | La política existente reserva `documents.finalize` al Gestor. El Gestor no podía ver determinadas cargas administradas clasificadas que seguían en estado temporal, aunque podía confirmarlas. Ahora puede ver solo esas cargas clasificadas en bibliotecas sin revisión obligatoria; los borradores sin clasificar siguen privados. El Colaborador recibe una explicación. | Prueba de visibilidad privada, denegación a otro Colaborador y confirmación por Gestor. Pasa. |
| Reasignación | `documents.reassign` requiere Gestor. El botón no indicaba por qué seguía inactivo y la búsqueda de la biblioteca podía fallar por paginación. Se muestra el requisito y se consulta la biblioteca exacta. | Prueba de rechazo al Colaborador y reasignación válida por Gestor con motivo y expediente destino. Pasa. |
| Carpeta local vinculada en Windows | La utilidad elevada comunicaba un resultado favorable incluso cuando `/finish` informaba que el servicio no podía leer la raíz. Tampoco comprobaba que el servicio existente ejecutara realmente como `NT SERVICE\AIDEX`. Ahora verifica esa identidad, aplica lectura y recorrido al destino, solo recorrido a sus ancestros y falla si el propio servicio no puede abrir la raíz. La configuración del servicio restablece la identidad en instalaciones existentes. | Script PowerShell parseado y paquete validado. Pendiente prueba nativa de ACL y UAC en Windows; la comprobación automática alcanza la raíz, y ACL explícitas en subcarpetas requieren diagnóstico individual. |
| Consolas de PowerShell durante NSIS | Las invocaciones `ExecWait` abrían consola. Las tres rutas (preparación, configuración y desinstalación) usan `nsExec::ExecToStack` con límite de tiempo y conservan código de salida y log UTF-16. | NSIS compiló; parser y estructura del instalador pasan. Falta comprobar visualmente en Windows, incluida actualización. |
| Primera instalación y reinstalación de macOS | El `postinstall` intentaba iniciar un servicio no preparado o hacer `bootstrap` de un trabajo ya cargado. Ahora espera el alta local del administrador en una instalación nueva, detecta trabajos heredados y conflicto de puerto, distingue trabajo cargado de no cargado, y comprueba la identidad AIDEX en `/health/live` durante un máximo de 30 intentos. La utilidad de configuración recupera una instalación parcial y se incluye la utilidad de cambio de puerto en el `.pkg`. | `sh -n`, `plutil` y estructura/firma ad hoc del paquete pasan. La reinstalación real requiere validación nativa sin tocar la instalación existente de desarrollo. |
| Dashboard lento en Windows | El servicio no conservaba en archivo los diagnósticos de tiempos que ya emite el backend. Ahora los registra en el directorio privado `%ProgramData%\AIDEX\logs\service.log`. No se alteraron consultas SQL sin evidencia de un cuello de botella. | HTTP con 10,000 documentos sintéticos en macOS: Dashboard ~128 ms, tarjetas ~25 ms, lista de bibliotecas ~0.5 ms. El caso de 7–15 s de Windows no se reprodujo aquí; consultar el log durante reconciliación/procesamiento real para separar espera de pool y etapas SQL. |
| Cierre de Brave en “Nombre visible” | El control es nativo y no transforma el valor ni mantiene un bucle de estado. No se encontró un desencadenante en el código. | Prueba de caracteres acentuados, apóstrofos, caracteres asiáticos y emoji en Brave/macOS pasa. Requiere versión de Brave/Chromium en Windows, extensiones, política empresarial, reproducción exacta y `brave://crashes` o Visor de eventos si reaparece. |
| Ubuntu AMD64 y ARM64 | Se revisaron `preinst`, `postinst`, `prerm`, `postrm`, unit systemd, cuenta, puerto, conservación de estado y datos. `postinst` y `aidex-admin` ahora comprueban respuesta del servicio AIDEX con espera limitada; `python3`, usado por las utilidades, quedó declarado como dependencia. | Sintaxis, arquitectura ELF, metadatos, propietarios y contenido de ambos `.deb` pasan. No hubo instalación nativa Ubuntu en esta Mac. |

## Regresión y seguridad

- `go test ./...`: pasa.
- Pruebas de desarrollo del flujo documental, privacidad, revisión y publicación: pasan.
- Compilación del frontend: pasa.
- Prueba de UI en Brave/macOS para nombre visible: pasa.
- Prueba HTTP de escala con 10,000 registros en base temporal: pasa; no se modificaron bases ni documentos reales.
- Pruebas Python del empaquetador: 7 pasan. Scripts shell: sintaxis correcta. PowerShell: sintaxis y parámetros correctos.
- Inspección de los cuatro paquetes: `PASS` en manifiestos y hashes internos, identidades, recursos gráficos, arquitectura, documentación y cierre de dependencias PE. No se incluyeron claves privadas, secretos ni códigos comerciales de prueba.
- Licenciamiento y CLUF de las etapas previas se conservaron; no se repitieron operaciones con licencias reales.

El runtime OCR temporal de Conda perdió sus metadatos durante esta compilación. Para Windows se reutilizaron únicamente 1,092 archivos de OCR/documentación del instalador anterior, verificados individualmente contra el manifiesto SHA-256 de ese paquete y comparados con el lock actual. El backend y el lanzador de este paquete sí se compilaron del código actualizado.

## Paquetes

| Plataforma | Archivo | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| Windows AMD64 | `dist/aidex-1.0.0-revision-20261027/AIDEX-1.0.0-Windows-amd64.exe` | 40,513,650 | `a569cd01412526916034b464093a3d86122a43083a0992ca29397fb26a6d65a1` |
| macOS ARM64 | `dist/aidex-1.0.0-revision-20261027/AIDEX-1.0.0-macOS-arm64.pkg` | 6,221,540 | `d51888654561aa81f55a8a1ac0d09ec50d213c9f0fa5e4de951cc76d4fc2611d` |
| Ubuntu AMD64 | `dist/aidex-1.0.0-revision-20261027/aidex_1.0.0_amd64.deb` | 6,615,006 | `ce322b1132836de4ee498f20a61c138ecf6c785be811a5dc40ab94233fff1b64` |
| Ubuntu ARM64 | `dist/aidex-1.0.0-revision-20261027/aidex_1.0.0_arm64.deb` | 6,110,070 | `1aa5a840885edb589e3d9877cc13596c6a11a70eb1d49e81c067e0cc19189e91` |

## Validación manual pendiente

En Windows, probar instalación limpia y actualización conservando datos, ausencia de consola, inicio del servicio y ACL de una carpeta local vinculada bajo un perfil de usuario. Inspeccionar identidad con `Get-CimInstance Win32_Service -Filter "Name='AIDEX'" | Select-Object StartName`, ACL con `icacls`, y el resultado de **Autorizar acceso**; verificar una carpeta con subdirectorios protegidos por ACL explícitas. Repetir el flujo documental con los tres usuarios independientes. Durante la carga de ~8,637 documentos y con reconciliación concurrente, medir Network y contrastar `%ProgramData%\AIDEX\logs\service.log` (campos `pool_ms`, `summary_ms`, `query_ms` y `total_ms`). Si Brave vuelve a cerrarse, recopilar versión, extensiones y reporte de fallo.

En macOS, probar el `.pkg` en un entorno aislado: primera configuración sin administrador, servicio heredado ocupando 18090, cambio de puerto, segunda instalación sobre el mismo AIDEX y recuperación de una instalación parcial. Confirmar datos/licencia intactos y respuesta del servicio. En Ubuntu 24.04 AMD64 y ARM64, instalar y actualizar con `apt`, crear administrador con `aidex-admin`, reiniciar con systemd, cambiar puerto, consultar una carpeta autorizada y desinstalar sin perder originales, base ni licencia. Estos pasos son necesarios antes de dar por resueltas las incidencias específicas del sistema operativo.
