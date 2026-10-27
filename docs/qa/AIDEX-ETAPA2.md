# AIDEX 1.0.0 — Etapa 2: diagnóstico y verificación

## Rendimiento

Las mediciones de 10 000, 50 000 y 100 000 documentos usan la base temporal de `TestSyntheticScale`; no modifican la base instalada ni crean PDF. En macOS arm64, antes del cambio, Dashboard tardó 149 ms, 1,36 s y 3,25 s; las tarjetas de bibliotecas tardaron 66 ms, 508 ms y 1,10 s. Después del cambio, Dashboard tardó 158 ms, 1,44 s y 2,96 s; las tarjetas tardaron 32 ms, 304 ms y 626 ms. Una repetición de 100 000 registros dio 2,99 s y 626 ms, respectivamente. Son mediciones individuales, no garantías de latencia.

La consulta de tarjetas cargaba el agregado de todos los documentos antes de paginar bibliotecas y calculaba datos de OCR y política que no presenta. Ahora pagina primero y calcula solo los tres conteos visibles. Dashboard reutiliza una vista materializada para sus agregados. `AIDEX_DIAG_TIMINGS=1` registra los tiempos por etapa; las solicitudes de más de dos segundos se registran automáticamente. Para Dashboard se informa `pool_ms` y `summary_ms`; para tarjetas, `pool_ms`, `query_ms` y `cursor_ms`. No se escriben rutas ni nombres de documentos en esos eventos. La traza local con 100 000 registros asignó 2,90 s de 2,96 s a `summary_ms`, 63 ms a `daily_ms` y menos de 1 ms a la adquisición de conexión.

El test HTTP `TestDashboardHTTPTiming` ejercita autenticación, enrutamiento y serialización con 10 000 filas temporales distribuidas entre una biblioteca vinculada y otra administrada. En esta Mac registró 127 ms para `/system/dashboard`, 26 ms para `/system/dashboard/libraries` y menos de 1 ms para `/libraries?archived=false`. Debe repetirse en Windows, con la biblioteca vinculada real y con el servicio instalado, porque aquí no se reprodujo la espera de 12–15 segundos reportada allí.

## Eliminación, recuperación y concurrencia

La eliminación mantiene la modificación masiva de la base en una sola transacción para conservar la frontera atómica. WAL permite lectores durante esa escritura. La prueba de 100 000 filas eliminó 80 000 documentos en aproximadamente cinco segundos, con 50 muestras simultáneas de lectura y máximo inferior a 2 ms; también consultó Dashboard, Bibliotecas, Estadísticas, historial y estado de eliminación durante el trabajo. Los originales vinculados permanecieron intactos. Las operaciones `running` en estado `planned` vuelven a `pending` al iniciar el servicio; la limpieza posterior al commit dispone de diario durable y pruebas de recuperación. No se dividió la transacción sin evidencia de que mejoraría el caso Windows sin comprometer atomicidad.

## Carpetas y permisos

El backend abre la carpeta y lee una pequeña muestra desde la **cuenta del servicio**, no desde la sesión del navegador. La inspección conserva códigos distintos para permiso denegado, ruta inexistente, fallo de red y fallo de lectura. El log incluye sistema operativo, número de error y si era ruta de red; no incluye la ruta. El código `ROOT_PERMISSION_DENIED` en macOS con `EPERM` indica también revisar los controles de privacidad. Un error genérico de red sigue sin demostrar por sí solo si falló DNS, SMB o la autorización remota.

En Windows, el servicio instalado usa `NT SERVICE\AIDEX`. Para una carpeta local, conceder **lectura y recorrido** a esa identidad en la carpeta y sus padres necesarios. Para una ruta UNC, comprobar permisos tanto de recurso compartido como NTFS para la identidad presentada por el servicio al equipo remoto; una cuenta virtual puede presentarse como la cuenta de equipo `DOMINIO\EQUIPO$`. Comprobar el principal efectivo en el servidor de archivos antes de conceder acceso. Evitar unidades mapeadas de la sesión del usuario y permisos generales de Control total. [Referencia de Microsoft sobre cuentas de servicio](https://learn.microsoft.com/windows-server/identity/ad-ds/manage/understand-service-accounts).

En macOS, el LaunchDaemon usa `_aidex`. Comprobar ACL y recorrido de los padres. Para Documentos, Escritorio, Descargas y volúmenes de red, revisar **Ajustes del Sistema → Privacidad y seguridad → Archivos y carpetas / Acceso total al disco** para el ejecutable del servicio y reiniciar el servicio tras el cambio. Si la política del equipo no permite esa autorización, elegir una ubicación compartida administrada con permisos específicos para `_aidex`. [Referencia de Apple sobre acceso a archivos](https://support.apple.com/en-gb/guide/security/secddd1d86a6/web).

## Navegador

El campo «Nombre visible» es un control nativo sin transformación de cada pulsación. La prueba de entrada con acentos, apóstrofo, caracteres asiáticos y emoji pasó en Chrome y Brave en macOS. No se reprodujo el cierre de Brave; falta repetirlo en la versión y equipo Windows que lo mostró, con extensiones/autocompletado y registro de fallos del navegador disponibles. No se atribuye al frontend sin esa evidencia.

## Verificación pendiente en equipos de destino

Después de instalar, repetir los endpoints con el conjunto real en Windows y revisar los eventos `dashboard timing` / `dashboard libraries timing`. Probar acceso a una carpeta local y una UNC con la cuenta efectiva, reconciliación, consulta durante eliminación, reinicio a mitad de trabajo, actualización y desinstalación conservando originales. El paquete Windows/Ubuntu generado desde macOS solo admite inspección estructural y compilación cruzada; la instalación nativa requiere los equipos respectivos.

## Comprobaciones y paquetes de esta etapa

`go test ./...` pasó con acceso a loopback; el sandbox sin acceso a sockets produjo únicamente fallos de apertura en `internal/network`. Pasaron la suite de bibliotecas/API en modo desarrollo, la prueba HTTP de escala, `npm --prefix web run build`, tres casos de interfaz de catálogos/Dashboard/biblioteca y el caso de nombre visible en Chrome y Brave de macOS. Pasaron siete pruebas de plantillas/parser del instalador Windows y la inspección estructural de los cuatro paquetes. La inspección no equivale a ejecutar una instalación nativa.

Directorio: `dist/aidex-1.0.0-etapa2-20261026/`.

| Sistema | Paquete | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| macOS arm64 | `AIDEX-1.0.0-macOS-arm64.pkg` | 6 216 321 | `d5b1aab8963ab96f75346e1fee302b0ccdb2fb1cf314e4f06b9d58902f207da8` |
| Windows amd64 | `AIDEX-1.0.0-Windows-amd64.exe` | 40 445 248 | `8c057d82e1d27d3531d93db05e1d6e21ed708a4a22110fa5e948eb83374ec0d3` |
| Ubuntu amd64 | `aidex_1.0.0_amd64.deb` | 6 612 448 | `39c78f0b21efe9ee0b0cf0488f83395738a7a597ff30722bde60bda212d89567` |
| Ubuntu arm64 | `aidex_1.0.0_arm64.deb` | 6 106 860 | `da728299edbd552ae573806162b823ae8ac3e70c9b926f33355355e315783d8a` |
