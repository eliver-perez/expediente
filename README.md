# AIBID

**Aplicación de Indexación de Bibliotecas Digitales**  
Tu biblioteca digital, ordenada y al alcance.

Producto comercial genérico de instalación local, con Go + React + SQLite WAL/FTS5
y acceso por LAN. **H5 implementado:** revisión, aprobación/rechazo,
guardado definitivo recuperable y verificación de integridad administrada. Incluye
bibliotecas híbridas, cargas privadas, expedientes, catálogos y clasificación. Conserva
identidad, auditoría, bibliotecas vinculadas, vigilancia y OCR de H2/H3.
Contraseñas de **6–128 caracteres**. Detalles: [entrega H5](docs/H5.md).
Licencias comerciales corresponden a H6 e instaladores/servicios del SO a H7.

Los cuatro logotipos SVG suministrados están en `web/public/assets/brand/`.
Se conservan el identificador técnico `gestor_documental`, el nombre de los
binarios y las rutas de estado existentes para mantener compatibilidad.

## Ejecutar en macOS / VS Code

Desde esta carpeta, con Go y Node 22.12+:

```sh
make deps
make build-dev
./build/gestor-documental-dev init
./build/gestor-documental-dev bootstrap
./build/gestor-documental-dev serve
```

Si ya utilizas H2/H3/H4, conserva su configuración y administrador: ejecuta únicamente
`make build-dev` y `serve`, con el servicio anterior detenido. La migración es automática.
El binario `-dev` habilita funciones documentales para estas pruebas; no es una release
comercial. `make build` sigue excluyendo el permiso simulado de licencia.

OCR requiere las herramientas PDF/Tesseract indicadas en [H3](docs/H3.md).
Abrir **http://127.0.0.1:8090**. `bootstrap` pide usuario, nombre y contraseña sin
mostrarla. No hay cuenta ni contraseña predeterminadas. `init` crea configuración
privada en el directorio de configuración del usuario, fuera de `htdocs`, sin
sobrescribir archivos existentes. Detalles de rutas y HTTPS: [INSTALL.md](INSTALL.md).

Para consultar bitácoras globales, el administrador puede asignarse explícitamente
**Auditor de accesos** desde Usuarios → Editar → Permisos globales. Mi cuenta
siempre muestra el historial propio.

```sh
make check
make test-e2e
```

El primer comando compila React, valida formato, ejecuta vet, pruebas Go con detector
de carreras y las comprobaciones SQL H1. El segundo requiere Chrome instalado;
usa servidor temporal en `127.0.0.1:8099`, cuentas ficticias y DB desechable.
Detalle del módulo y limitaciones: [entrega H5](docs/H5.md).

## Revisar primero

1. [Decisiones y alternativas](DECISIONS.md): D-01 a D-05 aprobadas.
2. [Arquitectura](ARCHITECTURE.md) y [modelo de datos](SCHEMA.md).
3. [Estados](docs/STATES.md), [permisos](docs/PERMISSIONS.md) y [pantallas](docs/UI.md).
4. [API interna](API.md), [seguridad](SECURITY.md) y [rutas/vigilancia](docs/STORAGE_WATCHER.md).
5. [Hitos y aceptación](docs/MILESTONES.md), [instalación](INSTALL.md) y
   [contrato externo de licencias](LICENSE_CONTRACT.md).

El [alcance recibido](docs/REQUIREMENTS.md) y el anexo de licencia se conservan
como referencia. Ninguna propuesta amplía ese alcance.

## Estructura inicial

```text
cmd/gestor-documental/  CLI y servidor Go
internal/               Configuración, SQLite, identidad, auditoría, HTTP y pruebas
web/                    React/TypeScript y pruebas de navegador
db/schema.proposed.sql  Modelo de referencia de V1; NO es una migración
db/migrations/          Migraciones H2/H3/H4/H5, checksum y reversión protegida
docs/                   Especificaciones y criterios de aceptación
packaging/              Instaladores por SO, a implementar en H7
testdata/license/       Vectores de desarrollo del contrato, a crear en H6
scripts/check_design.py Verificación aislada del diseño SQL y enlaces
```

No se crean módulos con lógica ficticia. Cada hito incorporará solo los archivos,
migraciones y pruebas necesarios para entregar su funcionalidad completa.

## Verificar esta entrega

Desde esta carpeta, en la terminal de Visual Studio Code:

```sh
python3 scripts/check_design.py
```

Requiere Python 3.9+ con su módulo `sqlite3` con FTS5. Usa una base temporal local y la elimina
al terminar; no inicializa el producto ni descarga paquetes. Comprueba integridad,
restricciones relevantes, sesión única, conservación y publicación de texto FTS,
y enlaces locales de documentación. Se ejecuta también dentro de make check.

El siguiente hito es **H6: cliente de licencias V1.0 online/offline**,
después de revisar esta entrega de H5.
