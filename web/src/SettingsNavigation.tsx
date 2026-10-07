export const settingsSections = [
  { path: '/admin/files', label: 'Archivos' },
  { path: '/admin/processing', label: 'Procesamiento y recursos' },
  { path: '/admin/advanced', label: 'Automatización, OCR y vigilancia' },
  { path: '/admin/previews', label: 'Vistas previas y caché' },
  { path: '/admin/network', label: 'Acceso y red' },
];
export const isSettingsPath = (path: string) => path === '/admin/settings' || settingsSections.some(section => section.path === path);

// These are links between addressable sections, not ARIA tabs with hidden panels.
// Old URLs and the browser's back/forward navigation remain valid.
export function SettingsNavigation({ path, navigate }: { path: string; navigate: (path: string) => void }) {
  const current = path === '/admin/settings' ? '/admin/files' : path;
  return <nav className="tabs settings-navigation" aria-label="Secciones de configuración">
    {settingsSections.map(section => <a key={section.path} href={section.path} aria-current={current === section.path ? 'page' : undefined} onClick={event => {
      if (event.button || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
      event.preventDefault(); navigate(section.path);
    }}>{section.label}</a>)}
  </nav>;
}
