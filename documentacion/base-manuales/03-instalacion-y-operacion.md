# AIBID — Instalación y operación

**Fecha de elaboración: 2026-10-09.** Actualización de la base elaborada el 8 de octubre; nombres y procedimientos contrastados con la transición de producción. Base de trabajo para los futuros manuales; no es un manual final.

## Clasificación de los procedimientos

Base para Manual de instalación. Instrucciones derivadas de scripts reales y evidencia previa. **No se ejecutó el instalador del sistema ni se modificaron servicios/datos reales. Se agregan pruebas aisladas y ejecución del binario macOS extraído sobre estado ficticio.**

- **Comprobado en pruebas previas:** binario macOS extraído, migraciones sobre estados sintéticos, PDF/OCR real de fixtures, validación de paquetes/PE/ELF/manifest y scripts PowerShell con mocks. No equivale a instalar en el SO destino.
- **Respaldado por código:** pasos del asistente/wrappers, cuentas, permisos, servicios y eliminación selectiva.
- **Pendiente manual:** ciclo nativo Windows/Ubuntu/macOS, LAN Windows efectivo, bundle offline y firma distribuible. Véase [VAL-001](07-pendientes-validacion.md#val-001) a [VAL-009](07-pendientes-validacion.md#val-009).

Evidencia actual: [informe de transición](../../docs/TRANSICION-PRODUCCION.md) y [VALIDACION.json](../../dist/aibid-1.0.0/VALIDACION.json). Las instrucciones siguientes describen instalaciones nuevas; las excepciones por actualización se separan en el apartado de migración. La evidencia previa de desarrollo permanece archivada y no certifica estos paquetes.

## Paquetes disponibles y soporte declarado

| Plataforma | Arquitectura / requisito | Formato | Alcance comprobado |
| --- | --- | --- | --- |
| Windows | Windows 10 o posterior64bits; amd64/x64 | NSIS .exe | Estructura/PE/iconos/scripts/mocks; nativo pendiente. |
| macOS | macOS 13+; AppleSilicon arm64 | Installer .pkg | Metadatos y binario extraído probado; Installer/launchd nativos pendientes. |
| Ubuntu | Ubuntu 24.04; amd64 o arm64 | Debian .deb, un paquete por arquitectura | Control/ELF/dependencias/scripts examinados; APT/systemd nativos pendientes. |

No hay objetivo distribuible MacIntel ni WindowsARM en este conjunto. Una VM UbuntuARM en Mac M1 utiliza paquete **arm64**, no amd64; no forzar arquitectura cruzada con dpkg.

Los cuatro paquetes nuevos corresponden a la primera edición comercial AIBID 1.0; su versión técnica es 1.0.0.

| Plataforma | Archivo real | Bytes | SHA-256 |
| --- | --- | --- | --- |
| Windows | [AIBID-1.0.0-Windows-amd64.exe](../../dist/aibid-1.0.0/AIBID-1.0.0-Windows-amd64.exe) | 40203869 | `52ba0851980dcbbba44b8d26f35094eec8ed04ef4a1eec22c1b07475980266c4` |
| macOS | [AIBID-1.0.0-macOS-arm64.pkg](../../dist/aibid-1.0.0/AIBID-1.0.0-macOS-arm64.pkg) | 5978466 | `fbb1ed795497cdfd2efa8596e4c4624ff297ebbfa278aa28849188ea86d069f4` |
| Ubuntu amd64 | [aibid_1.0.0_amd64.deb](../../dist/aibid-1.0.0/aibid_1.0.0_amd64.deb) | 6358014 | `32a3960cd2b633ecb9ac4d7e312b109466aee10311aabf1e5a360d856f6a0ae3` |
| Ubuntu arm64 | [aibid_1.0.0_arm64.deb](../../dist/aibid-1.0.0/aibid_1.0.0_arm64.deb) | 5875260 | `fa07b7b4e2325402455b3d6b97029c89f3e71a5cbbb8f30e93b7e4bae1000f0b` |

El manifiesto [SHA256SUMS](../../dist/aibid-1.0.0/SHA256SUMS) identifica los archivos finales. La inspección del paquete no equivale a instalarlo con su gestor nativo.

## Requisitos realmente definidos

| Recurso | Implementación / límite | Qué no debe afirmarse |
| --- | --- | --- |
| Backend/web | Binario Go con assets React incrustados. Navegador moderno con cookies y JavaScript. | No necesita Apache, PHP ni MySQL, ni Node/Go para ejecutar paquete instalado. |
| PDF/OCR | pdfinfo, pdftotext, pdftoppm, Tesseract; datos spa y eng para ambos idiomas. | OCR no procesa Word/Excel ni todo archivo de imagen. |
| Previews Office | LibreOffice opcional, ejecutable local detectado/configurado; fuentes instaladas. | Ausencia del convertidor no impide extracción DOCX/XLSX/búsqueda/descarga. |
| Windows | Herramientas PDF/OCR incluidas en tools\\Library\\bin; tessdata y configuración de fuentes empaquetados; PowerShell5.1. | Nativo real y políticas corporativas requieren prueba; no asumir elevación de backend. |
| Ubuntu | Depends:adduser, util-linux, systemd, poppler-utils, tesseract-ocr, tesseract-ocr-spa, tesseract-ocr-eng, fonts-dejavu-core. Recommends:libreoffice-writer, libreoffice-calc. | APT normal puede necesitar red para dependencias; archivo.deb solo no es bundle offline. |
| macOS | Herramientas externas típicamente /opt/homebrew/bin; idiomas disponibles para cuenta servicio; LibreOffice opcional. | Homebrew y dependencias no están contenidas en .pkg. |
| CPU/RAM | Se detectan núcleos/logical/availableCPUs/memoria; política automática conservadora. | No existe mínimo universal RAM/CPU validado para publicar; VAL-017. |
| Disco | DB/estado privados locales; uploads verifica espacio para máximo configurado+64 MiB; materialización compara espacio del destino. | No convertir tamaño instalador o máximo de archivo en espacio total mínimo. |
| SQLite | Motor modernc incrustado; WAL, FTS5; engine mínimo3.51.3. | No requiere instalar SQLite del SO; no poner DB en SMB/NAS. |
| Puertos | Instaladores18090/TCP; CLI init sin override8090/TCP. | No usar ambos indistintamente: leer config y diagnóstico. |
| Estado | Directorio absoluto privado y filesystem local; originales pueden ser rutas externas según permiso. | No almacenar estado bajo htdocs/wwwroot ni raíz del disco. |
| Privilegios | SO admin instala/crea cuenta/regla/servicio. Uso web y launcher normales no piden admin. | Permisos del usuario de escritorio y cuenta del servicio son distintos. |
| Licencia | Cliente proveedorHTTPS configurado por instalador; licencia comercial para acceso documental según política. | No hay activación automática con clave por defecto ni publicar claves reales. |

Fuentes: [scripts/build_installers.py](../../scripts/build_installers.py), [internal/config/config.go](../../internal/config/config.go), [internal/extraction/diagnostic.go](../../internal/extraction/diagnostic.go), [internal/libraries/uploads.go](../../internal/libraries/uploads.go), [internal/storage/database.go](../../internal/storage/database.go). Para hardware/espacio medido y fidelidad de fuentes: [VAL-017](07-pendientes-validacion.md#val-017), [VAL-018](07-pendientes-validacion.md#val-018).

## Ubicaciones, cuentas y servicios

| SO | Programa / cuenta | Configuración y estado | Servicio / logs |
| --- | --- | --- | --- |
| Windows | `%ProgramFiles%\\AIBID`; backend aibid-server.exe; launcher AIBID.exe; `NT SERVICE\\AIBID` | `%ProgramData%\\AIBID\\config.json`; estado `...\\state`; uploads por defecto `state\\uploads` | Nombre AIBID, display AIBID; EventLog Application fuente AIBID; salida del host y Errores UI. |
| Ubuntu | `/usr/lib/aibid/aibid-server`; cuenta/grupo aibid sin login | `/etc/aibid/config.json`; `/var/lib/aibid/state`; uploads debajo de estado | `aibid.service`; systemd/journald; docs `/usr/share/doc/aibid`. |
| macOS | `/Library/Application Support/AIBID/bin/aibid-server`; cuenta/grupo _aibid | `/Library/Application Support/AIBID/data/config.json`; `data/state` | `app.aibid`; plist en /Library/LaunchDaemons; `data/logs/service.log`; comandos en /Applications/AIBID. |

Estado contiene `documental.db`, posibles `documental.db-wal` y `documental.db-shm`, archivo privado de identidad y temporales/caché. Nunca compartir la carpeta completa como “diagnóstico”. Windows establece ACL privadas a SYSTEM/Administradores/SID de servicio; Unix usa directorios700 y archivos privados600. Originales vinculados y destinos administrados son independientes, no se deben ubicar dentro de estado privado.

Estos son los valores de una instalación nueva. Las actualizaciones mantienen rutas privadas y cuentas cuando es necesario; véase el apartado de migración. Identificadores incompatibles se rechazan con conservación; no forzar rename ni importar estado de otra instalación sin validar identidad/licencia.

## Windows: procedimiento de instalación y primer inicio

Relacionado: [FUN-110](01-inventario-funcional.md#fun-110), [FUN-111](01-inventario-funcional.md#fun-111), [FUN-112](01-inventario-funcional.md#fun-112).

1. Confirmar SO x64 compatible, paquete correcto y permisos administrativos. Revisar que otro programa no use18090; no borrar estado existente para “liberar” puerto.
2. Ejecutar el .exe; autorizar UAC; leer instrucciones del asistente.
3. **Instalar**. Preflight valida propietario, rutas sin reparse points, marcador compatible, detiene servicio hasta60 s y usa comprobador nuevo para exclusividad; ante rechazo conservar resultado de detalle.
4. Configure crea/valida servicio, cuenta virtual/SID y ACL. Inicializa config solo si falta; provisiona confianza pública de licencia y migra DB. Publica solo public_url en launcher.json.
5. Instalador prepara regla LAN limitada a servicio/ejecutable/puerto/subred y Domain/Private. No da administrador a la cuenta backend. Un fallo de configuración no se oculta: asistente puede anular instalación.
6. Abrir **Inicio → AIBID → Configurar AIBID**. La consola se eleva y ejecuta doctor sobre muestras empaquetadas. En primera vez pide usuario, nombre y contraseña confirmada (6–128; sin complejidad obligatoria).
7. Tras bootstrap habilita inicio automático demorado y arranca servicio. Se abre navegador en public_url. Por defecto **http://127.0.0.1:18090**.
8. Iniciar sesión; crear usuarios/perfiles según necesidad; activar licencia en pantalla Licencia. Bibliotecas y destino no se crean implícitamente por instalar.
9. Uso posterior mediante acceso **AIBID** del escritorio o **Abrir AIBID** de Inicio. AIBID.exe comprueba servicio/URL y abre navegador sin consola elevada.

No hay contraseña predeterminada. “Configurar AIBID” no es configuración web de red; prepara bootstrap/servicio. Si doctor falla, revisar herramientas/rutas/idiomas y mensaje sin desactivar protecciones indiscriminadamente. Ciclo nativo: [VAL-001](07-pendientes-validacion.md#val-001).

### Servicio, estado y logs en Windows

Comandos de referencia **no ejecutados aquí**, desde PowerShell administrativo cuando la operación lo requiera:

```powershell
Get-Service -Name AIBID
Start-Service -Name AIBID
Stop-Service -Name AIBID
Restart-Service -Name AIBID
Get-CimInstance Win32_Service -Filter "Name='AIBID'" | Select-Object Name, StartName, State, PathName
Get-WinEvent -FilterHashtable @{LogName='Application'; ProviderName='AIBID'} -MaxEvents 20
Get-NetTCPConnection -LocalPort 18090 -State Listen
Get-NetFirewallRule -Name AIBID-LAN-TCP
```

Leer `StartName` para identificar la cuenta efectiva. EventLog captura errores de terminación; no es un dashboard de todos los eventos. HTTP/worker logs usan sanitización, pero la vía err.Error() de arranque requiere validar ausencia de información sensible: [VAL-016](07-pendientes-validacion.md#val-016). No exportar datos privados junto al reporte.

## Ubuntu: instalación y primer inicio

Relacionado: [FUN-115](01-inventario-funcional.md#fun-115), [FUN-116](01-inventario-funcional.md#fun-116); fuente [packaging/ubuntu/postinst](../../packaging/ubuntu/postinst).

1. En terminal, comprobar `dpkg --print-architecture`; elegir arm64 o amd64 **igual al SO**.
2. Instalar con APT el paquete de esta arquitectura. El nombre en el ejemplo es marcador que se sustituye por archivo real:

```bash
dpkg --print-architecture
sudo apt install ./paquete-aibid-arquitectura-correcta.deb
sudo aibid-admin
```

3. Postinst crea cuenta aibid, prepara directorios privados, inicializa config si falta, configure-license y migrate. Nueva instalación espera bootstrap; no habilita servicio completo antes del marcador.
4. `aibid-admin` ejecuta diagnóstico como cuenta de servicio y crea administrador interactivo si aún no existe. Luego escribe marcador y `systemctl enable --now aibid.service`.
5. Desde navegador del equipo acceder a **http://127.0.0.1:18090**; desde fuera usar LAN después de habilitarlo y validar firewall.
6. Activar licencia; configurar biblioteca/rutas/módulos. LibreOffice Writer/Calc son paquetes recomendados para previews; no obligatorios para extracción nativa Office.

Los wrappers vigentes son `aibid-admin` y `aibid-uninstall`, sin alias de pruebas en una instalación nueva. No instalar Debian arbitrario como si estuviera certificado; objetivo de scripts offline y empaquetado Ubuntu 24.04.

### Servicio y logs en Ubuntu

```bash
systemctl status aibid.service --no-pager
sudo systemctl start aibid.service
sudo systemctl stop aibid.service
sudo systemctl restart aibid.service
sudo journalctl -u aibid.service -n 100 --no-pager
ss -ltn '( sport = :18090 )'
```

Unidad define User/Group, NoNewPrivileges, PrivateTmp, ProtectSystem=full, ReadWritePaths de estado/config, UMask0077, familias AF_UNIX/AF_INET/AF_INET6, reinicio on-failure y parada60 s. Estas medidas no confieren permisos sobre NAS. ACL/montaje de originales debe admitir cuenta de servicio. No copiar DB/config bajo Apache; AIBID sirve su propia interfaz.

### Bundle de dependencias sin red

Relacionado [FUN-117](01-inventario-funcional.md#fun-117); **implementado, no certificado aquí** ([VAL-004](07-pendientes-validacion.md#val-004)). En Ubuntu 24.04 con arquitectura destino y red:

```bash
sudo ./prepare-offline.sh ./paquete-aibid-arquitectura-correcta.deb "/ruta/nueva/Documentación José/bundle"
```

El script requiere carpeta nueva; descarga cierre de dependencias desde estado dpkg vacío, con recommends, genera Packages y SHA256SUMS, copia install.sh. Conservar checksum por canal independiente. Transferir carpeta completa a equipo Ubuntu de **misma arquitectura**, desconectado; ejecutar desde ella:

```bash
sudo ./install.sh
sudo aibid-admin
```

install.sh valida hashes/arquitectura y solo usa un repositorio file: temporal para la invocación; no modifica permanentemente fuentes APT. Los scripts están disponibles en [packaging/ubuntu/prepare-offline.sh](../../packaging/ubuntu/prepare-offline.sh) y [packaging/ubuntu/install-offline.sh](../../packaging/ubuntu/install-offline.sh); no confundirlos con un bundle ya construido. Tamaño/tiempo mínimos no medidos.

## macOS: instalación y primer inicio

Relacionado [FUN-119](01-inventario-funcional.md#fun-119), [FUN-120](01-inventario-funcional.md#fun-120); fuentes [packaging/macos/postinstall](../../packaging/macos/postinstall) y [scripts/build_installers.py](../../scripts/build_installers.py).

1. Confirmar macOS 13+AppleSilicon y paquete arm64. Intel no forma parte del alcance actual.
2. Preparar herramientas externas accesibles a cuenta servicio. La instrucción que el proyecto conserva en [docs/H7.md](../../docs/H7.md) es `brew install poppler tesseract tesseract-lang` como usuario normal. Es un procedimiento externo previo, **no se ejecutó ni se instaló Homebrew aquí**; distribución/dependencias actuales y permisos requieren validación nativa.
3. Para previews Word/Excel instalar LibreOffice externo autorizado. Su ruta automática habitual se verifica desde Ajustes → Vistas previas y caché. No asumir que el .pkg incluye convertidor.
4. Abrir .pkg y completar Installer con permisos administrativos. Preinstall valida/para instalación anterior compatible; postinstall crea cuenta _aibid y directorios privados, configura licencia/migración, marcador y accesos.
5. Abrir **/Applications/AIBID/Configurar AIBID.command**; autorizar administración; leer doctor; crear administrador interactivo cuando falta; habilitar launchd.
6. Acceder a URL local18090 mediante **Abrir AIBID.webloc** o URL del navegador; activar licencia y registrar bibliotecas.

El paquete emplea firma ad hoc para binario; eso no es firma DeveloperID/notarización de distribución. No recomendar saltarse Gatekeeper como paso rutinario; resolver [VAL-006](07-pendientes-validacion.md#val-006) antes de manual distribuible.

### Servicio y logs en macOS

Comandos operativos basados en etiqueta/plist existentes, **no ejecutados durante esta tarea**:

```bash
sudo launchctl print system/app.aibid
sudo launchctl bootout system/app.aibid
sudo launchctl bootstrap system /Library/LaunchDaemons/app.aibid.plist
sudo launchctl kickstart -k system/app.aibid
sudo tail -n 100 "/Library/Application Support/AIBID/data/logs/service.log"
lsof -nP -iTCP:18090 -sTCP:LISTEN
```

`kickstart -k` interrumpe proceso: utilizar únicamente como mantenimiento autorizado, no para producir una captura. Para primera habilitación preferir wrapper que verifica bootstrap/marker. Plist tiene cuenta dedicada, KeepAlive on-failure, ThrottleInterval10, ExitTimeOut60 y umask077. El archivo plist se **genera desde scripts/build_installers.py**; no existe como fuente independiente packaging/macos/app.aibid.plist en este repositorio.

## Acceso local, LAN, TLS y firewall

Relacionado [FUN-096](01-inventario-funcional.md#fun-096), [FUN-097](01-inventario-funcional.md#fun-097).

1. Ingresar como cuenta con **system.configure**. Abrir **Ajustes → Acceso y red**.
2. Leer **Estado del servicio**, puerto y modo. **Solo este equipo** escucha127.0.0.1; **Permitir acceso desde la red local** escucha0.0.0.0 sobre mismo puerto, IPv4.
3. Elegir modo y pulsar **Guardar acceso**. Backend precomprueba listeners, intenta bind real y persiste; si puerto/permisos/persistencia fallan revierte al estado anterior.
4. Pulsar **Ejecutar diagnóstico**; revisar URLs sugeridas e **Interfaces IPv4 disponibles**. Adaptadores virtuales/VPN pueden no ser destino útil.
5. Desde otro equipo autorizado acceder **http://IP-LAN-DEL-SERVIDOR:18090** (sustituir IP/puerto reales). No usar127.0.0.1 del equipo cliente.
6. Al volver a local un usuario remoto pierde acceso; guardar desde cliente local es preferible para poder comprobar resultado.

| SO | Qué hace AIBID | Qué debe validar soporte |
| --- | --- | --- |
| Windows | Instalador elevado configura AIBID-LAN-TCP, solo ejecutable backend/Service AIBID/TCPpuerto/LocalSubnet/Domain+Private/EdgeTraversalBlock. La regla puede permanecer habilitada con bind local sin abrir escucha LAN. | Estado efectivo, perfil red, GPO; una red Public o firewall externo puede bloquear. Backend virtual no debe elevarse para cambiar regla. |
| Ubuntu | Listener local/LAN y diagnóstico readonly de UFW con timeout; no prepara regla automáticamente. | Firewall/nftables/UFW externo cuando activo. Procedimiento autorizado específico del entorno; no inventar regla automática. |
| macOS | Listener local/LAN y diagnóstico readonly de firewall de aplicaciones. | Permiso del binario/servicio y política de firewall; acceso entre VM y host. |
| HTTPS/proxy personalizado | Config valida TLS directo o proxies CIDR explícitos. UI muestra modo externo y no lo sobrescribe con toggle local/LAN. | Certificados/configproxy/origen público y headers solo de proxies confiados. Procedimiento final VAL-009. |

Host/Origin y CSRF siguen activos en LAN; permitir bind no elimina permisos ni sesiones. HTTP LAN no cifra credenciales/documentos: la interfaz lo advierte. Esta base no incluye despliegue público de internet ni receta universal de proxy.

### Permisos de archivos y carpetas de red

La inspección y los workers usan **cuenta del backend**, no credencial web ni usuario del Explorador. Windows virtual account accede a recursos remotos de dominio como identidad del equipo en condiciones normales de dominio; confirmarlo con StartName/servidor SMB. Una concesión a usuario interactivo puede no servir para `NT SERVICE\\AIBID` local o cuenta de equipo remota.

Para biblioteca vinculada conceder lectura/listado en filesystem y, en SMB, en **Compartir** y **Seguridad** a identidad efectiva; preferir UNC completa `\\servidor\\recurso\\carpeta`. Para destino administrado se necesita crear/escribir/leer/verificar. No conceder Control Total a raíz de volumen por defecto ni abrir System Volume Information/$RECYCLE.BIN: AIBID las excluye siempre. En Linux/macOS montajes/ACL deben admitir cuenta del servicio, y no depender de una sesión de escritorio. Este párrafo describe la diferencia de identidades, no verifica un dominio concreto ([VAL-012](07-pendientes-validacion.md#val-012)).

## Configuración inicial de producto

Después del bootstrap y activación:

1. Crear usuarios (FUN-007) y perfiles globales mínimos (FUN-009); asignar perfiles de biblioteca por separado (FUN-016).
2. Crear Vinculada para consultar originales o Administrada/Híbrida para cargas (FUN-014/FUN-015).
3. Registrar raíces/destino (FUN-022 a FUN-024). En administrada elegir destino para futuras incorporaciones en Configuración (FUN-019).
4. Elegir expedientes/catálogos/plantillas y revisión si se necesitan. No imponer expedientes a biblioteca que no los usa.
5. Revisar **Archivos y procesamiento**: almacenar e indexar son diferentes; indexación global inicial soloPDF. Habilitar DOCX/XLSX/TXT/CSV para indexar donde se desee (FUN-084/FUN-085).
6. Revisar Automatización/OCR, recursos y previews (FUN-087 a FUN-095). Verificar idiomas y convertidor detectados; ajustar a hardware real.
7. Habilitar LAN únicamente si se requiere y comprobar desde otro equipo; revisar Dashboard/Procesamiento/Errores según permisos.

Activación online/offline, importación, desactivación y recuperación se detallan en FUN-103 a FUN-109 y FUN-124. No hay claves por defecto. Desinstalar/borrar DB no libera automáticamente activación del proveedor.

## Comandos locales implementados

`aibid-server` (en Windows .exe) acepta `subcomando --config RUTA`; no se intercambia con launcher AIBID.exe. La ruta de configuración debe ser la de plataforma anterior. Esta tabla registra efectos reales para evitar ejecutar una “consulta” que abre/migra DB.

| Subcomando | Función / campos | Efectos y condiciones |
| --- | --- | --- |
| version | Información de build | No requiere config/estado; el nombre técnico/binario y versión pueden aparecer en cuerpo de manual, no encabezados documentales. |
| init | --state,--listen,--tools,--tessdata,--fontconfig | Solo instalación nueva; O_EXCL no sobrescribe; configura listenlocal; crea archivo privado. |
| configure-license | Provisión de proveedor y claves públicas | Puede crear backup config.before-license-... y cambiar confianza; no consulta readonly; instaladores la llaman. |
| doctor | --sample-directory opcional | Valida config/tools/idiomas; muestras PDF/OCR pueden usar temporales; no se ejecutó aquí. |
| check-state | Carga segura y lock | No abre SQLite ni migra; adquiere lock exclusivo. No usar mientras sirve salvo para comprobar rechazo. |
| bootstrap-ready | Verifica cuenta inicial | Comando interno usado por wrappers, abre DB/migra; no debe confundirse con consulta de archivo. |
| bootstrap | Credenciales interactivas | Crea administrador inicial solo una vez; no entrada arbitraria contraseña por argumento/pipeline. |
| recover-admin | Usuario y nueva contraseña interactiva | Cuenta administrativa existente; cierra sesiones y audita. |
| recover-license | --confirm-license-recovery,--reason | Nueva identidad/activación requerida; preserva evidencia; proveedor debe resolver transferencia. |
| migrate | Migraciones aditivas | Open aplica migraciones y backup condicionado; conserva registros; requiere lock. |
| rollback-empty | Reversión estricta | Solo estado nunca usado; rechaza evidencia existente. No es downgrade de datos reales. |
| erase-internal-data | --confirm-erase-internal-data | Destructivo para DB/índices/extracción/usuarios/config/identidad; preserva físicos privados/externos desconocidos. |
| serve | Servidor y workers | Requiere bootstrap, assets y estado exclusivo; aplica migraciones; inicia licencia/runtime/listener. No consulta readonly. |

Fuente: [cmd/gestor-documental/main.go](../../cmd/gestor-documental/main.go); ver [FUN-122](01-inventario-funcional.md#fun-122) a [FUN-127](01-inventario-funcional.md#fun-127). No ejecutar rollback/erase/recovery como parte de redactar manuales.

## Respaldo y restauración: alcance real

**No se encontró una herramienta integral implementada por UI, endpoint o CLI para respaldo/restauración**. Los permisos reservados no bastan. No redactar “Pulse Respaldar/Restaurar” ni un comando inexistente.

Sí se implementa [internal/storage/upgrade_backup.go](../../internal/storage/upgrade_backup.go): con schema previo poblado y antes de aplicar migración0014, crea `state/upgrade-backups/before-processing-FECHA.db` mediante **VACUUM INTO**, incluyendo WAL confirmado y protección600. No corre en instalación nueva ni cada arranque ya actualizado. **Es una copia de DB, no de originales, config, identidad privada ni uploads**. No existe restore simétrico certificado.

Para elaborar runbook de respaldo completo se debe resolver [VAL-007](07-pendientes-validacion.md#val-007): inventariar estado/config/identidad privada, originales administrados y vinculados, uploads y referencias; obtener copia coherente con servicio detenido o mecanismo aprobado; ensayar restauración en clon y verificar licencia/binding/hash/permisos. Aquí se registra la necesidad, **no se declara un procedimiento seguro de restauración ya comprobado**. Copiar solamente documental.db con servicio activo puede omitir WAL y no es instrucción válida.

## Actualización: procedimiento respaldado por código

1. Identificar arquitectura y marcador existente, y revisar compatibilidad admitida por instalador. No renombrar manualmente cuentas/rutas privadas; el instalador selecciona el perfil compatible.
2. Contar con respaldo integral/retorno validado antes de actuar sobre información real ([VAL-007](07-pendientes-validacion.md#val-007)). No desinstalar ni borrar DB para actualizar.
3. Ejecutar paquete compatible sobre instalación existente. Preflight/preinstall/preinst paran servicio y validan propietario, rutas sin enlaces y exclusividad.
4. Nueva configuración no sobrescribe config existente; configure-license provisiona confianza compatible; DBOpen aplica cadena incremental y verifica checksums.
5. Antes de actualizar marcador puede crear snapshot interno condicionado. Un checksum cambiado o esquema posterior desconocido detiene proceso; no editar schema_migrations a mano.
6. Si instalación ya tenía service-enabled, postinstall puede reanudar servicio. En primera configuración ejecutar wrapper administrativo.
7. Revisar login, roles/bibliotecas, originales, texto/OCR, búsqueda, estado de cola, licencia y LAN. No dar por aprobado por aparecer “instalación completada”.

Migraciones y fixtures de upgrade tienen pruebas previas aprobadas; el ciclo nativo por SO aún necesita evidencia de [VAL-001](07-pendientes-validacion.md#val-001), [VAL-003](07-pendientes-validacion.md#val-003), [VAL-005](07-pendientes-validacion.md#val-005).

## Desinstalación y reinicio de datos

| SO / procedimiento | Conservar (predeterminado) | Eliminar datos internos (opcional) | Protecciones |
| --- | --- | --- | --- |
| Windows: Desinstalar AIBID / Desinstalar.exe | Checkbox Eliminar los datos de AIBID al desinstalar desmarcado; datos/config quedan en ProgramData. | Checkbox y segunda confirmación; manage Remove invoca erase-internal-data; nueva instalación requiere bootstrap/activación. | Parada60 s/check-state; rule/SCM retiro; manifiesto SHA protege archivos modificados/desconocidos. |
| Ubuntu: sudo aibid-uninstall | Responder no a pregunta; apt-get remove; datos/config/cuenta conservados incluso purge directo. | Responder sí y escribir ELIMINAR; wrapper borra internos y marcadores de paquete identificados. | Servicio parado; check-state; erase protegida; no recorre originales para borrar. |
| macOS: Desinstalar AIBID.command | Elevar y conservar por defecto; bootout/manifest; datos físicos preservados. | Confirmar ELIMINAR para erase; cuenta/directorios con físicos pueden seguir. | Validación de todas rutas/manifiesto antes de borrar; solo packaged hashes intactos; receipt eliminado. |

**Siempre** quedan intactos documentos físicos externos y administrados; también cargas privadas .document preservadas aunque borrar DB pierda referencia. No prometer limpieza total de carpeta del usuario ni eliminación remota de licencia. Solo metadata/índices/texto/usuarios/historial/config/identidad reconocidos se eliminan; archivos desconocidos se preservan.

Antes de eliminar interno, si se desea reutilizar licencia en nueva instalación, desactivarla y esperar confirmación de proveedor. Borrado local no garantiza plaza libre. Las opciones destructivas solo se prueban con estado desechable; nunca sobre datos reales. Fuentes: [internal/maintenance/erase.go](../../internal/maintenance/erase.go), [packaging/windows/manage.ps1](../../packaging/windows/manage.ps1), [packaging/ubuntu/aibid-uninstall](../../packaging/ubuntu/aibid-uninstall), [packaging/macos/uninstall.command](../../packaging/macos/uninstall.command).

## Diagnóstico operativo sin improvisación

| Síntoma | Lectura recomendada / causa a distinguir | Acción segura de soporte |
| --- | --- | --- |
| Solo responde en localhost | Diagnóstico modo/listener; red/filtro firewall | Consultar estado y UIAcceso; no alterar DB ni elevar backend. |
| Ruta correcta no inspeccionable | Cuenta del servicio, permisos locales/SMB, enlaces/solapamientos | Ver identidad efectiva y lectura como servicio en ambiente autorizado; no asumir usuario web. |
| No hay texto Office | Política indexar vs almacenar; unidad vacía vs documento entero; Último procesamiento | Revisar reglas/contexto/error; no suponer que requiere LibreOffice para extracción. |
| No preview Office | Estado/convertidor/fuentes/tamaño/cache | Revisar AjustesPreviews; extracción/búsqueda siguen independientes. |
| Trabajo parece bloqueado | Tarea, intentos, próximo retry, pause/módulo, SCAN_PARTIAL, PROCESS_RESTARTED | Observar progreso/diagnóstico; no lanzar retries concurrentes sobre misma raíz. |
| Disco insuficiente | UPLOAD_SPACE / STORAGE_SPACE / límites caché; espacio estado y destino separados | Medir espacio y liberar solo mediante procedimiento autorizado, no borrar estado/originales. |
| Licencia solo lectura | Revocada/vencida vs invalid/unactivated, módulos y reloj | Leer status/artefactos y proveedor; no resetear identidad para evitar licencia. |
| Fallo de actualización | Marker/checksum/lock/ownership/archivo modificado | Conservar evidencia y estado; no forzar chmod/rename o edición del esquema. |

Utilizar reporte sanitizado de **Errores y diagnósticos → Copiar información de diagnóstico**. Nunca enviar contraseñas/config completo/identidad privada/archivos de clientes. Evidencia suficiente: fecha, SO/arquitectura, operación, código, request/job/root/document ID autorizado, último progreso y condiciones reproducibles. Las capturas necesarias están en [05](05-catalogo-capturas.md).

<a id="migracion-instalaciones-anteriores"></a>

## Migración de instalaciones anteriores

Exclusivamente para soporte de instalaciones de desarrollo existentes; no trasladar estos nombres al procedimiento de instalación nueva.

| Plataforma | Identificadores conservados | Transición y precauciones |
| --- | --- | --- |
| Windows | Servicio/SID `AIBIDTest`, cuenta `NT SERVICE\AIBIDTest`, carpeta de programa anterior y `%ProgramData%\AIBID-Test` cuando ya existen. | Nuevo backend aibid-server.exe y launcher AIBID.exe; comando del servicio incluye --windows-service AIBIDTest. Registro de desinstalación pasa a AIBID tras configurar correctamente. Los comandos de SCM/EventLog de este documento usan AIBID para instalaciones nuevas; en este perfil consultar AIBIDTest. La regla AIBID-LAN-TCP se actualiza para el programa/servicio reales y solo si pertenece a AIBID. |
| Ubuntu | Cuenta/grupo `aibid-test`, `/etc/aibid-test/config.json` y `/var/lib/aibid-test`. | Nuevo programa /usr/lib/aibid/aibid-server y unidad aibid.service. Drop-in /etc/systemd/system/aibid.service.d/installation.conf conserva la cuenta/paths anteriores. El campo Replaces permite tomar los dos wrappers públicos que antes pertenecían al paquete anterior, sin Conflicts ni eliminación automática de ese paquete; su unidad aibid-test.service se deshabilita/enmascara tras verificar que la nueva esté activa. No usar sus wrappers antiguos. |
| macOS | Cuenta `_aibidtest` y `/Library/Application Support/AIBID-Test/data`. | Nuevo programa bajo /Library/Application Support/AIBID y etiqueta app.aibid. El plist nuevo se adapta a la cuenta/paths anteriores; el plist y recibo app.aibid.test se conservan para recuperación y su arranque se deshabilita tras verificar el servicio nuevo. |

Los marcadores admiten las entregas históricas 0.7.0-test.1, 2.0.0-alpha.1 a alpha.8 y 2.0.0-rc.1, además de 1.0.0. No son versiones comerciales anteriores. Dos instalaciones con datos, servicio con ruta/cuenta ajena, enlaces o marcador desconocido se rechazan; no fusionar ni forzar marcadores. Configuración, SQLite, identidad privada de licencia y archivos originales no se mueven por cambio de marca. Las unidades y programas históricos retenidos permanecen como compatibilidad/recuperación, no son otro servicio que deba arrancarse simultáneamente. Tras un borrado interno confirmado, Ubuntu retira únicamente el drop-in generado por el paquete y no modificado, evitando que una reinstalación nueva apunte a la configuración eliminada. Al comenzar desde cero, revisar permisos de las bibliotecas para la cuenta nueva antes de vincularlas. Validación real de actualización: VAL-001/003/005.

## Caracteres y diagnóstico del instalador

Se admiten rutas con espacios, acentos, ñ y Unicode compatibles con el SO; no cambiar el nombre del usuario ni la configuración regional. Windows muestra la ruta de un registro Unicode temporal para conservar el detalle de fallos; no es necesario conocer páginas de códigos para instalar. Las acciones no interactivas del asistente se acotan a diez minutos; SCM a sesenta segundos. Revisar el estado y el registro antes de reintentar una acción que agotó su plazo.

La corrección de transmisión de mensajes y quoting tiene pruebas automatizadas; Windows PowerShell5.1/NSIS visual, APT/systemd y Installer/launchd con nombres de usuario José, María López y Muñoz requieren las pruebas nativas indicadas en 07. El binario macOS extraído puede probarse con esos nombres en /private/tmp sin certificar el asistente del sistema.
