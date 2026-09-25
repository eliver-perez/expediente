# AIBID — Seguridad y controles por hito

D-01 a D-05 fueron aprobadas por el propietario. H2 implementa identidad, cookies,
CSRF, permisos globales, límites y auditoría descritos en [H2](docs/H2.md). Los
controles documentales/licencias se entregan en sus respectivos hitos.

H3 aplica permisos por biblioteca, no sigue enlaces internos, abre originales bajo
handles `os.Root`, valida identidad de volumen/archivo, restringe consultas exactas
a auditores de todos sus ámbitos y limita procesos PDF/OCR. [Implementación y
límites comprobables](docs/H3.md). El permiso simulado solo entra con el build tag
`development`; el build normal exige desde H6 una licencia verificada.

## Alta, contraseñas y sesiones

Primer administrador mediante CLI local interactiva, ejecutada por operador
autorizado del equipo; lock y transacción impiden dos altas iniciales. Solo procede
si no existe administrador inicial. Sin contraseña predeterminada, argumentos con
secretos, URL de bootstrap LAN o mecanismo público para «reabrir» el alta. La
recuperación posterior es un comando local distinto, auditado y con servicio en
modo mantenimiento. La licencia no impide este acceso de recuperación.

Argon2id con sal aleatoria por contraseña, formato PHC y parámetros versionados.
Parámetros H2: 64 MiB, 3 pasadas, paralelismo 1, salida de 32 bytes; como máximo
dos hashes simultáneos. Reajustar tras medir el equipo. No truncar contraseñas
silenciosamente. Verificar límites de entrada antes de consumir memoria. Go ofrece
[Argon2id mediante IDKey](https://pkg.go.dev/golang.org/x/crypto/argon2).

Política H2: mínimo 6 caracteres, máximo 128, permitir espacios/Unicode,
sin reglas de composición arbitrarias. Verificación de contraseña actual para
«Mi cuenta»; administrador con `users.reset_password` puede restablecer sin conocerla.
Toda modificación incrementa `credential_version`, invalida sesiones y exige login.
Nunca registrar contraseña, PHC, token de sesión, CSRF, claves privadas o comerciales.

Cookie opaca de 32 bytes aleatorios; guardar digest SHA-256 del token, no token.
`HttpOnly`, `SameSite=Lax`, `Path=/`, sin `Domain`, `Secure` con HTTPS; prefijo
`__Host-` en producción HTTPS. Desarrollo HTTP loopback usa nombre distinto.
Valores iniciales H2: inactividad 30 minutos, duración absoluta 12 horas, configurables.
`GET /auth/session` cada 30 segundos mientras la app está abierta y al recuperar foco;
este sondeo no prolonga la inactividad. Actividad humana real sí, actualizaciones
de `last_activity_at` agrupadas para no escribir en cada request. Límites y
expiración se comprueban en cada acceso aunque el último toque sea agrupado.

Una sesión cerrada por `new_login` devuelve 401 `SESSION_REPLACED`. React elimina
su estado y muestra exactamente: «Tu sesión fue cerrada porque se inició sesión
con tu cuenta desde otro equipo o navegador». Las otras causas tienen códigos
distintos. No atribuir identidad física o ubicación a una IP o user agent.

Un 401 genérico no borra la cookie: una respuesta tardía de una petición anterior
no debe eliminar la cookie de un login nuevo. React descarta su estado de sesión;
logout y cambio de contraseña sí envían la eliminación explícita de la cookie.

Autenticación fallida: respuesta genérica, hash ficticio de costo equivalente para
usuario inexistente, rate limit por IP y clave de identificador normalizado, espera
progresiva con expiración. Valores iniciales H2: 5 fallos por identificador/15 min y
30 por IP/15 min, ajustar para LAN compartida; sin bloqueo permanente que permita
denegar servicio a otra cuenta. Persistir intentos y reinicio no borra límites.
Identifier saneado/limitado, fecha UTC, IP observada, agente y resultado, incluso
sin usuario existente. Fallos de login no revelan si existe la cuenta.

## Autorización y acceso documental

En Go: sesión vigente → permiso en biblioteca/recurso → capacidad/licencia → estado
válido → operación. Denegación por defecto; filtros en SQL antes de paginar/contar.
Revalidar dentro de mutaciones y publicación de jobs. Los workers son actores
`system` con tareas concretas; no usan una sesión humana permanente ni permisos
obsoletos capturados al encolar. Lectura de temporales además exige ser autor o
revisor autorizado del documento según su estado.

Obtener un ID no concede acceso. Para recurso ajeno/no visible: 404 sin título,
ruta, conteo ni existencia. Para acción prohibida sobre recurso visible: 403.
Rutas físicas solo con permiso específico; jamás DTO de entidades DB completo.
Administrar usuarios/licencia no concede búsqueda global de contenido.

CSRF derivado con HMAC del token aleatorio de sesión, guardando solo su digest,
para métodos mutantes, incluido logout;
cabecera `X-CSRF-Token`, validación de `Origin` y política CORS cerrada. Login JSON
comprueba origen del sitio y rechaza cross-origin; no aceptar formularios simples
ni habilitar CORS `*` con credenciales. Seguridad de sesión revalidada durante
transacciones; una descarga en curso se puede cancelar al revocarse, pero los bytes
ya entregados no son recuperables.

## LAN, proxy y sistema operativo

Escucha en loopback por defecto; interface/puerto/LAN explícitos. Producción LAN
usa TLS directo o proxy con certificado del cliente/CA local confiable. Validar
host/origin permitidos. En H2, `X-Forwarded-For` solo se interpreta si la conexión
inmediata proviene de proxies configurados por CIDR, recorriendo la cadena de
confianza; en otro caso se usa la IP del socket. `Forwarded` se ignora. El proxy
confiable sobrescribe `X-Forwarded-Proto: https`. Documentar IPv4/IPv6, firewall y
red elegida; configuración concreta en [INSTALL.md](INSTALL.md).

Cuenta de servicio con mínimos permisos: linked lectura; managed escritura solo
en destinos seleccionados; DB, claves y temporales privados, sin exposición por
servidor estático. ACL de Windows o modos/propietario POSIX equivalentes. No
prometer ocultamiento criptográfico de DB/documentos frente al administrador del
equipo. Cifrado del volumen queda a la instalación; no se inventa cifrado de BD
como requisito de V1. Backups contienen información sensible y requieren ACL.

## Archivos no confiables

PDF por tipo/magic y análisis, no solo extensión. Límites configurables de archivo,
lote, páginas, CPU, memoria, tiempo y disco. MIME consistente y nombre de descarga
saneado; previews desde endpoint por ID, nunca `file://` o rutas recibidas como
URL. Los subprocesos reciben argumentos separados sin shell, usuario restringido,
directorio temporal privado y entorno mínimo; terminación del grupo al exceder
presupuesto. No pasar paths arbitrarios del cliente al extractor.

La comprobación de prefijo textual no basta para containment. Abrir bajo raíz
autorizada con primitivas seguras del SO, verificar handle/identidad tras resolver,
rechazar enlaces no autorizados, traversal, rutas de dispositivo y escapes UNC.
Validar de nuevo al usar, no solo al elegir una raíz. Ver
[especificación de rutas](docs/STORAGE_WATCHER.md).

React renderiza consultas/extractos como texto escapado. Convertir resaltado FTS
a segmentos seguros; no insertar HTML arbitrario. Visor PDF empaquetado localmente
sin scripts externos/CDN, CSP y cabeceras de contenido. Dependencias fijadas y
actualizadas con pruebas; revisar permisos/licencias de redistribución PDF/OCR.

## Auditoría, privacidad y retención

Evento de negocio + mutación + tareas en la misma transacción. Auditoría requerida
para búsquedas, previews y descargas: si no puede persistirse, respuesta de error
controlado y diagnóstico, sin afirmar falsamente que quedó registrada.
La consulta exacta solo está en `search_audit_details`, accesible con permiso
`audit.search_details`; no repetirla en access logs. Nunca registrar cuerpos HTTP
de auth/licencias, OCR completo ni rutas privadas mediante mensajes de error.

Bitácora append-only en operaciones normales; controles SO y respaldo protegen
contra modificaciones ordinarias. Retención aprobada por administrador y registrada;
sin purgas automáticas hasta configurarla. Separar detalle sensible de búsquedas
para retirarlo sin borrar el evento de que hubo una búsqueda. Usuario puede ver
su historial; auditor global de accesos requiere permiso independiente.

Licencia: claves privadas locales en almacén/archivo protegido, respaldadas solo
mediante operación administrativa segura; trust store público separado por entorno
y propósito. Ninguna clave de fixture entra en una build productiva. Verificar JWS
antes de confiar en campos, `alg`/`typ`/`kid`, producto, instalación, clave, huella,
licencia/activación y revisión. Límites de tamaño y JSON estricto sin claves duplicadas.

## Pruebas de seguridad exigidas por hitos

H2: login simultáneo, invalidación por contraseña, CSRF, enumeración, límites y
ausencia de secretos en logs. H3: path traversal, alias entre bibliotecas, symlink
swap, junction/UNC, raíz caída y filtrado FTS. H4/H5: temporal no revelado ni abierto
por otro usuario, aprobación cruzada, fallo entre copia/commit. H6: firmas, entornos,
retroceso de revisión, reloj y read-only. H7: instalación, upgrade, restore y revisión
de superficie expuesta en cada SO. Los casos concretos están en
[hitos](docs/MILESTONES.md).

## Controles entregados en H4

Temporales en disco local privado (directorio 0700, archivos 0600 en Unix), fuera
de raíces públicas/documentales. Recepción por streaming con dos plazas, hasta
100 intentos por lote, tamaño máximo de `indexing.maximum_file_mb`, reserva de
64 MiB además del máximo y deadline HTTP de cinco minutos. Validación de cabecera
y estructura PDF mediante `pdfinfo` antes de crear el documento; publicación con
hash, comprobación de sesión/permisos y licencia nuevamente en transacción.
El multipart exige clave y un único archivo, y rechaza partes adicionales.

Ficha, original, descarga, texto, historial, búsquedas, duplicados, trabajos y
conteos de expedientes filtran temporales al autor/revisor autorizado. Ser gestor
o administrador global no concede acceso implícito a cargas privadas ajenas.
Revocar el rol de revisor tiene efecto en la siguiente petición. Los DTO no
incluyen localizadores ni rutas temporales, tampoco con `storage.view_paths`.

La recuperación al arrancar limpia exclusivamente nombres de transferencias
`receiving` registrados y las marca fallidas. Cancelar conserva bytes y evidencia;
no hay purga automática. Cambiar `upload_directory` requiere mantenimiento y
trasladar íntegramente su contenido preservando permisos; no hay migración de
almacenamiento por interfaz en H4. [Evidencia y límites](docs/H4.md).

## Controles entregados en H5

Separación autor/solicitante/revisor, asignación, revisión optimista y versión de
contenido. Un permiso explícito de autoaprobación exige evento y se revalida antes
del commit asíncrono, junto con usuario activo, capacidades y permiso de decisión.
La publicación no reemplaza archivos existentes y verifica bytes/ubicación antes
de aprobar. Los handles de directorio impiden seguir enlaces internos. El fencing
rechaza trabajadores anteriores a un reinicio. La limpieza ocurre después del
commit y vuelve a verificar el definitivo. Retención de rechazados/cancelados
conserva la privacidad aun cuando ya no quedan bytes temporales.
[Pruebas y límites de filesystem](docs/H5.md).

## Controles entregados por H6

Solo claves públicas de producción con propósito `license` verifican JWS en el
binario normal. Se rechaza la clave pública conocida de prueba aun cambiando su
`kid`. El simulador y su semilla de prueba se excluyen mediante build tags. TLS
verifica certificados, prohíbe redirecciones y acota tiempos/tamaños; las respuestas
remotas y errores nunca reflejan claves comerciales en logs o interfaz.

Clave privada local fuera de DB, acceso privado del SO, referencia pública anclada
a SQLite y recuperación con diario local/auditoría. No se regenera silenciosamente
una identidad perdida. El hash de huella reemplaza al identificador de hardware
original en archivos y red. Renovaciones/importaciones verifican binding completo
y piso persistido de revisión; estado, evidencia y auditoría se guardan juntos.

Gates por operación y workers, permisos/CSRF de administración y autorización
repetida al confirmar una operación de red. La solicitud offline de desactivación
no declara una liberación remota. Consultas/descargas conservan sus permisos en
vencida/revocada; recuperación y política de backup siguen disponibles. La hora
máxima persistida limita retrocesos, sin prometer protección frente al administrador
del SO. [Pruebas, recuperación y límites de interoperabilidad](docs/H6.md).
