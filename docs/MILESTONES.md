# Hitos y criterios de aceptación

Un hito se cierra tras demostración y pruebas; el siguiente empieza con revisión
del propietario. Mantener los siete documentos raíz sincronizados. No entregar
un módulo como completo si solo tiene interfaces, mocks o endpoints sin controles.

| Hito | Entrega acotada | Migración y verificación | Puerta de aceptación |
| --- | --- | --- | --- |
| H1 aprobado | Arquitectura, alternativas, SQL propuesto, estados, permisos, API, pantallas, rutas y plan | Verificador aislado de DDL, invariantes y enlaces | Revisar D-01…D-05; no aprobarlas implícitamente |
| H2 | Servicio Go básico, SQLite WAL/FTS5, config privada, migraciones, bootstrap local, usuarios, sesiones, auditoría, React login/Mi cuenta/admin básico | Migraciones solo identidad/permisos/auditoría; tests unit/integración y transacciones concurrentes; comandos reales de desarrollo | Login B invalida A y UI exacta; contraseña revoca; intentos fallidos trazables; API deniega sin sesión |
| H3 | Bibliotecas linked, múltiples raíces, vistas/colisiones/consolidación, watcher+reconciliación, cola/OCR/explorador/buscador | Tablas raíces/identidades/versión/páginas/trabajos; corpus de fixtures por SO; fallos NAS/overflow/reinicio | Búsqueda autorizada con texto exacto auditado; ausencia conserva OCR; altas y cambios idempotentes |
| H4 | Managed/hybrid, temporales, multifile, expedientes, catálogos, plantillas/requerimientos, asociación sin copia | Migraciones de organización/cargas; pruebas de aislamiento temporal y snapshots de requerimientos | Vinculada→híbrida conserva IDs/OCR; expediente propio, asociación única; nueva raíz no bloquea cargas |
| H5 | Revisión/aprobación/rechazo, finalización directa donde corresponde, materialización, integridad managed y avance | Migraciones de workflow/journal; pruebas inyectando fallos entre discos y reinicios | Aprobación solo tras materializar; linked modificado conserva aprobación; managed exige nueva decisión |
| H6 | Cliente licencias online/offline V1.0, Ed25519/JWS, identidad/huella, revisión máxima y módulos en Go | Vectores compartidos con otro proyecto; mock local solo desarrollo, exclusión productiva | Estados/15 días/revocación/renovación/read-only exactos; ningún documento/ruta sale |
| H7 | Endurecimiento, carga/recuperación, backups, CI/release e instaladores de los cuatro objetivos | Actualización N-1, desinstalación conservadora, bundle offline en VM sin red, firma/checksum | Instalación/upgrade/restore preservan DB/PDF/OCR/licencia y tienen evidencia reproducible |

En H2–H5, el adaptador de licencia de desarrollo debe estar identificado y excluido
de builds productivas; middleware/capacidades se diseñan desde H2. No publicar un
producto sin H6 ni construir el servidor comercial para poder probar el cliente.
El artefacto de desarrollo no se anuncia como instalador comercial.

## Evidencia de esta entrega H1

Verificado el 2026-09-23 en macOS arm64 con `python3 scripts/check_design.py`:
**14 comprobaciones correctas** sobre DDL, claves/restricciones, sustitución y
rollback de sesión, identidad/origen, hashes duplicados, OCR retenido, publicación
FTS transaccional, bitácora/consulta exacta, límites de jobs y enlaces locales.
El anexo `LICENSE_CONTRACT.md` se comparó además con el archivo original recibido:
contenido idéntico (con salto final), SHA-256
`18064e5ebf3302d5be5b3bad1e62ba86b645fca8acfe9c9c11fdfbc9626a87e4`.

En el cierre de H1 todavía no se habían implementado ni probado endpoints, React,
sesiones HTTP concurrentes, filesystem por SO, OCR real, JWS ni instaladores.
D-01 a D-05 fueron aprobadas posteriormente por el propietario.

