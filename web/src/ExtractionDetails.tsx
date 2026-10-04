import { diagnosticLabel } from './libraryTypes';

export interface ProcessingRun {
  id: string; content_version_id: string; extractor: { id: string; version: string; format: string };
  extractor_revision: string; started_at: string; completed_at: string; status: string; error_code: string;
  unit_count: number; summary: Record<string, unknown>; warnings: string[];
}
const summaries: Record<string, string> = {
  pages: 'Páginas', ocr_pages: 'Páginas con OCR', paragraphs: 'Párrafos', tables: 'Tablas',
  section_properties: 'Definiciones de sección', parts: 'Partes leídas', sheets: 'Hojas', cells: 'Celdas con contenido o fórmula',
  formula_cells: 'Celdas con fórmula', formulas_without_cached_value: 'Fórmulas sin valor guardado',
  encoding: 'Codificación', lines: 'Líneas', characters_without_line_breaks: 'Caracteres sin saltos de línea',
  delimiter: 'Delimitador', records: 'Registros (incluida la primera fila)', columns: 'Columnas', nonempty_cells: 'Celdas con contenido', header: 'Encabezado',
};
const warnings: Record<string, string> = {
  CSV_HEADER_INFERRED: 'La primera fila parece un encabezado. Se utilizó como referencia y también se conservó íntegra en el índice.',
  EXTERNAL_REFERENCES_IGNORED: 'Las referencias externas no se consultaron.',
  TRACKED_DELETIONS_IGNORED: 'Se omitió el texto marcado como eliminado en el control de cambios.',
  FIELD_INSTRUCTIONS_IGNORED: 'Se conservaron los textos visibles de campos sin ejecutar sus instrucciones.',
  EMBEDDED_TEXT_PART_IGNORED: 'No se extrajo una parte de contenido incrustada.',
  NON_WORKSHEET_SHEET_IGNORED: 'Se omitieron hojas que no contienen celdas, como hojas de gráficos.',
  CELL_FORMATTING_NOT_APPLIED: 'Se leyeron valores guardados sin aplicar el formato visual de Excel; las fechas pueden aparecer como números de serie.',
  FORMULAS_NOT_EVALUATED: 'Las fórmulas no se recalcularon. Se usaron sus valores guardados, que pueden estar desactualizados.',
  FORMULA_VALUE_UNAVAILABLE: 'Hay fórmulas sin un resultado guardado; su valor no pudo indexarse.',
};
export function ExtractionDetails({ run }: { run?: ProcessingRun }) {
  if (!run) return null;
  const status: Record<string, string> = { complete: 'Completado', complete_with_warnings: 'Completado con advertencias', failed: 'Error', superseded: 'Sustituido' };
  return <details className="document-accordion"><summary>Último procesamiento · {status[run.status] || run.status}</summary>
    <p>Extractor: {run.extractor.id || run.extractor_revision}{run.extractor.version && ` · versión ${run.extractor.version}`} · {run.extractor.format.toUpperCase()}</p>
    {run.completed_at && <p>Finalizado: {new Date(run.completed_at).toLocaleString('es-MX')}</p>}
    {run.error_code && <p role="status">{diagnosticLabel[run.error_code] || run.error_code}</p>}
    <dl>{Object.entries(run.summary).filter(([key]) => summaries[key]).map(([key, value]) => <div key={key}><dt>{summaries[key]}</dt><dd>{value === 'inferred_first_record' ? 'Primera fila probable (inferencia)' : value === 'not_declared' ? 'No declarado; la primera fila se conserva como contenido' : value === '\t' ? 'Tabulador' : String(value)}</dd></div>)}</dl>
    {run.warnings.length > 0 && <ul>{run.warnings.map(code => <li key={code}>{warnings[code] || code}</li>)}</ul>}
  </details>;
}
