import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice, usePage } from './components';
import { DashboardChart, type ChartValue } from './DashboardChart';
import { StatisticsChart, type StatisticsSeries } from './StatisticsChart';

interface Group { key: string; label: string; total: number; evaluable: number; complete: number; incomplete: number; without_documents: number; without_template: number; required: number; fulfilled: number; unavailable: number; complete_percent: number | null; progress_percent: number | null }
interface Missing { name: string; cases: number; missing_units: number }
interface Stats { totals: Group; by_exercise: Group[]; by_template: Group[]; missing_requirements: Missing[] }
const number = (value: number) => value.toLocaleString('es-MX');
const percent = (value: number | null) => value === null ? 'Sin base evaluable' : `${value.toLocaleString('es-MX', { minimumFractionDigits: 1, maximumFractionDigits: 1 })} %`;
const chartLabel = (value: string) => value.length > 28 ? `${value.slice(0, 27)}…` : value;

function StatisticsMetric({ label, value, note, tone, icon, onClick }: { label: string; value: string; note: string; tone: string; icon: 'files' | 'check' | 'pending' | 'progress'; onClick?: () => void }) {
  const paths = { files: 'M7 3h7l4 4v14H7z M14 3v5h5 M4 7v17h11', check: 'M5 13l5 5L21 6 M4 3h16v19H4z', pending: 'M4 4h18v18H4z M8 9h10 M8 14h7', progress: 'M4 20V12 M10 20V7 M16 20V4 M22 20V10' };
  const contents = <><div className="statistics-metric-content"><span>{label}</span><strong>{value}</strong><small>{note}</small></div><svg className="metric-symbol" width="44" height="44" viewBox="0 0 26 26" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[icon]} /></svg></>;
  return onClick ? <button className={`metric dashboard-metric metric-${tone} statistics-kpi-button`} onClick={onClick} aria-label={`${label}: ${value}. ${note}. Ver expedientes`}>{contents}</button> : <article className={`metric dashboard-metric metric-${tone}`}>{contents}</article>;
}
function SeriesLegend({ series }: { series: StatisticsSeries[] }) {
  return <ul className="statistics-series-legend">{series.map(item => <li key={item.label}><span className="legend-dot" style={{ backgroundColor: item.color }} />{item.label}</li>)}</ul>;
}

