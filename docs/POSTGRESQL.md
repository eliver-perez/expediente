# AIDEX 1.0.0 — PostgreSQL

## Base y alcance

La rama `feature/postgresql` parte de `5e3c44f`, sincronizado con
`origin/main` el 13 de noviembre de 2026. Incluye `220c711` (rendimiento,
procesamiento y navegación en Windows) y `5e3c44f` (eliminación masiva,
manifiestos y progreso). No se cambia la versión comercial ni se generan
instaladores. No se modifica el servidor de licencias.

PostgreSQL será el único motor. No habrá conversión de datos de SQLite,
compatibilidad con sus bases ni selección de motor. El esquema inicial de
PostgreSQL recoge las entidades, permisos, restricciones e índices existentes.

## Decisiones de implementación

- Conservar `database/sql` y los límites separados de conexiones para control,
  progreso y analítica. Usar pgx y varias conexiones de escritura.
- SQL nativo de PostgreSQL, claves foráneas diferibles para relaciones cíclicas,
  restricciones y triggers de inmutabilidad. Migraciones con checksum y bloqueo
  asesor de PostgreSQL, sin depender del bloqueo de archivos para proteger DDL.
- Búsqueda con `tsvector`, GIN y consultas de términos/frases. Mantener palabras
  vacías, búsqueda sin distinción de acentos, filtros, autorización, agrupación,
  cursores y fragmentos seguros; no introducir stemming que cambie resultados.
- Reclamar trabajos mediante bloqueos de filas y `SKIP LOCKED`; conservar leases,
  fencing tokens, cancelación, recuperación y los índices de rendimiento.
- Respaldos lógicos con una instantánea consistente. Validación en un esquema
  temporal aislado de PostgreSQL y restauración atómica con referencias
  comprobadas. Conservar las exclusiones de licencias y secretos.
- Clúster privado por instalación, binarios y datos separados, credenciales
  protegidas, autenticación SCRAM, acceso por loopback y puerto propio. El
  aprovisionamiento no debe usar ni modificar clústeres ajenos.
- El futuro paquete incluirá los binarios apropiados para macOS, Windows y
  Ubuntu AMD64/ARM64. En esta etapa se desarrolla el ciclo de vida del motor y
  se prueba con los binarios locales de desarrollo, sin editar instaladores.

