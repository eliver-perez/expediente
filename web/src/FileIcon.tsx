const formats: Record<string, { color: string; label: string }> = {
  pdf: { color: '#bf342c', label: 'PDF' }, docx: { color: '#185abd', label: 'Word' },
  xlsx: { color: '#107c41', label: 'Excel' }, txt: { color: '#526777', label: 'Texto' },
  csv: { color: '#087e8b', label: 'CSV' },
};

// Local vector artwork has fixed geometry: long extensions never wrap vertically.
export function FileIcon({ format = 'pdf' }: { format?: string }) {
  const kind = formats[format] || formats.txt;
  return <svg className="file-icon" viewBox="0 0 40 48" aria-hidden="true" data-format={format} style={{ color: kind.color }}>
    <path d="M10 2h17l10 10v32a2 2 0 0 1-2 2H10a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2Z" fill="currentColor" opacity=".12" />
    <path d="M10 2h17l10 10v32a2 2 0 0 1-2 2H10a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2Zm17 0v11h10" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
    {format === 'docx' ? <><path d="M18 21h13m-13 6h13m-13 6h13m-13 6h13" stroke="currentColor" strokeWidth="2"/><rect x="1" y="18" width="21" height="23" rx="2" fill="currentColor"/><path d="m5 24 2 11 4-8 4 8 2-11" fill="none" stroke="white" strokeWidth="1.8" strokeLinejoin="round"/></>
      : format === 'xlsx' ? <><path d="M19 20h12v20H19zm0 7h12m-12 6h12m-6-13v20" fill="none" stroke="currentColor" strokeWidth="1.5"/><rect x="1" y="18" width="21" height="23" rx="2" fill="currentColor"/><path d="m7 24 9 11m0-11L7 35" stroke="white" strokeWidth="2.3"/></>
      : format === 'pdf' ? <><path d="M14 33c8-11 9-19 6-18-4 2 1 15 11 15 7 0 0-7-12-1-12 5-11 10-5 4Z" fill="none" stroke="currentColor" strokeWidth="1.6"/><text x="22" y="42" textAnchor="middle" fill="currentColor" fontSize="8" fontWeight="700">PDF</text></>
      : format === 'csv' ? <><path d="M13 21h19v19H13zm0 6h19m-19 6h19m-12-12v19m6-19v19" fill="none" stroke="currentColor" strokeWidth="1.6"/></>
      : <path d="M13 22h19m-19 6h19m-19 6h19m-19 6h13" stroke="currentColor" strokeWidth="2"/>}
  </svg>;
}
