# AIDEX — Validación final de licenciamiento

Fecha: 2026-10-10. Cliente de producción 1.0.0, servidor https://licensing.aidex.com.mx, confianza exclusiva lic-prod-2026-a / production / license. Ciclo expresamente autorizado con la licencia de prueba proporcionada, sin modificar el servidor ni debilitar ninguna validación.

| Prueba | Resultado | Evidencia |
|---|---|---|
| Activación online | PASS | HTTP 200; licencia perpetua, cinco módulos, revisión 1 |
| Ed25519/KID y vínculo | PASS | EdDSA, lic-prod-2026-a, producto e identidad verificados con la clave pública autorizada |
| Persistencia y uso offline | PASS | Cierre y reapertura SQLite, identidad/estado/artefacto/módulos conservados; validación e importación sin servidor disponible |
| Renovación/actualización | PASS | HTTP 200; artefacto vigente verificado y guardado, contacto actualizado, revisión no decreciente |
| Idempotencia | PASS | Replay de activación, renovación y desactivación sin duplicar la operación |
| Desactivación/liberación | PASS | HTTP 200; solo lectura tras reiniciar; artefacto retirado rechazado |
| Nueva activación | PASS | Misma instalación/licencia, nueva activación, revisión 3; firma válida y renovación posterior aprobada |
| Clave inválida | PASS | HTTP 404 LICENSE_NOT_FOUND; no concede permisos |
| Manipulación y confianza | PASS | Firma/payload manipulados, KID desconocido, formato inválido, vínculo ajeno y activaciones retiradas rechazados; estado confiable intacto |
| Liberación final | PASS | HTTP 200; estado final desactivado, dos activaciones retiradas y cero operaciones pendientes |
| Confidencialidad | PASS | Clave ausente de journal, artefactos, logs y entrega; copia temporal eliminada |
| Regresión local | PASS | Suites de licenciamiento production y development |

No se encontraron fallos del cliente ni del servidor durante el ciclo confirmado. No fue necesario cambiar lógica de ejecución, API, claves de confianza ni regenerar instaladores. Se amplió la prueba explícitamente opt-in para cubrir este ciclo y sus rechazos.

El antiguo HTTP 503 quedó resuelto después de que el propietario activara el firmante. La solicitud conservada recibió HTTP 401 INVALID_PROOF porque su desafío había vencido; tras el rechazo definitivo se creó una nueva solicitud usando la misma identidad. Ese rechazo es el comportamiento esperado, no un fallo de seguridad ni una activación duplicada. El primer intento de esta sesión no tenía acceso de red desde el sandbox; la ejecución autorizada con acceso de red pasó.

La licencia real probada es perpetua, con todos los módulos. Renovar consulta y persiste el estado firmado actual; no se modificó la licencia comercial ni se extendió una suscripción. Expiración/tolerancia de suscripciones, reloj e indisponibilidad están cubiertos localmente. La solicitud offline se firma y verifica, pero no se ejerció su cumplimiento en el servidor; tampoco se esperaron 24 horas para comprobar la programación periódica en tiempo real.

El licenciamiento está listo para usarse en el alcance verificado. Firma comercial/notarización y pruebas nativas de instaladores siguen siendo pendientes de distribución separados, descritos en [el informe de producción](AIDEX-PRODUCCION.md).

Evidencia: [ciclo real](../dist/aidex-1.0.0/validacion/licensing-production-cycle.log), [estado final sin datos secretos](../dist/aidex-1.0.0/validacion/licensing-production-state.json), [validación consolidada](../dist/aidex-1.0.0/VALIDACION.json). Los paquetes conservan sus hashes y no contienen estado preactivado.
