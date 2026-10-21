export function Brand({ variant = 'horizontal', dark = false, className = '' }: { variant?: 'horizontal' | 'vertical'; dark?: boolean; className?: string }) {
  const filename = `logo-${dark ? 'dark-' : ''}${variant === 'vertical' ? 'v' : 'h'}.svg`;
  return <img className={`aidex-brand ${className}`} src={`/assets/brand/${filename}`} alt="AIDEX — Aplicación de Indexación Digital de Expedientes. Tu biblioteca digital, ordenada y al alcance." width={variant === 'vertical' ? 520 : dark ? 740 : 720} height={variant === 'vertical' ? 210 : dark ? 140 : 130} />;
}
