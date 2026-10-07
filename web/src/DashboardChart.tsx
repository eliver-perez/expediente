import { useEffect, useRef } from 'react';
import { Chart, ArcElement, BarElement, LineElement, PointElement, DoughnutController, BarController, LineController, CategoryScale, LinearScale, Tooltip, Filler, type ChartConfiguration } from 'chart.js';

Chart.register(ArcElement, BarElement, LineElement, PointElement, DoughnutController, BarController, LineController, CategoryScale, LinearScale, Tooltip, Filler);
export interface ChartValue { label: string; count: number; color: string }
type Kind = 'doughnut' | 'bar' | 'line';

// Canvas is an enhancement: visible legends and a data table carry the same
// values for keyboard/screen-reader users. No HTML tooltip or remote assets.
export function DashboardChart({ kind, title, values }: { kind: Kind; title: string; values: ChartValue[] }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const serialized = JSON.stringify(values);
  useEffect(() => {
    if (!canvas.current) return;
    const items: ChartValue[] = JSON.parse(serialized);
    const common = {
      responsive: true, maintainAspectRatio: false,
      animation: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? false as const : { duration: 350 },
      font: { family: 'system-ui, sans-serif' },
      plugins: { legend: { display: false }, tooltip: { backgroundColor: '#12343e', padding: 12, cornerRadius: 8, displayColors: kind !== 'line', callbacks: { label: (item: { dataIndex: number }) => `${items[item.dataIndex].label}: ${items[item.dataIndex].count.toLocaleString('es-MX')}` } } },
    };
    const config: ChartConfiguration<Kind, number[], string> = {
      type: kind,
      data: { labels: items.map(item => item.label), datasets: [{
        data: items.map(item => item.count),
        backgroundColor: kind === 'line' ? '#007d6e18' : items.map(item => item.color),
        borderColor: kind === 'line' ? '#007d6e' : '#ffffff', borderWidth: kind === 'line' ? 2.5 : kind === 'doughnut' ? 4 : 0,
        ...(kind === 'line' ? { fill: true, tension: 0, pointRadius: 2, pointHoverRadius: 5, pointBackgroundColor: '#007d6e' } : {}),
        ...(kind === 'bar' ? { borderRadius: 5, maxBarThickness: 22 } : {}),
      }] },
      options: kind === 'doughnut' ? { ...common, cutout: '76%' } : {
        ...common, indexAxis: kind === 'bar' ? 'y' : 'x',
        scales: {
          x: kind === 'bar' ? { beginAtZero: true, ticks: { precision: 0, color: '#526770' }, grid: { color: '#e9efec' }, border: { display: false } } : { ticks: { maxTicksLimit: 6, maxRotation: 0, color: '#526770' }, grid: { display: false }, border: { display: false } },
          y: kind === 'bar' ? { grid: { display: false }, border: { display: false }, ticks: { color: '#344e55', font: { size: 11 } } } : { beginAtZero: true, ticks: { precision: 0, color: '#526770' }, grid: { color: '#e9efec' }, border: { display: false } },
        },
      },
    };
    const chart = new Chart(canvas.current, config);
    return () => chart.destroy();
  }, [serialized, kind]);
  return <div className={`dashboard-chart chart-${kind}`}><canvas ref={canvas} role="img" aria-label={`${title}. ${values.map(item => `${item.label}: ${item.count}`).join('; ')}`} /></div>;
}
