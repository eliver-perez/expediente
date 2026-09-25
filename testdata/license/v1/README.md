# Vectores de licencia V1.0

Material público exclusivamente de desarrollo. No contiene claves comerciales ni
claves privadas de producción. La firma conocida usa el primer vector de prueba
Ed25519 de RFC 8032; esa clave pública está expresamente bloqueada en producción.
La identidad de instalación de los vectores utiliza una semilla ficticia de 32
bytes `0x42`. Ninguna identifica una instalación de usuario.

`manifest.json` fija SHA-256 y resultados de cada caso. Se incluyen payload/JWS,
peticiones `.licreq`, prueba de posesión con sus bytes exactos, vinculaciones
incorrectas, firma corrupta, módulos, revisiones y los cuatro tiempos exclusivos
alrededor del vencimiento y de los 15 días de gracia. `lower-revision.lic` debe
rechazarse después de aceptar `higher-revision.lic`. La licencia perpetual debe
seguir activa independientemente de la fecha de mantenimiento/contacto.

Regenerar determinísticamente desde la raíz:

```sh
go run -tags development ./cmd/license-mock vectors --out testdata/license/v1
```

Las pruebas comparan los archivos byte por byte. Compartir **esta carpeta completa**
con el proyecto del servidor, incluyendo `manifest.json`, y ejecutar allí las
mismas expectativas. Aún no se han contrastado contra ese proyecto. El simulador
local demuestra las suposiciones del cliente, no una compatibilidad externa ya
certificada. Escenarios de reintento, límites de activación, renovación offline,
TLS, privacidad y módulos están en las pruebas ejecutables Go de H6.
