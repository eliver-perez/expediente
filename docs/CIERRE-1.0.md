# AIBID — ajustes para el cierre de 1.0 (r7)

Fecha: 2026-09-29. Cambios sobre la arquitectura existente: Go, SQLite y React.
Se conservan los cambios previos de r6 y el soporte Ubuntu ARM64. No se incorporan funciones de 2.0.

## Uso

Como administrador, abrir **Acceso y red** y elegir:

- **Solo este equipo**: `127.0.0.1:PUERTO`.
- **Permitir acceso desde la red local**: `0.0.0.0:PUERTO` (IPv4).

El puerto actual permanece fijo, normalmente 18090. Se muestran las URLs por IP del equipo y las
interfaces útiles. El cambio se guarda y se aplica sin reiniciar los trabajos documentales.
Al volver a local desde otro equipo, la respuesta informa que debe continuarse en el equipo de AIBID.
La dirección local se muestra como texto para evitar que un cliente remoto la confunda con su propio PC.

## Decisiones y compatibilidad

- HTTP por LAN requiere una elección explícita (`network_mode: "lan"`). Una configuración anterior
  con `0.0.0.0` sin ese campo continúa exigiendo HTTPS. Las configuraciones HTTPS/proxy existentes
  funcionan igual y no se convierten automáticamente a HTTP desde la pantalla nueva.
- Se conserva el puerto para mantener los accesos directos existentes. La validación ahora exige
  un puerto numérico canónico entre 1 y 65535. No se añade una pantalla de cambio de puerto.
- El cambio reemplaza solamente el listener HTTP. Se mantiene la base de datos abierta, las sesiones,
  el procesamiento, las bibliotecas y el cliente de licencias. Los HTTP aceptados terminan normalmente;
  las conexiones anteriores no pueden seguir accediendo remotamente tras volver a local.
- Se valida y reserva la nueva escucha antes de guardar. Si falla el puerto o la escritura de la
  configuración, se restaura la escucha anterior. Si otro proceso toma el puerto justo durante la
  restauración, se conserva el archivo anterior y se comunica el error al supervisor del servicio.
- macOS/BSD admite ciertas combinaciones de escucha general e IP específica. Se comprueba también
  que ninguna de las IP propias atienda ya el puerto antes de reservar la escucha LAN. No se exploran
  otros equipos ni otros puertos.
- El archivo privado se sustituye mediante un temporal protegido, sincronización y renombrado.
  Se rechazan cambios si el archivo fue editado fuera del servicio, para no pisar otra configuración.
  Ubuntu concede escritura únicamente a `/etc/aibid-test` dentro de la protección de `/etc` del servicio.
- Se mantienen autorización administrativa, cookie HttpOnly/SameSite, tokens CSRF y comparación
  exacta de Origin. En LAN el Host debe ser una IPv4 real del servidor con su puerto, o 127.0.0.1.
  No se confía en cabeceras de proxy en el modo HTTP local/LAN.
- El navegador genera UUID v4 con `crypto.getRandomValues` cuando `crypto.randomUUID` no está
  disponible en HTTP por IP. No se utiliza `Math.random` para claves de idempotencia.
- No se necesita Apache, Nginx o IIS. No se implementan UPnP, NAT, redirecciones de router, túneles ni cloud.

## Firewall e instaladores

La implementación es deliberadamente de **diagnóstico sin mutaciones**. El servicio se ejecuta con
cuentas restringidas y no tiene una sesión de escritorio para elevar privilegios de forma interactiva.
Se usa la alternativa expresamente permitida en los requisitos: informar cuando no es posible verificar
o gestionar automáticamente, y mostrar instrucciones específicas del sistema.

- Linux detecta UFW, firewalld, nftables e iptables, sin asumir que estén instalados o activos.
  Consulta UFW/firewalld cuando los permisos lo permiten; identifica nftables/iptables como herramientas
  que requieren revisión administrativa. La ausencia de una herramienta no prueba ausencia de filtrado.
- Windows consulta los perfiles de Windows Defender Firewall mediante PowerShell, sin ejecutarlo elevado.
  No interpreta perfiles habilitados como prueba de que una regla concreta permita AIBID.
