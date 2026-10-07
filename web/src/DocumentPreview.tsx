import { useEffect, useState } from 'react';
import { api, message } from './api';
import { Notice } from './components';

interface Preview { id?: string; state: string; error_code?: string; url?: string; paused: boolean; generated_at?: string }
const labels: Record<string,string> = { not_generated:'No generada', queued:'Pendiente de generación', generating:'Generando vista previa…', ready:'Disponible', obsolete:'Obsoleta', error:'Error', unavailable:'No disponible' };
const errors: Record<string,string> = {
 PREVIEW_DISABLED:'Las vistas previas están deshabilitadas por el administrador.',
 PREVIEW_UNSUPPORTED:'No hay vista previa para este formato. Puedes consultar el texto y descargar el original según tus permisos.',
 PREVIEW_CONVERTER_MISSING:'Falta LibreOffice en el equipo servidor. Solicita al administrador que revise Ajustes → Vistas previas y caché.',
 PREVIEW_SOURCE_LIMIT:'El archivo supera el tamaño configurado para generar una vista previa.',
 PREVIEW_SIZE_LIMIT:'La vista previa supera el límite de tamaño permitido.',
 PREVIEW_QUEUE_FULL:'La cola de vistas previas está llena. Intenta nuevamente más tarde.',
 PREVIEW_TIMEOUT:'La conversión superó los dos minutos permitidos.',
 PREVIEW_OBSOLETE:'El archivo o el generador cambiaron. Verifica la biblioteca y solicita una nueva vista previa.',
 DOCUMENT_UNAVAILABLE:'No se puede leer el original en este momento.',
 PROCESS_RESTARTED:'La conversión se interrumpió al reiniciar AIBID. Puedes volver a intentarlo.',
};
export function DocumentPreview({ documentID, hash, format }: { documentID:string; hash:string; format:string }) {
 const [preview,setPreview]=useState<Preview|null>(null); const [error,setError]=useState(''); const [busy,setBusy]=useState(false);
 const [expanded,setExpanded]=useState(false); const [refresh,setRefresh]=useState(0);
 useEffect(()=>{
  const abort=new AbortController(); let timer:ReturnType<typeof setTimeout>|undefined;let requested=false;
  setPreview(null);setError('');
  async function poll(){
   try{
    let value=await api<Preview>(`/documents/${documentID}/preview`,{signal:abort.signal});
    if (!requested && ['not_generated','obsolete'].includes(value.state)) {
     requested=true;value=await api<Preview>(`/documents/${documentID}/preview`,{method:'POST',body:{retry:false},signal:abort.signal});
    }
    if(abort.signal.aborted)return;setPreview(value);
    if(['queued','generating'].includes(value.state))timer=setTimeout(()=>void poll(),1500);
   }catch(cause){if(!abort.signal.aborted){setError(message(cause));setPreview({state:'error',paused:false});}}
  }
  void poll();return()=>{abort.abort();if(timer)clearTimeout(timer);};
 },[documentID,hash,refresh]);
 async function retry(){
  setBusy(true);setError('');
  try{const value=await api<Preview>(`/documents/${documentID}/preview`,{method:'POST',body:{retry:true}});setPreview(value);setRefresh(v=>v+1);}catch(cause){setError(message(cause));}finally{setBusy(false);}
 }
 return <section className={`generated-preview${expanded?' generated-preview-expanded':''}`} aria-label="Vista previa del documento">
  <div className="section-heading"><h3>Vista previa</h3><span role="status">{preview ? labels[preview.state] : 'Consultando…'}</span></div>
  <Notice text={error}/>
  {preview?.state==='ready' && preview.url ? <><div className="actions"><button className="secondary compact" onClick={()=>setExpanded(!expanded)}>{expanded?'Reducir vista previa':'Ampliar vista previa'}</button><button className="secondary compact" onClick={()=>{setPreview(null);setRefresh(v=>v+1);}}>Comprobar vigencia</button></div><iframe title={`Vista previa de ${format.toUpperCase()}`} src={preview.url}/><p className="muted">Representación en PDF. El diseño puede variar según las fuentes instaladas. El original permanece intacto.</p></>
   : <div className="preview-placeholder"><p>{preview?.paused?'La generación comenzará al reanudar el procesamiento.':preview?.state==='generating'||preview?.state==='queued'?'Puedes seguir consultando el texto mientras se prepara la vista.':!preview?'Consultando la vista previa…':errors[preview?.error_code||'']||'No fue posible generar una vista previa. El archivo original continúa disponible.'}</p>{preview && ['error','obsolete'].includes(preview.state)&&<button className="secondary" disabled={busy} onClick={()=>void retry()}>Reintentar vista previa</button>}</div>}
 </section>;
}
