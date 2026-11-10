# AIDEX — Informe técnico de producción

Fecha inicial: 2026-10-09. Licenciamiento actualizado: 2026-10-10. Primera edición comercial AIDEX 1.0; versión técnica 1.0.0. Este informe documenta la transición y sus límites. No sustituye los futuros manuales de instalación, usuario y técnico.

## A. Identidad

Aplicación de Indexación Digital de Expedientes. Se actualizan textos de interfaz, título/metadatos del navegador, nombre del frontend, mensajes/diagnósticos, namespaces temporales, configuración de usuario, backend, launcher, scripts, servicios, cuentas, firewall, recursos, herramientas de construcción, tareas del editor y CI. El launcher está en cmd/aidex-launcher; los recursos Windows se llaman aidex.ico y aidex-icon.svg; las herramientas Ubuntu aidex-admin, aidex-uninstall y aidex.service. Se retiran de los instaladores los selectores de perfiles, registros y marcadores históricos y drop-ins de transición.

Los cuatro SVG proporcionados por el propietario se conservan sin editar. Se verifica la geometría común de la carpeta; ICO y recursos .syso conservan el mismo icono sin modificaciones de colores/proporciones. El favicon SVG deriva de esa misma figura. No había manifest PWA ni aplicación macOS .app/ICNS que renombrar. Los recursos compilados anteriores se sustituyen al reconstruir el frontend. Un DOCX sintético de diagnóstico contenía una marca anterior dentro de XML comprimido: se actualiza únicamente esa muestra, conservando los documentos originales del usuario.

Se mantienen el módulo técnico Go gestor-documental y product_id gestor_documental, así como los campos, endpoints, firmas y módulos del contrato V1. No cambia el esquema de datos ni la lógica documental. La nueva instalación no importa estado de otros productos. Las salidas históricas build/dist se conservan fuera de la entrega oficial; incluyen documentos originales de prueba del propietario y no se limpian indiscriminadamente.

Archivos clave: internal/licensing/provider.go; internal/config/config.go; internal/buildinfo/service.go; internal/launcher/target.go; packaging/windows/manage.ps1, installer-action.ps1 e installer.nsi; packaging/ubuntu y packaging/macos; scripts/build_installers.py, check_production_packages.py y audit_identity.py; web/src/Brand.tsx, web/index.html y assets/brand.

## B. Instaladores y servicios

| Plataforma | Programa/servicio/cuenta | Configuración y estado |
|---|---|---|
| Windows x64 | C:\Program Files\AIDEX\aidex-server.exe; launcher AIDEX.exe; servicio AIDEX; NT SERVICE\AIDEX | C:\ProgramData\AIDEX\config.json; subcarpeta state; eventos Aplicación/AIDEX |
| Ubuntu amd64/arm64 | /usr/lib/aidex/aidex-server; aidex.service; usuario/grupo aidex | /etc/aidex/config.json; /var/lib/aidex/state; journal de aidex.service |
| macOS Apple Silicon | /Library/Application Support/AIDEX/bin/aidex-server; app.aidex; usuario/grupo _aidex | /Library/Application Support/AIDEX/data/config.json; data/state y data/logs/service.log |

Windows registra su propia ficha de desinstalación y accesos de Escritorio/Inicio. La ACL privada admite SYSTEM, Administradores y el SID de AIDEX; el backend no es administrador. La regla AIDEX-LAN-TCP, grupo AIDEX, se limita a binario/servicio, puerto TCP configurado, LocalSubnet y perfiles Domain/Private, sin edge traversal. El modo local no escucha en LAN. Ubuntu/macOS conservan su diagnóstico de firewall y no modifican reglas externas. La configuración local/LAN, verificación y restauración existentes se preservan.

Unix conserva datos 0700 y archivos sensibles 0600, cuentas sin login y servicios propios. El plist macOS es root:wheel 0644; paquete/LaunchDaemon app.aidex, aplicaciones bajo /Applications/AIDEX. PowerShell 5.1 mantiene BOM único UTF-8, lectura JSON UTF-8, quoting CRT y transporte NSIS UTF-16LE; Unix usa argumentos entre comillas y UTF-8 sin BOM. Se mantienen verificaciones de enlaces, dueño/cuenta, lock, timeout y eliminación por hash. No se eliminan documentos originales ni se cambian ACL de bibliotecas externas.

Procedimientos/comandos exactos: [Instalación y mantenimiento](PRODUCCION.md). Para retirar instalaciones personales previas conservando datos: [Retiro manual](RETIRAR-INSTALACION-ANTERIOR.md). La cuenta nueva requiere sus propios permisos de acceso local/SMB; los permisos anteriores no se transfieren.

