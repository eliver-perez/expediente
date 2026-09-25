# Mapa de pantallas y bocetos

Bocetos funcionales para revisar; React se implementará por flujo completo. Todas
las acciones dependen de Go. Ocultar un botón no sustituye permisos/licencia.

| Pantalla / ruta React | Contenido y acciones |
| --- | --- |
| `/login` | Inicio, error genérico, aviso de sesión reemplazada |
| `/account` | Nombre, cambio de contraseña con la actual, historial propio y motivos de cierre |
| `/libraries` | Bibliotecas autorizadas, modalidad, salud de raíces, búsqueda global autorizada |
| `/libraries/:id/explorer` | Árbol de raíces/vistas, filtros disponibles, resultados por nombre/ruta/contenido, aviso de ausentes |
| `/documents/:id` | Ficha, clasificación, disponibilidad/aprobación/OCR separados, visor o texto retenido, historial |
| `/libraries/:id/uploads` | Multifile/arrastrar, lote y progreso individual, preview privado, clasificación y envío |
| `/libraries/:id/cases` | Lista, identificador con etiqueta configurable, avance y pendientes |
| `/cases/:id` | Requisitos propios desde creación, estados reales, archivos, asociar existente sin copia |
| `/pending` | Solo responsabilidades autorizadas, preview, revisión, motivo de rechazo, línea de tiempo |
| `/libraries/:id/settings` | Modalidad/capacidades, etiqueta, ejercicio opcional, política y nomenclatura con preview |
| `/libraries/:id/roots` | Rutas del servidor, escaneos, verificar ahora, conflictos, vistas y planes de consolidación/retiro |
| `/libraries/:id/catalogs` | Categorías/tipos, Varios con título obligatorio, plantillas versionadas |
| `/admin/users` | Cuentas, roles por biblioteca, restablecer, revocar; no mostrar contraseñas existentes |
| `/admin/events` | Filtros de evento/actor/fecha/recurso; consultas exactas y filtros solo con permiso sensible |
| `/admin/access` | Historial de sesiones e intentos fallidos, IP/agente como observaciones |
| `/admin/license` | Estado/módulos, activar/refrescar, exportar `.licreq`, importar `.lic`, desactivación y diagnóstico |
| `/admin/operations` | Cola/errores, reintentos, configuración de escucha, backup y recuperación autorizada |

## Explorador

```text
Biblioteca: Documentos del cliente   [Híbrida]   [Cargar] [Verificar ahora]
Buscar: [ frase, archivo o identificador                     ] [Buscar]
Categoría [Todas]  Tipo [Todos]  Identificador [     ]  Ejercicio [si existe]

Carpetas y vistas             Resultado          Disponibilidad  Aprobación
▾ Raíz A                     Informe.pdf p. 4   Disponible      Aprobado
  └─ Proyectos               «...texto hallado...»
▾ Raíz B [Inaccesible]        Plano.pdf p. 2     No disponible   Aprobado
  └─ Vista: Archivo          «...OCR retenido...»

Estado de escaneo: Raíz C, en curso [Ver progreso]
```

Vista global no filtra por bibliotecas que el usuario no puede consultar. Mostrar
página y extractos escapados; click abre esa página si existe original accesible.
Descarga es una acción separada. Ausente abre ficha/texto retenido, sin visor vacío
ni descarga falsa. Ruta visible solo con permiso; temporal nunca revela ruta.

## Documento y revisión

```text
Título descriptivo                  [Visualizar] [Descargar si autorizado]
Origen: Vinculado   Disponible: No   Revisión: Aprobado   OCR: Conservado
Expediente: PR-008  Categoría: Planos  Tipo: Plano general
Aviso: Original no disponible. Se conserva el texto de la última extracción.

[Ficha] [Texto por página] [Historial autorizado]
Página 2: texto nativo/OCR retenido...
```

```text
Pendiente de revisión: lote de 4 PDF
Documento | Clasificación | Estado de archivo | Revisor | Acción
Plano     | PR-008/Planos | Temporal privado  | Usuario | [Revisar]

Preview PDF                         Decisión: [Aprobar] [Rechazar]
                                    Motivo [obligatorio al rechazar]
Línea de tiempo: cargado → clasificado → enviado → materializando → aprobado
```

No mostrar `approved` durante `materializing`; errores ofrecen diagnóstico y
reintento autorizado. Biblioteca sin revisión ofrece «Confirmar incorporación»
con permiso distinto; sigue verificando destino. Asociar linked indica «Se conserva
su ubicación original» y selecciona solo documentos libres de esa biblioteca.

## Gestión de raíces y estructura

```text
Añadir carpeta del servidor [ruta elegida por administrador] [Comprobar]
Resultado: esta carpeta pertenece a la raíz «Archivo».
[Crear vista en su biblioteca]      [Cancelar]

Destino de archivos nuevos: [raíz administrada autorizada]
Estructura: [{identificador}/{categoria}]
Nombre: [{tipo_documento}_{consecutivo}]
Vista previa: PR-008/Planos/Plano_general_0001.pdf
Los cambios se aplicarán a incorporaciones futuras.
```

Conflicto con otra biblioteca sin permiso no revela su ruta/contenido; identifica
que requiere administrador con acceso a ambas. Retirar raíz muestra recuentos,
expedientes afectados y opción explícita de conservar/quitar referencias.

## Estados y accesibilidad

Carga, vacío, error, paginación y reintento definidos en cada pantalla. Indicadores
no dependen solo de color; etiquetas, navegación por teclado y foco tras errores.
Mensaje de sesión exacto en [SECURITY.md](../SECURITY.md). Licencia en gracia avisa
al administrador; read-only explica el motivo sin ocultar descarga/respaldo permitidos.
No se añade panel de ventas, campos institucionales ni filtros obligatorios ajenos.

## Pantallas entregadas H4

AIBID incorpora Cargas (arrastre multifile, progreso, reintento y lotes), Expedientes
(plantilla, requisitos propios, avance y asociación de vinculados), Catálogos
(categorías/tipos/versiones) y Configuración (modalidad, campos, patrones y preview).
La ficha clasifica/reasigna/cancela según permisos. El buscador incorpora filtros
de organización al seleccionar biblioteca, y el retiro advierte su impacto en
expedientes. Navegación responsive verificada a 390 px. Marca/eslogan y SVG locales
claros/oscuros se usan en login, navegación y cabecera. La revisión/materialización
se incorporará en H5; no hay acciones simuladas. [Recorrido de prueba](H4.md).

## Entrega H5

Configuración incorpora revisión independiente por origen y días de retención.
Revisiones muestra responsabilidades asignadas/compartidas de cada biblioteca.
La ficha presenta envío, confirmación directa, decisión con motivo y etapas del
guardado/reintento. Distingue integridad alterada, disponibilidad, aprobación y
frescura del texto. Procesamiento distingue verificación administrada y guardado.
Los avisos explican si un cambio administrado exige decisión o uno vinculado
conserva aprobación. [Recorrido H5](H5.md).
