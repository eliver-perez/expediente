# AIDEX 1.0.0 — informe de estabilización para pruebas internas

Fecha: 23 de octubre de 2026. El instalador se compiló en macOS Apple Silicon para Windows AMD64. **No se ejecutó el instalador ni el servicio en Windows en este entorno.** El CLUF incluido sigue siendo provisional y pendiente de revisión jurídica.

## A. Bloqueo durante eliminación

La eliminación de una biblioteca sintética de 80 000 documentos tardó alrededor de 70 s antes de la corrección. La instrucción que eliminaba las versiones de contenido consumió 65,5 s: SQLite comprobaba claves foráneas sin índices de las columnas hijas. WAL ya permitía lecturas; además, la renovación de la sesión intentaba tomar el escritor y podía bloquear peticiones de lectura.

La migración 0019 añade los índices de referencia necesarios y otros índices de consulta. La misma eliminación pasó a 4,55–5,14 s y la instrucción de versiones dejó de aparecer entre las consultas lentas. La autenticación evita renovar la sesión en cada lectura y acota a 200 ms un intento de renovación cuando corresponde. Durante el borrado se comprobaron Dashboard, estadísticas, historial, estado de eliminación, bibliotecas ajenas, sesión y licencia. En la muestra repetida de lectura/sesión/licencia, la latencia máxima fue 1,13 ms. El archivo original vinculado de prueba quedó intacto. La eliminación conserva su transacción y el flujo de limpieza seguro existente.

## B. Contrato de licencia

Mientras el CLUF está pendiente, React muestra siempre el contrato y oculta la navegación, incluso después de cambiar de URL, usar Atrás o recargar. El backend niega el acceso a API documental hasta que una cuenta autorizada registre la versión y huella exactas; las cuentas sin permiso no pueden aceptar. El renderizador presenta títulos, cláusulas, negritas, listas y separadores como elementos React escapados, sin ejecutar HTML. El texto contractual almacenado y su SHA-256 no se alteraron. El modal de «Acerca de AIDEX» solo consulta el documento.

Se verificaron el bloqueo de API y la aceptación en pruebas de backend sin el modo de desarrollo; las pruebas de navegador comprobaron el renderizado y el modal. La aceptación del CLUF provisional no sustituye la revisión jurídica.

## C. Bibliotecas e interfaz

La creación asigna al administrador creador como gestor desde el backend, sin casilla manual. La descripción opcional (máximo 500 caracteres) se guarda en la biblioteca y admite edición; si está vacía, la tarjeta muestra el texto de modalidad sin persistirlo. Se ajustaron alineación, tamaño, foco y disposición responsiva de tarjetas. Los enlaces de descarga de manifiesto y al sitio oficial siguen el patrón visual de botones. Las pruebas de navegador se actualizaron para la asignación automática.

## D. Dashboard

Los indicadores muestran esqueletos de carga y la estructura de la página desde el inicio. Las bibliotecas se consultan por separado, por lo que una sección puede aparecer mientras la otra sigue cargando; los fallos parciales se presentan sin inventar ceros. Los índices de la migración 0019 apoyan las consultas de fechas y bibliotecas. No se añadió una caché de estadísticas que pudiera quedar desactualizada.

## E. Rendimiento sintético

Base SQLite temporal con tres bibliotecas híbridas, raíces, 300 expedientes, plantillas, requisitos, documentos en distintos estados, 25 000 versiones de contenido y trabajos sintéticos. No se crearon 100 000 PDF ni se tocó información del usuario.

| Registros | Dashboard | Conteos por biblioteca | Estadísticas de expedientes | Búsqueda por nombre | Tamaño SQLite |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 10 000 | 0,15 s | 0,06 s | 0,06 s | 0,01 s | 11,4 MiB |
| 50 000 | 1,42–1,60 s | 0,48–0,53 s | 0,53–0,59 s | 0,10–0,11 s | 54,2 MiB |
| 100 000 | 3,06–3,22 s | 0,99–1,07 s | 1,12–1,16 s | 0,20–0,21 s | 108 MiB |

El listado simple de bibliotecas se mantuvo por debajo de 3 ms. El heap observado llegó a aproximadamente 82 MiB durante el primer lote de carga y bajó después de GC. No se midió CPU de forma aislada; la última corrida coincidió con pruebas de navegador. Dashboard a 100 000 documentos aún tarda cerca de 3 s y puede necesitar una optimización adicional si el objetivo operativo exige respuesta subsegundo.

## F. Puerto configurable

