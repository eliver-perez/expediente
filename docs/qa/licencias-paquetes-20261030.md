# AIDEX 1.0.0 — Revisión de licencias y cuatro instaladores

Fecha: 30 de octubre de 2026. Versión comercial conservada: 1.0.0.

## Diagnóstico

La aclaración del usuario confirma que el segundo registro se efectuó mediante una solicitud offline, aprobada en el servidor, y la importación de su respuesta. No fue una transferencia online iniciada desde el cliente.

El valor predeterminado del cliente de producción es **24 horas desde el último contacto confirmado**. `Client.Run`, iniciado por el servicio en todas las plataformas, comprueba cada minuto si corresponde contactar; ese minuto no es el intervalo de solicitudes al servidor. El primer ciclo también se ejecuta al iniciar el servicio. Tras un contacto fallido se programa otro intento a los **15 minutos**, respetando además las restricciones del servidor. No depende de abrir la pantalla de licencia ni de pulsar un botón.

Por tanto, que la instalación anterior continúe funcionando después de dos horas es compatible con la periodicidad existente. El archivo firmado que ya posee no se modifica a distancia. Cuando la instalación contacte y reciba la revisión terminal firmada, suspende las modificaciones sin aplicar una tolerancia adicional. Una perpetua continúa funcionando durante una desconexión real; una falla de red no constituye una revocación.

Se mantiene el intervalo diario: es coherente con el modelo perpetuo offline aprobado y evita comprobaciones frecuentes. El intercambio es explícito: una sustitución no garantiza bloquear instantáneamente otro equipo, aunque tenga conexión, sino cuando corresponda su siguiente validación y esta pueda completarse. No se redujo arbitrariamente el intervalo ni se añadieron botones.

No se inspeccionó el archivo de configuración, los registros ni el último contacto de la computadora Windows del usuario. Las configuraciones existentes se conservan y pueden tener otro valor, incluido `refresh_hours=0`, que deshabilita la revisión automática. Si la instalación concreta sigue activa después de su próxima validación prevista, debe comprobarse ese valor y el resultado de su contacto; la observación de dos horas por sí sola no demuestra un fallo.

## Revisión del servidor y del cliente

Se revisó el código disponible en `/Applications/XAMPP/xamppfiles/htdocs/aidexlicense`, sin modificarlo, desplegarlo ni utilizar su base comercial. Las verificaciones dinámicas utilizaron una copia temporal del código, MySQL 8.4.11 aislado sin puerto TCP, Nginx/PHP-FPM/TLS locales, claves aleatorias y licencias exclusivamente sintéticas.

- La aprobación offline que sustituye una instalación bloquea y actualiza los registros dentro de una transacción; publica una revisión terminal firmada para la activación anterior y una activa para la nueva.
- La renovación autenticada de la instalación anterior devuelve su revisión terminal firmada. No se limita a devolver un error HTTP de revocación que el cliente pudiera interpretar como una interrupción temporal.
- El cliente verifica firma Ed25519, KID, identidad vinculada, revisión y persistencia antes de aplicar el resultado. Los errores HTTP sin un estado firmado válido no reemplazan la licencia auténtica.
- Una revocación firmada persiste tras reabrir el cliente y bloquea escritura y módulos sujetos a licencia. Conserva consulta y exportación conforme a la política existente. No borra documentos ni índices.
- La programación, la interpretación y la persistencia se implementan en código Go compartido. Se comprobó el arranque del verificador desde el servicio; no se ejecutó Windows nativamente.
- Las solicitudes pendientes conservan su identificador para reintentos idempotentes después de una interrupción o reinicio.

No se encontró un defecto funcional reproducible que requiriera cambiar el cliente o el servidor en esta revisión. **No hubo cambios en el servidor de licencias**, en datos comerciales, periodicidad ni protecciones criptográficas. Se añadieron pruebas para cubrir directamente el comportamiento reportado.

## Pruebas efectivamente ejecutadas

| Comprobación | Resultado y alcance |
| --- | --- |
| Go: licenciamiento, bibliotecas, API y auxiliar macOS | PASS con `go test -tags development ./internal/licensing ./internal/libraries ./internal/httpapi ./internal/macosfolders ./cmd/macos-folder-helper -count=1 -timeout=180s`. |
| Configuración y conservación del proveedor | PASS con `go test ./internal/config -count=1 -timeout=45s`. |
| Nuevas pruebas de revisión automática | PASS, 61.322 s: no revisar a las dos horas, vencimiento de 24 horas, revocación firmada, persistencia, desconexión, firma manipulada, recuperación y reintento de 15 minutos. |
| Servicio continuamente encendido | PASS: se dejó `Run` activo y se esperó el tick real de un minuto después de adelantar únicamente el reloj de prueba hasta la revisión prevista. No se reinició el bucle ni se llamó manualmente a `Online`. |
| Servidor: pruebas unitarias | PASS: 57 pruebas, 20,116 aserciones. |
| Servidor: activaciones y operaciones offline con MySQL 8 real aislado | PASS: 39 pruebas, 344 aserciones, incluidas respuestas terminales e idempotencia. |
| Cliente Go real ↔ servidor PHP real por HTTPS local | PASS: 1 prueba, 28 aserciones. Activación online, sustitución offline, una sola activación activa en el servidor, revocación del origen al contactar, persistencia, fallo y recuperación de conexión, revocación comercial sintética y conservación del documento centinela. Cada invocación reabre identidad y base persistidas. |
| Frontend | PASS: TypeScript y compilación Vite. Advertencia existente sobre tamaño del bundle, sin impedir la compilación. |
| Playwright | PASS: 5 pruebas; licencia HTTPS/offline/recuperación, incorporación y corrección/sustitución conservando versiones, flujo del colaborador y destino administrado/autorización macOS. |
| Scripts macOS | PASS: 33 pruebas, 63.738 s. |
| Ciclo usando scripts y backend extraídos del PKG entregado | PASS: instalación limpia, actualización directa, conservar/reinstalar con administrador anterior, dos ciclos de eliminación interna/reinstalación, recuperación de residuos antiguos, rechazo seguro de datos recuperables sin configuración, documentos externos e incorporados conservados. HTTP y login reales sobre puertos temporales; cuentas, launchd y recibo simulados. |
| Cuatro paquetes | PASS: arquitectura, versión, identidad de producción, proveedor/KID/clave pública, scripts, recursos oficiales, manifiestos SHA-256 y ausencia de configuración/base de usuario precargada. Windows: sintaxis PowerShell, BOM, iconos y cierre de dependencias PE/amd64. macOS: cuatro ProgramArguments exactos, aplicaciones nativas, auxiliar ARM64 y firmas ad hoc válidas. Ubuntu: ELF y metadatos/dependencias de cada arquitectura. |
| Espacios y formato del diff | PASS: `git diff --check`. |

