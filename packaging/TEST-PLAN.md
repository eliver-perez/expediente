# Pruebas de instaladores H7

Registrar SO/versión/arquitectura, versión del paquete, SHA256, fecha, resultado y error exacto.
Usar documentos de prueba. La URL local del canal es http://127.0.0.1:18090.
La contraseña admite de 6 a 128 caracteres. No hay contraseña inicial.

| Paso | Resultado esperado |
| --- | --- |
| Instalar el paquete del SO | Crea el servicio detenido hasta configurar administrador; datos privados y código separados |
| Ejecutar Configurar / `sudo aibid-test-admin` | Diagnóstico PDF nativo y OCR español OK; crea administrador e inicia servicio |
| Cerrar Terminal y abrir navegador | La aplicación sigue disponible |
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

Linux: `systemctl status aibid-test`, `journalctl -u aibid-test --no-pager -n 100`.
macOS: `sudo launchctl print system/app.aibid.test`, log bajo `data/logs/service.log`.
Windows: Servicios → AIBID Pruebas; Visor de eventos → Registros de Windows → Aplicación → AIBIDTest.
No enviar DB, contraseñas, identidad privada o documentos reales como evidencia. Basta el error y el paso.

La actualización N-1, backup/restore integral, firma de releases, ensayos de carga y certificación
operativa completa pertenecen al cierre posterior de H7. No se consideran aprobados por compilar un paquete.
