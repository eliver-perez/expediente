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
2. Implementación y pruebas: pendientes.

Este documento registra avances comprobados; no certifica la migración hasta
que se documenten las pruebas sobre PostgreSQL real.
