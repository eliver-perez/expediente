import { useEffect, useRef } from 'react';
import { BarController, BarElement, CategoryScale, Chart, LinearScale, Tooltip } from 'chart.js';

Chart.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip);

export interface StatisticsSeries {
  label: string;
  color: string;
  values: (number | null)[];
}

export function StatisticsChart({ title, labels, series, percent = false, horizontal = false }: {
  title: string;
  labels: string[];
  series: StatisticsSeries[];
  percent?: boolean;
  horizontal?: boolean;
}) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const serialized = JSON.stringify({ labels, series });
  useEffect(() => {
    if (!canvas.current) return;
    const data: { labels: string[]; series: StatisticsSeries[] } = JSON.parse(serialized);
    const chart = new Chart(canvas.current, {
      type: 'bar',
      data: {
        labels: data.labels,
        datasets: data.series.map(item => ({
          label: item.label,
          data: item.values,
          backgroundColor: item.color,
          borderRadius: 5,
          maxBarThickness: 34,
        })),
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        indexAxis: horizontal ? 'y' : 'x',
        animation: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? false : { duration: 350 },
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: '#12343e', padding: 12, cornerRadius: 8,
            callbacks: { label: item => `${item.dataset.label}: ${Number(horizontal ? item.parsed.x : item.parsed.y).toLocaleString('es-MX', { maximumFractionDigits: 1 })}${percent ? ' %' : ''}` },
          },
        },
        scales: {
          x: horizontal ? { beginAtZero: true, max: percent ? 100 : undefined, ticks: { color: '#526770', callback: value => percent ? `${value} %` : value }, grid: { color: '#e9efec' }, border: { display: false } } : { ticks: { color: '#344e55', maxRotation: 0 }, grid: { display: false }, border: { display: false } },
          y: horizontal ? { ticks: { color: '#344e55', autoSkip: false, font: { size: 11 } }, grid: { display: false }, border: { display: false } } : { beginAtZero: true, max: percent ? 100 : undefined, ticks: { color: '#526770', callback: value => percent ? `${value} %` : value }, grid: { color: '#e9efec' }, border: { display: false } },
        },
      },
    });
    return () => chart.destroy();
  }, [serialized, percent, horizontal]);
  const description = labels.map((label, index) => {
    const values = series.map(item => `${item.label} ${item.values[index] === null ? 'sin base evaluable' : `${item.values[index]}${percent ? ' %' : ''}`}`);
    return `${label}: ${values.join(', ')}`;
  }).join('; ');
  return <div className="statistics-chart" style={{ height: horizontal ? Math.max(270, labels.length * (series.length > 1 ? 70 : 42)) : 290 }}>
    <canvas ref={canvas} role="img" aria-label={`${title}. ${description}`} />
  </div>;
}
