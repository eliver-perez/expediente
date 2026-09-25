# Matriz de permisos propuesta

Roles son plantillas de asignación, no comprobaciones rígidas en componentes.
Go verifica claves de permiso, biblioteca, recurso y licencia en cada operación.
`✓` concedido por plantilla; `—` ausente. Asignar más de un rol suma permisos en
esa biblioteca. Sin asignación no hay acceso. Permisos por grupo/área/tipo son una
extensión opcional del alcance; no se simulan con filtros solo en React.

## Roles por biblioteca

| Permiso | Lector | Colaborador | Revisor | Gestor de biblioteca | Auditor |
| --- | --- | --- | --- | --- | --- |
| `documents.read`, `search.execute` | ✓ | ✓ | ✓ | ✓ | — |
| `documents.download` | ✓ | ✓ | ✓ | ✓ | — |
| `storage.view_paths` | — | — | — | ✓ | — |
| `documents.upload`, `documents.classify` | — | ✓ | — | ✓ | — |
| `documents.submit`, `documents.cancel_own` | — | ✓ | — | ✓ | — |
| `documents.associate` | — | ✓ | — | ✓ | — |
| `documents.reassign` | — | — | — | ✓ | — |
| `documents.review`, `documents.approve`, `documents.reject` | — | — | ✓ | — | — |
| `documents.finalize` (sin flujo de revisión) | — | — | — | ✓ | — |
| `documents.approve_own` (excepción auditada) | — | — | — | — | — |
| `documents.remove_index` | — | — | — | ✓ | — |
| `cases.manage`, `requirements.edit` | — | — | — | ✓ | — |
| `catalogs.manage`, `templates.manage` | — | — | — | ✓ | — |
| `libraries.configure`, `storage.manage_roots` | — | — | — | ✓ | — |
| `indexing.run`, `indexing.retry` | — | — | — | ✓ | — |
| `permissions.manage_library` | — | — | — | ✓ | — |
| `audit.read_library` | — | — | — | ✓ | ✓ |
| `audit.search_details` | — | — | — | — | ✓ |

Estos son valores propuestos, configurables antes de desplegar. Dar `approve_own`
no concede `approve`; exige ambos. Crear roles/asignaciones exige no poder elevar
privilegios fuera del alcance concedido. Un gestor no se concede a sí mismo un
permiso global ni una biblioteca ajena. El permiso de auditor no concede lectura
del PDF ni viceversa. Para ver consultas exactas se requieren lectura de bitácora
y `audit.search_details` en las bibliotecas consultadas; eventos globales con
múltiples bibliotecas se presentan solo a auditores autorizados para todas ellas.

## Permisos globales

| Perfil | Permisos iniciales |
| --- | --- |
| Administrador de instalación | `users.manage`, `users.reset_password`, `sessions.revoke`, `libraries.create`, `permissions.manage_global`, `license.manage`, `system.configure`, `backup.manage`, `restore.manage` |
| Auditor de accesos de organización | `sessions.read_all`, `authentication_attempts.read`, `audit.read_global` |
| Cualquier usuario autenticado | Mi cuenta, cambiar su contraseña actual, historial de sesiones propio, cerrar su sesión |

Restaurar exige control local adicional y mantenimiento; el permiso no permite
sobrescribir silenciosamente la instalación por una petición remota. Administrador
de instalación puede asignar gestores, acción auditada; no recibe acceso documental
por su título. La exportación integral/backup es un privilegio sensible, separado
de `documents.download`; conservarlo en read-only no lo concede a cualquier usuario.

## Capacidades y políticas adicionales

| Operación | Capacidad | Restricción extra |
| --- | --- | --- |
| Añadir/verificar raíz linked, indexar | `linked_libraries` | Raíz propia, sin colisión y permiso de gestión/indexación |
| Ejecutar OCR | `ocr` y capacidad del origen | Leer texto ya retenido sigue posible al perder módulo |
| Cargar/materializar | `managed_libraries` | Destino accesible, cuota de recursos; `expedientes` si usa expediente |
| Crear expediente/asociar | `expedientes` + capacidades de modalidad | Documento de misma biblioteca y aún libre, o permiso de reasignación |
| Revisar/aprobar/rechazar | `review_workflow` + `expedientes` | Estado y revisión esperada; separación de autor/revisor según D-04 |
| Consultar/abrir/descargar | No requiere recuperar módulos perdidos | Permiso de contenido y licencia válida o estado read-only permitido |
| Cambiar catálogos/rutas/clasificación | Capacidades aplicables | Bloqueado por vencimiento/revocación |

Sin activación o firma inválida, lectura general no se habilita por defecto:
asistente y exportación administrativa de recuperación. Auth, seguridad, auditoría
y diagnóstico necesarios pueden operar sin escritura documental. Los pendientes se
filtran por biblioteca, permiso de revisar y asignación al usuario (o cola compartida
de revisores autorizados); no imponer ejercicio ni tipo de recurso.

En híbrida, las escrituras comprueban además ambas capacidades de modalidad
(`linked_libraries` y `managed_libraries`); no se introduce una clave `hybrid`.

## Roles implementados en H4

Colaborador añade carga, clasificación, asociación y cancelación propia a lectura.
Revisor añade consulta de temporales ajenos en su biblioteca; las decisiones de
aprobación/rechazo se entregarán en H5. Gestor administra catálogos, plantillas,
expedientes, requerimientos y reasignaciones, y puede cargar/clasificar; no tiene
acceso implícito a temporales ajenos. Los perfiles pueden combinarse expresamente.
Los controles de privacidad preceden a los totales y a la paginación de resultados.

## Roles implementados en H5

Colaborador y gestor añaden `documents.submit`; gestor añade `documents.finalize`.
Revisor añade `documents.approve` y `documents.reject`. La consulta de la bandeja
requiere `documents.review` y filtra asignación/cola compartida. Ningún rol estándar
recibe `documents.approve_own`; concederlo explícitamente no elimina la auditoría
ni la comprobación de aprobación. El revisor que autorizó puede reintentar su
materialización; el gestor necesita `indexing.retry` y acceso al documento.
