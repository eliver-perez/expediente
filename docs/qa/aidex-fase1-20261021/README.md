# Evidencia de QA — AIDEX Fase 1

21 de octubre de 2026. [Informe A–G](../../AIDEX-FASE1-IMPLEMENTACION.md).

Entorno macOS ARM, Go 1.27.1, cachés de Go aisladas en `/private/tmp`; Chrome mediante Playwright. Se utilizaron fixtures sintéticas. Ningún instalador fue generado.

| Archivo | Comando / resultado |
| --- | --- |
| [backend-initial.txt](backend-initial.txt) | `go test -tags development ./... -count=1 -timeout=180s`; primera regresión con cuatro fallos descritos y corregidos en el informe. Los demás paquetes pasaron. |
| [backend-final.txt](backend-final.txt) | `go test -tags development ./internal/libraries ./internal/storage -count=1 -timeout=180s`; salida 0. |
| [race.txt](race.txt) | `go test -race -tags development ./internal/libraries -run '^TestLifecycle' -count=1 -timeout=120s`; salida 0. |
| [unsafe-preparation.txt](unsafe-preparation.txt) | Prueba `TestLifecycleCleanupPreparationRejectsUnsafeOwnership`; salida 0. Añadida después de race, prueba cuatro sustituciones/rutas inseguras antes de preparar el journal. |
| [upgrade-14-to-15.txt](upgrade-14-to-15.txt) | `go test ./internal/storage -run '^TestLibraryLifecycleUpgradeFromProduction$' -count=1 -timeout=45s`; salida 0. |
| [browser.txt](browser.txt) | `npm --prefix web run test:e2e -- lifecycle.spec.ts dashboard-settings.spec.ts observability.spec.ts libraries.spec.ts organization.spec.ts workflow.spec.ts processing.spec.ts identity.spec.ts license.spec.ts network.spec.ts`; 13/13, salida 0. |
| [frontend-build.txt](frontend-build.txt) | `npm --prefix web run build`; salida 0. Incluye advertencia de tamaño, sin errores de TypeScript. |
| [go-vet.txt](go-vet.txt) | `go vet -tags development ./...`; salida 0, sin diagnósticos (archivo vacío). |
| `build-{darwin-arm64,windows-amd64,linux-amd64,linux-arm64}.txt` | Cuatro compilaciones con `CGO_ENABLED=0 GOOS=… GOARCH=… go build -o /private/tmp/aidex-phase1-verify/… ./cmd/gestor-documental`; salida 0 en cada plataforma, sin diagnósticos. Binarios temporales, no paquetes. |
| [npm-audit.json](npm-audit.json) | Auditoría de dependencias con dos hallazgos altos preexistentes en herramientas de desarrollo. No se presenta como una auditoría sin incidencias. |

Los comandos se ejecutaron con un supervisor externo que termina el grupo de procesos al superar su límite: 60–240 segundos para verificaciones focalizadas/builds y 480 segundos para la regresión inicial o navegador. Se comprobó la terminación de todas las sesiones. `git diff --check` terminó sin errores.

Capturas revisadas: [Toast móvil con desplazamiento](toast-mobile.png), [tabla y progreso de Procesamiento](processing.png). Las capturas contienen exclusivamente nombres y rutas de fixtures temporales.

La compilación cruzada no prueba cuentas de servicio, NTFS/junctions, SMB ni ejecución nativa en Ubuntu. Esas pruebas manuales se detallan en G del informe.
