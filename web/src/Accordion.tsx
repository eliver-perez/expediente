import type { ReactNode } from 'react';

// Native details preserves keyboard support and independent open/closed state.
// All sections share the original “Huella y trazabilidad” presentation.
export function Accordion({ title, children }: { title: string; children: ReactNode }) {
  return <details className="document-accordion"><summary>{title}</summary>{children}</details>;
}
