import { useEffect, useRef, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';
import { DocumentList } from './Documents';
import type { Document, SearchInput, SearchResult } from './libraryTypes';

// Each mounted section owns its cursor history, size and scroll position.
export function SearchSection({ input, name, count, open }: { input: SearchInput; name: string; count?: number; open: (document: Document, page?: number) => void }) {
 const [size,setSize]=useState(10);const [cursor,setCursor]=useState('');const [previous,setPrevious]=useState<string[]>([]);const [collapsed,setCollapsed]=useState(false);
 const [result,setResult]=useState<SearchResult|null>(null);const [busy,setBusy]=useState(false);const [error,setError]=useState('');const container=useRef<HTMLDivElement>(null);
 const scope=JSON.stringify(input);
 useEffect(()=>{
  const abort=new AbortController();setBusy(true);setError('');
  api<SearchResult>('/search',{method:'POST',body:{...JSON.parse(scope),summary_only:false,limit:size,cursor},signal:abort.signal}).then(value=>{setResult(value);if(container.current)container.current.scrollTop=0;}).catch(cause=>{if(!abort.signal.aborted)setError(message(cause));}).finally(()=>{if(!abort.signal.aborted)setBusy(false);});return()=>abort.abort();
 },[scope,cursor,size]);
 return <section className="card search-library-section" aria-label={`Resultados de ${name}`}>
 <div className="section-heading"><div><h2>{name}</h2><p>{result?.result_count??count??'…'} coincidencias</p></div><button className="secondary compact" aria-expanded={!collapsed} onClick={()=>setCollapsed(!collapsed)}>{collapsed?'Expandir resultados':'Contraer resultados'}</button></div>
 <div hidden={collapsed}><Notice text={error}/><div className="section-heading"><label>Resultados por página<select value={size} onChange={event=>{setSize(Number(event.target.value));setCursor('');setPrevious([]);}}><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option></select></label><span role="status">{busy?'Buscando…':`Página ${previous.length+1}`}</span></div>
 <div ref={container} className={size>10?'search-results-scroll':''} aria-busy={busy}><DocumentList documents={result?.items??[]} open={open}/>{!busy&&result?.items.length===0&&<p className="empty-state">No hay coincidencias con estos filtros.</p>}</div>
 <div className="actions"><button className="secondary" disabled={busy||!previous.length} onClick={()=>{setCursor(previous[previous.length-1]);setPrevious(previous.slice(0,-1));}}>Anterior</button><span>Página {previous.length+1}</span><button className="secondary" disabled={busy||!result?.next_cursor} onClick={()=>{setPrevious([...previous,cursor]);setCursor(result?.next_cursor||'');}}>Siguiente</button></div></div>
 </section>;
}
