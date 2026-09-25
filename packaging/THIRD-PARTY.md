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
