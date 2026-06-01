import { useState, useEffect } from 'react';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';

interface StatItem { status: string; count: number; }

const STATUS_COLORS: Record<string, string> = {
  'New': '#94A3B8',
  'Open': '#3B82F6',
  'In Progress': '#6366F1',
  'Fixed': '#10B981',
  'Ready for Retest': '#06B6D4',
  'Verified': '#16A34A',
  'Reopened': '#EF4444',
  'Rejected': '#F97316',
  "Can't Reproduce": '#EAB308',
};

export default function AnalyticsPage() {
  const [stats, setStats] = useState<StatItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [orgs, setOrgs] = useState<any[]>([]);
  const [projects, setProjects] = useState<any[]>([]);
  const [selectedOrgId, setSelectedOrgId] = useState(() => Number(localStorage.getItem('selectedOrgId') || '0'));
  const [selectedProjectId, setSelectedProjectId] = useState(() => Number(localStorage.getItem('selectedProjectId') || '0'));

  useEffect(() => {
    apiFetch(`${API_URL}/orgs`)
      .then(r => r.json()).then(d => setOrgs(Array.isArray(d) ? d : [])).catch(() => {});
  }, []);

  useEffect(() => {
    if (!selectedOrgId) return;
    apiFetch(`${API_URL}/projects?org_id=${selectedOrgId}`)
      .then(r => r.json()).then(d => setProjects(Array.isArray(d) ? d : [])).catch(() => {});
  }, [selectedOrgId]);

  useEffect(() => {
    if (!selectedProjectId) { setStats([]); return; }
    setLoading(true);
    apiFetch(`${API_URL}/stats?project_id=${selectedProjectId}`)
      .then(r => r.json())
      .then(d => setStats(Array.isArray(d) ? d : []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [selectedProjectId]);

  const total = stats.reduce((s, x) => s + x.count, 0);
  const sorted = [...stats].sort((a, b) => b.count - a.count);

  const getCount = (status: string) => stats.find(s => s.status === status)?.count ?? 0;

  const summaryCards = [
    { label: 'Открыто:', value: getCount('Open') },
    { label: 'Найдено:', value: getCount('Fixed') },
    { label: 'Новых:', value: getCount('New') },
    { label: 'Переотк.:', value: getCount('Reopened') },
  ];

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-xl font-bold text-gray-900 text-center mb-6">Аналитика</h1>

      {/* Filters */}
      <div className="flex gap-3 mb-6">
        <div className="flex items-center gap-2 bg-white border border-gray-200 rounded-xl px-3 py-2.5 flex-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9CA3AF" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          <select
            value={selectedOrgId || ''}
            onChange={e => { const v = Number(e.target.value); setSelectedOrgId(v); localStorage.setItem('selectedOrgId', String(v)); }}
            className="flex-1 text-sm text-gray-700 bg-transparent outline-none"
          >
            <option value="">Выберите организацию</option>
            {orgs.map(o => <option key={o.id} value={o.id}>{o.name}</option>)}
          </select>
        </div>
        <div className="flex items-center gap-2 bg-white border border-gray-200 rounded-xl px-3 py-2.5 flex-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9CA3AF" strokeWidth="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 12l2 2 4-4"/></svg>
          <select
            value={selectedProjectId || ''}
            onChange={e => { const v = Number(e.target.value); setSelectedProjectId(v); localStorage.setItem('selectedProjectId', String(v)); }}
            className="flex-1 text-sm text-gray-700 bg-transparent outline-none"
            disabled={!selectedOrgId}
          >
            <option value="">Выберите проект</option>
            {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-5 gap-3 mb-6">
        <div className="col-span-2 rounded-2xl p-5 flex flex-col justify-center" style={{ background: P }}>
          <p className="text-white/80 text-xs font-medium">Всего багов:</p>
          <p className="text-white text-4xl font-black mt-1">{total}</p>
        </div>
        {summaryCards.map(c => (
          <div key={c.label} className="bg-white rounded-2xl p-4 border border-gray-100 flex flex-col items-center justify-center text-center">
            <p className="text-gray-500 text-[11px] font-medium leading-tight mb-1">{c.label}</p>
            <p className="text-2xl font-black text-gray-900">{c.value}</p>
          </div>
        ))}
      </div>

      {/* Bar chart */}
      <div className="bg-white rounded-2xl p-5 border border-gray-100">
        <h2 className="font-bold text-gray-900 mb-4">По статусам</h2>
        {loading && <p className="text-sm text-gray-400">Загрузка...</p>}
        {!loading && total === 0 && <p className="text-sm text-gray-400">Нет данных</p>}
        <div className="space-y-3">
          {sorted.map(s => {
            const pct = total > 0 ? Math.round((s.count / total) * 100) : 0;
            const color = STATUS_COLORS[s.status] ?? '#CBD5E1';
            return (
              <div key={s.status}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-gray-700">{s.status}</span>
                  <span className="text-gray-400">{s.count} ({pct}%)</span>
                </div>
                <div className="w-full bg-gray-100 rounded-full h-2">
                  <div className="h-2 rounded-full transition-all" style={{ width: `${pct}%`, background: color }} />
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
