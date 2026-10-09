# AIBID 2.0 RC1 — ajustes de cierre y QA

Entrega **2.0.0-rc.1-r1**. Ajustes puntuales posteriores a fase 8. Se conserva el
esquema **14**, el puerto existente, los usuarios, la configuración, la licencia y
las bibliotecas. No se añade una migración de base de datos ni se mueve el estado.

Incluye [la revisión final de QA](QA-V2-RC1.md): recuperación de escrituras de
estado, aislamiento de archivos administrados inaccesibles, paginación al cambiar
filtros y respuestas HTTP con espera limitada. El informe distingue defectos
corregidos, cobertura existente y validaciones nativas pendientes.

## Diagnóstico del cambio de acceso en Windows

El mensaje anterior procedía de `network.Apply`: después de cerrar la aceptación
del listener local llamaba a `listen(next)`. Antes de intentar `net.Listen`,
`checkLANPort` conectaba a cada dirección del propio equipo con un timeout de
**250 ms**. Cualquier timeout abortaba y restauraba el acceso local.

Esa comprobación no es válida en Windows: rechazar una conexión a un puerto cerrado
puede tardar segundos. Los propios tests de Go documentan esa diferencia. Por tanto
un puerto libre podía clasificarse como no disponible **antes de intentar abrirlo**.
El mensaje mezclaba ese caso con un conflicto real o una denegación del bind.
El síntoma descrito es compatible con este defecto; sin la traza original del equipo
no se atribuye el fallo concreto a los permisos de `NT SERVICE\AIBIDTest`.

Además, el código de firewall era exclusivamente de diagnóstico: ni el servicio ni
el instalador creaban reglas. La ausencia de una regla no era una creación fallida;
esa funcionalidad no estaba implementada.

La corrección en Windows consulta `GetExtendedTcpTable` con IPv4 y
`TCP_TABLE_OWNER_PID_LISTENER`. Solo un puerto en estado LISTEN es un conflicto.
Las conexiones aceptadas y TIME_WAIT no bloquean artificialmente el cambio; la
petición HTTP que guarda el ajuste puede terminar. Se mantienen la apertura real
del socket, persistencia, reversión si falla y comprobación posterior de escucha.
Linux/macOS conservan su comprobación anterior. No se habilita `SO_REUSEADDR`.

Los errores distinguen comprobación del sistema fallida, puerto ocupado y apertura
del socket fallida. No se devuelven rutas ni salida cruda de PowerShell al navegador.

