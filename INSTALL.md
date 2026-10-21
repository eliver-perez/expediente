# AIDEX — Desarrollo, distribución y operación

**AIDEX 1.0** es la primera edición comercial (versión interna 1.0.0). Distribución para Windows x64, Ubuntu 24.04 amd64/arm64 y macOS Apple Silicon. [Instalación vigente](docs/PRODUCCION.md), [transición y verificación](docs/TRANSICION-PRODUCCION.md). Los apartados de desarrollo usan configuración propia; no son instrucciones del paquete instalado. No se requiere Apache/XAMPP.

## Ejecutar en macOS / Visual Studio Code

Se necesitan Go (descarga automática del toolchain fijado 1.27.1) y Node 22.12+.
Desde la terminal de esta carpeta:

```sh
make deps
make build-dev
./build/aidex-server-dev init
./build/aidex-server-dev bootstrap
./build/aidex-server-dev serve
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

En macOS: `~/Library/Application Support/aidex/config.json`; la DB vive
en su subcarpeta `state/`. Linux usa `os.UserConfigDir()` (normalmente `~/.config/`)
y Windows el directorio de configuración del usuario. Son rutas de desarrollo,
distintas de las rutas del servicio instalado indicadas en la guía vigente. Todos los comandos admiten:

```sh
./build/aidex-server-dev init --config /ruta/local/privada/config.json
./build/aidex-server-dev bootstrap --config /ruta/local/privada/config.json
./build/aidex-server-dev serve --config /ruta/local/privada/config.json
```

Elegir ruta fuera de `htdocs`, `wwwroot` y cualquier raíz pública del cliente.
No usar enlaces simbólicos en estado; para pruebas desechables macOS usar
`/private/tmp`. Se verifica filesystem local, permisos privados y lock exclusivo.
El repositorio fuente tampoco debe exponerse como raíz de un virtual host Apache.

En Windows sin Make: `go mod download`, `npm --prefix web ci`,
`npm --prefix web run build`, `go build -tags development -o build/aidex-server-dev.exe ./cmd/gestor-documental`;
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

Con servicio detenido, `./build/aidex-server recover-admin` solicita un
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

## Paquetes y operación vigentes

Primera edición comercial: AIDEX 1.0; versión técnica 1.0.0. Windows amd64, Ubuntu 24.04 amd64/arm64 y macOS Apple Silicon arm64; Intel queda excluido. Cuatro paquetes en dist/aidex-1.0.0. Instalación nueva: servicio Windows AIDEX, Ubuntu aidex.service y macOS app.aidex; backend aidex-server (en Windows .exe), launcher AIDEX.exe.

Los comandos, rutas, cuentas, permisos, actualización y desinstalación actuales están en [Instalación](docs/PRODUCCION.md). Las actualizaciones de AIDEX preservan su cuenta, rutas e identidad. No se importan perfiles de otros productos; retira manualmente cualquier instalación anterior que ocupe el puerto, conservando sus datos.

Construir no certifica el instalador nativo. Validación y pendientes reales: [transición](docs/TRANSICION-PRODUCCION.md). No se incluye Mac Intel, no se publica una actualización automática firmada ni se promete respaldo/restauración integral. Los documentos de hitos anteriores quedan como evidencia histórica, no como instrucciones vigentes.
