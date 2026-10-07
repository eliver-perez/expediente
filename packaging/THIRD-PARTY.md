# Dependencias de los paquetes de prueba

AIBID se entrega compilado con React embebido; los instaladores no incluyen su código Go/React.
PDF/OCR se ejecuta en procesos externos. El contrato de licencias de AIBID no sustituye
las licencias de estas dependencias.

- Go y dependencias: inventario exacto en go.mod/go.sum del proyecto; runtime enlazado en el binario.
- React y frontend: inventario en web/package-lock.json del proyecto, versiones compiladas.
- macOS: Poppler/Tesseract/tesseract-lang instalados por Homebrew; este paquete no los redistribuye.
- Ubuntu: dependencias suministradas por los repositorios Ubuntu 24.04, avisos en /usr/share/doc de cada paquete.
- Windows: compilaciones conda-forge. `windows-ocr.lock.json` identifica URL, versión, build, licencia y SHA256
  del paquete original. `docs/third-party/` conserva avisos, recetas de construcción y parches de sus proveedores.
  Se distribuyen DLL, cuatro ejecutables PDF/OCR, datos de Poppler, fuentes e idiomas spa/eng/osd seleccionados.
  Las recetas apuntan a los archivos fuente originales y sus hashes. No son fuentes de AIBID.

Este canal es para pruebas internas. Antes de una publicación comercial hay que preparar y verificar
la distribución de fuentes correspondientes y los avisos/ofertas que exija cada dependencia copyleft;
la presencia de enlaces o recetas por sí sola no acredita ese cumplimiento.

Referencias de proveedores: [Tesseract](https://github.com/tesseract-ocr/tesseract),
[Poppler](https://poppler.freedesktop.org/), [conda-forge](https://conda-forge.org/docs/),
[NSIS](https://nsis.sourceforge.io/License).

## Vistas previas de Word y Excel (fase 4)

LibreOffice es una dependencia opcional instalada por el administrador en el servidor.
AIBID no redistribuye sus ejecutables ni lo descarga automáticamente. Se utiliza un
proceso headless con perfil privado para producir una representación PDF descartable.
En Ubuntu, Writer y Calc se declaran como `Recommends` (apt los instala salvo que se desactiven los recomendados); las herramientas OCR anteriores
siguen siendo las dependencias requeridas. Versiones y licencias dependen de la
instalación elegida; ver los [avisos oficiales de LibreOffice](https://www.libreoffice.org/licenses/).
La validación utiliza LibreOffice 26.2.6 para macOS ARM, temporalmente.

## Gráficas locales (fase 8)

Chart.js **4.5.1**, licencia MIT, y su dependencia @kurkle/color **0.3.4**, MIT,
se integran en los recursos web del ejecutable mediante Vite. Versiones e integridad
están fijadas en `web/package-lock.json`; avisos en `third-party/frontend/` de cada
paquete. Se carga el módulo de gráficas al abrir Dashboard. No se usan CDN, claves,
servicios externos ni descargas en el equipo del usuario.

La ausencia de LibreOffice se indica en el instalador Windows/macOS, en la guía y
en Ajustes → Vistas previas y caché. Ubuntu lo recomienda desde el paquete y la
preparación offline incluye recomendados. No se descarga automáticamente en macOS
o Windows. Los avisos de sus proveedores continúan aplicándose.

[Chart.js: integración](https://www.chartjs.org/docs/latest/getting-started/integration.html),
[accesibilidad](https://www.chartjs.org/docs/latest/general/accessibility.html),
[licencia](https://github.com/chartjs/Chart.js/blob/v4.5.1/LICENSE.md).
