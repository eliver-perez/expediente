# Windows — corrección de configuración y acceso desde el Escritorio

Revisión **0.7.0-test.1-r2**, compatible con los datos/configuración de **0.7.0-test.1**.
No cambia el esquema 5 ni las migraciones, el mínimo de contraseña de 6 caracteres o el contrato de licencia.
La integración con el servidor comercial continúa aplazada.

## Instalar sobre la prueba anterior

1. Cierra la ventana de configuración que mostró el error.
2. Ejecuta **AIBID-Pruebas-0.7.0-test.1-r2-Windows-amd64.exe** y acepta la elevación del instalador.
   Se puede instalar directamente sobre la revisión anterior. No hace falta desinstalar ni borrar carpetas.
3. Desde Inicio abre **AIBID Pruebas → Configurar AIBID Pruebas** y acepta la elevación para esta configuración inicial.
4. Completa usuario/nombre/contraseña. Si el administrador ya existía, se conserva.
5. Para el uso diario abre **AIBID Pruebas** en el Escritorio o **Abrir AIBID** en Inicio.

El acceso diario ejecuta `AIBID.exe`, una pequeña aplicación sin consola y sin elevación que abre
la dirección configurada en el navegador predeterminado. Solo consulta el estado del servicio;
si falta configurarlo o está detenido muestra instrucciones. No inicia otro servidor ni solicita contraseñas.

La URL inicial es **http://127.0.0.1:18090**. El administrador puede cambiarla en la configuración
privada y ejecutar Configurar de nuevo. El lanzador recibe únicamente `public_url` en
`%ProgramFiles%\AIBID-Test\launcher.json`, legible por usuarios locales; no accede a la configuración
privada, la DB ni las claves del servicio. Se conservan sus permisos restringidos.

## Fallo corregido

El diagnóstico de herramientas PDF/OCR terminaba correctamente. Después la ruta Windows se
convertía en `file://C:/ProgramData/...`, donde SQLite interpretaba `C:` como autoridad de red.
Se construye ahora `file:///C:/ProgramData/...`, tanto para el escritor como para el lector,
conservando escapes, parámetros, WAL y permisos. Es el formato que documenta
[SQLite para unidades Windows](https://www.sqlite.org/uri.html#the_uri_path).

El preflight del nuevo instalador usa el binario nuevo extraído en su carpeta temporal y el comando
`check-state`: comprueba acceso exclusivo sin abrir/migrar SQLite. Esto evita que el binario anterior
impida su propia reparación. El desinstalador utiliza la misma comprobación sin abrir la DB.
La compatibilidad sigue siendo la versión de datos 0.7.0-test.1; `r2` identifica la revisión del ejecutable
y del instalador. No habilita actualizaciones generales N-1 ni cambios de esquema.

## Comprobación en Windows

- Reinstalar r2 sobre la configuración fallida sin eliminar `%ProgramData%\AIBID-Test`.
- Configurar: herramientas/OCR OK, ausencia de `invalid uri authority`, administrador y servicio iniciados.
- Abrir desde Escritorio e Inicio con una cuenta estándar: navegador sin UAC ni consola.
- Reiniciar la PC y abrir nuevamente; si el inicio diferido aún no terminó, el lanzador lo indica.
- Detener el servicio administrativamente: el lanzador muestra instrucciones, sin elevarse por sí solo.
- Reinstalar r2 con datos existentes: conserva usuarios, configuración y documentos.

Las pruebas automatizadas cubren la reproducción del error original, la URI corregida, nombres con
espacios/acentos/%/#, lectura/escritura y reapertura con WAL, lector de solo lectura, exclusión de estado
y validación de la URL pública. La ejecución completa del instalador/SCM/lanzador requiere Windows;
compilarlo en macOS no sustituye esa comprobación nativa.
