# Decisiones para revisión

Fecha: 2026-09-23. **D-01 a D-05 aprobadas por el propietario** al responder
«Adelante con las propuestas, continuemos». H2 autorizado. El nombre visible acordado posteriormente es AIBID. No se necesita resolver ahora el dominio del
repositorio, tamaño de corpus ni equipo de cada cliente; se documentan después.

## Alcance ya fijado por el usuario

Go; React compilado; SQLite local WAL/FTS5; servicio por SO; tres modalidades
por biblioteca; usuarios ilimitados; sesión única; archivos fuera de SQLite;
raíces linked de lectura; watcher y reconciliación; permisos en Go; solo CLIENTE
de licencias V1.0. No implementar MySQL, servidor de licencias, extracción avanzada,
integraciones institucionales, acuses ni reglas de contratación.

## D-01 — Alta inicial, transporte y autorización

**Recomendación:** alta del primer administrador mediante comando local interactivo
del servicio, sin contraseña en argumentos ni HTTP público de bootstrap. Escucha
inicial en `127.0.0.1`; habilitación LAN explícita con HTTPS, directo o por proxy.
HTTP solo para desarrollo loopback. Cookies opacas; permisos por biblioteca,
denegación por defecto; administrar el sistema no concede leer todas las bibliotecas.

**Alternativa:** asistente web con token local de un solo uso. Facilita instalación
gráfica, pero añade custodia del token, caducidad y superficie de bootstrap. HTTP
en LAN expone credenciales/sesiones; no se recomienda como valor de producción.

**Consecuencias:** el instalador necesita un paso local de alta; asignar bibliotecas
es explícito. Separar permisos globales de contenido desde las primeras tablas.
Aprobada e implementada en H2 para identidad y permisos globales; permisos de
biblioteca se incorporan en H3. Detalle: [seguridad](SECURITY.md).

## D-02 — Identidad física, documentos y temporales

**Recomendación:** una identidad física pertenece a una biblioteca y tiene como
máximo un documento V1. Ese documento tiene cero o un expediente. Una carga nueva
crea una identidad `managed` antes de tener ubicación definitiva; su localizador
privado vive en `upload_items`, separado de las rutas definitivas. Alias físicos
(por ejemplo hard links) se guardan como ubicaciones de la misma identidad, no como
otro documento. Revisiones de contenido y páginas OCR son entidades separadas.

**Alternativa:** varios documentos para un archivo físico, o raíz falsa de temporales.
La primera complica propiedad y permisos y permitiría múltiples expedientes sobre
el mismo original; la segunda mezcla almacenamiento transitorio con raíces que se
exploran y se muestran. Ninguna se recomienda para V1.

**Consecuencias:** claves compuestas impiden mezclar bibliotecas; índice único sobre
`documents.physical_file_id`. `storage_source` es inmutable. SHA-256 no es una clave
de identidad. El SQL de referencia expresa esta decisión aprobada para H3/H4.

## D-03 — Almacenamiento, enlaces y recuperación

**Recomendación:** DB y temporal en disco local privado; raíces finales configurables
por biblioteca, incluso montajes. Materialización recuperable: copiar a un archivo
parcial en destino, verificar SHA-256 y tamaño, sincronizar según el SO, publicar
sin sobrescritura y confirmar aprobación en DB; solo entonces limpiar temporal.
Conservar historial de metadatos/OCR; no prometer conservar bytes históricos de
originales linked. Resolver la raíz elegida; omitir enlaces simbólicos/junctions
en su interior en V1 con diagnóstico, sin seguirlos silenciosamente.

**Alternativas:** mover primero y aprobar después reduce E/S pero dificulta recuperar
fallos entre discos. Seguir enlaces internos añade comprobaciones por apertura,
ciclos y alias entre bibliotecas. Guardar copias históricas linked cambiaría su
naturaleza de lectura y el costo de almacenamiento; requiere acuerdo nuevo.

**Consecuencias:** historial no significa versiones descargables de todo original.
Un NAS debe ofrecer las operaciones requeridas o se rechaza como destino final;
no se asume que `rename` entre discos sea atómico. Decisión aprobada para H3/H5.

## D-04 — Revisión y separación de responsabilidades

**Recomendación:** donde esté activada la revisión, el autor no aprueba su propia
carga/asociación. Cualquier excepción exige permiso específico y evento de auditoría.
Sin revisión configurada, una acción de confirmar con permiso `documents.finalize`
usa la misma materialización y deja aprobación automática identificada como tal.
No obliga a comprar `review_workflow` para usar `managed_libraries`.

**Alternativa:** permitir autoaprobación a todo revisor simplifica equipos pequeños
pero elimina separación de funciones. Exigir revisión siempre impediría una
biblioteca administrada con los módulos comerciales básicos.

**Consecuencias:** permisos distintos para cargar, enviar, revisar y finalizar;
política versionada por biblioteca. Una modificación externa managed aprobada
queda inválida para cumplimiento hasta nueva decisión; linked conserva aprobación.
Decisión aprobada para los permisos documentales y el flujo de H5.

## D-05 — SQLite y límite de las abstracciones

**Recomendación:** evaluar `modernc.org/sqlite` (sin CGO) con `database/sql`; elegir y
fijar versión en H2 después de verificar FTS5, WAL, backup y los cuatro objetivos
de release. Un escritor serializado y lectores acotados; transacciones breves,
`foreign_keys=ON`, `busy_timeout` acotado y `synchronous=FULL`. SQL visible por módulo,
sin repositorio genérico universal. Conservar interfaces para transacciones y
búsqueda sin implementar un segundo motor.

