export function Brand({ variant = 'horizontal', dark = false, className = '' }: { variant?: 'horizontal' | 'vertical'; dark?: boolean; className?: string }) {
  const filename = `logo-${dark ? 'dark-' : ''}${variant === 'vertical' ? 'v' : 'h'}.svg`;
  return <img className={`aibid-brand ${className}`} src={`/assets/brand/${filename}`} alt="AIBID — Aplicación de Indexación de Bibliotecas Digitales. Tu biblioteca digital, ordenada y al alcance." width={variant === 'vertical' ? 520 : dark ? 740 : 720} height={variant === 'vertical' ? 210 : dark ? 140 : 130} />;
}
