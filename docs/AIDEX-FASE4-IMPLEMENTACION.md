# AIDEX 1.0 — Fase 4: licenciamiento comercial del cliente

Fecha: 22 de octubre de 2026. Referencias técnicas conservadas en la raíz: `INTEGRACION-LICENCIAMIENTO.md` y `LICENSE_CONTRACT-SERVER.md`. Este informe distingue lo comprobado en el servidor real de las pruebas con artefactos sintéticos. No contiene códigos comerciales, JWS ni claves privadas.

## A. Implementación

- El cliente acepta el código comercial 3-4-4-4 del alfabeto contractual, sin prefijo anterior. Convierte minúsculas a mayúsculas, admite pegado compacto o agrupado y rechaza caracteres ambiguos sin sustituirlos. El backend normaliza antes del desafío y conserva la idempotencia de las solicitudes.
- Se mantienen el desafío HTTPS, la prueba de posesión Ed25519, el identificador técnico de producto del contrato, el JWS, la vinculación a la instalación, los módulos, las revisiones y el estado persistente. La limitación HTTP 429 tiene una pausa local de 15 minutos; una interrupción de red no altera la licencia firmada.
- La solicitud `.licreq` sigue firmada con la identidad de instalación. La importación `.lic` comprueba el JWS con las claves públicas confiables configuradas, producto, instalación, clave, huella, modalidad, estado y revisión. Una licencia perpetua válida no exige conexión periódica para conservar el uso; las suscripciones mantienen la tolerancia contractual.
- La desactivación voluntaria confirmada y la solicitud de desactivación offline bloquean lectura y escritura documentales. Una licencia vencida o revocada con evidencia firmada conserva solo los derechos de consulta y exportación que exige `LICENSE_CONTRACT-SERVER.md`; no se confunde con una desactivación voluntaria.

## B. Seguridad

- El backend comprueba aceptación del CLUF y derecho de licencia en rutas autenticadas, incluidas bibliotecas, búsquedas, expedientes, documentos, vistas previas, descargas y restauración. Solo se exceptúan autenticación básica, licencia y CLUF. Las operaciones automáticas comprueban licencia y módulos antes de procesar trabajo.
- Se rechaza importar un JWS anterior de una activación desactivada; también se rechazan revisiones inferiores, la firma alterada, un KID no confiable, producto equivocado y vinculación a otra instalación. El firmante de producción permanece fijado por configuración confiable: `lic-prod-2026-a`; no se obtiene del artefacto importado.
- La semilla privada de la instalación se conserva fuera de los respaldos en un archivo privado, con permisos `0600` y protección de ruta/ACL del sistema. La identidad se compara con su registro en la base local. Esta es protección de software: una persona con control suficiente del sistema y una copia completa de identidad **y** base de datos podría reproducir o retroceder el estado local. No hay atestación de hardware ni garantía de clave Ed25519 no exportable en las cuatro plataformas. La revocación de un equipo permanentemente aislado solo se conoce al recibir evidencia autenticada.
- Los archivos `.aidexbak` mantienen su formato y su lista cerrada de tablas. No incluyen identidad privada, licencia, artefactos de licencia ni aceptación del CLUF. Restaurar datos en otro equipo requiere la licencia propia de ese equipo.

## C. Interfaz

- Campo accesible de un solo renglón con grupos automáticos 3-4-4-4, edición del cursor, mensaje de formato y botón deshabilitado hasta tener un código válido. No se escribe el código en logs ni diagnósticos. En caso de fallo de red se conserva el valor introducido para reintentar la misma operación.
- La pantalla de licencia conserva las opciones online y offline, el estado, modalidad, módulos y archivos de solicitud/respuesta. La interfaz explica la desactivación y muestra los módulos suspendidos cuando el acceso queda bloqueado. La pantalla principal remite al administrador cuando no hay acceso.
- Un usuario sin permiso de administración puede consultar el CLUF, pero la interfaz le indica que un administrador debe aceptarlo.

## D. Contrato de licencia

- Se añadió un borrador separado, marcado **pendiente de revisión jurídica**, porque no se encontró un CLUF definitivo aprobado. La aceptación explícita requiere casilla desmarcada por defecto, versión y SHA-256 exactos. La migración `0016_eula` conserva instalación, usuario, versión, huella y fecha, además del evento de auditoría.
- El CLUF puede consultarse desde «Acerca de AIDEX · Contrato de licencia», incluso sin activación. Si cambia la versión o el texto, la instalación deberá aceptarlo de nuevo sin perder los registros anteriores.
- **Antes de distribución comercial:** sustituir el borrador por el texto aprobado, fijar su versión y decidir cómo presentarlo también en cada instalador. No se generaron ni modificaron paquetes de instalación en esta fase.