## C. Licenciamiento

Servidor definitivo https://licensing.aidex.com.mx. La configuración se inicializa en config.json de cada plataforma, bloque license, mediante el mecanismo existente. Confianza única predeterminada: kid lic-prod-2026-a, environment production, purpose license. La clave pública proporcionada se conserva exactamente y decodifica como Ed25519 de 32 bytes Base64URL. refresh_hours 24 y development_bypass false. El binario oficial no se compila con development; ese modo se conserva para uso expreso de desarrollo. No se generan claves alternativas ni se aceptan certificados inválidos.

Se conserva product_id gestor_documental; app_version 1.0.0. No se modifica el servidor de licencias ni la lógica de seguridad del cliente. El 2026-10-10, con el firmante activado por el propietario, **el ciclo real de licenciamiento pasó**.

Resultados diferenciados:

- Activación HTTPS HTTP 200, JWS EdDSA verificado con Ed25519 y KID lic-prod-2026-a, vínculo de instalación y producto correctos. Licencia perpetua, sin vencimiento de uso, con los cinco módulos contratados.
- Renovación HTTP 200, validación/persistencia del artefacto firmado y contacto confirmado. La revisión inicial fue 1; la renovación devolvió la revisión vigente. No se cambió el contrato comercial en el servidor.
- Persistencia al cerrar/reabrir ambos pools SQLite y recrear el cliente: identidad, activación, revisión y módulos conservados. Validación/importación offline con origen HTTPS inaccesible y prueba criptográfica de solicitud de renovación offline aprobadas.
- Desactivación HTTP 200, modo solo lectura persistente, replay idempotente y rechazo de la licencia de la activación retirada. Nueva activación HTTP 200 con la misma identidad/licencia, activación distinta y revisión 3: la instalación previamente desactivada no impidió activar nuevamente. Renovación y desactivación final HTTP 200. Dos activaciones retiradas, cero operaciones pendientes y plaza temporal liberada.
- Rechazos aprobados: clave inválida HTTP 404 LICENSE_NOT_FOUND; firma y payload manipulados, KID desconocido, artefacto malformado, vínculo de otra instalación y artefacto de una activación anterior. Ningún rechazo sustituyó el estado confiable.
- Producción y development: las suites locales de licenciamiento pasan. Suscripción, tolerancia, reloj, indisponibilidad, TLS, respuestas inválidas, idempotencia y contrato offline siguen cubiertos por sus pruebas locales; el ciclo comercial real utilizó la licencia perpetua proporcionada. No se esperaron 24 horas ni se solicitó cumplimiento offline al servidor.

La prueba histórica HTTP 503 se conserva como evidencia, ahora resuelta tras la activación del firmante. El desafío guardado había vencido y el servidor respondió HTTP 401 INVALID_PROOF; únicamente después de ese rechazo definitivo se generó una nueva solicitud, conservando la misma instalación. No hubo nuevos HTTP 503 durante el ciclo confirmado.

La clave de producto se leyó desde un archivo temporal 0600 y la copia se eliminó incluso en fallos. No se incorporó al código, argumentos, informes, logs ni paquetes; el journal y los artefactos tampoco guardan la clave. La identidad y el journal privados en /private/tmp/aidex-live-license-state conservan evidencia de las dos desactivaciones confirmadas. No se implementó bypass, fallback de confianza ni modificación del servidor. Los cambios de esta revisión se limitan a la prueba opt-in y evidencia: los cuatro instaladores conservan sus hashes.

Detalle y límites: [Informe de licenciamiento](AIDEX-LICENCIAMIENTO.md). Evidencia: validacion/licensing-production-cycle.log, las dos suites de regresión y licensing-production-state.json de la entrega.

## D. Compilación y distribución

Cuatro paquetes nuevos, versión 1.0.0, en dist/aidex-1.0.0. Las dependencias/toolchains previamente verificadas se reutilizaron como herramientas, nunca los instaladores anteriores. Fue necesario reempaquetar después de detectar el DOCX sintético comprimido; la última revisión pasó sobre los hashes siguientes.

| Paquete | Bytes | SHA256 |
|---|---:|---|
| AIDEX-1.0.0-Windows-amd64.exe | 40202923 | `a574fcd3dbe31156fa1dfbaeafe7a121fe9339ae80b718c643492a9407d09c20` |
| AIDEX-1.0.0-macOS-arm64.pkg | 5979040 | `eaec05915410128ec9c4b2950ff15ca429cf9ac3b34f377327786a0bc63f8cf5` |
| aidex_1.0.0_amd64.deb | 6357504 | `679d9dfa6bf3f3cc0ad0426dabef8d90de06baed919b8a53a4a19040afb3f776` |
| aidex_1.0.0_arm64.deb | 5875366 | `c3807ae43ab63618ceec2330f0e411cf8be0730e09c7366ed19b6c0541744316` |

