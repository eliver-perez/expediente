import { AdvancedSettings } from './AdvancedSettings';
import { FileSettings } from './FileSettings';
import { ProcessingSettings } from './ProcessingSettings';
import { PreviewSettings } from './PreviewSettings';
import { NetworkSettings } from './NetworkSettings';
import { SettingsNavigation } from './SettingsNavigation';

export function Settings({ path, navigate }: { path: string; navigate: (path: string) => void }) {
  const section = path === '/admin/settings' ? '/admin/files' : path;
  const content = section === '/admin/advanced' ? <AdvancedSettings /> : section === '/admin/processing' ? <ProcessingSettings /> : section === '/admin/previews' ? <PreviewSettings /> : section === '/admin/network' ? <NetworkSettings /> : <FileSettings />;
  return <><header className="page-heading"><p className="eyebrow">ADMINISTRACIÓN</p><h1>Ajustes</h1><p>Configura los archivos, el procesamiento y el acceso a esta instalación.</p></header>
    <SettingsNavigation path={path} navigate={navigate} /><div className="settings-content" key={section}>{content}</div></>;
}
