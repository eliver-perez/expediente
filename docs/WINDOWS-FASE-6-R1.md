# Corrección del instalador Windows — fase 6, revisión f6r1

6 de octubre de 2026. Versión de aplicación 2.0.0-alpha.6; revisión del paquete
**f6r1**. No cambia el esquema 13 ni las funcionalidades de la fase 6.

## Causa y alcance

El instalador anterior abortaba al analizar `manage.ps1`, antes de ejecutar
Preflight. El archivo fuente tenía una marca UTF-8 BOM; el constructor lo leía
como texto UTF-8 conservando esa marca y añadía otra al escribir UTF-8 con BOM.
El resultado empezaba por `EF BB BF EF BB BF`. PowerShell dejaba de reconocer
`param(...)` como bloque de parámetros y devolvía `InvalidLeftHandSide` en la
asignación de InstallRoot, exactamente el error de la captura.

La corrección decodifica la marca de entrada y escribe una sola marca de salida,
conservando los caracteres españoles necesarios para Windows PowerShell 5.1.
No se cambian permisos de carpetas, servicio, licencia, datos ni migraciones.
El fallo de análisis mostrado ocurre antes de detener el servicio o escribir datos.

## Prevención y validación

- Tres pruebas de transformación: fuente con/sin BOM, script real y plantilla Unix.
- Cuatro pruebas con el parser PowerShell: acepta el script real corregido y
  rechaza doble BOM, sintaxis inválida y pérdida de un parámetro del instalador.
- El constructor comprueba el script generado antes de compilar y empaquetar:
  BOM único, versión sustituida, sintaxis y los cuatro parámetros esperados.
  Requiere un intérprete PowerShell; no omite silenciosamente esta comprobación.
- En Windows prefiere `powershell.exe` (Windows PowerShell), también en el workflow
  de CI. La ejecución remota de CI no se afirma como realizada.
- En esta Mac se reproduce el error anterior y se valida la corrección con el
  parser de [PowerShell oficial 7.6.6](https://github.com/PowerShell/PowerShell/releases/tag/v7.6.6).
  Esto verifica análisis sintáctico, no ejecuta las acciones Windows del instalador.
- El paquete se vuelve a construir con NSIS y las comprobaciones de iconos y
  dependencias PE. La instalación real y el servicio Windows requieren prueba allí.

Archivos: `scripts/build_installers.py`, `scripts/check_windows_script.ps1`,
`scripts/test_build_installers.py` y `.github/workflows/h7-installers.yml`.
Se conserva `packaging/windows/manage.ps1` con su BOM único de origen.

## Instalar la corrección

1. Cierra el instalador fallido. No desinstales AIBID ni borres su base de datos.
2. Ejecuta `AIBID-Pruebas-2.0.0-alpha.6-f6r1-Windows-amd64.exe` y autoriza la elevación.
3. Conserva la carpeta de instalación existente. Puede actualizar directamente
   desde alpha.4 o alpha.5; no necesitas instalar la fase 5 primero.
4. Abre AIBID y comprueba el acceso, bibliotecas y configuración. Si es una
   instalación nueva, usa Configurar AIBID Pruebas para crear el administrador.

El instalador nuevo se entrega en `dist/aibid-2.0-fase-6-r1/`, separado de los
paquetes anteriores. Los instaladores macOS y Ubuntu de fase 6 no requieren este
arreglo. La fase 7 no se inicia.