Referencias primarias:
[prueba de Go sobre puertos cerrados](https://github.com/golang/go/blob/master/src/net/dial_test.go),
[tabla TCP de Microsoft](https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable),
[estructura de filas IPv4](https://learn.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid).

## Firewall sin configuración manual por el usuario

El instalador, que ya se ejecuta elevado, prepara **AIBID-LAN-TCP** de forma
idempotente. Configurar AIBID también puede repararla. Su alcance es:

- Entrada TCP, solo el puerto de esta instalación.
- Solo el ejecutable instalado y el servicio `AIBIDTest`.
- Dirección remota `LocalSubnet` y perfiles **Domain/Private**.
- Sin perfil público, sin edge traversal y sin cambiar la política global.

La regla queda preparada incluso en modo local. **Esto no activa el acceso LAN**:
en ese modo el servicio escucha en `127.0.0.1`; al habilitar red desde Ajustes cambia
a `0.0.0.0`. Al volver a local desaparece la escucha LAN y el control HTTP también
rechaza nuevas peticiones remotas en conexiones anteriores. No se eleva la cuenta
virtual, no se le da pertenencia a Administradores y no se añade un servicio auxiliar
privilegiado. No es necesario modificar reglas en cada cambio de modo.

El diagnóstico consulta la regla **efectiva** de Windows y sus filtros, distingue
ausencia, deshabilitación/discrepancia, perfil público y restricciones a reglas
locales por política. No promete conectividad externa únicamente porque exista
una regla: bloqueos explícitos, GPO y firewalls ajenos pueden prevalecer. Una política
de dominio que impida reglas locales debe atenderla su administrador; AIBID no la
elude. La desinstalación retira solo su regla identificada y comprueba su pertenencia.

[Parámetros oficiales de reglas](https://learn.microsoft.com/en-us/powershell/module/netsecurity/new-netfirewallrule),
[política efectiva y actualización de reglas](https://learn.microsoft.com/en-us/powershell/module/netsecurity/set-netfirewallrule).

## Controles deshabilitados

Botones, campos, selectores y controles con `aria-disabled` usan `not-allowed`.
Se conserva la atenuación visual. El cursor de espera ya no se usa para indicar
una acción no disponible, incluidas las entradas deshabilitadas del explorador.

## Transición visible a AIBID

Instaladores Windows/macOS, accesos del Escritorio/Inicio, mensajes del lanzador,
nombre visible del servicio, comandos de configuración y descripción de paquetes
usan **AIBID**. Windows sustituye los accesos conocidos de AIBID Pruebas. En macOS
los accesos pasan a `/Applications/AIBID`; solo se retiran los antiguos que siguen
coincidiendo con el manifiesto del paquete anterior. Los modificados se conservan.
Ubuntu incorpora `aibid-admin` y `aibid-uninstall` y conserva los comandos anteriores.

La versión es candidata de publicación, no una declaración de firma/notarización
comercial. El canal del binario pasa a `release-candidate`.

| Identificador conservado temporalmente | Motivo |
| --- | --- |
| Servicio `AIBIDTest` y cuenta `NT SERVICE\AIBIDTest` | Mantener SID, permisos NTFS/SMB existentes y configuración del SCM. Nombre visible: AIBID. |
| `%ProgramFiles%\AIBID-Test`, `%ProgramData%\AIBID-Test` y clave de desinstalación `AIBIDTest` | Actualización en el mismo lugar, configuración/activación intactas y una única entrada de desinstalación. |
| macOS: `/Library/Application Support/AIBID-Test`, `_aibidtest`, `app.aibid.test` | Conservar propietario, UID/GID, launchd, recibo del paquete y rutas absolutas existentes. |
| Ubuntu: paquete/servicio/cuenta `aibid-test`, `/etc/aibid-test`, `/var/lib/aibid-test`, `/usr/lib/aibid-test` | apt actualiza el paquete actual sin una instalación paralela ni pérdida de estado; se añaden alias públicos AIBID. |
| Puerto 18090 y rutas técnicas del ejecutable | Mantener accesos, URLs y configuración de las instalaciones anteriores. |
| Fixtures, cuentas de test y documentación histórica | Son pruebas o evidencia de entregas previas, no nombres visibles de producción. |

Estos identificadores se mantienen también en instalaciones nuevas de RC1 para
conservar un único mecanismo de actualización. Su migración futura requiere un
procedimiento específico; no se fuerza con sustituciones globales de texto.

## Archivos y validación

Red: `internal/network/portcheck_windows.go`, `portcheck_other.go`, `porttable.go`,
`service.go`, `firewall_windows.go`, `firewall_report.go` y pruebas relacionadas.
Instalación: `packaging/windows/manage.ps1`, NSIS, scripts macOS/Ubuntu, constructor,
lanzador, versión y CI. Cursor: `web/src/styles.css` y comprobación en Chrome.

Pruebas: cambio de modo desde una petición HTTP activa, persistencia/reinicio,
conflicto y reversión; interpretación de tabla Windows y estados de firewall;
funciones de instalación con NetSecurity simulado; codificación/analizador PowerShell;
compilación Windows y regresiones Go/Chrome. Se añade una prueba nativa de la tabla
TCP a la suite de Windows, sin privilegios ni cambios de firewall.

La evidencia ejecutada se guarda en `dist/aibid-2.0-rc.1/VALIDACION-RC1.json`.
Los paquetes y sus hashes están en esa misma carpeta. La comprobación nativa del
paquete macOS usa datos temporales de fase 8. **Windows no se ejecuta en esta Mac**:
las pruebas de su tabla TCP real, SCM y firewall deben confirmarse en Windows o CI.

## Comprobación en tu Windows

1. Ejecuta el instalador RC1 sobre la instalación actual, sin desinstalar ni borrar
   la base de datos. Confirma que tus bibliotecas, usuarios y licencia permanecen.
2. Abre **AIBID** y entra a **Ajustes → Acceso y red**. El diagnóstico debe mostrar
   la regla preparada. La cuenta virtual y el puerto siguen siendo los anteriores.
3. Selecciona **Permitir acceso desde la red local**, guarda y entra desde otro
   equipo usando una dirección que muestre AIBID. La escucha debe ser `0.0.0.0:18090`.
4. Vuelve a **Solo este equipo**. La escucha debe regresar a `127.0.0.1:18090` y el
   otro equipo debe perder el acceso. La regla preparada puede seguir apareciendo
   habilitada; por sí sola no crea un puerto en escucha.
5. Reinicia el servicio y comprueba que conserva el modo. Comprueba los nuevos
   accesos del menú Inicio y el cursor de Guardar acceso cuando está deshabilitado.

En Ubuntu y macOS, comprobar la actualización y los nombres visibles. No se cambia
su gestión de firewall ni su mecanismo de cambio de listener en esta revisión.
