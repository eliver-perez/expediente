# AIDEX — Identidad y validación de producción

AIDEX 1.0, versión técnica 1.0.0, canal production. Los cuatro paquetes se construyen nuevamente desde frontend, backend y scripts de esta identidad. No se renombran paquetes anteriores. Servicios, cuentas, reglas, comandos, rutas y recursos propios usan AIDEX; el módulo Go gestor-documental y product_id gestor_documental conservan su contrato técnico. No cambia el esquema de datos ni la lógica documental.

Se utilizan los cuatro SVG proporcionados por el propietario sin alterar colores, composición o proporciones. La carpeta gráfica no cambió: favicon e iconos ICO/.syso conservan esa misma figura, con nombres de archivo actuales. La compilación incluye los SVG y textos nuevos; los recursos anteriores generados se sustituyen al construir el frontend.

Los perfiles de instalación actuales no importan datos ni identidades de otro producto. Solo administran sus propios servicios, permisos, reglas y archivos. Se conserva la verificación exclusiva de estado, protección frente a enlaces, desinstalación por hash, conservación predeterminada de datos y confirmación expresa para borrar metadatos.

La configuración de licencia tiene un único servidor predeterminado HTTPS y la clave pública lic-prod-2026-a proporcionada para production/license. Se mantiene el contrato V1, TLS validado, firma Ed25519, vínculo de instalación, persistencia, importación offline y renovación. Los binarios oficiales no usan la etiqueta development ni omiten licencias. El modo de desarrollo requiere compilación expresa separada. Los paquetes nunca contienen la licencia utilizada en las pruebas ni estado preactivado.

La evidencia de construcción, inspección, pruebas automatizadas y hashes se entrega separadamente en VALIDACION.json y validacion/ del directorio de distribución. Estos resultados deben distinguirse de instalación nativa, activación real y requisitos externos; no sustituyen pruebas de servicios reales en Windows/Ubuntu/macOS ni firma comercial o notarización.

Las instrucciones de instalación, diagnóstico, red y desinstalación se encuentran en [Producción](PRODUCCION.md). Los informes históricos y los inventarios funcionales anteriores se conservan como investigación, sin distribuirlos como manuales vigentes. No se generan manuales Word/PDF en esta transición.