## Evidencia de esta entrega H2

Implementación y verificación local completadas el 2026-09-23. `make check` aprobó
compilación React/TypeScript, formato, vet, 21 pruebas Go con detector de carreras,
política de licencia de desarrollo y los 14 checks H1. Chrome aprobó tres escenarios
E2E, incluida la notificación automática de sesión sustituida. Los cuatro objetivos
de distribución compilan sin CGO; la ejecución local fue en macOS arm64.
La CLI se comprobó con configuración temporal, sin crear credenciales reales.
Alcance, evidencia y limitaciones en [la entrega H2](H2.md). La revisión del
propietario precedió al inicio de H3; en el cierre H2, OCR, JWS e instaladores
seguían pendientes.

## Evidencia de esta entrega H3

Implementación y verificación local completadas el 2026-09-24, tras autorización
del propietario. Se aplicó además el mínimo de contraseña de seis caracteres.
`make check` aprobó React/TypeScript, formato, vet, pruebas Go con detector de
carreras en builds normal/development (33 funciones distintas) y 14 checks H1.
Chrome aprobó los cuatro escenarios H2/H3, con OCR real, PDF visible y consulta
de texto retenido tras ausencia. Migración H2 con usuarios/auditoría comprobada;
CLI probada en estado temporal incluyendo reversión vacía y nueva migración.

Compilación sin CGO correcta para Windows amd64, Linux amd64, macOS amd64 y arm64;
build de desarrollo macOS arm64 listo. La ejecución nativa fue en macOS arm64.
NAS desconectado y pérdida de eventos se simularon; la matriz nativa completa,
ensayos de carga e instaladores siguen en H7. Licencia comercial/JWS sigue en H6.
Alcance, archivos, checksum, comandos y límites en [la entrega H3](H3.md).
El siguiente módulo es H4, después de revisar esta entrega.

## Pruebas funcionales vinculadas al alcance

