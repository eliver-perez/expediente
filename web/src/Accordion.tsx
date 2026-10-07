import type { ReactNode } from 'react';

// Native details preserves keyboard support and independent open/closed state.
// All sections share the original “Huella y trazabilidad” presentation.
export function Accordion({ title, children, open, className = '' }: { title: string; children: ReactNode; open?: boolean; className?: string }) {
  return <details className={`document-accordion ${className}`} open={open}><summary>{title}</summary>{children}</details>;
}
