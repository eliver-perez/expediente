# Retiro manual de instalaciones anteriores

Este procedimiento corresponde únicamente a las instalaciones personales anteriores. No se distribuye en los paquetes AIDEX. No hay adopción automática de su estado ni traslado de documentos. Antes de intervenir, conserva una copia de configuración, estado, identidad de licencia y documentos. Si necesitas transferir una licencia activa, desactívala desde esa instalación y confirma la respuesta del servidor; desinstalar no libera activaciones.

## Windows

En Aplicaciones instaladas selecciona AIBID o AIBID Pruebas y ejecuta su desinstalador. Deja desmarcado el borrado de datos. En PowerShell elevado consulta `Get-Service AIBID,AIBIDTest -ErrorAction SilentlyContinue`; si el servicio sigue ejecutándose, detenlo mediante Servicios antes de instalar el producto nuevo. No borres C:\ProgramData\AIBID ni C:\ProgramData\AIBID-Test. Un servicio pendiente de eliminación puede requerir cerrar Servicios o reiniciar Windows. No uses sc delete o icacls sobre cuentas ajenas ni borres reglas manualmente sin comprobar su propietario.

## Ubuntu

Identifica el paquete con `dpkg-query -W aibid aibid-test`. Detén solo la unidad que exista: `sudo systemctl stop aibid.service` o `sudo systemctl stop aibid-test.service`. Retira el paquete instalado con `sudo apt remove aibid` o `sudo apt remove aibid-test`; elige conservar datos. No ejecutes borrado de metadatos. Conserva /etc/aibid*, /var/lib/aibid* y las cuentas correspondientes. Comprueba que el puerto 18090 ya no esté ocupado antes de instalar AIDEX.

## macOS

Ejecuta `/Applications/AIBID/Desinstalar AIBID.command` o `/Applications/AIBID Pruebas/Desinstalar AIBID.command`, según la carpeta existente, y responde No al borrado de metadatos. Comprueba `sudo launchctl print system/app.aibid` o `sudo launchctl print system/app.aibid.test`; si la unidad sigue cargada, detén únicamente la correspondiente con `sudo launchctl bootout system/app.aibid` o `sudo launchctl bootout system/app.aibid.test`. Conserva /Library/Application Support/AIBID, /Library/Application Support/AIBID-Test y sus cuentas. No borres ni copies manualmente su identidad de licencia a la instalación nueva.

Una vez retirado el servicio anterior, sigue las instrucciones actuales de [AIDEX](PRODUCCION.md). Verifica que los servicios y permisos del producto nuevo correspondan a las cuentas nuevas; los permisos SMB/NTFS otorgados a la cuenta anterior no se transfieren automáticamente.
