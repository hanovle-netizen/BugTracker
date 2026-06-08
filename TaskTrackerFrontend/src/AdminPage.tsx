import { useEffect, useState } from 'react';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';

type Group = { id: number; name: string; role?: string };
type Project = { id: number; org_id: number; name: string; role?: string };
type Member = { user_id: number; login: string; role: string; position?: string };

const inputCls = 'flex-1 px-3 py-2.5 rounded-xl border border-gray-200 text-sm outline-none focus:border-[#7C5CBF] bg-white';
const selectCls = 'px-3 py-2.5 rounded-xl border border-gray-200 text-sm outline-none focus:border-[#7C5CBF] bg-white';

const ROLE_LABELS: Record<string, string> = {
  owner: 'Владелец', admin: 'Администратор', member: 'Участник',
  pm: 'Менеджер', dev: 'Разработчик', qa: 'Тестировщик', viewer: 'Наблюдатель',
  developer: 'Разработчик',
};
const roleLabel = (r: string) => ROLE_LABELS[r] ?? r;

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white rounded-2xl border border-gray-100 p-5">
      <h2 className="text-sm font-bold text-gray-900 mb-4">{title}</h2>
      {children}
    </div>
  );
}

export default function AdminPage() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [orgMembers, setOrgMembers] = useState<Member[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number>(0);
  const [selectedProjectId, setSelectedProjectId] = useState<number>(
    () => Number(localStorage.getItem('selectedProjectId') || '0')
  );
  const [loading, setLoading] = useState(false);

  const [newGroupName, setNewGroupName] = useState('');
  const [newProjectName, setNewProjectName] = useState('');

  // Project member form
  const [memberLogin, setMemberLogin] = useState('');
  const [memberRole, setMemberRole] = useState<string>('qa');
  const [memberPosition, setMemberPosition] = useState('');
  const [memberHint, setMemberHint] = useState<string | null>(null);

  // Org member form
  const [orgMemberLogin, setOrgMemberLogin] = useState('');
  const [orgMemberRole, setOrgMemberRole] = useState<string>('member');
  const [orgMemberHint, setOrgMemberHint] = useState<string | null>(null);

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/orgs`);
      const data = await res.json().catch(() => []);
      const list: Group[] = Array.isArray(data) ? data : [];
      setGroups(list);
      const stored = Number(localStorage.getItem('selectedOrgId') || '0');
      const first = list[0]?.id || 0;
      const next = stored && list.some(g => g.id === stored) ? stored : first;
      if (next) { setSelectedGroupId(next); localStorage.setItem('selectedOrgId', String(next)); }
    } finally { setLoading(false); }
  };

  const fetchProjects = async (groupId: number) => {
    if (!groupId) return;
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/projects?org_id=${groupId}`);
      const data = await res.json().catch(() => []);
      const list: Project[] = Array.isArray(data) ? data : [];
      setProjects(list);
      const stored = Number(localStorage.getItem('selectedProjectId') || '0');
      const first = list[0]?.id || 0;
      const next = stored && list.some(p => p.id === stored) ? stored : first;
      if (next) { setSelectedProjectId(next); localStorage.setItem('selectedProjectId', String(next)); }
      else { setSelectedProjectId(0); }
    } finally { setLoading(false); }
  };

  const fetchMembers = async (projectId: number) => {
    if (!projectId) { setMembers([]); return; }
    const res = await apiFetch(`${API_URL}/projects/${projectId}/members`);
    const data = await res.json().catch(() => []);
    setMembers(Array.isArray(data) ? data : []);
  };

  const fetchOrgMembers = async (orgId: number) => {
    if (!orgId) { setOrgMembers([]); return; }
    const res = await apiFetch(`${API_URL}/orgs/${orgId}/members`);
    const data = await res.json().catch(() => []);
    setOrgMembers(Array.isArray(data) ? data : []);
  };

  useEffect(() => { fetchGroups(); }, []);
  useEffect(() => {
    if (selectedGroupId) {
      fetchProjects(selectedGroupId);
      fetchOrgMembers(selectedGroupId);
    }
  }, [selectedGroupId]);
  useEffect(() => { fetchMembers(selectedProjectId); }, [selectedProjectId]);

  const createGroup = async () => {
    const name = newGroupName.trim();
    if (!name) return;
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/orgs`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      setNewGroupName('');
      await fetchGroups();
    } catch (e: any) { alert(e.message); }
    finally { setLoading(false); }
  };

  const createProject = async () => {
    const name = newProjectName.trim();
    if (!name || !selectedGroupId) return;
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/projects`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ org_id: selectedGroupId, name }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      setNewProjectName('');
      await fetchProjects(selectedGroupId);
    } catch (e: any) { alert(e.message); }
    finally { setLoading(false); }
  };

  const addMember = async () => {
    const login = memberLogin.trim();
    if (!login || !selectedProjectId) return;
    setMemberHint(null);
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/projects/${selectedProjectId}/members`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ login, role: memberRole, position: memberPosition.trim() }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      setMemberLogin('');
      setMemberPosition('');
      await fetchMembers(selectedProjectId);
    } catch (e: any) {
      if (e.message === 'user_not_found') setMemberHint('Пользователь не зарегистрирован');
      else setMemberHint(e.message);
    } finally { setLoading(false); }
  };

  const removeMember = async (userId: number) => {
    if (!selectedProjectId || !confirm('Удалить участника?')) return;
    await apiFetch(`${API_URL}/projects/${selectedProjectId}/members/${userId}`, { method: 'DELETE' });
    await fetchMembers(selectedProjectId);
  };

  const addOrgMember = async () => {
    const login = orgMemberLogin.trim();
    if (!login || !selectedGroupId) return;
    setOrgMemberHint(null);
    setLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/orgs/${selectedGroupId}/members`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ login, role: orgMemberRole }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data?.error || 'Ошибка');
      setOrgMemberLogin('');
      await fetchOrgMembers(selectedGroupId);
    } catch (e: any) {
      if (e.message === 'user_not_found') setOrgMemberHint('Пользователь не зарегистрирован');
      else setOrgMemberHint(e.message);
    } finally { setLoading(false); }
  };

  const removeOrgMember = async (userId: number) => {
    if (!selectedGroupId || !confirm('Удалить участника из организации?')) return;
    await apiFetch(`${API_URL}/orgs/${selectedGroupId}/members/${userId}`, { method: 'DELETE' });
    await fetchOrgMembers(selectedGroupId);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold text-gray-900">Управление</h1>
        {loading && <span className="text-xs text-gray-400">Загрузка…</span>}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">

        {/* Create group */}
        <Section title="Создать группу">
          <div className="flex gap-2">
            <input
              value={newGroupName}
              onChange={e => setNewGroupName(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && createGroup()}
              placeholder="Название группы"
              className={inputCls}
            />
            <button onClick={createGroup} className="text-white px-4 py-2.5 rounded-xl font-semibold text-sm shrink-0" style={{ background: P }}>
              Создать
            </button>
          </div>
          {groups.length > 0 && (
            <div className="mt-3 space-y-1">
              {groups.map(g => (
                <button
                  key={g.id}
                  onClick={() => { setSelectedGroupId(g.id); localStorage.setItem('selectedOrgId', String(g.id)); }}
                  className="w-full text-left text-sm px-3 py-2 rounded-xl transition-colors"
                  style={selectedGroupId === g.id ? { background: '#EDE9F7', color: P, fontWeight: 600 } : { color: '#374151' }}
                >
                  {g.name}
                </button>
              ))}
            </div>
          )}
        </Section>

        {/* Create project */}
        <Section title="Создать проект">
          {!selectedGroupId ? (
            <p className="text-sm text-gray-400">Сначала выберите группу.</p>
          ) : (
            <>
              <div className="flex gap-2 mb-3">
                <input
                  value={newProjectName}
                  onChange={e => setNewProjectName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && createProject()}
                  placeholder="Название проекта"
                  className={inputCls}
                />
                <button onClick={createProject} className="bg-gray-900 text-white px-4 py-2.5 rounded-xl font-semibold text-sm shrink-0">
                  Создать
                </button>
              </div>
              {projects.length > 0 && (
                <div className="space-y-1">
                  {projects.map(p => (
                    <button
                      key={p.id}
                      onClick={() => { setSelectedProjectId(p.id); localStorage.setItem('selectedProjectId', String(p.id)); }}
                      className="w-full text-left text-sm px-3 py-2 rounded-xl transition-colors"
                      style={selectedProjectId === p.id ? { background: '#EDE9F7', color: P, fontWeight: 600 } : { color: '#374151' }}
                    >
                      {p.name}
                    </button>
                  ))}
                </div>
              )}
            </>
          )}
        </Section>

        {/* Add project member */}
        <Section title="Добавить участника в проект">
          {!selectedProjectId ? (
            <p className="text-sm text-gray-400">Сначала выберите проект.</p>
          ) : (
            <div className="space-y-2">
              <div className="flex gap-2">
                <input
                  value={memberLogin}
                  onChange={e => { setMemberLogin(e.target.value); setMemberHint(null); }}
                  onKeyDown={e => e.key === 'Enter' && addMember()}
                  placeholder="Логин участника"
                  className={inputCls}
                />
                <select
                  value={memberRole}
                  onChange={e => setMemberRole(e.target.value)}
                  className={selectCls}
                >
                  <option value="qa">QA</option>
                  <option value="dev">Dev</option>
                  <option value="pm">PM</option>
                  <option value="viewer">Viewer</option>
                </select>
                <button onClick={addMember} className="text-white px-4 py-2.5 rounded-xl font-semibold text-sm shrink-0" style={{ background: '#10B981' }}>
                  Добавить
                </button>
              </div>
              <input
                value={memberPosition}
                onChange={e => setMemberPosition(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addMember()}
                placeholder="Должность (необязательно), напр. Руководитель проекта, Тимлид"
                className={inputCls + ' w-full'}
              />
              <div className="flex items-center justify-between">
                <p className="text-xs text-gray-400">Участник должен быть зарегистрирован</p>
                <button
                  onClick={async () => { await navigator.clipboard.writeText(`${window.location.origin}/login`); alert('Ссылка скопирована'); }}
                  className="text-xs font-bold"
                  style={{ color: P }}
                >
                  Скопировать ссылку
                </button>
              </div>
              {memberHint && <p className="text-xs text-orange-600 bg-orange-50 border border-orange-100 rounded-xl p-2">{memberHint}</p>}
            </div>
          )}
        </Section>

        {/* Project members list */}
        <Section title="Участники проекта">
          {!selectedProjectId ? (
            <p className="text-sm text-gray-400">Выберите проект.</p>
          ) : members.length === 0 ? (
            <p className="text-sm text-gray-400">Нет участников</p>
          ) : (
            <div className="space-y-2">
              {members.map(m => (
                <div key={m.user_id} className="flex items-center gap-2 p-2 rounded-xl border border-gray-100">
                  <div className="flex-1 min-w-0">
                    <span className="text-sm text-gray-700 truncate block">{m.login}</span>
                    {m.position && <span className="text-[11px] font-semibold" style={{ color: P }}>{m.position}</span>}
                  </div>
                  <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full shrink-0">{roleLabel(m.role)}</span>
                  <button onClick={() => removeMember(m.user_id)} className="text-xs font-bold text-red-400 hover:text-red-600 shrink-0">
                    Удалить
                  </button>
                </div>
              ))}
            </div>
          )}
        </Section>

        {/* Add org member */}
        <Section title="Добавить участника в организацию">
          {!selectedGroupId ? (
            <p className="text-sm text-gray-400">Сначала выберите группу.</p>
          ) : (
            <div className="space-y-2">
              <div className="flex gap-2">
                <input
                  value={orgMemberLogin}
                  onChange={e => { setOrgMemberLogin(e.target.value); setOrgMemberHint(null); }}
                  onKeyDown={e => e.key === 'Enter' && addOrgMember()}
                  placeholder="Логин участника"
                  className={inputCls}
                />
                <select
                  value={orgMemberRole}
                  onChange={e => setOrgMemberRole(e.target.value)}
                  className={selectCls}
                >
                  <option value="member">Участник</option>
                  <option value="admin">Администратор</option>
                </select>
                <button onClick={addOrgMember} className="text-white px-4 py-2.5 rounded-xl font-semibold text-sm shrink-0" style={{ background: '#10B981' }}>
                  Добавить
                </button>
              </div>
              {orgMemberHint && <p className="text-xs text-orange-600 bg-orange-50 border border-orange-100 rounded-xl p-2">{orgMemberHint}</p>}
            </div>
          )}
        </Section>

        {/* Org members list */}
        <Section title="Участники организации">
          {!selectedGroupId ? (
            <p className="text-sm text-gray-400">Выберите группу.</p>
          ) : orgMembers.length === 0 ? (
            <p className="text-sm text-gray-400">Нет участников</p>
          ) : (
            <div className="space-y-2">
              {orgMembers.map(m => (
                <div key={m.user_id} className="flex items-center gap-2 p-2 rounded-xl border border-gray-100">
                  <span className="text-sm text-gray-700 flex-1 truncate">{m.login}</span>
                  <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full shrink-0">{roleLabel(m.role)}</span>
                  {m.role !== 'owner' && (
                    <button onClick={() => removeOrgMember(m.user_id)} className="text-xs font-bold text-red-400 hover:text-red-600 shrink-0">
                      Удалить
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </Section>

      </div>
    </div>
  );
}