export function CaseStatistics({ libraryID, openCases }: { libraryID: string; openCases: (exercise: string, status: string) => void }) {
  const exercises = usePage<string>(`/libraries/${libraryID}/exercises`);
  const [exercise, setExercise] = useState('');
  const [revision, setRevision] = useState(0);
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    setStats(null); setError('');
    void api<Stats>(`/libraries/${libraryID}/case-statistics?exercise=${encodeURIComponent(exercise)}`, { signal: controller.signal })
      .then(setStats).catch(cause => { if (!controller.signal.aborted) setError(message(cause)); });
    return () => controller.abort();
  }, [libraryID, exercise, revision]);
  const totals = stats?.totals;
  const byExercise = [...(stats?.by_exercise ?? [])].sort((a, b) => b.key.localeCompare(a.key));
  const byTemplate = [...(stats?.by_template ?? [])].sort((a, b) => a.label.localeCompare(b.label));
  const comparison: StatisticsSeries[] = [
    { label: 'Expedientes completos', color: '#007d6e', values: [] },
    { label: 'Avance documental', color: '#416a99', values: [] },
  ];
  const exerciseSeries = comparison.map((item, index) => ({ ...item, values: byExercise.map(group => index === 0 ? group.complete_percent : group.progress_percent) }));
  const templateSeries = comparison.map((item, index) => ({ ...item, values: byTemplate.map(group => index === 0 ? group.complete_percent : group.progress_percent) }));
  const distribution: ChartValue[] = totals ? [
    { label: 'Completos', count: totals.complete, color: '#007d6e' },
    { label: 'Incompletos', count: totals.incomplete, color: '#416a99' },
  ] : [];
  return <div className="statistics-dashboard">
    <section className="card statistics-intro"><div className="section-heading"><div><p className="eyebrow">PERSPECTIVA DOCUMENTAL</p><h2>Estadísticas documentales</h2><p className="muted">Un expediente está completo cuando todos sus requisitos obligatorios tienen documentos aprobados y disponibles.</p></div><button className="secondary" onClick={() => { setRevision(n => n + 1); exercises.reload(); }}>Actualizar estadísticas</button></div>
      <Notice text={error || exercises.error} />
      <label className="statistics-filter">Ejercicio<select value={exercise} onChange={event => setExercise(event.target.value)}><option value="">Todos los ejercicios</option>{exercises.items.map(year => <option key={year} value={year}>{year}</option>)}</select></label>
    </section>
    {!stats && !error && <section className="card" role="status">Calculando estadísticas…</section>}
    {totals && <>
      <div className="dashboard-kpis statistics-kpis">
        <StatisticsMetric label="Total de expedientes" value={number(totals.total)} note={`${number(totals.evaluable)} con plantilla evaluable`} tone="navy" icon="files" />
        <StatisticsMetric label="Expedientes completos" value={number(totals.complete)} note={`${percent(totals.complete_percent)} de los evaluables`} tone="teal" icon="check" onClick={() => openCases(exercise, 'complete')} />
        <StatisticsMetric label="Expedientes incompletos" value={number(totals.incomplete)} note={`${percent(totals.evaluable ? 100 * totals.incomplete / totals.evaluable : null)} de los evaluables`} tone="blue" icon="pending" onClick={() => openCases(exercise, 'incomplete')} />
        <StatisticsMetric label="Avance documental" value={percent(totals.progress_percent)} note={`${number(totals.fulfilled)} de ${number(totals.required)} requisitos obligatorios`} tone="slate" icon="progress" />
      </div>
      {totals.total === 0 ? <section className="card"><p className="chart-empty">No hay expedientes para este ejercicio.</p></section> : <>
        <div className="statistics-chart-grid">
          <section className="card dashboard-card statistics-card"><div className="section-heading"><div><h2>Distribución de expedientes</h2><p className="muted">Completos e incompletos entre los expedientes evaluables.</p></div></div>
            {totals.evaluable > 0 ? <><div className="donut-layout"><div className="donut-container"><DashboardChart kind="doughnut" title="Distribución de expedientes evaluables" values={distribution} /><div className="donut-total" aria-hidden="true"><strong>{number(totals.evaluable)}</strong><span>evaluables</span></div></div><ul className="chart-legend">{distribution.map(item => <li key={item.label}><span className="legend-dot" style={{ backgroundColor: item.color }} /><span>{item.label}</span><strong>{number(item.count)}</strong><small>{percent(100 * item.count / totals.evaluable)}</small></li>)}</ul></div></> : <p className="chart-empty">No hay expedientes con requisitos obligatorios evaluables.</p>}
            <div className="statistics-context"><strong>{number(totals.without_documents)} sin documentos</strong><span>Este grupo puede estar incluido entre los incompletos; no se suma otra vez al gráfico. {totals.without_template > 0 && `${number(totals.without_template)} expedientes sin plantilla evaluable quedan fuera del anillo.`}</span><button className="text-button" onClick={() => openCases(exercise, 'empty')}>Ver expedientes sin documentos →</button></div>
          </section>
          <section className="card dashboard-card statistics-card"><div className="section-heading"><div><h2>Avance por ejercicio</h2><p className="muted">Porcentaje de expedientes completos frente al cumplimiento de requisitos.</p></div></div>
            {byExercise.length ? <><StatisticsChart title="Avance por ejercicio" labels={byExercise.map(group => group.label)} series={exerciseSeries} percent /><SeriesLegend series={comparison} /></> : <p className="chart-empty">Sin ejercicios registrados.</p>}
          </section>
          <section className="card dashboard-card statistics-card"><div className="section-heading"><div><h2>Avance por plantilla</h2><p className="muted">Comparación porcentual entre plantillas documentales.</p></div></div>
            {byTemplate.length ? <><StatisticsChart title="Avance por plantilla" labels={byTemplate.map(group => chartLabel(group.label))} series={templateSeries} percent horizontal /><SeriesLegend series={comparison} /></> : <p className="chart-empty">Sin plantillas evaluables.</p>}
          </section>
          <section className="card dashboard-card statistics-card"><div className="section-heading"><div><h2>Requisitos más frecuentemente incumplidos</h2><p className="muted">Expedientes afectados por tipo documental; un requisito se cuenta una vez por expediente.</p></div></div>
            {stats && stats.missing_requirements.length ? <><StatisticsChart title="Requisitos más frecuentemente incumplidos" labels={stats.missing_requirements.map(item => chartLabel(item.name))} series={[{ label: 'Expedientes afectados', color: '#b77922', values: stats.missing_requirements.map(item => item.cases) }]} horizontal /><SeriesLegend series={[{ label: 'Expedientes afectados', color: '#b77922', values: [] }]} /></> : <p className="chart-empty">No hay requisitos obligatorios pendientes en la selección.</p>}
          </section>
        </div>
        <div className="statistics-detail-grid">
          <section className="card statistics-detail-card"><h2>Desglose por ejercicio</h2>{byExercise.length ? <div className="table-wrap"><table><thead><tr><th>Ejercicio</th><th>Total</th><th>Completos</th><th>Incompletos</th><th>% completos</th><th>% avance</th></tr></thead><tbody>{byExercise.map(group => <tr key={group.key}><td><button className="text-button" onClick={() => setExercise(group.key)}>{group.label}</button></td><td>{number(group.total)}</td><td>{number(group.complete)}</td><td>{number(group.incomplete)}</td><td>{percent(group.complete_percent)}</td><td>{percent(group.progress_percent)}</td></tr>)}</tbody></table></div> : <p>Sin ejercicios registrados.</p>}</section>
          <section className="card statistics-detail-card"><h2>Desglose por plantilla</h2>{byTemplate.length ? <div className="table-wrap"><table><thead><tr><th>Plantilla</th><th>Total</th><th>Completos</th><th>Incompletos</th><th>% completos</th><th>% avance</th></tr></thead><tbody>{byTemplate.map(group => <tr key={group.key}><td>{group.label}</td><td>{number(group.total)}</td><td>{number(group.complete)}</td><td>{number(group.incomplete)}</td><td>{percent(group.complete_percent)}</td><td>{percent(group.progress_percent)}</td></tr>)}</tbody></table></div> : <p>Sin plantillas evaluables.</p>}</section>
          <section className="card statistics-detail-card"><h2>Requisitos pendientes</h2>{stats && stats.missing_requirements.length ? <div className="table-wrap"><table><thead><tr><th>Tipo documental</th><th>Expedientes afectados</th><th>Unidades pendientes</th></tr></thead><tbody>{stats.missing_requirements.map(item => <tr key={item.name}><td>{item.name}</td><td>{number(item.cases)}</td><td>{number(item.missing_units)}</td></tr>)}</tbody></table></div> : <p>No hay requisitos obligatorios pendientes en la selección.</p>}</section>
        </div>
        <section className="statistics-note"><span className="badge">Cómo leer los porcentajes</span><p>“Expedientes completos” divide expedientes completos entre evaluables. “Avance documental” divide requisitos obligatorios cumplidos entre requeridos. Los requisitos opcionales no intervienen.</p></section>
      </>}
    </>}
  </div>;
}
