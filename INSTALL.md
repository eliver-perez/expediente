# AIBID — Desarrollo, distribución y operación

**H6 ejecutable:** cliente de licencias online/offline y bibliotecas vinculadas/administradas/híbridas, cargas privadas,
expedientes, revisión, guardado definitivo, integridad, OCR/búsqueda e identidad. Los instaladores
y el registro como servicio del SO corresponden a H7. No se configura Apache/XAMPP.

## Ejecutar en macOS / Visual Studio Code

Se necesitan Go (descarga automática del toolchain fijado 1.27.1) y Node 22.12+.
Desde la terminal de esta carpeta:

```sh
make deps
make build-dev
./build/gestor-documental-dev init
./build/gestor-documental-dev bootstrap
./build/gestor-documental-dev serve
```

Para actualizar H2–H5, detener su proceso y ejecutar `make build-dev` y `serve`.
No repetir `init`/`bootstrap`; los datos se conservan. El sufijo `-dev` identifica
el bypass de desarrollo sin JWS; H6 aplica la licencia después de activarla. No distribuir ese build.
[Configuración de extracción y herramientas](docs/H3.md).

Visitar **http://127.0.0.1:8090**. Ctrl+C detiene el servicio. `bootstrap` exige una
terminal local y pide usuario, nombre y contraseña de 6–128 caracteres sin
mostrarla. No hay credenciales predeterminadas ni contraseñas por argumentos/pipes.
`init` no sobrescribe configuración existente. El binario ya compilado no necesita
Node/npm ni conexión a Internet para operar. H3 requiere las herramientas PDF/OCR
e idiomas instalados en el equipo servidor.

En macOS: `~/Library/Application Support/gestor-documental/config.json`; la DB vive
en su subcarpeta `state/`. Linux usa `os.UserConfigDir()` (normalmente `~/.config/`)
y Windows el directorio de configuración del usuario. Son rutas de desarrollo,
distintas de las rutas de servicio propuestas para H7. Todos los comandos admiten:

```sh
./build/gestor-documental-dev init --config /ruta/local/privada/config.json
./build/gestor-documental-dev bootstrap --config /ruta/local/privada/config.json
./build/gestor-documental-dev serve --config /ruta/local/privada/config.json
```

Elegir ruta fuera de `htdocs`, `wwwroot` y cualquier raíz pública del cliente.
No usar enlaces simbólicos en estado; para pruebas desechables macOS usar
`/private/tmp`. Se verifica filesystem local, permisos privados y lock exclusivo.
El repositorio fuente tampoco debe exponerse como raíz de un virtual host Apache.

En Windows sin Make: `go mod download`, `npm --prefix web ci`,
`npm --prefix web run build`, `go build -tags development -o build/gestor-documental-dev.exe ./cmd/gestor-documental`;
después ejecutar `init`, `bootstrap`, `serve` con ese `.exe` de desarrollo.

## Pruebas

```sh
make check
make test-e2e
```

El primer comando incluye compilación/tipado React, formato Go, vet, pruebas con
race detector y comprobaciones SQL H1 (Python 3.9+ con FTS5, sin paquetes pip).
El segundo requiere Chrome y puerto loopback 8099 disponible: utiliza cuentas
ficticias, DB temporal y dos contextos independientes de navegador. No toca el
estado de trabajo. Si falta Chrome, instalarlo desde `web/` con
`npx playwright install chrome`. Alcance y evidencia en [H4](docs/H4.md).