Los primeros intentos de las nuevas pruebas detectaron dos defectos de la fixture, no del producto: una comparación textual de fechas con formatos distintos y un servidor simulado que no conservaba respuestas idempotentes al reintentar una firma dañada. Se corrigieron las fixtures y la ejecución final pasó. No se modificó el comportamiento de producción para satisfacer una simulación incorrecta.

Evidencia de integración y ciclos en `evidencias/licencias-paquetes-20261030/`: `servidor-integracion.json`, `servidor-integracion.txt`, `cliente-servidor-real.txt`, `macos-scripts.txt`, `macos-ciclo.txt` y `paquetes.json`.

Pruebas añadidas: `internal/licensing/automatic_validation_test.go` y `scripts/test_license_revocation.php`. La segunda se ejecutó copiándola al directorio `tests/System` de la copia aislada del servidor, con sus fixtures y `TEST_CLIENT_SOURCE` apuntando a este proyecto. No se incorporó al servidor desplegado.

## Instaladores generados desde el código actual

Directorio absoluto: `/Applications/XAMPP/xamppfiles/htdocs/expediente/dist/aidex-1.0.0-validacion-licencias-20261030/`.

Se recompilaron backend y frontend actuales, y el auxiliar nativo macOS. Se utilizaron los scripts existentes con `--target all`. No se reutilizó un instalador antiguo como entregable. El runtime externo de OCR de Windows se tomó del almacén local verificado: 1,092 archivos contrastados contra el manifiesto fijado del proveedor; los ejecutables y recursos propios de AIDEX son los recién generados.

| Archivo | Tamaño (bytes) | SHA-256 |
| --- | ---: | --- |
| `AIDEX-1.0.0-Windows-amd64.exe` | 40,534,736 | `af762f536bd76208236bfc83902aa8e07a7c55c5a1fc51b1335a51d16328a1db` |
| `AIDEX-1.0.0-macOS-arm64.pkg` | 10,109,355 | `04458c0e2247cb14d6f10d91629a2c1e6014f696686cb25103c7edce475960a3` |
| `aidex_1.0.0_amd64.deb` | 6,655,614 | `acec4c7ad837289f103102360e0a0fdbed7441d98f881cfac7abda854be68af9` |
| `aidex_1.0.0_arm64.deb` | 6,145,664 | `a08b2857b617e644e07d3f99cc961d7047a4a8e6d8c36e57d21813af105305c0` |

Hashes recalculados y contrastados con `SHA256SUMS`. No se empaquetaron claves privadas ni códigos comerciales de prueba. Los cambios funcionales compartidos y las últimas correcciones del auxiliar de carpetas están incluidos. No se modificaron manuales, instalaciones existentes ni documentos del usuario.

## Límites y validación pendiente

1. No se efectuaron operaciones sobre licencias reales ni se verificó una sustitución en el servidor desplegado. La prueba real de protocolo utiliza una copia aislada del código disponible; no demuestra que el despliegue remoto sea idéntico.
2. No se realizaron instalaciones nativas mediante el asistente de Windows, Installer.app o apt/systemd en Ubuntu. La inspección de un paquete y la fixture de ciclo macOS no equivalen a esas instalaciones.
3. La autorización gráfica macOS completa con elevación, servicio real `_aidex`, incorporación y lectura desde Finder sigue pendiente de la validación nativa descrita en `macos-permisos-20261030.md`. El PKG incorpora la implementación más reciente; esta revisión no la presenta como una incidencia nativa completamente cerrada.
4. Las firmas macOS verificadas son ad hoc; no hay firma Developer ID ni notarización. No se publicó una versión oficial ni se creó una etiqueta.

El comportamiento de revocación del código revisado pasó las pruebas, manteniendo el uso perpetuo offline. Para concluir sobre las dos instalaciones concretas del usuario falta observar el contacto automático previsto en Windows y su resultado autenticado; no corresponde declarar una avería ni una reparación de ese equipo basándose únicamente en dos horas de coexistencia.
