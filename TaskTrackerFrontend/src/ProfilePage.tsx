import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';

const ROLE_LABELS: Record<string, string> = {
  owner: 'Владелец', admin: 'Администратор', member: 'Участник',
  pm: 'Менеджер', dev: 'Разработчик', qa: 'Тестировщик', viewer: 'Наблюдатель',
  developer: 'Разработчик',
};
const roleLabel = (r: string) => ROLE_LABELS[r] ?? r;

function PencilIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
    </svg>
  );
}

export default function ProfilePage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('');
  const [loading, setLoading] = useState(false);
  const [orgs, setOrgs] = useState<any[]>([]);
  const [projects, setProjects] = useState<any[]>([]);
  const [orgsOpen, setOrgsOpen] = useState(false);
  const [projectsOpen, setProjectsOpen] = useState(false);

  const [editEmail, setEditEmail] = useState(false);
  const [newEmail, setNewEmail] = useState('');
  const [emailPass, setEmailPass] = useState('');

  const [editPass, setEditPass] = useState(false);
  const [currentPass, setCurrentPass] = useState('');
  const [newPass, setNewPass] = useState('');

  const initial = email.slice(0, 1).toUpperCase();

  const fetchMe = async () => {
    const res = await apiFetch(`${API_URL}/me`);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) return;
    setEmail(data.login || '');
    setRole(data.role || '');
    setNewEmail(data.login || '');
  };

  const fetchOrgs = async () => {
    const res = await apiFetch(`${API_URL}/orgs`);
    const data = await res.json().catch(() => []);
    setOrgs(Array.isArray(data) ? data : []);
  };

  const fetchProjects = async () => {
    const orgId = localStorage.getItem('selectedOrgId');
    if (!orgId) return;
    const res = await apiFetch(`${API_URL}/projects?org_id=${orgId}`);
    const data = await res.json().catch(() => []);
    setProjects(Array.isArray(data) ? data : []);
  };

  useEffect(() => {
    fetchMe();
    fetchOrgs();
    fetchProjects();
  }, []);

  const changeEmail = async () => {
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/me/login`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_login: newEmail.trim(), current_password: emailPass }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      localStorage.setItem('userLogin', newEmail.trim());
      setEmailPass('');
      setEditEmail(false);
      await fetchMe();
    } catch (e: any) { alert(e.message); }
    finally { setLoading(false); }
  };

  const changePassword = async () => {
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/me/password`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: currentPass, new_password: newPass }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      localStorage.clear();
      alert('Пароль изменён. Войди заново.');
      window.location.href = '/login';
    } catch (e: any) { alert(e.message); }
    finally { setLoading(false); }
  };

  const logout = async () => {
    try { await apiFetch(`${API_URL}/me/logout-all`, { method: 'POST' }); } catch {}
    localStorage.clear();
    navigate('/login');
    window.location.reload();
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-xl font-bold text-gray-900 text-center mb-6">Профиль</h1>

      {/* Avatar + name */}
      <div className="flex flex-col items-center mb-6">
        <div
          className="w-20 h-20 rounded-full flex items-center justify-center text-white text-3xl font-bold mb-3"
          style={{ background: P }}
        >
          {initial}
        </div>
        <p className="font-semibold text-gray-900">{email || '—'}</p>
        <span className="text-xs text-gray-400 mt-0.5">{roleLabel(role)}</span>
        <button
          onClick={() => setEditEmail(!editEmail)}
          className="mt-3 flex items-center gap-1.5 text-sm font-medium px-4 py-1.5 rounded-xl border border-gray-200 text-gray-600 hover:border-[#7C5CBF] hover:text-[#7C5CBF] transition-colors"
        >
          <PencilIcon /> Редактировать
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
        {/* Left: accordions */}
        <div className="space-y-3">
          {/* Организации */}
          <div className="bg-white border border-gray-100 rounded-xl overflow-hidden">
            <button
              onClick={() => setOrgsOpen(!orgsOpen)}
              className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-gray-700"
            >
              <div className="flex items-center gap-2">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9CA3AF" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                Организации
              </div>
              <span className="text-gray-400 text-xs">{orgsOpen ? '▲' : '▼'}</span>
            </button>
            {orgsOpen && (
              <div className="px-4 pb-3 space-y-1">
                {orgs.length === 0 ? <p className="text-xs text-gray-400">Нет организаций</p> : orgs.map(o => (
                  <div key={o.id} className="text-sm text-gray-600 py-1 border-t border-gray-50">{o.name} <span className="text-xs text-gray-400">({roleLabel(o.role ?? '')})</span></div>
                ))}
              </div>
            )}
          </div>

          {/* Проекты */}
          <div className="bg-white border border-gray-100 rounded-xl overflow-hidden">
            <button
              onClick={() => setProjectsOpen(!projectsOpen)}
              className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-gray-700"
            >
              <div className="flex items-center gap-2">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9CA3AF" strokeWidth="2"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-4 0v2"/><path d="M8 7V5a2 2 0 0 1 4 0v2"/></svg>
                Проекты
              </div>
              <span className="text-gray-400 text-xs">{projectsOpen ? '▲' : '▼'}</span>
            </button>
            {projectsOpen && (
              <div className="px-4 pb-3 space-y-1">
                {projects.length === 0 ? <p className="text-xs text-gray-400">Нет проектов</p> : projects.map(p => (
                  <div key={p.id} className="text-sm text-gray-600 py-1 border-t border-gray-50">{p.name}</div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right: credentials */}
        <div className="space-y-3">
          {/* Email */}
          <div className="bg-white border border-gray-100 rounded-xl p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Логин:</p>
                <p className="text-sm text-gray-700">{email}</p>
              </div>
              <button onClick={() => setEditEmail(!editEmail)} className="text-gray-400 hover:text-[#7C5CBF] transition-colors">
                <PencilIcon />
              </button>
            </div>
            {editEmail && (
              <div className="mt-3 space-y-2 pt-3 border-t border-gray-50">
                <input value={newEmail} onChange={e => setNewEmail(e.target.value)} placeholder="Новый логин" className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg outline-none" />
                <input type="password" value={emailPass} onChange={e => setEmailPass(e.target.value)} placeholder="Текущий пароль" className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg outline-none" />
                <button disabled={loading} onClick={changeEmail} className="w-full py-2 rounded-lg text-white text-sm font-medium disabled:opacity-60" style={{ background: P }}>Сохранить</button>
              </div>
            )}
          </div>

          {/* Password */}
          <div className="bg-white border border-gray-100 rounded-xl p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-400 mb-0.5">Пароль:</p>
                <p className="text-sm text-gray-700">••••••••••••••</p>
              </div>
              <button onClick={() => setEditPass(!editPass)} className="text-gray-400 hover:text-[#7C5CBF] transition-colors">
                <PencilIcon />
              </button>
            </div>
            {editPass && (
              <div className="mt-3 space-y-2 pt-3 border-t border-gray-50">
                <input type="password" value={currentPass} onChange={e => setCurrentPass(e.target.value)} placeholder="Текущий пароль" className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg outline-none" />
                <input type="password" value={newPass} onChange={e => setNewPass(e.target.value)} placeholder="Новый пароль (мин. 8, A-z, 0-9)" className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg outline-none" />
                <button disabled={loading} onClick={changePassword} className="w-full py-2 rounded-lg text-white text-sm font-medium disabled:opacity-60 bg-gray-900">Сменить пароль</button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Logout */}
      <div className="flex justify-end mt-4">
        <button
          onClick={logout}
          className="px-6 py-2.5 rounded-xl text-white text-sm font-semibold bg-red-500 hover:bg-red-600 transition-colors"
        >
          Выйти
        </button>
      </div>
    </div>
  );
}
