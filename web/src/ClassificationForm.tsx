import { useState } from 'react';
import { api, message } from './api';
import { Notice, PageEnd, usePage } from './components';
import type { Document, Library } from './libraryTypes';
import type { CaseRecord, CatalogEntry } from './organizationTypes';

export function ClassificationForm({ document, preferredCaseID = '', saved }: { document: Document; preferredCaseID?: string; saved: (document: Document) => void }) {
  const libraries = usePage<Library>('/libraries'); const library = libraries.items.find(library => library.id === document.library_id);
  const categories = usePage<CatalogEntry>(`/libraries/${document.library_id}/categories`); const types = usePage<CatalogEntry>(`/libraries/${document.library_id}/document-types`);
  const [query, setQuery] = useState(''); const cases = usePage<CaseRecord>(`/libraries/${document.library_id}/cases?q=${encodeURIComponent(query)}`);
  const [caseID, setCaseID] = useState(document.case_id || preferredCaseID); const [categoryID, setCategoryID] = useState(document.category_id); const [typeID, setTypeID] = useState(document.document_type_id);
  const [fields, setFields] = useState(Object.entries(document.metadata || {})); const [error, setError] = useState(''); const [busy, setBusy] = useState(false);
  const changingCase = document.case_id !== caseID;
  const operation = changingCase && document.case_id ? 'reassign' : changingCase && document.storage_source === 'linked' ? 'associate' : 'classification';
  const allowed = operation === 'reassign' ? document.can_reassign : operation === 'associate' ? document.can_associate : document.can_classify;
  if (!library || library.mode === 'linked') return null;
  return <details className="classification-panel" open={!!preferredCaseID}><summary>Clasificar o asociar a expediente</summary><Notice text={error || categories.error || types.error || cases.error} />
    <form className="filters" onSubmit={event => { event.preventDefault(); setQuery(String(new FormData(event.currentTarget).get('query') || '')); }}><label>Buscar expediente<input name="query" maxLength={160} /></label><button className="secondary">Buscar identificador</button></form>
    <form className="form-grid" onSubmit={async event => { event.preventDefault(); const data = new FormData(event.currentTarget); setBusy(true); setError(''); try { await api(`/documents/${document.id}/${operation}`, { method: operation === 'classification' ? 'PATCH' : 'POST', revision: document.revision, body: { case_id: caseID, category_id: categoryID, document_type_id: typeID, title: data.get('title'), metadata: Object.fromEntries(fields.filter(([key]) => key.trim())), reason: data.get('reason') || '' } }); saved(await api<Document>(`/documents/${document.id}`)); } catch (error) { setError(message(error)); } finally { setBusy(false); } }}>
      {library.settings.cases_enabled && <label>Expediente de destino<select value={caseID} onChange={event => setCaseID(event.target.value)}><option value="">Sin expediente</option>{document.case_id && !cases.items.some(item => item.id === document.case_id) && <option value={document.case_id}>{document.case_identifier}</option>}{cases.items.map(item => <option key={item.id} value={item.id}>{item.identifier}{item.exercise ? ` · ${item.exercise}` : ''}</option>)}</select></label>}
      <div className="editor-grid"><label>Categoría del documento<select value={categoryID} onChange={event => { setCategoryID(event.target.value); setTypeID(''); }} required={!!caseID}><option value="">Sin categoría</option>{categories.items.filter(item => !item.archived || item.id === document.category_id).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label>Tipo del documento<select value={typeID} onChange={event => setTypeID(event.target.value)} required={!!caseID}><option value="">Sin tipo</option>{types.items.filter(item => item.category_id === categoryID && (!item.archived || item.id === document.document_type_id)).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label></div>
      <label>Título descriptivo<input name="title" defaultValue={document.title} maxLength={250} required={types.items.find(item => item.id === typeID)?.requires_descriptive_title} /></label>
      {operation === 'associate' && <p>El archivo conserva su ubicación original y el texto ya extraído.</p>}{operation === 'reassign' && <label>Motivo de reasignación<input name="reason" required maxLength={1000} /></label>}
      <details><summary>Campos adicionales</summary>{fields.map(([key, value], index) => <div className="metadata-row" key={index}><label>Campo {index + 1}<input value={key} maxLength={60} onChange={event => setFields(fields.map((field, position) => position === index ? [event.target.value, value] : field))} /></label><label>Valor {index + 1}<input value={value} maxLength={500} onChange={event => setFields(fields.map((field, position) => position === index ? [key, event.target.value] : field))} /></label><button type="button" className="secondary" onClick={() => setFields(fields.filter((_, position) => position !== index))}>Quitar campo</button></div>)}{fields.length < 20 && <button type="button" className="secondary" onClick={() => setFields([...fields, ['', '']])}>Añadir campo</button>}</details>
      <button disabled={busy || !allowed}>{operation === 'associate' ? 'Asociar archivo' : operation === 'reassign' ? 'Reasignar archivo' : 'Guardar clasificación'}</button>
    </form><PageEnd {...cases} empty={false} />
  </details>;
}
