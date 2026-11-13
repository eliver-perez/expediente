# AIDEX

**Aplicación de Indexación Digital de Expedientes**
Tu biblioteca digital, ordenada y al alcance.

**AIDEX 1.0**, primera versión comercial; versión técnica **1.0.0**.
[Instalación y actualización](docs/PRODUCCION.md).
[Transición, codificación y validación](docs/TRANSICION-PRODUCCION.md).
Paquetes nuevos: `dist/aidex-1.0.0/`; entregas anteriores conservadas como evidencia histórica.

Producto comercial genérico de instalación local, con Go + React + PostgreSQL 18 con búsqueda GIN
y acceso por LAN. **H6 implementado:** cliente de licencias V1.0 online/offline, con simulador HTTPS,
control de módulos y solo lectura al vencer. Incluye revisión, guardado definitivo
recuperable, verificación de integridad,
bibliotecas híbridas, cargas privadas, expedientes, catálogos y clasificación. Conserva
identidad, auditoría, bibliotecas vinculadas, vigilancia y OCR de H2/H3.
Contraseñas de **6–128 caracteres**. Detalles y prueba del simulador: [entrega H6](docs/H6.md).
Los instaladores actuales usan el nombre **AIDEX**, para Mac Apple Silicon,
Windows y Ubuntu 24.04 (amd64/arm64), conservando el puerto 18090. [H7](docs/H7.md) conserva las instrucciones
históricas del canal de pruebas; las instrucciones vigentes están en la guía de instalación.
macOS Intel queda excluido.

Los cuatro logotipos SVG suministrados están en `web/public/assets/brand/`.
Se conservan el identificador técnico `gestor_documental`, las importaciones técnicas de Go y el contrato V1. Las nuevas instalaciones tienen rutas propias de AIDEX y no importan datos anteriores.

## Ejecutar en macOS / VS Code

Desde esta carpeta, con Go y Node 22.12+:

```sh
make deps
make build-dev
./build/aidex-server-dev init
./build/aidex-server-dev bootstrap
./build/aidex-server-dev serve
```

El desarrollo usa su configuración propia bajo el directorio de usuario aidex;
no importa automáticamente configuraciones anteriores.
El binario `-dev` permite las pruebas anteriores mientras no se importe una licencia.
Al activarla aplica sus condiciones. `make build` excluye claves y herramientas de
desarrollo; requiere licencia firmada por una clave de producción configurada.

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
Detalle del módulo y limitaciones: [entrega H6](docs/H6.md).

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
internal/               Configuración, PostgreSQL, identidad, auditoría, HTTP y pruebas
web/                    React/TypeScript y pruebas de navegador
db/migrations/          Esquema inicial y evolución de PostgreSQL
docs/                   Especificaciones y criterios de aceptación
packaging/              Instaladores y servicios por SO
testdata/license/       Vectores V1 versionados para compartir con el servidor
scripts/test_postgresql.sh Pruebas con PostgreSQL temporal aislado
```

No se crean módulos con lógica ficticia. Cada hito incorporará solo los archivos,
migraciones y pruebas necesarios para entregar su funcionalidad completa.

## Verificar esta entrega

Desde esta carpeta, en la terminal de Visual Studio Code:

```sh
scripts/test_postgresql.sh
```

El arnés inicia PostgreSQL 18 real en un directorio temporal privado, sin TCP,
ejecuta las pruebas Go y el detector de carreras y destruye solo su clúster.
Los binarios PostgreSQL son herramientas de desarrollo; el producto final los
incluirá en su paquete. No requiere Python SQLite ni conversiones de datos.
Consulta [la migración PostgreSQL](docs/POSTGRESQL.md) para conocer la arquitectura,
el aprovisionamiento autónomo y las pruebas verificadas.

La entrega actual es **AIDEX 1.0**. Consulta la guía de transición para conocer las verificaciones ejecutadas y las pruebas nativas pendientes.
