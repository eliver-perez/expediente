import { useState } from 'react';
import { Accordion } from './Accordion';
import { api, message } from './api';
import { Notice, usePage } from './components';
import { useToastState } from './feedback';
import type { Library } from './libraryTypes';
import type { CatalogEntry, RequirementInput, Template } from './organizationTypes';

const sortNames = (a: { name: string }, b: { name: string }) => a.name.localeCompare(b.name, 'es', { sensitivity: 'base' });
const matches = (text: string, query: string) => text.toLocaleLowerCase('es').includes(query.trim().toLocaleLowerCase('es'));
const Pencil = () => <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L9 17l-4 1 1-4Z"/></svg>;

export function Catalogs({ library }: { library: Library }) {
  const categories = usePage<CatalogEntry>(`/libraries/${library.id}/categories`);
  const types = usePage<CatalogEntry>(`/libraries/${library.id}/document-types`);
  const [category, setCategory] = useState<CatalogEntry | null>(null);
  const [type, setType] = useState<CatalogEntry | null>(null);
  const [categoryKey, setCategoryKey] = useState(0);
  const [typeKey, setTypeKey] = useState(0);
  const [categoryQuery, setCategoryQuery] = useState('');
  const [typeQuery, setTypeQuery] = useState('');
  const [typeCategory, setTypeCategory] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useToastState();
  const [busy, setBusy] = useState(false);
  const canEdit = library.permissions.includes('catalogs.manage');
  const orderedCategories = [...categories.items].sort(sortNames);
  const visibleCategories = orderedCategories.filter(item => matches(item.name, categoryQuery));
  const visibleTypes = [...types.items].sort(sortNames).filter(item => (!typeCategory || item.category_id === typeCategory) && matches(`${item.name} ${categories.items.find(category => category.id === item.category_id)?.name || ''}`, typeQuery));

  async function save(kind: 'categories' | 'document-types', selected: CatalogEntry | null, data: FormData) {
    setBusy(true); setError('');
    try {
      await api(selected ? `/${kind}/${selected.id}` : `/libraries/${library.id}/${kind}`, {
        method: selected ? 'PATCH' : 'POST', revision: selected?.revision,
        body: { name: data.get('name'), category_id: selected?.category_id || data.get('category_id') || '', allows_multiple: data.get('multiple') === 'on', requires_descriptive_title: data.get('title_required') === 'on', archived: data.get('archived') === 'on' }
      });
      categories.reload(); types.reload();
      if (kind === 'categories') { setCategory(null); setCategoryKey(value => value + 1); }
      else { setType(null); setTypeKey(value => value + 1); }
      setSuccess('Catálogo guardado.');
    } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  return <>
    <Notice text={error || categories.error || types.error} /><Notice text={success} kind="success" />
    <div className="catalog-section-grid">
      <section className="card catalog-table-card"><div className="section-heading"><div><h2>Categorías documentales</h2><p className="muted">{categories.items.length} registradas</p></div></div>
        <label className="catalog-search">Buscar categoría<input type="search" value={categoryQuery} onChange={event => setCategoryQuery(event.target.value)} placeholder="Nombre de categoría" /></label>
        <div className="catalog-scroll table-wrap"><table className="compact-table"><thead><tr><th>Categoría</th><th className="catalog-actions-column">Acciones</th></tr></thead><tbody>{visibleCategories.map(item => <tr key={item.id}><td><strong>{item.name}</strong>{item.archived && <span className="badge catalog-inline-badge">Archivada</span>}</td><td className="catalog-actions-column">{canEdit && <button type="button" className="catalog-icon-button" aria-label={`Editar categoría ${item.name}`} title={`Editar ${item.name}`} onClick={() => setCategory(item)}><Pencil /></button>}</td></tr>)}</tbody></table></div>
        {!categories.busy && !visibleCategories.length && <p className="catalog-empty">{categoryQuery ? 'No hay categorías que coincidan con la búsqueda.' : 'Todavía no hay categorías. Crea la primera para organizar tus documentos.'}</p>}
        {categories.busy && <p className="catalog-empty" role="status">Cargando categorías…</p>}
      </section>
      {canEdit && <section className="card catalog-form-card"><p className="eyebrow">{category ? 'MODO EDICIÓN' : 'NUEVO REGISTRO'}</p><h2>{category ? 'Editar categoría' : 'Nueva categoría'}</h2>
        <form key={`${category?.id || 'new'}-${categoryKey}`} className="form-grid" onSubmit={event => { event.preventDefault(); void save('categories', category, new FormData(event.currentTarget)); }}>
          <label>Nombre de categoría<input name="name" defaultValue={category?.name || ''} required maxLength={160} /></label>
          {category && <label className="check"><input name="archived" type="checkbox" defaultChecked={category.archived} />Archivar categoría</label>}
          <div className="actions"><button disabled={busy}>{category ? 'Guardar categoría' : 'Crear categoría'}</button>{category && <button type="button" className="secondary" onClick={() => setCategory(null)}>Cancelar edición</button>}</div>
        </form>
      </section>}
    </div>
    <div className="catalog-section-grid">
      <section className="card catalog-table-card"><div className="section-heading"><div><h2>Tipos documentales</h2><p className="muted">{types.items.length} registrados</p></div></div>
        <div className="catalog-filters"><label>Buscar tipo<input type="search" value={typeQuery} onChange={event => setTypeQuery(event.target.value)} placeholder="Nombre o categoría" /></label><label>Filtrar por categoría<select value={typeCategory} onChange={event => setTypeCategory(event.target.value)}><option value="">Todas las categorías</option>{orderedCategories.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label></div>
        <div className="catalog-scroll table-wrap"><table className="compact-table"><thead><tr><th>Tipo documental</th><th>Categoría</th><th>Modalidad</th><th>Título descriptivo</th><th className="catalog-actions-column">Acciones</th></tr></thead><tbody>{visibleTypes.map(item => <tr key={item.id}><td><strong>{item.name}</strong>{item.archived && <span className="badge catalog-inline-badge">Archivado</span>}</td><td>{categories.items.find(category => category.id === item.category_id)?.name || '—'}</td><td>{item.allows_multiple ? 'Múltiples archivos' : 'Archivo único'}</td><td>{item.requires_descriptive_title ? 'Sí' : 'No'}</td><td className="catalog-actions-column">{canEdit && <button type="button" className="catalog-icon-button" aria-label={`Editar tipo ${item.name}`} title={`Editar ${item.name}`} onClick={() => setType(item)}><Pencil /></button>}</td></tr>)}</tbody></table></div>
        {!types.busy && !visibleTypes.length && <p className="catalog-empty">{typeQuery || typeCategory ? 'No hay tipos documentales con esos filtros.' : 'Todavía no hay tipos documentales. Crea una categoría y luego agrega un tipo.'}</p>}
        {types.busy && <p className="catalog-empty" role="status">Cargando tipos…</p>}
      </section>
      {canEdit && <section className="card catalog-form-card"><p className="eyebrow">{type ? 'MODO EDICIÓN' : 'NUEVO REGISTRO'}</p><h2>{type ? 'Editar tipo documental' : 'Nuevo tipo documental'}</h2>
        <form key={`${type?.id || 'new'}-${typeKey}`} className="form-grid" onSubmit={event => { event.preventDefault(); void save('document-types', type, new FormData(event.currentTarget)); }}>
          <label>Categoría del tipo<select name="category_id" defaultValue={type?.category_id || ''} disabled={!!type} required><option value="">Selecciona una categoría</option>{orderedCategories.filter(item => !item.archived || item.id === type?.category_id).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>Nombre del tipo<input name="name" defaultValue={type?.name || ''} required maxLength={160} /></label>
          <label className="check"><input type="checkbox" name="multiple" defaultChecked={type?.allows_multiple} />Admite múltiples archivos</label>
          <label className="check"><input type="checkbox" name="title_required" defaultChecked={type?.requires_descriptive_title} />Exige título descriptivo (por ejemplo, Varios)</label>
          {type && <label className="check"><input name="archived" type="checkbox" defaultChecked={type.archived} />Archivar tipo</label>}
          <div className="actions"><button disabled={busy}>{type ? 'Guardar tipo' : 'Crear tipo'}</button>{type && <button type="button" className="secondary" onClick={() => setType(null)}>Cancelar edición</button>}</div>
        </form>
      </section>}
    </div>
    {library.settings.cases_enabled && <Templates library={library} categories={categories.items} types={types.items} />}
  </>;
}

function Templates({ library, categories, types }: { library: Library; categories: CatalogEntry[]; types: CatalogEntry[] }) {
  const templates = usePage<Template>(`/libraries/${library.id}/templates`);
  const [selected, setSelected] = useState<Template | null>(null);
  const [requirements, setRequirements] = useState<RequirementInput[]>([]);
  const [formKey, setFormKey] = useState(0);
  const [availableCategory, setAvailableCategory] = useState('');
  const [availableQuery, setAvailableQuery] = useState('');
  const [templateQuery, setTemplateQuery] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useToastState();
  const [busy, setBusy] = useState(false);
  const canEdit = library.permissions.includes('templates.manage');
  const orderedCategories = [...categories].sort(sortNames);
  const availableTypes = [...types].sort(sortNames).filter(item => !item.archived && !categories.find(category => category.id === item.category_id)?.archived && (!availableCategory || item.category_id === availableCategory) && matches(`${item.name} ${categories.find(category => category.id === item.category_id)?.name || ''}`, availableQuery));
  const visibleTemplates = [...templates.items].sort(sortNames).filter(item => matches(item.name, templateQuery));
  const requiredUnits = requirements.reduce((total, item) => total + (Number.isFinite(item.required_count) ? item.required_count : 0), 0);
  const mandatoryCount = requirements.filter(item => item.mandatory).length;
  const summary = [
    { value: requirements.length, label: 'Tipos documentales', icon: '▤', tone: 'teal' },
    { value: requiredUnits, label: 'Documentos requeridos', icon: '◫', tone: 'blue' },
    { value: mandatoryCount, label: 'Requisitos obligatorios', icon: '✓', tone: 'green' },
    { value: requirements.length - mandatoryCount, label: 'Requisitos opcionales', icon: '◌', tone: 'amber' }
  ];
  function resetForm() { setSelected(null); setRequirements([]); setAvailableCategory(''); setAvailableQuery(''); setFormKey(value => value + 1); }
  function startVersion(template: Template) {
    setSelected(template);
    setRequirements(template.versions[0]?.requirements.map(item => ({ category_id: item.category_id, document_type_id: item.document_type_id, required_count: item.required_count, mandatory: item.mandatory })) || []);
    setAvailableCategory(''); setAvailableQuery(''); setError('');
  }
  async function saveTemplate(data: FormData) {
    setBusy(true); setError('');
    try {
      await api(selected ? `/templates/${selected.id}/versions` : `/libraries/${library.id}/templates`, { method: 'POST', revision: selected?.versions[0]?.revision, body: { name: data.get('name') || selected?.name, requirements } });
      templates.reload(); resetForm(); setSuccess('Versión de plantilla creada.');
    } catch (cause) { setError(message(cause)); } finally { setBusy(false); }
  }
  return <section className="card templates-section"><div className="section-heading"><div><h2>Plantillas de expedientes</h2><p className="muted">Cada expediente conserva sus requisitos. Una nueva versión se aplica a expedientes nuevos.</p></div></div>
    <Notice text={error || templates.error} /><Notice text={success} kind="success" />
    <label className="catalog-search">Buscar plantilla<input type="search" value={templateQuery} onChange={event => setTemplateQuery(event.target.value)} placeholder="Nombre de plantilla" /></label>
    <div className="template-list">{visibleTemplates.map(template => { const latest = template.versions[0]; const latestRequirements = latest?.requirements || []; return <Accordion key={template.id} title={<span className="template-summary-line"><strong>{template.name} · versión {latest?.revision || 1}</strong><span className="template-overview"><span className="badge positive">{latestRequirements.length} tipos</span><span className="badge">{latestRequirements.reduce((total, item) => total + item.required_count, 0)} documentos</span><span className="badge">{latestRequirements.filter(item => item.mandatory).length} obligatorios</span><span className="badge">{template.versions.length} versiones</span></span></span>} className="template-accordion">
      {canEdit && <button type="button" className="secondary compact" onClick={() => startVersion(template)}>Crear nueva versión de {template.name}</button>}
      {template.versions.map(version => <Accordion key={version.id} title={`Versión ${version.revision} · ${version.requirements.length} tipos`} className="template-version"><div className="table-wrap"><table className="compact-table"><thead><tr><th>Tipo documental</th><th>Categoría</th><th>Cantidad</th><th>Requisito</th></tr></thead><tbody>{version.requirements.map(item => <tr key={item.id}><td>{types.find(type => type.id === item.document_type_id)?.name || 'Tipo archivado'}</td><td>{categories.find(category => category.id === item.category_id)?.name || '—'}</td><td>{item.required_count}</td><td>{item.mandatory ? 'Obligatorio' : 'Opcional'}</td></tr>)}</tbody></table>{!version.requirements.length && <p className="catalog-empty">Sin requisitos en esta versión.</p>}</div></Accordion>)}
    </Accordion>; })}</div>
    {!templates.busy && !visibleTemplates.length && <p className="catalog-empty">{templateQuery ? 'No hay plantillas que coincidan con la búsqueda.' : 'Todavía no hay plantillas. Configura los requisitos para crear la primera.'}</p>}
    {templates.busy && <p className="catalog-empty" role="status">Cargando plantillas…</p>}
    {canEdit && <form key={`${selected?.id || 'new'}-${formKey}`} className="template-form" onSubmit={event => { event.preventDefault(); void saveTemplate(new FormData(event.currentTarget)); }}>
      <div className="template-form-heading"><div><p className="eyebrow">{selected ? 'NUEVA VERSIÓN' : 'NUEVA PLANTILLA'}</p><h3>{selected ? `Nueva versión de ${selected.name}` : 'Crear plantilla'}</h3></div>{selected && <button type="button" className="secondary compact" onClick={resetForm}>Cancelar versión</button>}</div>
      {!selected && <label className="template-name">Nombre de plantilla<input name="name" required maxLength={160} /></label>}
      <div className="template-builder">
        <div className="template-panel"><div className="template-panel-heading"><h3>Tipos disponibles</h3><span className="badge">{availableTypes.length}</span></div>
          <div className="template-filters"><label>Categoría<select value={availableCategory} onChange={event => setAvailableCategory(event.target.value)}><option value="">Todas las categorías</option>{orderedCategories.filter(item => !item.archived).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label>Buscar tipo<input type="search" value={availableQuery} onChange={event => setAvailableQuery(event.target.value)} placeholder="Nombre o categoría" /></label></div>
          <div className="template-scroll">{availableTypes.map(item => { const included = requirements.some(requirement => requirement.document_type_id === item.id); return <div className="template-type-row" key={item.id}><div><strong>{item.name}</strong><small>{categories.find(category => category.id === item.category_id)?.name || 'Sin categoría'} · {item.allows_multiple ? 'Múltiples archivos' : 'Archivo único'}</small></div><button type="button" className={included ? 'secondary compact' : 'compact'} disabled={included} aria-label={`${included ? 'Agregado' : 'Agregar'} ${item.name}`} onClick={() => setRequirements(current => [...current, { category_id: item.category_id, document_type_id: item.id, required_count: 1, mandatory: true }])}>{included ? 'Agregado' : '+ Agregar'}</button></div>; })}{!availableTypes.length && <p className="catalog-empty">No hay tipos disponibles con estos filtros.</p>}</div>
        </div>
        <div className="template-panel"><div className="template-panel-heading"><h3>Requisitos incluidos</h3><span className="badge positive">{requirements.length}</span></div>
          <div className="template-scroll">{requirements.map(item => { const documentType = types.find(type => type.id === item.document_type_id); return <div className="template-requirement-row" key={item.document_type_id}><div className="template-requirement-head"><div><strong>{documentType?.name || 'Tipo archivado'}</strong><small>{categories.find(category => category.id === item.category_id)?.name || 'Sin categoría'}</small></div><button type="button" className="text-button" aria-label={`Quitar ${documentType?.name || 'tipo'} de la plantilla`} onClick={() => setRequirements(current => current.filter(entry => entry.document_type_id !== item.document_type_id))}>Quitar</button></div><div className="template-requirement-controls"><label>Cantidad requerida<input type="number" min={1} max={documentType?.allows_multiple ? 1000 : 1} value={item.required_count} required onChange={event => setRequirements(current => current.map(entry => entry.document_type_id === item.document_type_id ? { ...entry, required_count: Number(event.target.value) } : entry))} /></label><label className="check"><input type="checkbox" checked={item.mandatory} onChange={event => setRequirements(current => current.map(entry => entry.document_type_id === item.document_type_id ? { ...entry, mandatory: event.target.checked } : entry))} />Obligatorio</label></div></div>; })}{!requirements.length && <p className="catalog-empty">Agrega tipos documentales desde el panel izquierdo.</p>}</div>
        </div>
      </div>
      <div className="template-summary">{summary.map(item => <div className={`template-summary-card ${item.tone}`} key={item.label}><span className="template-summary-icon" aria-hidden="true">{item.icon}</span><div><strong>{item.value}</strong><span>{item.label}</span></div></div>)}</div>
      <div className="actions"><button disabled={busy}>{selected ? 'Guardar nueva versión' : 'Crear plantilla'}</button></div>
    </form>}
  </section>;
}