**Alternativa:** `mattn/go-sqlite3` con CGO y FTS5 habilitado ofrece SQLite nativo pero
requiere toolchains C por plataforma. `synchronous=NORMAL` puede rendir mejor con
menor garantía de durabilidad ante pérdida de energía; medir antes de cambiarlo.

**Consecuencias:** binario Go portable no elimina dependencias nativas PDF/OCR.
El motor SQLite de producción debe incluir la corrección WAL documentada en
3.51.3 o posterior (o backport oficial verificado), no asumir que el `sqlite3`
del equipo es el motor embebido. [SQLite WAL](https://www.sqlite.org/wal.html),
[driver candidato](https://pkg.go.dev/modernc.org/sqlite).
**Resultado H2:** seleccionado y fijado `modernc.org/sqlite` 1.59.0 con su
`libc` 1.75.7. Verificados WAL, FTS5, FKs, backup del driver y transacciones
concurrentes en macOS arm64; compilación sin CGO para los cuatro objetivos.
El arranque exige SQLite 3.51.3 o posterior. Un escritor y hasta cuatro lectores.
Las pruebas nativas en los demás sistemas e instaladores siguen pendientes;
compilar no acredita permisos ni operación en esos equipos.

## Decisiones diferidas, sin cambiar el contrato

| Tema | Propuesta a validar antes de su hito |
| --- | --- |
| Retención | Configurar antes de activar purgas; ninguna purga automática en el estado inicial. Separar temporales, sesiones, búsquedas y auditoría general. |
| Tiempo | UTC RFC 3339 externo; representación UTC canónica de precisión fija en DB. Reloj retrocedido genera diagnóstico, no extiende tolerancia. |
| Huella por SO | H6: componentes estables y minimizados; comprobar autorización local y recuperación de hardware antes de fijar algoritmo V1. |
| OCR/PDF | H3: evaluar herramientas, licencias redistribuibles y corpus; español inicial, inglés opcional. Presupuestos de CPU/RAM/disco configurables. |
| Grupos/tipos | El alcance los hace opcionales. V1 inicial propone roles por biblioteca; añadir alcance por área/tipo solo tras revisión, antes de implementarlo. |
| Binarios históricos managed | Historial, hash y OCR sí; copias de todas las versiones no son un compromiso implícito. Definir retención antes de ofrecer restauración de versiones. |
| Actualizaciones firmadas | Claves y manifiesto distintos de licencias. Definir formato en H7; no alterar los cuatro endpoints externos. |

## Ambigüedades del contrato externo a contrastar con el otro proyecto

Conservar literalmente V1.0. En el ejemplo firmado `fingerprint_version` es la
cadena `"1"`; no convertirla a número unilateralmente. Fijar vectores compartidos
para su representación en todas las solicitudes antes de declarar conformidad.

La desactivación offline no define un archivo de acuse firmado distinto del `.lic`.
Generar la solicitud y mostrar «pendiente de confirmación» no libera la plaza;
no inventar un endpoint/acuse ni interpretar cualquier licencia revocada como
liberación confirmada. Documentar la confirmación manual del servidor.

V1.0 define prueba de posesión sobre desafío/identidad, no sobre todo el cuerpo HTTP.
Implementarla exactamente y exigir TLS; no añadir campos al texto firmado. El
formato detallado de huella, la política de igualdad de revisión (reimportación
idéntica sí; misma revisión con contenido distinto no), el JSON exacto de respuesta
de desactivación y la retención de idempotencia requieren vectores del otro proyecto.
Una incompatibilidad se registra como propuesta V1.1, jamás como cambio silencioso.

## Aplicación en H3

El propietario autorizó H3 y cambió el mínimo de contraseña a seis caracteres.
Se conservan Argon2id y el resto de controles de H2. `fsnotify` 1.10.1 implementa
vigilancia nativa con reconciliación; Poppler y Tesseract son programas externos
para extracción por página. Español por defecto. Se conserva el límite de V1:
consolidación propia, conflictos entre bibliotecas sin reorganización automática.
Detalles, presupuestos, licencias de terceros y validación en [H3](docs/H3.md).

## Aplicación en H4 y marca

El propietario autorizó continuar H4 y eligió **AIBID — Aplicación de Indexación
de Bibliotecas Digitales**, con el eslogan **Tu biblioteca digital, ordenada y al
alcance.** Los cuatro SVG originales se conservan íntegros; las variantes oscuras
se usan sobre fondos oscuros. La marca visual no modifica el product identifier
del contrato de licencias, el módulo Go, binarios ni directorios de instalaciones
anteriores. `LICENSE_CONTRACT.md` permanece byte por byte sin cambios.

H4 entrega cargas/organización según D-02/D-03. Los temporales están separados
de raíces managed/linked y el autor/revisor controla su consulta. Versionar una
plantilla nunca propaga cambios a expedientes existentes. La conversión a híbrida
amplía capacidades sin mover ni reextraer documentos. D-04 (decisiones de revisión
y finalización) se aplicará en H5. [Implementación y verificación H4](docs/H4.md).

## Aplicación de D-03/D-04 en H5

H5 incorpora revisión por biblioteca y materialización con diario persistente.
La publicación sin reemplazo utiliza un enlace duro dentro del destino después
de copiar/verificar/sincronizar el parcial. Un filesystem que no soporte esa
operación falla conservando el temporal. Confirmación directa y aprobación por
revisor comparten el protocolo; las diferencias linked/managed se mantienen en
integridad y cumplimiento. [Entrega, recuperación y límites](docs/H5.md).