Dependencias fijadas en `go.mod`/`go.sum` y `web/package-lock.json`. Node usado
localmente: 22.15.0. SQLite del sistema/Python: 3.51.0, utilizado solo para checks
H1; el producto usa el motor embebido y exige corrección WAL (3.51.3 o posterior).
[Referencia SQLite](https://www.sqlite.org/wal.html). Las herramientas PDF/OCR de H3 están instaladas localmente.
Su configuración y presupuestos se describen en [H3](docs/H3.md).

## Recuperación administrativa y migraciones

Con servicio detenido, `./build/gestor-documental recover-admin` solicita un
administrador existente y nueva contraseña, rehabilita esa cuenta, invalida sus
sesiones y deja auditoría. No crea una cuenta oculta. Usar el mismo `--config` de
la instalación. El lock impide recuperación/bootstrap/migración con servicio activo.

`migrate` verifica/aplica migraciones embebidas. `rollback-empty` revierte H4, H3 y H2
en una sola transacción únicamente si no hay bibliotecas, usuarios, sesiones,
intentos, contadores ni auditoría. Si ya existen datos se niega sin alterar el
esquema. No hay reversión destructiva automática.

## HTTPS en LAN

Editar config con el servicio detenido. Para TLS directo: `listen_address` igual
a `0.0.0.0:8443`, `public_url` igual a `https://documental.cliente.local:8443`, y
`tls_certificate`/`tls_private_key` con rutas absolutas al certificado y clave.
Usar CA confiable para los navegadores LAN y firewall limitado a la red elegida.
No se generan certificados ni se expone Internet público automáticamente.

Con proxy HTTPS: mantener Go en `127.0.0.1:8090`, configurar URL pública HTTPS y
`trusted_proxies: ["127.0.0.1/32"]` (añadir `::1/128` solo si se utiliza). El proxy
conserva Host público, **sobrescribe** `X-Forwarded-Proto: https` y anexa la IP
observada a `X-Forwarded-For`. Go solo confía en CIDRs explícitos y recorre la cadena
desde el salto inmediato. El encabezado `Forwarded` se ignora en H2.

Cookie HTTPS: `__Host-documental-session`; HTTP loopback:
`documental-local-session`. La URL del navegador debe coincidir con `public_url`,
incluyendo puerto. Usar localhost en lugar de 127.0.0.1 requiere cambiar esa config.

Para recarga React durante desarrollo: ejecutar Go en puerto predeterminado y
`npm --prefix web run dev`; abrir `http://127.0.0.1:5173`. Su proxy solo adapta el
Origin de ese origen local explícito. La distribución integra assets compilados.

## Configuración prevista

Configuración del servicio: interface/puerto, URL pública, TLS/proxy confiable,
estado local, temporal local, ejecutables PDF/OCR y presupuestos. Sin claves
privadas embebidas en config general. Configuración por biblioteca y sus revisiones
vive en DB: raíces linked/managed, nomenclatura, campos, políticas y vigilancia.

Validar espacio y permisos al configurar y antes de operaciones grandes. SQLite
rechaza montajes de red como ubicación. Documentos finales pueden usar discos/NAS
verificados. Cambiar estado/temporal requiere mantenimiento y plan explícito;
cambiar destino futuro de biblioteca no mueve sus archivos históricos.

## Matriz de releases a implementar en H7

| Plataforma | Artefacto | Servicio | Estado y configuración propuestos |
| --- | --- | --- | --- |
| Windows amd64 | `.exe` instalador + desinstalador | Servicio Windows sin consola persistente | Binarios en Program Files; config/DB/clave en ProgramData con ACL específica |
| Ubuntu 24.04 LTS amd64 | `.deb` online y bundle offline | systemd, usuario restringido | `/etc/gestor-documental/`, `/var/lib/gestor-documental/`, journald |
| macOS Intel amd64 | `.pkg` | launchd | `/Library/Application Support/GestorDocumental/` protegido; LaunchDaemon propio |
| macOS Apple Silicon arm64 | `.pkg` | launchd | Mismos criterios de permisos y actualización |

Las ubicaciones internas por defecto no fijan dónde van todos los PDF. Seleccionar
y conceder ACL a cada raíz; no dar al servicio acceso total al disco por conveniencia.
Redistribuir dependencias OCR/PDF e idiomas conforme a sus licencias; inventario/SBOM,
notices y fuentes/ofertas de componentes de terceros cuando su licencia lo requiera.
Esto se revisa antes de elegir herramientas: el código propio se entrega compilado.

Los cuatro objetivos necesitan builds y pruebas nativas con dependencias reales;
cross-compilar Go no prueba instalador, permisos, servicio ni OCR. CI en H2 agrega
format/lint/tests/migraciones de ese módulo; en H3 añade FTS/rutas; H6 contrato y
aislamiento de claves; H7 matriz de empaquetado, smoke tests, checksums/firma y
proceso de publicación. No descargar dependencias sin fijar versión/integridad.

## Ubuntu y operación offline

Cuando exista el artefacto H7, instalación online prevista:

```sh
sudo apt install ./gestor-documental_VERSION_amd64.deb
```

El bundle offline contiene el `.deb`, todas las dependencias de la matriz Ubuntu
24.04 amd64 y repositorio/índice local verificable o instalador que configure ese
repositorio para APT. Incluir Tesseract, español y herramienta PDF elegida; probar
con red deshabilitada en VM limpia. `dpkg -i` solo no resuelve dependencias ausentes.
No afirmar instalación offline hasta completar ese ensayo y guardar evidencia.

Para NAS: montar en SO con credenciales protegidas, identidad de servicio y permisos
de lectura/escritura apropiados, reconexión y comprobación de identidad del recurso.
Documentar SMB/NFS, UID/GID o ACL según caso; no almacenar contraseñas de montaje
en rutas de biblioteca ni enviarlas a licencias. systemd/launchd/Windows deben
gestionar recurso tardío/desconectado sin declarar eliminados todos los archivos.

## Actualización y desinstalación

1. Verificar firma/checksum del release y `published_at` firmado contra
   `entitled_release_until`. La firma del manifiesto usa claves distintas a licencias.
2. Modo mantenimiento, drenar jobs/materializaciones, snapshot verificado y comprobar
   espacio. No ejecutar migraciones con dos instancias del servicio.
3. Actualizar binarios/dependencias y migrar esquema por versión/checksum.
4. Comprobar salud, FKs/FTS y muestra de archivos, reanudar cola/watcher.
5. Si falla, restaurar conjunto compatible de binario/DB/estado; no bajar únicamente
   el binario tras una migración irreversible. Conservar evidencia de recuperación.

Conservar DB, raíz/ubicaciones externas, OCR, configuración, identidad/clave local y
licencia entre upgrades. Desinstalación elimina programa/servicio, **no documentos
ni DB por defecto**; borrado explícito posterior es operación separada. Perder
mantenimiento no deshabilita la versión instalada de una perpetua válida.

## Backup/restore consistente

Respaldar solo `.db` en caliente ignorando WAL no es un backup válido. Propuesta V1:
ventana breve de mantenimiento para escrituras de negocio, drenar materializaciones,
fijar corte de metadatos y usar [Online Backup API](https://www.sqlite.org/backup.html)
para snapshot DB que incorpore transacciones comprometidas del WAL. La copia
resultante no necesita transportar un WAL separado incongruente.

Copiar archivos managed referenciados y temporales que deban recuperarse, revisiones
de configuración, licencia, identidad y clave local protegida, y manifiesto de
hashes/tamaños/versiones. Definir alcance visible: las raíces linked pertenecen al
cliente y pueden cambiar externamente; incluir copia de originales solo por opción
administrativa explícita o snapshot del volumen. Si no se incluyen, manifest indica
«índice/OCR conservado, originales externos no incluidos». No prometer consistencia
de bytes linked contra cambios externos sin snapshot/control del cliente.

Validar origen antes/después de copiar. Una ventana de solo lectura de la app no
impide editores externos; si managed cambia externamente, reportar inconsistencia
y reintentar/verificar, no declarar respaldo íntegro. Snapshot DB/documentos se
asocian a un mismo corte; no omitir materializaciones en tránsito.

Restaurar en directorio privado aislado con servicio detenido: verificar manifiesto,
integridad DB/FKs, relación documentos/archivos y credenciales locales. Registrar
restore, invalidar sesiones restauradas y reconciliar raíces con ausencia segura.
Remapear rutas mediante plan auditado conservando historial; no tratarlas como
carpetas nuevas para duplicar índice. En otro equipo, usar recuperación/transferencia
de licencia, no aceptar silenciosamente huella anterior ni bajar revisión máxima.
Ensayar pérdida de energía, disco lleno y restore completo en los cuatro objetivos.

## Almacenamiento privado de cargas H4

Por defecto se usa `state/uploads`. `upload_directory` permite una ruta absoluta
local y privada, fuera de `htdocs`, `wwwroot` y cualquier raíz documental. Puede
omitirse en configuraciones H2/H3. No requiere cambiar las rutas de instalación
al adoptar la marca AIBID; los nombres técnicos anteriores permanecen compatibles.

Para cambiar esa ruta después de cargar archivos: detener el servicio, conservar
una copia de DB/estado/temporales, trasladar **todo** el directorio preservando
nombres y permisos, actualizar `upload_directory` y comprobar las cargas al
reiniciar. No apuntar a una carpeta vacía dejando archivos pendientes en la
anterior. H4 no ofrece traslado automático ni purgas. Los destinos managed se
registran separadamente en Carpetas y se seleccionan en Configuración; su uso
para materializar documentos está disponible en H5. [Recorrido y recuperación](docs/H5.md).

### Corrección del escaneo en bibliotecas híbridas

Detener el proceso anterior y arrancar el binario actualizado evita que el escaneo
linked marque como ausentes las nuevas cargas temporales managed. Esta corrección
no cambia el esquema ni repara los datos de prueba anteriores por indicación del
propietario. [Detalle y regresión](docs/H4.md).

## Actualizar a H5

Conservar copia consistente de estado/configuración y temporales con el servicio
detenido. Iniciar el nuevo binario sobre la misma configuración aplica 0004 sin
reparar ni reclasificar documentos H4. Registrar/seleccionar un destino administrado
y decidir si se exige revisión. Por defecto ambas políticas están desactivadas y
la retención conserva temporales. Las cargas previas sanas pueden confirmarse o
enviarse a revisión. No se debe mover el destino ni borrar sus parciales durante
un guardado. [Diagnósticos y reintentos](docs/H5.md).

## H6: licencias y simulador

La pantalla **Licencia** permite activar y renovar con HTTPS o archivos. La
[guía H6](docs/H6.md) incluye una instalación aislada en el puerto 8100, simulador
HTTPS en 9443, escenarios públicos y recuperación de identidad. Ninguna prueba
requiere modificar la instalación habitual en 8090. No ejecutar ambos servicios
con el mismo directorio de estado.

Producción requiere `license.server_url` y claves públicas autorizadas por `kid`
para activar online; para archivos basta la confianza pública configurada. No
acepta CA/bypass/clave de desarrollo. Conservar `state/license/installation.json`
junto a SQLite al respaldar; perderlo requiere recuperación local y nueva
autorización del proveedor. La migración 0005 es automática y deja intactos los
originales, cargas, expedientes, usuarios y migraciones anteriores.
