export interface DeletionProgress {
  phase: string;
  done: number;
  total: number;
  unit: string;
}

interface TrackedDeletion {
  kind: string;
  name: string;
  state: string;
  progress?: DeletionProgress;
  cleanup_diagnostics?: string[];
}

const failureDescriptions: Record<string, string> = {
  PROCESS_TIMEOUT: 'La ejecución anterior superó su tiempo permitido. Puedes reintentar la misma operación.',
  DELETION_PLAN_STALE: 'El contenido cambió desde que se preparó el plan. Inicia una eliminación nueva para revisar el alcance actualizado.',
  INTERRUPTED: 'El servicio se interrumpió. La operación volverá a la cola sin duplicar la eliminación.',
  SOURCE_UNAVAILABLE: 'La base no pudo completar la operación. Descarga el manifiesto y revisa el diagnóstico antes de reintentar.',
};

const descriptions: Record<string, string> = {
  queued: 'La eliminación está en espera. AIDEX la iniciará en segundo plano.',
  validating: 'Comprobando que el alcance y los registros no cambiaron desde la confirmación.',
  preparing_manifest: 'Preparando el manifiesto y la limpieza de archivos temporales.',
  stopping_jobs: 'Retirando las tareas de procesamiento relacionadas.',
  removing_documents: 'Retirando las referencias documentales internas.',
  removing_index: 'Retirando páginas extraídas y entradas del índice de búsqueda.',
  removing_extraction: 'Retirando las páginas de extracción por lotes. El contador muestra las páginas retiradas en la transacción actual.',
  rebuilding_search: 'Reconstruyendo el índice de búsqueda a partir de los documentos que permanecen.',
  finalizing: 'Finalizando las relaciones y el registro de la biblioteca o carpeta.',
  cleanup: 'Los registros se retiraron. Revisando únicamente archivos temporales de AIDEX.',
  complete: 'La eliminación terminó. Los documentos originales permanecen intactos.',
  failed: 'La operación se interrumpió y la transacción se revirtió. Consulta el diagnóstico y reintenta.',
};

const stageNames = ['Detener tareas relacionadas', 'Retirar registros documentales', 'Finalizar eliminación'];

function stageIndex(phase: string): number {
  if (['queued', 'validating', 'preparing_manifest', 'stopping_jobs'].includes(phase)) return 0;
  if (['removing_documents', 'removing_index', 'removing_extraction', 'rebuilding_search'].includes(phase)) return 1;
  return 2;
}

export function DeletionTracker({ item }: { item: TrackedDeletion }) {
  const noun = item.kind === 'library' ? 'biblioteca' : 'carpeta';
  const active = item.state === 'pending' || item.state === 'running';
  const phase = item.progress?.phase || (item.state === 'pending' ? 'queued' : item.state === 'running' ? 'validating' : item.state);
  const currentStage = stageIndex(phase);
  const progress = item.progress;
  const measurable = active && !!progress && progress.total > 0;
  const percent = measurable ? Math.min(100, Math.round(100 * progress!.done / progress!.total)) : null;
  const unit = progress?.unit === 'pages' ? 'páginas' : progress?.unit === 'jobs' ? 'tareas' : 'documentos';
  const heading = active ? `Eliminando ${noun} ${item.name}` : item.state === 'failed' ? `Eliminación de ${noun} ${item.name} interrumpida` : `${noun === 'biblioteca' ? 'Biblioteca' : 'Carpeta'} ${item.name} eliminada`;
  return <div className="deletion-tracker" role="status" aria-live="polite">
    <h3>{heading}</h3>
    <p>{descriptions[phase] || descriptions.validating}</p>
    {item.state === 'failed' && item.cleanup_diagnostics?.map(code => <p key={code} role="alert">{failureDescriptions[code] || code}</p>)}
    {active && <div className="deletion-meter">
      <progress aria-label={`Avance de la eliminación de ${noun}`} max={measurable ? progress!.total : undefined} value={measurable ? progress!.done : undefined} />
      {measurable && <span>{progress!.done.toLocaleString()} de {progress!.total.toLocaleString()} {unit} · {percent} %</span>}
      {!measurable && <span>Actividad en curso; esta etapa no tiene un total calculable.</span>}
    </div>}
    <ol className="deletion-stages">
      {stageNames.map((name, index) => {
        const state = item.state === 'complete' ? 'done' : item.state === 'failed' ? 'pending' : index < currentStage ? 'done' : index === currentStage ? 'current' : 'pending';
        return <li key={name} className={`deletion-stage-${state}`}>{name} <span>{state === 'done' ? 'Terminada' : state === 'current' ? 'En curso' : 'Pendiente'}</span></li>;
      })}
    </ol>
  </div>;
}