Referencias técnicas: [pgx stdlib](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib),
[búsqueda textual](https://www.postgresql.org/docs/18/textsearch-controls.html),
[bloqueos](https://www.postgresql.org/docs/18/explicit-locking.html) e
[inicialización](https://www.postgresql.org/docs/18/app-initdb.html).

## Validación exigida

Se ejecutarán las pruebas de servicios y HTTP existentes sobre PostgreSQL real,
además de pruebas de esquema, integridad, búsqueda, escrituras simultáneas,
reclamo de trabajos, reinicios, respaldos/restauración y bibliotecas grandes.
Los originales físicos se mantienen sujetos a las reglas existentes de
bibliotecas vinculadas, administradas e híbridas.

## Avance

1. Preparación: árbol limpio, remoto actualizado y rama creada; inventario de
   dependencias realizado en almacenamiento, migraciones, bibliotecas, HTTP,
   identidad, cliente de licencias, respaldos, mantenimiento, CLI y pruebas.
2. Persistencia implementada en `a2ce193`: esquema inicial PostgreSQL, SQL de
   servicios, índices GIN, conexiones concurrentes, respaldos, mantenimiento y
   aprovisionamiento. Se retiraron el controlador y las migraciones SQLite.
3. Validación macOS aprobada: suites de producción y desarrollo con detector de
   carreras, 143 casos/subcasos de bibliotecas, 37 flujos de navegador y cinco
   flujos adicionales contra el motor aprovisionado por AIDEX. Cargas de 100 000
   documentos, 200 000 páginas y 9 000 archivos físicos verificadas. Evidencia y
   límites en [el registro de QA](qa/postgresql-macos-20261113.md).

La validación nativa de Windows también terminó, con pruebas funcionales,
servicio temporal, navegador y documentos físicos reales. Sus resultados y
límites están en [el registro de QA Windows](qa/postgresql-windows-20261113.md).
El empaquetado y la validación nativa de Linux siguen pendientes.


## Ejecución local y contrato del runtime

En desarrollo, `AIDEX_POSTGRES_BIN` permite elegir los binarios PostgreSQL 18;
si se omite se busca el runtime junto al ejecutable y después en PATH. Los
binarios con canal `production` utilizan exclusivamente el runtime del paquete,
ubicado en `tools/postgresql/bin` junto al backend. Este runtime deberá contener
los ejecutables, bibliotecas, extensiones (`unaccent`) y datos compartidos de su
plataforma/arquitectura; copiar solo los ejecutables de Homebrew no constituye
un runtime distribuible.

Cada estado reserva `postgresql/installation.json` (credenciales, formato y
puerto), `postgresql/data` (clúster), `postgresql/server.log` y su bloqueo privado.
No se usa el puerto 5432 ni un servicio/clúster global. `initdb` activa checksums,
UTF-8 y SCRAM. El administrador del motor y `aidex_app` tienen contraseñas
aleatorias distintas; la aplicación usa `aidex_app`, sin SUPERUSER, CREATEDB ni
CREATEROLE. El servicio AIDEX inicializa, arranca y supervisa su motor local;
recupera una caída sin reinstalación y lo detiene tras cerrar sus pools.

Las herramientas de diagnóstico se conectan al clúster ya iniciado mediante
`storage.Connect`: no aplican migraciones ni toman control de su ciclo de vida.
El estado no admite enlaces ni sistemas de archivos de red. Un directorio de
datos sin descriptor de propiedad se conserva y se rechaza; no se inicializa
sobre él. Las bases del motor anterior también se conservan y se rechazan.

El esquema se aplica en una transacción protegida por bloqueo asesor, con
checksum y rechazo de versiones desconocidas. Las actualizaciones mayores del
motor requieren el runtime y el procedimiento de mantenimiento correspondientes;
no se intenta ejecutar otro major contra un clúster existente. Autovacuum y
recuperación WAL pertenecen al motor privado y no dependen de Internet.

El CLUF conserva la identidad histórica de quien aceptó, como la bitácora;
no se exporta ni reemplaza su evidencia local al restaurar otra instalación.
Los respaldos portátiles incluyen correcciones y sustituciones documentales,
excluyen licencias/sesiones/secretos y regeneran los vectores GIN al importar las
páginas. La restauración ajusta las secuencias IDENTITY antes de reactivar trabajo.

## Reproducción de pruebas

```sh
scripts/test_postgresql.sh
scripts/test_postgresql.sh npm --prefix web run test:e2e
AIDEX_PERF_SCALE=1 AIDEX_PERF_FOLDERS=1 scripts/test_postgresql.sh go test -v -tags development -timeout 12m -run 'TestSyntheticScale|TestFolderNavigationScale' ./internal/libraries
AIDEX_PERF_POSTGRES=1 scripts/test_postgresql.sh go test -v -tags development -timeout 5m -run TestPostgresLargeIndexConcurrentDeletion ./internal/libraries
```

El arnés usa una carpeta temporal privada y un socket Unix sin TCP. Su modo
trust se limita a esa fixture; el aprovisionamiento del producto y sus pruebas
usan SCRAM. `AIDEX_TEST_POSTGRES_DSN` solo sirve para aislar esquemas en pruebas
con binarios de canal `source`; no configura los paquetes de producción.

## Fuera de la entrega de esta etapa

No se han generado ni modificado instaladores. La inclusión y firma de runtimes,
las cuentas y el arranque por el sistema operativo y las actualizaciones de
paquetes se integrarán cuando se autorice el empaquetado. La compilación cruzada
no sustituye las pruebas nativas de Windows y Ubuntu AMD64/ARM64. El servidor de
licencias, la versión 1.0.0 y la interfaz del producto se mantienen.
