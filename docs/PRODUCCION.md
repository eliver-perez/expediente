# AIDEX 1.0 — Instalación y mantenimiento

Versión técnica 1.0.0, canal production. Aplicación de Indexación Digital de Expedientes. Paquetes para Windows x64, Ubuntu 24.04 amd64/arm64 y macOS Apple Silicon (13 o posterior). Verifica SHA256SUMS antes de instalar. No hay paquete Mac Intel.

## Windows

Ejecuta AIDEX-1.0.0-Windows-amd64.exe con elevación. Programa: C:\Program Files\AIDEX. Backend: aidex-server.exe. Lanzador sin consola ni elevación: AIDEX.exe. Servicio y Visor de eventos: AIDEX. Cuenta virtual: NT SERVICE\AIDEX. Configuración: C:\ProgramData\AIDEX\config.json. Estado privado: C:\ProgramData\AIDEX\state. La ACL privada admite SYSTEM, Administradores y el SID del servicio; launcher.json publica únicamente URL y nombre de servicio.

Abre Configurar AIDEX desde Inicio para comprobar PDF/OCR, crear el administrador (contraseña de 6 a 128 caracteres) e iniciar el servicio. Después usa AIDEX en Escritorio o Abrir AIDEX en Inicio. Diagnóstico elevado: `& 'C:\Program Files\AIDEX\manage.ps1' -Action Doctor`. Consulta: `Get-Service AIDEX`. Desinstala mediante Aplicaciones instaladas; conservar datos es el valor predeterminado. El borrado interno exige selección y confirmación explícitas y conserva documentos físicos y cargas.

El instalador elevado prepara AIDEX-LAN-TCP en PersistentStore, grupo AIDEX, limitada al binario, servicio y puerto configurado, TCP, LocalSubnet, perfiles Domain y Private; bloquea edge traversal. Nunca modifica reglas ajenas ni eleva el backend. En modo local el servicio no escucha en LAN aunque la regla esté provisionada. Redes públicas y políticas de dominio requieren revisión administrativa independiente.

## Ubuntu

Comprueba `dpkg --print-architecture`. Instala `sudo apt install ./aidex_1.0.0_arm64.deb` o su equivalente amd64; ejecuta `sudo aidex-admin`. Servicio aidex.service, usuario/grupo aidex sin login, backend /usr/lib/aidex/aidex-server, configuración /etc/aidex/config.json, datos /var/lib/aidex/state. Directorios privados 0700; archivos sensibles 0600. Las dependencias incluyen Poppler y Tesseract spa/eng; LibreOffice Writer/Calc son recomendaciones para previews.

Administración: `sudo systemctl start aidex.service`, `sudo systemctl stop aidex.service`, `sudo systemctl status aidex.service`, `sudo journalctl -u aidex.service --no-pager -n 100`. Diagnóstico: `sudo -u aidex /usr/lib/aidex/aidex-server doctor --config /etc/aidex/config.json --sample-directory /usr/share/doc/aidex/pdf`. Desinstalación: `sudo aidex-uninstall`; APT remove/purge conserva configuración, cuenta y datos. El asistente ofrece borrado de metadatos con confirmación, manteniendo documentos físicos.

El .deb no contiene todas las dependencias. Para equipos desconectados prepara el bundle en Ubuntu 24.04 de igual arquitectura con offline/prepare-offline.sh y usa offline/install-offline.sh. El instalador verifica hashes y arquitectura; el repositorio temporal admite rutas con espacios. No se deshabilita ni reconfigura el firewall del sistema.

## macOS

Instala AIDEX-1.0.0-macOS-arm64.pkg. Abre /Applications/AIDEX/Configurar AIDEX.command. Backend: /Library/Application Support/AIDEX/bin/aidex-server. Configuración: /Library/Application Support/AIDEX/data/config.json; estado y logs bajo data. Cuenta/grupo _aidex sin login y oculto; LaunchDaemon app.aidex, plist /Library/LaunchDaemons/app.aidex.plist. Datos privados 0700 y archivos sensibles 0600; plist root:wheel 0644. Poppler/Tesseract con idiomas spa/eng deben estar disponibles en /opt/homebrew/bin. LibreOffice es opcional para previews Word/Excel.

Diagnóstico de servicio: `sudo launchctl print system/app.aidex`. Detener: `sudo launchctl bootout system/app.aidex`. Iniciar: `sudo launchctl bootstrap system /Library/LaunchDaemons/app.aidex.plist`. Diagnóstico documental: `sudo -u _aidex '/Library/Application Support/AIDEX/bin/aidex-server' doctor --config '/Library/Application Support/AIDEX/data/config.json' --sample-directory '/Library/Application Support/AIDEX/docs/pdf'`. Desinstala con /Applications/AIDEX/Desinstalar AIDEX.command. Solo se eliminan archivos propios no modificados según su SHA256; conserva originales, archivos ajenos y cuenta del servicio.

## Red, licencias y actualizaciones

Acceso inicial http://127.0.0.1:18090. Ajustes → Acceso y red permite cambiar a LAN, con las verificaciones y restauración existentes. La URL pública, TLS/reverse proxy y permisos de carpetas externas deben configurarse para el entorno. El usuario del navegador y la cuenta del servicio son identidades distintas: concede a la cuenta del servicio únicamente el acceso documental requerido.

El bloque license de config.json usa https://licensing.aidex.com.mx, confianza Ed25519 lic-prod-2026-a (production/license), refresh_hours 24 y development_bypass false. product_id técnico gestor_documental; app_version 1.0.0. Activa desde la pantalla Licencia; no hay contraseña ni licencia predeterminadas. La identidad local privada se conserva durante actualizaciones. La desactivación confirmada en el servidor libera la plaza; borrar metadatos o desinstalar no la libera. Consulta [Contrato V1](../LICENSE_CONTRACT.md).

Las actualizaciones de esta edición mantienen nombres, cuentas, configuración, identidad y datos. Se valida el marcador 1.0.0 y acceso exclusivo antes de migrar con el nuevo binario. No se importan instalaciones de otro producto. Retira manualmente instalaciones anteriores que ocupen el puerto antes de validar esta edición; nunca borres sus datos para resolver un conflicto de puerto.

Windows conserva NSIS Unicode, BOM único UTF-8 para PowerShell 5.1, JSON UTF-8, comandos UTF-16LE y salida de instalador UTF-16LE. No cambies la configuración regional ni renombres usuarios para ocultar problemas. Operaciones no interactivas del asistente: máximo diez minutos; espera de servicio: sesenta segundos. Guarda el registro temporal indicado si falla la operación. Unix usa shell UTF-8 sin BOM y argumentos entre comillas.

Antes de publicar valida instalación, actualización y desinstalación con servicios reales en las cuatro plataformas, LAN/GPO/SMB, Unicode, respaldos y restauración. La firma comercial de Windows, firma/notarización de macOS y cumplimiento de redistribución de terceros son requisitos externos. Compilar e inspeccionar un paquete no certifica estos ciclos. Véase [Validación](TRANSICION-PRODUCCION.md) y [Dependencias](../packaging/THIRD-PARTY.md) en el paquete.