Evidencia: [VALIDACION.json](../dist/aidex-1.0.0/VALIDACION.json), SHA256SUMS y validacion/ de la entrega.

Pruebas completadas:

- TypeScript/Vite, Go production y development; gofmt y go vet. Dos grupos development inicialmente fallaron por PATH sin Poppler/Tesseract; se repitieron únicamente esos grupos con las herramientas instaladas y pasaron.
- 22 pruebas Chrome: autenticación/sesiones, usuarios/permisos, bibliotecas/expedientes/cargas, búsqueda, extracción Office/OCR, procesamiento/watchers, duplicados, reindexación, previews/cache, auditoría/observabilidad, ajustes, red y licencias simuladas.
- 11 pruebas de templates/perfiles/administración y 14 comprobaciones de diseño/contrato/documentación activa.
- PowerShell 7.6.6 en macOS: parser, perfiles de servicio, rechazo de cuenta/ruta ajena, UTF-8 JSON, quoting, mensajes/rutas Unicode, salida UTF-16LE y código de salida. Firewall NetSecurity con mocks: provisión, ámbito, propiedad, idempotencia y eliminación. No certifica PowerShell 5.1/SCM/firewall reales.
- Cuatro paquetes: nombres y contenido completo, XML de Office descomprimido, metadatos, servicios/scripts, control Debian/arquitecturas ELF, PE/íconos ocho tamaños, closure DLL, manifiestos SHA, confianza del backend, documentación y firma ad hoc del Mach-O.
- macOS nativo desde PKG extraído con datos ficticios: PDF/OCR español, bootstrap seis caracteres, HTTP/login, local→LAN IPv4 y restauración local, persistencia al reiniciar, locks, integridad SQLite, permisos y borrado interno confirmado conservando PDF original. No instala launchd ni modifica la instalación del propietario.

Pendientes: ciclos nativos de instalación/actualización/desinstalación y cuenta del servicio en Windows/Ubuntu/macOS; comprobación visual PowerShell 5.1/NSIS y rutas Unicode; LAN/GPO/SMB y APT offline; respaldo/restauración integral; firma comercial Windows, firma/notarización macOS y avisos/fuentes correspondientes de terceros antes de publicar.

## E. Auditoría final

Auditoría inicial: **1,470 apariciones de contenido en 139 archivos**, incluyendo XML de DOCX sintético y ZIP de inventarios. Se excluyen dependencias/cachés y copias de compilaciones históricas para no mezclar código del producto con proveedores ni contar cada paquete como fuente. Se inspeccionan por separado los paquetes nuevos y sus proveedores.

**735 referencias originales corregidas**: 691 en componentes activos, 43 en recursos compilados y 1 en la muestra DOCX comprimida. **735 originales se conservan**: 235 en informes históricos y 500 en los inventarios y su ZIP. El documento nuevo de retiro manual añade **24 menciones deliberadas** a los identificadores anteriores, necesarias para los comandos de desinstalación personal. Total actual de contenido: **759**, todas históricas o de retiro; **0 en código/recursos activos**.

Nombres/rutas originales auditados: **17**; **8 renombrados**, **9 conservados** como nombres de informes históricos. Cero rutas antiguas en componentes activos o nuevos paquetes. Los siete inventarios, ocho imágenes y ZIP permanecen idénticos por SHA256. No se genera Word/PDF ni se reescriben inventarios. Sus enlaces históricos representan la implementación anterior, no instrucciones actuales.

En los binarios externos de iconv/ICU se localizaron **11 coincidencias incidentales de bytes**: tablas numéricas Unicode y traducciones gaélicas de «alfabeto». No son referencias comerciales. Se conservan los DLL originales fijados por SHA256; el inspector valida exactamente hashes y posiciones para distinguir estos datos de una referencia nueva. Cero referencias comerciales anteriores en los cuatro paquetes. No hay referencia obsoleta pendiente en los componentes activos. Los artefactos previos permanecen separados como historia y no deben distribuirse como esta edición.

## F. Estado final

**Licenciamiento real aprobado con la licencia perpetua proporcionada.** Identidad, configuración y cuatro paquetes completos, con verificaciones locales y de contenido aprobadas. El ciclo real de activación, firma, renovación, persistencia, desactivación y nueva activación está confirmado y la plaza temporal quedó liberada. La distribución comercial completa continúa condicionada a los requisitos nativos/externos indicados; esos pendientes son distintos del ciclo de licenciamiento ya aprobado.
