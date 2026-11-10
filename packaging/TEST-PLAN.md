# AIDEX — Pruebas de instaladores

Registrar SO/versión/arquitectura, versión del paquete, SHA256, fecha, resultado y error exacto.
Usar documentos de prueba. La URL local del canal es http://127.0.0.1:18090.
La contraseña admite de 6 a 128 caracteres. No hay contraseña inicial.

| Paso | Resultado esperado |
| --- | --- |
| Instalar el paquete del SO | Crea el servicio detenido hasta configurar administrador; datos privados y código separados |
| Ejecutar Configurar / `sudo aidex-admin` | Diagnóstico PDF nativo y OCR español OK; crea administrador e inicia servicio |
| Cerrar Terminal y abrir navegador | La aplicación sigue disponible |
| Windows: abrir AIDEX desde Escritorio/Inicio como usuario estándar | Abre navegador con URL configurada, sin consola ni UAC; no lee estado privado |
| Windows r2: instalar sobre el fallo `invalid uri authority: C:` | Permite reinstalar y configurar sin borrar configuración ni ejecutar el binario defectuoso |
| Reiniciar el equipo | Servicio vuelve a estar disponible y conserva la sesión/datos según las políticas normales |
| Crear biblioteca híbrida y sus raíces | Permisos limitados a las carpetas elegidas; sin modificar permisos de carpetas ajenas |
| Cargar native.pdf y scanned.pdf | Vista previa/descarga y búsqueda `PR-008` / `ESCANEADO` funcionan |
| Crear expediente, categoría y tipo; agregar ambos PDF | Relaciones conservadas y archivos disponibles |
| Agregar nueva raíz vinculada a la biblioteca híbrida | El documento administrado sigue disponible (regresión ya corregida en H4) |
| Reinstalar EXACTAMENTE este mismo paquete | Servicio se detiene, configura sin reemplazar JSON/DB y vuelve a iniciar si estaba habilitado |
| Desinstalar | Servicio/código eliminados; DB, licencia, subidas, originales y configuración conservados |
| Instalar de nuevo la misma versión | Vuelve a los mismos usuarios, bibliotecas, expedientes y documentos |
| Cuenta local ajena sin permisos | No puede leer configuración, DB ni identidad privada del servicio |
| Windows sin conexión | Instalación y OCR funcionan con las herramientas/idiomas empaquetados |
| Ubuntu offline, VM limpia, red desactivada | Bundle resuelve dependencias solo desde su repositorio local; repetir reinstalación |
| Otro programa usa 18090 | Servicio falla de forma visible; no desplaza al otro programa ni anuncia disponibilidad |

Linux: `systemctl status aidex.service`, `journalctl -u aidex.service --no-pager -n 100`.
macOS: `sudo launchctl print system/app.aidex`, log bajo `data/logs/service.log`.
Windows: Servicios → AIDEX; Visor de eventos → Registros de Windows → Aplicación → AIDEX.
No enviar DB, contraseñas, identidad privada o documentos reales como evidencia. Basta el error y el paso.

La actualización N-1, backup/restore integral, firma de releases, ensayos de carga y certificación
operativa completa pertenecen al cierre posterior de H7. No se consideran aprobados por compilar un paquete.

## Ubuntu ARM64

Para una VM sobre Apple Silicon comprobar `dpkg --print-architecture` = `arm64` y usar
`aidex_1.0.0_arm64.deb`. Repetir instalación, diagnóstico PDF/OCR, inicio del
servicio, acceso HTTP y ambas opciones de desinstalación. Preparar el bundle offline en
Ubuntu 24.04 arm64; los scripts rechazan un paquete/bundle de otra arquitectura.


## Red y licenciamiento

En macOS ARM64, Windows amd64 y Ubuntu 24.04 (amd64/arm64):

1. Actualizar conservando datos y licencia. Confirmar `version` informa 1.0.0 y canal production.
2. En modo local, abrir 127.0.0.1:18090; comprobar que otro equipo no conecta por la IP del servidor.
3. Guardar modo LAN, entrar por una IPv4 indicada y comprobar inicio de sesión, biblioteca, PDF y licencia.
4. Reiniciar el servicio y el equipo; confirmar que mantiene LAN y el puerto. Revertir a local y comprobar
   que deja de aceptar conexiones remotas, conservando la interfaz local y los trabajos documentales.
5. Comprobar que un usuario sin `system.configure` no ve ni modifica la configuración de red.
6. Revisar el diagnóstico del firewall del sistema. Si indica verificación incompleta, seguir sus
   instrucciones con el administrador. No debe desactivar protecciones ni crear/modificar/borrar reglas.
7. Mantener una licencia de prueba activada sin Internet, reiniciar y comprobar consulta/operaciones dentro
   de su vigencia. Suscripciones: tolerancia de 15 días desde vencimiento. No usar la licencia real para
   simular vencimientos ni cambiar el reloj del equipo de producción.
8. Revisar los cinco filtros antes de buscar, en escritorio y móvil; probar los seis acordeones sin perder
   contenido ni acciones. En LAN verificar biblioteca nueva, cargas y solicitudes de licencia.
9. Desinstalar conservando datos y luego con borrado interno en una instalación desechable. El instalador configura y elimina únicamente su regla AIDEX-LAN-TCP; las reglas ajenas o de políticas externas se conservan.

## Identidad, actualización y Unicode

Ejecutar instalación nueva y actualización desde una entrega anterior con datos ficticios. Los valores vigentes están en docs/PRODUCCION.md; las excepciones históricas se limitan a cuentas/rutas privadas y SID de Windows. Reinstalar idempotentemente; rechazar instalaciones dobles, servicio ajeno, marcador desconocido y enlaces. Comprobar que el servicio antiguo no arranca simultáneamente con el nuevo y que el borrado interno requiere confirmación.

En cada SO probar usuarios/rutas José, María López, Muñoz, Documentación, espacios y Unicode adicional compatible. En Windows comprobar visualmente áéíóú/ÁÉÍÓÚ/Ññ/ü/¿¡ con PowerShell5.1 y NSIS; validar código de salida, registro Unicode, binPath de SCM y timeout. En Ubuntu repetir APT offline con carpeta Unicode y espacios. En macOS repetir Installer, plist generado y launchd. Registrar por separado prueba automatizada, inspección del paquete y evidencia visual/nativa; las primeras no certifican las últimas.
