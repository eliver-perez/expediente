import type { ReactNode } from 'react';

// The contract is rendered as text nodes only. HTML in the stored Markdown is
// never interpreted, and rendering cannot change the bytes used for its hash.
function inline(text: string): ReactNode[] {
  return text.split(/(\*\*[^*]+\*\*)/g).map((part, index) =>
    part.startsWith('**') && part.endsWith('**') ? <strong key={index}>{part.slice(2, -2)}</strong> : part);
}

export function LegalMarkdown({ text }: { text: string }) {
  const lines = text.replace(/\r\n/g, '\n').split('\n');
  const blocks: ReactNode[] = [];
  let paragraph: string[] = [];
  let list: { ordered: boolean; entries: string[] } | null = null;
  const flushParagraph = () => {
    if (paragraph.length) blocks.push(<p key={blocks.length}>{inline(paragraph.join(' ').replace(/  +/g, ' '))}</p>);
    paragraph = [];
  };
  const flushList = () => {
    if (list) {
      const entries = list.entries.map((entry, index) => <li key={index}>{inline(entry)}</li>);
      blocks.push(list.ordered ? <ol key={blocks.length}>{entries}</ol> : <ul key={blocks.length}>{entries}</ul>);
    }
    list = null;
  };
  for (const raw of lines) {
    const line = raw.trim();
    if (!line) { flushParagraph(); flushList(); continue; }
    const heading = /^(#{1,6})\s+(.+)$/.exec(line);
    const item = /^(\d+\.|[-*])\s+(.+)$/.exec(line);
    if (heading) {
      flushParagraph(); flushList();
      const level = Math.min(heading[1].length + 1, 6);
      const Heading = `h${level}` as 'h2';
      blocks.push(<Heading key={blocks.length}>{inline(heading[2])}</Heading>);
    } else if (/^(-{3,}|\*{3,})$/.test(line)) {
      flushParagraph(); flushList(); blocks.push(<hr key={blocks.length} />);
    } else if (item) {
      flushParagraph();
      const ordered = item[1] !== '-' && item[1] !== '*';
      if (list && list.ordered !== ordered) flushList();
      if (!list) list = { ordered, entries: [] };
      list.entries.push(item[2]);
    } else {
      flushList(); paragraph.push(line);
    }
  }
  flushParagraph(); flushList();
  return <div className="legal-markdown">{blocks}</div>;
}
