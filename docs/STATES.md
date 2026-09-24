# Estados independientes

Una sola columna «estado» no representa disponibilidad, integridad, revisión y OCR.
Estos diagramas son contratos a implementar, no flujos disponibles todavía.

## Disponibilidad física e integridad

```mermaid
stateDiagram-v2
  [*] --> staged: carga managed privada
  [*] --> available: archivo linked reconocido
  staged --> available: materialización verificada + commit
  available --> missing: ausencia confirmada con raíz accesible
  available --> unknown: raíz inaccesible o permisos perdidos
  unknown --> available: verificación satisfactoria
  unknown --> missing: escaneo completo confirma ausencia
  missing --> available: reaparece y se reconoce identidad
```

La inaccesibilidad de una raíz domina la disponibilidad **efectiva** de sus archivos:
se muestra `unknown` sin reescribir miles de filas ni declarar borrados. Si existe
otro alias autorizado, accesible y verificado, puede usarse. `integrity_status` es
`unknown`, `verified` o `changed`; una identidad managed aprobada con integridad
alterada nunca se presenta como su versión aprobada vigente.

| Suceso | Origen | Aprobación | Disponibilidad / cumplimiento |
| --- | --- | --- | --- |
| Modificación externa confirmada | linked | Se conserva | Nueva versión, aviso único, OCR anterior hasta reemplazo completo |
| Original desaparece | linked | Se conserva | No disponible; texto/FTS retenidos; cumplimiento separado |
| Modificación externa confirmada | managed | `needs_review` si aprobado | Alerta, no cuenta como aprobado válido, conserva historia |
| Archivo definitivo desaparece | managed | Evento previo se conserva | No disponible; no válido para cumplimiento hasta restaurar/verificar |
| NAS inaccesible | ambos | Sin cambio masivo | Estado efectivo desconocido; pausar/verificar |
| Cambio producido por materialización registrada | managed | Según operación | Correlación por operación+hash evita alarma externa falsa |

## Revisión documental

```mermaid
stateDiagram-v2
  [*] --> draft: indexación o carga
  draft --> pending_review: envío con clasificación válida
  pending_review --> draft: cambiar clasificación e invalidar revisión anterior
  pending_review --> rejected: motivo obligatorio
  pending_review --> materializing: aprobar managed aún temporal
  materializing --> approved: destino íntegro y commit
  pending_review --> approved: aprobar asociación linked
  pending_review --> approved: revisar managed ya definitivo y verificar bytes
  draft --> materializing: confirmar managed sin revisión habilitada
  draft --> approved: asociación linked sin revisión inicial
  rejected --> draft: corregir y reenviar
  draft --> cancelled: cancelar
  pending_review --> cancelled: cancelar con permiso
  approved --> needs_review: managed alterado externamente
  needs_review --> pending_review: presentar nueva decisión
```

Fallo de copia deja `materializing` con operación fallida/reintentable; no se anuncia
aprobación final. Cancelar una materialización requiere reconciliar cualquier
publicación ya existente; se habilitará solo con ese protocolo completo. Edición
de clasificación durante revisión invalida esa solicitud (revisión optimista) y
requiere reenvío; nunca aprobar metadatos diferentes de los revisados.

Un linked asociado tiene revisión inicial si lo exige su biblioteca. Su siguiente
modificación externa no vuelve a `pending_review` ni `needs_review`. Si cambia
mientras se revisa inicialmente, el revisor debe confirmar qué versión observó.
Un indexed linked sin clasificación permanece `draft`; eso no impide buscarlo.
Baja lógica `deleted_at` es independiente; no borrar aprobación/auditoría histórica.

## OCR y cola

```mermaid
stateDiagram-v2
  [*] --> queued
  queued --> running: adquirir lease y fencing token
  running --> succeeded: publicación completa y versión vigente
  running --> retry_wait: error transitorio y menos de 5 intentos
  retry_wait --> running: backoff cumplido
  running --> failed: error permanente o intento 5
  running --> queued: recuperar lease tras fallo sin aceptar worker viejo
  queued --> paused: licencia, raíz o presupuesto
  paused --> queued: condición resuelta
  queued --> cancelled: operación retirada
```

Recuperar lease consume/controla intentos; jamás reinicia automáticamente el máximo.
Un trabajo obsoleto termina sin publicar. Reintento manual es un job nuevo con
`retry_of_job_id`; requiere permiso y evento. Las ejecuciones de extracción son
`pending/running/complete/failed/superseded`. La frescura del texto publicado es
`none/current/stale`; «stale» sigue consultable.

## Licencia local

```mermaid
stateDiagram-v2
  [*] --> unactivated
  unactivated --> active: aceptar JWS válido
  active --> grace: subscription y now >= expires_at
  grace --> expired: now >= expires_at + 15 días
  grace --> active: renovación válida
  expired --> active: renovación válida
  active --> revoked: JWS revocado autenticado
  grace --> revoked: JWS revocado autenticado
  expired --> revoked: JWS revocado autenticado
  active --> invalid: no existe evidencia local válida
  invalid --> active: recuperación validada
```

Una importación inválida se rechaza sin reemplazar un JWS válido existente. Un error
de red no produce `invalid`. Perpetua no transita por tiempo a `grace/expired`;
`maintenance_until` no vence el uso. Revocación offline solo se conoce al recibir
evidencia autenticada. `expired/revoked`: consulta, visualización, descarga y backup;
`invalid/unactivated`: recuperación administrativa, sin escrituras documentales.