El puerto predeterminado sigue siendo 18090. El instalador pregunta el puerto solo en instalaciones nuevas y comprueba conflictos evidentes; una actualización conserva el puerto de la configuración privada. En Ajustes, un administrador puede cambiar el puerto local con revisión optimista, enlace al nuevo origen, enlace previo recuperable y validación de disponibilidad. El ejecutable `configure-port` permite cambiarlo con el servicio detenido y bloqueo exclusivo de estado, también en Linux y macOS.

En Windows, «Configurar servidor AIDEX» en Inicio ofrece el cambio offline con elevación, actualiza la regla propia de firewall, el destino público del lanzador y el registro de puerto, reinicia el servicio y restaura el puerto anterior ante un fallo. Un cambio de puerto LAN desde la web se remite a esa herramienta elevada: la cuenta virtual del servicio no debe recibir facultades administrativas para editar el firewall. Un cambio a LAN desde la web exige confirmar que ya exista una regla administrada válida. El instalador limita la regla a subred local y perfiles privados o de dominio; no abre el router.

En Ubuntu/macOS, el comando `configure-port` reutiliza la validación y persistencia del backend. Quedaron preparados en código fuente `packaging/ubuntu/aidex-port` y `packaging/macos/port.command`: coordinan systemd/launchd, verifican el endpoint y restauran el puerto anterior si falla. Su sintaxis shell pasó; requieren pruebas nativas e integración de los accesos directos con puerto personalizado antes de incorporarse a esos paquetes. No se generaron paquetes Ubuntu ni macOS.

## G. Pruebas generales

- `go test -tags development ./... -timeout 8m`: pasó con permiso de sockets locales.
- `go test ./internal/httpapi ./internal/network -count=1 -timeout 3m`: pasó; incluye la validación de producción del CLUF.
- Prueba optativa `AIDEX_PERF_SCALE=1`: pasó con 100 000 registros y eliminación concurrente de 80 000.
- `npm run build`: pasó.
- Playwright completo: 27 de 27 pruebas pasaron. Se actualizaron selectores anteriores de la casilla de gestor y del botón de red.
- Simulación PowerShell de identidad de servicio, rechazo de servicios ajenos y rutas/mensajes Unicode: pasó. El parser validó los scripts empaquetados.

Las primeras corridas fallaron por la versión 19 no reconocida en respaldos y por selectores obsoletos de pruebas. Se corrigieron y repitieron. La importación de un respaldo anterior admite la columna de descripción ausente con su valor predeterminado. Las pruebas nativas de instalación, ACL, servicio, firewall, reinicio, desinstalación y OCR en Windows siguen pendientes.

## H. Instalador Windows

Archivo: `dist/aidex-1.0.0-estabilizacion-20261023-windows/AIDEX-1.0.0-Windows-amd64.exe`.

Tamaño: **40 420 756 bytes**. SHA-256: `833c4bf21e4e08ed8ed882478afe2df0e58de48eeb1ba12e3a7ef2217db82eb1`.

NSIS compiló sin advertencias. Los dos ejecutables incluidos tienen recursos de icono AIDEX y cabeceras PE AMD64; el stub de NSIS es de 32 bits, habitual para instaladores que entregan binarios de 64 bits. Se comprobó el cierre de importaciones PE y que `SHA256SUMS` coincide con todos los archivos del directorio. El instalador conserva datos por defecto y no elimina documentos originales. Las instrucciones de prueba están en `docs/PRUEBAS-WINDOWS-1.0.0.md`, también copiadas junto al instalador.

## I. Cambios técnicos y pendientes

Archivos principales: migraciones 0018/0019, servicios de bibliotecas, identidad, red y respaldo; componentes React de CLUF, bibliotecas, Dashboard y red; scripts NSIS/PowerShell y lanzador Windows; pruebas Go, PowerShell y Playwright. Las migraciones se aplican en la actualización normal. La columna de descripción se añade con cadena vacía; los índices 0019 no modifican documentos.

Antes de considerar distribución comercial: validar el instalador en Windows 11 real (instalación nueva y actualización, puerto alternativo ocupado/libre, firewall LAN, arranque, recuperación y desinstalación conservadora), probar las utilidades de puerto en Ubuntu/macOS antes de empaquetarlas, y concluir la revisión jurídica del CLUF. La compatibilidad con un respaldo de esquema 15 anterior a la columna de descripción se implementó, pero no se ensayó con un archivo histórico real. No se modificó el servidor de licencias ni se usó una licencia real en esta ronda.
