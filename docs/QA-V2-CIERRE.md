# AIBID V2 RC1 — cierre de verificaciones

Estado revisado el **7 de octubre de 2026**. Se reutilizaron los paquetes y
resultados existentes. En esta revisión no se repitieron pruebas, no se
regeneraron instaladores y no se modificó el código de la aplicación.

## Qué ocurrió

El primer `go test -race` con la carga de 9.000 archivos comenzó a las **15:33:12**
y terminó a las **15:43:15** por el límite global del ejecutor: `test timed out
after 10m0s`. En ese momento `TestLargeLibrary9000` llevaba 4 min 44 s; el resto
del tiempo correspondía a pruebas anteriores del paquete. La traza muestra su
goroutine ejecutable dentro de SQLite durante la incorporación de archivos, no
una espera indefinida demostrada. Esa ejecución quedó fallida por timeout; no se
contabiliza como aprobada.

La recuperación ya se había completado antes de esta revisión:

| Comprobación existente | Resultado registrado |
| --- | --- |
| Desarrollo completo, incluida carga | 175 pruebas principales aprobadas; 15:45:53–15:47:53 |
| Carga de 9.000 archivos | Incorporación/interrupción/reapertura: 20,90 s; pasada incremental: 6,92 s; 9.000 trabajos únicos |
| Concurrencia, separada de carga | 104 pruebas principales aprobadas; 15:45:56–15:56:19; presupuesto explícito de 30 min |
| Navegador | 22 escenarios aprobados en 2 min; finalizó a las 15:48:16 |
| Build de producción | 80 pruebas principales aprobadas; 21:55:41–21:55:49 |
| Empaquetado de los cuatro destinos | Completado a las 21:56:31 |
| Prueba aislada del paquete macOS | Aprobada a las 21:56:40, incluida actualización desde fase 8 |
| Inspección final de los paquetes | Cuatro resultados PASS; último registro a las 21:59:40 |
| Integridad de la entrega | 54 entradas verificadas, 54 OK; registro a las 22:00:23 |

Las horas son las registradas en este equipo (UTC−06:00). Los recuentos de suites
se solapan: no deben sumarse como si fueran pruebas distintas. No hay evidencia
de un único comando ejecutándose durante 37 minutos. Los registros tampoco
permiten determinar por qué la interfaz pudo seguir mostrando actividad.

## Estado de procesos

Las consultas de procesos y puertos se ejecutaron con **timeout de 10 segundos**.
No quedan procesos de pruebas Go, Playwright/Chromium, empaquetado, inspección,
LibreOffice u OCR correspondientes a estas verificaciones.

Se identificaron dos procesos anteriores a esta tarea:

- Servicio instalado `gestor-documental`, PID 83299, iniciado el 25 de septiembre.
- Servidor Go de desarrollo, PID 16559 (padre 16553), iniciado el 19 de agosto,
  con escucha en el puerto 8081.

No son procesos residuales de esta ejecución de QA y no se detuvieron. La ausencia
de verificadores activos y los finales PASS permiten cerrar las comprobaciones
locales; no había un proceso de QA bloqueado que reparar.

## Estado de los cuatro paquetes

Todos corresponden a **AIBID 2.0 RC1**, esquema 14. La revisión de cierre confirmó
su presencia, tamaño coincidente con el informe, las 17 evidencias registradas y
el resultado previo de integridad. No volvió a calcular hashes ni ejecutó pruebas
ya aprobadas.

| Paquete | Bytes | Validación completada | Pendiente nativo |
| --- | ---: | --- | --- |
| [Windows amd64](../dist/aibid-2.0-rc.1/AIBID-2.0.0-rc.1-r1-Windows-amd64.exe) | 40.258.350 | Compilación, dependencias PE, scripts extraídos/PowerShell, manifiesto y simulación de firewall | Instalador/SCM/PowerShell 5.1, LAN y SMB en Windows real |
| [macOS Apple Silicon](../dist/aibid-2.0-rc.1/AIBID-2.0.0-rc.1-macOS-arm64.pkg) | 6.033.719 | Manifiesto, binario nativo, PDF/OCR, actualización aislada, LAN/local y reinicio | Instalador del sistema, launchd y accesos interactivos |
| [Ubuntu amd64](../dist/aibid-2.0-rc.1/aibid-test_2.0.0~rc.1_amd64.deb) | 6.411.306 | ELF, arquitectura, control, dependencias, contenido y actualización prevista | Instalación, actualización y desinstalación con systemd |
| [Ubuntu ARM64](../dist/aibid-2.0-rc.1/aibid-test_2.0.0~rc.1_arm64.deb) | 5.926.670 | ELF, arquitectura, control, dependencias, contenido y actualización prevista | Ciclo nativo en Ubuntu 24.04 ARM64 |

## Informe de QA y decisión

El [informe completo de QA](QA-V2-RC1.md) detalla lo revisado, los defectos reales
corregidos y la cobertura previa conservada. Las correcciones incluyen la
comprobación LAN/firewall de Windows, cursor y marca, paginación con filtros,
timeout/mensajes HTTP, recuperación al guardar estados finales y aislamiento de
archivos administrados inaccesibles.

Los resultados y sus omisiones están en
[VALIDACION-RC1.json](../dist/aibid-2.0-rc.1/VALIDACION-RC1.json), con sus
[hashes de entrega](../dist/aibid-2.0-rc.1/SHA256SUMS). Las pruebas de licencia real
y corpus privado no se ejecutaron; se conservaron pruebas con simulador/fixtures.
LibreOffice sí se probó en desarrollo y navegador. La carga no mide 9.000 OCR ni
rendimiento de SMB.

**No quedan verificaciones locales pendientes.** La entrega continúa como
candidata: falta la validación nativa descrita en la tabla, el ensayo de corpus
real/SMB y recuperación de almacenamiento/backup, y la firma comercial/notarización
antes de distribución general. Ninguno de esos pendientes justifica repetir las
pruebas locales aprobadas ni regenerar los paquetes sin cambios.