- macOS consulta Application Firewall y, cuando es posible, el estado del ejecutable. También informa
  cuando no puede determinarlo. Permisos concedidos al ejecutable no equivalen a conectividad de extremo
  a extremo; pueden existir otros filtros o políticas.
- Las consultas tienen tiempos y salida limitados. No se muestran comandos ni salidas técnicas crudas
  en el flujo de uso; sí instrucciones para el administrador cuando hagan falta.
- AIBID r7 **no crea, cambia ni elimina reglas de firewall**. Por tanto, no quedan reglas propias que
  duplicar, migrar o retirar al desinstalar. Los desinstaladores conservan reglas ajenas, tanto si se
  guardan los datos como si se elimina la información interna. No se intenta adivinar propiedad por puerto.
- No se desactiva ninguna protección global. Una autorización que el administrador configure por su
  cuenta continúa siendo suya y no se elimina por coincidencia de nombre, ejecutable o puerto.

Referencias revisadas: [Windows Defender Firewall](https://learn.microsoft.com/en-us/powershell/module/netsecurity/get-netfirewallprofile),
[Application Firewall de macOS](https://support.apple.com/en-za/guide/mac-help/mh34041/mac),
[firewall de Ubuntu](https://ubuntu.com/server/docs/firewalls/),
[consulta de firewalld](https://firewalld.org/documentation/man-pages/firewall-cmd.html).

## Licenciamiento

Se revisó el contrato V1, sus verificaciones de Ed25519, enlace con la instalación/huella, límites,
revisiones aceptadas, estado local, archivo de identidad, solicitudes online y archivos offline.

- Una instalación y usuarios ilimitados, con los módulos firmados por el proveedor.
- Licencia perpetua sin vencimiento de uso; mantenimiento y versiones incluidas permanecen separados.
- Suscripción: 15 días exactos de tolerancia desde su vencimiento; finalizada la tolerancia conserva
  consulta/descarga y administración. No se extiende la tolerancia cambiando el reloj hacia atrás.
- Fallos de conexión, DNS, timeout o servidor no sustituyen el estado firmado por una revocación.
  Se conserva la información válida localmente, también después de reiniciar.
- Renovación periódica según la configuración existente, normalmente 24 horas. Los reintentos pendientes
  conservan su identificador; el bucle espacia nuevos intentos 15 minutos. Se muestra la siguiente
  validación programada o pendiente; tras reiniciar una validación ya vencida puede intentarse al arrancar.
- La interfaz distingue perpetua activa, suscripción activa, tolerancia, vencimiento, activación pendiente,
  revocación/desactivación y fallo temporal de validación. Los códigos técnicos se muestran en soporte.
- Activación/desactivación/reactivación mantienen la identidad local; una nueva activación no reutiliza
  la activación retirada. Un problema transitorio no libera la plaza ni inventa una activación.
- Las pruebas usan claves generadas o el simulador. No se activó/desactivó la licencia real del propietario.
  No se cambió el contrato ni se modificó el servidor de licencias.

## Interfaz

Los cinco filtros del explorador preceden a la barra de búsqueda: raíz, vista lógica, disponibilidad,
tipo de búsqueda y ámbito. Se conservan sus valores, handlers y consultas. La distribución se adapta
al ancho disponible.

`Accordion.tsx` reutiliza el patrón nativo `details/summary` y la clase visual de Huella y trazabilidad.
Lo usan Huella y trazabilidad, Vista previa del formato guardado, Información avanzada · registro original,
Configurar carpeta, Ver huella completa del grupo e Identificación para soporte. Se conservan sus contenidos,
acciones y estados independientes; no cambian la generación de nombres, watcher, SHA-256 ni consultas.

## Archivos de esta revisión

| Área | Archivos principales |
| --- | --- |
| Configuración privada | `internal/config/config.go`, `save.go`, `config_test.go` |
| Servicio de red | `internal/network/service.go`, `interfaces.go`, `diagnostic.go`, `firewall.go`, `firewall_linux.go`, `firewall_windows.go`, `firewall_darwin.go`, `service_test.go` |
| Arranque y API | `cmd/gestor-documental/main.go`, `internal/httpapi/server.go`, `network_settings.go`, `network_test.go`, `server_test.go` |
| Licencias | `internal/licensing/client.go`, `network.go`, `availability_test.go`, `online_development_test.go`; `web/src/License.tsx` |
| Pantalla de red | `web/src/NetworkSettings.tsx`, `App.tsx`, `styles.css` |
| UUID sobre HTTP | `web/src/requestID.ts`, `Libraries.tsx`, `License.tsx`, `Uploads.tsx` |
| Acordeones y filtros | `web/src/Accordion.tsx`, `Documents.tsx`, `ManagedSettings.tsx`, `AuditTable.tsx`, `LibrarySettings.tsx`, `Duplicates.tsx`, `Libraries.tsx`, `License.tsx` |
| Pruebas de navegador | `internal/e2eserver/main.go`; `web/tests/network.spec.ts`, `license.spec.ts`, `navigation.spec.ts`, `revision-r6.spec.ts`, `workflow.spec.ts` |
| Empaquetado y documentación | `packaging/ubuntu/aibid-test.service`, `scripts/build_installers.py`, `scripts/smoke_macos_installer.py`, `docs/H7.md`, `docs/MILESTONES.md`, `packaging/TEST-PLAN.md`, este informe |

## Validación

- Suite Go normal: aprobada.
- Suite Go con `-tags development`: aprobada, incluyendo procesamiento documental y simulador de licencias.
- Detector de carreras en red, licencias y API: aprobado.
- `go vet ./...`, formato Go y `git diff --check`: aprobados.
- Compilación React/TypeScript/Vite: aprobada.
- Prueba Chrome por IPv4 real del equipo: acceso local → LAN, login, bibliotecas, licencia y regreso a local,
  incluyendo el fallback UUID y ancho móvil: aprobada.
- Suite completa Chrome: 11 escenarios aprobados, incluyendo filtros, los seis acordeones, licencias,
  cargas, workflow, búsquedas y procesamiento existente.

- Compilación y empaquetado: Windows amd64, macOS arm64, Ubuntu amd64 y Ubuntu arm64 completados.
  Se verificaron arquitectura ELF de Ubuntu, dependencias PE de Windows, manifiestos y SHA-256.
- Binario de producción extraído del `.pkg` macOS: aprobado con datos desechables. Incluye PDF/OCR,
  bootstrap interactivo, login, acceso por IPv4, persistencia LAN tras reiniciar el proceso, retorno a local,
  bloqueo de estado, integridad SQLite, permisos privados y borrado interno que conserva el PDF original.
  Esta prueba no instala launchd ni cambia cuentas/permisos del sistema anfitrión.
- Tras la revisión visual final se repitieron navegación y red en Chrome: ambos escenarios aprobados.

## Verificación manual y límites

1. Probar desde **otro equipo físico** hacia Windows, Ubuntu ARM64/amd64 y macOS, con el firewall real
   y las políticas del entorno. La prueba automática por IP propia valida aplicación y enlace, no el trayecto externo.
2. Comprobar instalación/actualización, reinicio del servicio y del equipo, cambio local/LAN y desinstalación
   en los sistemas destino. Compilar para Windows/Ubuntu no equivale a ejecutar sus servicios nativos.
3. HTTP LAN no cifra credenciales, sesiones ni documentos: utilizarlo solo en una red de confianza.
   Las instalaciones HTTPS existentes conservan sus protecciones. No se añade publicación en Internet.
4. La selección de interfaces es heurística: los adaptadores virtuales/VPN conocidos se apartan de las URLs
   sugeridas, pero pueden consultarse en el diagnóstico. Una VM conserva su Ethernet principal.
5. El guardado por renombrado mantiene el mecanismo existente del proyecto; no se promete resistencia
   absoluta frente a fallos del sistema de archivos o pérdida física de energía.
6. Se mantiene el canal de instaladores de prueba y la versión base `0.7.0-test.1`, con revisión de binario
   `r7` en los cuatro objetivos, para no romper las restricciones de actualización existentes. Esta entrega
   no representa una publicación comercial firmada/notarizada de la versión 1.0.