| ID | Hito | Escenario y resultado observable |
| --- | --- | --- |
| ACC-01 | H2 | PC/navegador B inicia sesión; A recibe `SESSION_REPLACED` incluso inactivo, texto exacto, solo una sesión abierta en DB |
| ACC-02 | H2 | Fallos con usuario existente/inexistente: respuesta genérica, intento auditado sin contraseña; rate limit resiste reinicio |
| ACC-03 | H2 | Login concurrente y cambio/restablecimiento concurrente: no sesiones duplicadas ni credencial vieja activa; rollback preserva sesión anterior |
| ACC-04 | H3 | Vinculada indexa PDF nativo/escaneado por página; OCR selectivo español; búsquedas con acentos, frases, guiones y sintaxis inválida |
| ACC-05 | H3 | Raíz igual/subcarpeta se detecta; vista lógica no agrega archivos, OCR ni watcher; alias con otra biblioteca no evade permisos |
| ACC-06 | H3 | Padre consolidado con confirmación mantiene IDs, OCR, expedientes/permisos e historial; solo indexa archivos desconocidos |
| ACC-07 | H3 | Se desconecta NAS o pierde permisos durante recorrido: raíz inaccesible, no miles de eliminaciones; reanuda y reconcilia |
| ACC-08 | H3 | Crear subdirectorios después del escaneo; perder eventos/overflow/reiniciar: reconciliación detecta cambios sin duplicados |
| ACC-09 | H3 | Borrar PDF linked: buscar/leer texto retenido; original/descarga deshabilitados; reaparición reconocida no duplica |
| ACC-10 | H3/H5 | Modificar linked aprobado: un aviso por cambio, sin revocar aprobación; A→B→A genera dos observaciones distintas |
| ACC-11 | H3 | Búsqueda registra texto exacto, tipo, filtros, ámbitos efectivos, usuario/sesión y total; auditor sin permiso sensible no ve consulta |
| ACC-12 | H3 | Copias con mismo SHA-256 se informan separadas; renombrado probado conserva identidad; coincidencia débil no fusiona |
| ACC-13 | H4 | Convertir a híbrida y añadir raíz durante cargas: UI/cola siguen disponibles, IDs/OCR permanecen |
| ACC-14 | H4 | Temporal no aparece en JSON/HTML/logs/errores ni es accesible por usuario ajeno; preview autorizado sí |
| ACC-15 | H4 | Asociación linked a expediente sin copiar/mover/reextraer; segunda asociación falla; reasignación autorizada queda auditada |
| ACC-16 | H4 | Plantilla nueva no modifica expedientes existentes; inicialización repetida no duplica; conteos derivados y sin obligatorios → porcentaje null |
| ACC-17 | H5 | Aprobar entre discos: fallos antes/después de publicar/commit/limpieza; nunca aprobación con destino faltante o temporal borrado prematuramente |
| ACC-18 | H5 | Managed editado externamente aprobado pasa a needs_review; no cuenta válido; escritura propia no genera falsa alerta |
| ACC-19 | H5 | Cambio de patrón/categoría/nombre no mueve históricos; colisión/reservados/longitud generan nombres seguros y estables |
| ACC-20 | H6 | Firma/alg/kid/producto/equipo/huella/revisión incorrectos fallan; fixture de desarrollo no es aceptado en producción |
| ACC-21 | H6 | subscription: un instante antes/en expires_at y antes/en +15 días; offline no extiende gracia; perpetua funciona sin consultas periódicas |
| ACC-22 | H6 | Activación/reintentos idempotentes, duplicada, refresh revocado, `.licreq/.lic`, renovación offline y módulos con vectores del otro proyecto |
| ACC-23 | H6 | Vencida/revocada permite consulta/descarga/backup autorizado y bloquea escrituras; perder módulos nunca elimina datos |
| ACC-24 | H7 | `.deb` online y bundle offline se instalan/actualizan conservando rutas/DB/licencia/PDF; no depende de que dpkg descargue dependencias |
| ACC-25 | H7 | Windows/macOS: servicio sin consola, ACL, OCR español, upgrades y desinstalación conservadora probados nativamente |
| ACC-26 | H7 | Backup/restore con WAL y archivos en corte consistente; detectar hash incorrecto/faltantes y recuperar cola sin duplicar |
| ACC-27 | H3/H7 | Symlink/junction/UNC, case sensitivity, hard links, traversal y cambio de enlace entre validación/apertura no escapan raíces |
| ACC-28 | H3/H7 | Corpus grande con carga/búsqueda/OCR simultáneos: medir p50/p95, CPU/RAM/disco, queue lag y cancelación; fijar capacidad después de medir |

## Contrato de licencias: evidencia mínima compartida

Crear en H6 vectores de desarrollo versionados e idénticos en ambos repositorios:
payload exacto, bytes de proof, JWS Compact conocido, `.licreq`/`.lic`, tiempos UTC,
firmas corruptas, kid desconocido, installation/public-key/fingerprint ajenos,
revoked, perpetua sin mantenimiento, cambio de features y revisión menor/igual/mayor.
Además del mock HTTP, correr los mismos vectores contra el otro proyecto; un mock
que replica nuestras suposiciones no prueba compatibilidad real. Nunca guardar
claves privadas de producción ni claves comerciales reales en fixtures.

## Cierre de cada módulo

Incluir lista real de archivos, migración/checksum/reversión segura, comandos de
arranque/prueba con versiones fijadas, resultados y limitaciones. Revisar UI y
documentación del comportamiento entregado, no solo compilación. No correr todos
los tests repetidamente si no hay cambios/fallos; ampliar cuando lo justifique el
riesgo. Las pruebas H1 solo verifican el diseño SQL, no cubren ACC del backend.