## E. Respaldos y restauración

- La migración de aceptación no altera el esquema portable `.aidexbak`. El servicio de respaldo acepta la migración de base local 16 manteniendo el manifiesto portable anterior. Las rutas de restauración siguen sujetas al control de licencia del backend. Una licencia perpetua offline válida puede restaurar sin contacto con el servidor.
- Las regresiones de respaldo/restauración y de identidad se ejecutaron dentro de las suites Go de producción y desarrollo. No se modificaron documentos físicos del propietario.

## F. Pruebas y evidencia

| Comprobación | Resultado |
| --- | --- |
| Suite Go completa, compilación de producción (`go test ./...`) | Pasó |
| Suite Go completa con etiqueta `development` | Pasó |
| Pruebas enfocadas de HTTP/licencia tras el último cambio | Pasaron |
| Compilación web y TypeScript | Pasó; Vite advierte un bloque JS mayor a 500 KiB, sin error de compilación |
| Playwright de licencia con simulador: captura, edición, operaciones y derechos | Pasó |
| Compilación cruzada del servidor: Windows AMD64, Ubuntu AMD64, Ubuntu ARM64, macOS ARM64 | Pasó; no equivale a ejecución nativa |
| HTTPS y desafío del servidor de producción | Pasó; TLS validado |
| Activación comercial real en instalación temporal | Pasó; licencia perpetua, cinco módulos, JWS Ed25519 y vinculación verificados |
| Persistencia después de recrear el cliente y renovación real | Pasó |
| Desactivación real de **esa instalación temporal** y bloqueo local | Pasó; el servidor confirmó `deactivated`, condición que libera la plaza según el contrato. No se consultó por separado un contador de plazas |
| Reimportación del JWS anterior después de desactivar | Rechazada |
| JWS manipulado, firma/KID/producto/equipo/revisión incorrectos; offline y errores de red | Cubiertos mediante pruebas sintéticas; no se manipuló la licencia comercial real |

La instalación temporal y su archivo privado se eliminaron **después** de la desactivación confirmada. La primera ejecución del ensayo se detuvo antes de contactar al servidor porque el directorio temporal de macOS atravesaba un enlace simbólico; se repitió con `/private/tmp`, aceptado por la protección de rutas. Esa primera ejecución no ocupó plaza.

## G. Cambios técnicos

- Backend: `internal/licensing` (normalización, estado, desactivación y tratamiento de 429); `internal/httpapi` (control central, CLUF, sesión); `internal/backups/service.go` (migración local 16); `db/migrations/0016_eula.*.sql`.
- Web: `LicenseKeyField.tsx`, `License.tsx`, `Eula.tsx`, `App.tsx`, tipos de sesión y estilos. Los fixtures de desarrollo usan únicamente códigos sintéticos con formato contractual.
- API añadida: `GET /api/v1/eula` y `POST /api/v1/eula/accept`. Las rutas documentales existentes ahora comprueban licencia y aceptación después de autenticar y validar CSRF. No se añadió una dependencia de ejecución ni se cambió la versión del producto.

## H. Pendientes antes de distribución

1. Aprobar y colocar el CLUF definitivo; probar su aceptación en instaladores y la primera ejecución en Windows, Ubuntu y macOS. El borrador actual es infraestructura técnica, no un contrato comercial final.
2. Ejecutar ensayos nativos en Windows AMD64, Ubuntu AMD64/ARM64 y macOS ARM64. La compilación cruzada no valida permisos del servicio, selección de archivos, reloj ni interacción con el instalador.
3. Si se requiere resistencia contra administradores del sistema o clonaciones completas, definir un ancla de confianza propia de cada plataforma/hardware y un procedimiento de recuperación. Los permisos de archivo y el ancla en la base local no pueden garantizar esa propiedad. No se añadió un contacto online obligatorio a las licencias perpetuas.
4. Para el manual: documentar captura 3-4-4-4, solicitud/importación offline, permisos para aceptar CLUF, efectos de desactivación, derechos de solo lectura de suscripciones vencidas/revocadas y respaldo antes de trasladar equipos.

No se generaron instaladores, no se modificó el servidor de licencias y no se alteraron documentos reales.
