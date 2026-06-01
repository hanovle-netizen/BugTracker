import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';

type User = { id: number; login: string; role: string; created_at?: string };

const ROLE_LABELS: Record<string, string> = {
  admin: 'Администратор', developer: 'Разработчик', qa: 'Тестировщик',
};
const SYSTEM_ROLES = ['admin', 'developer', 'qa'];
const roleLabel = (r: string) => ROLE_LABELS[r] ?? r;

export default function UsersPage() {
  const navigate = useNavigate();
  const role = localStorage.getItem('userRole') || '';

  useEffect(() => {
    if (role !== 'admin') navigate('/', { replace: true });
  }, []);

  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resetUserId, setResetUserId] = useState<number | null>(null);
  const [resetPassword, setResetPassword] = useState('');

  const fetchUsers = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch(`${API_URL}/admin/users`);
      if (res.status === 404 || res.status === 501) {
        setError('Эндпоинт /admin/users ещё не реализован на бэкенде.');
        return;
      }
      if (!res.ok) {
        setError('Не удалось загрузить пользователей.');
        return;
      }
      const data = await res.json().catch(() => []);
      setUsers(Array.isArray(data) ? data : []);
    } catch {
      setError('Ошибка соединения с сервером.');
    } finally { setLoading(false); }
  };

  useEffect(() => { fetchUsers(); }, []);

  const changeRole = async (userId: number, newRole: string) => {
    try {
      const res = await apiFetch(`${API_URL}/admin/users/${userId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole }),
      });
      if (!res.ok) throw new Error();
      setUsers(prev => prev.map(u => u.id === userId ? { ...u, role: newRole } : u));
    } catch { alert('Не удалось изменить роль'); }
  };

  const deleteUser = async (userId: number, login: string) => {
    if (!confirm(`Удалить пользователя ${login}?`)) return;
    try {
      const res = await apiFetch(`${API_URL}/admin/users/${userId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error();
      setUsers(prev => prev.filter(u => u.id !== userId));
    } catch { alert('Не удалось удалить пользователя'); }
  };

  const resetUserPassword = async (userId: number) => {
    if (!resetPassword.trim()) return;
    try {
      const res = await apiFetch(`${API_URL}/admin/users/${userId}/password`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_password: resetPassword }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data?.error || 'Ошибка');
      }
      setResetUserId(null);
      setResetPassword('');
      alert('Пароль изменён');
    } catch (e: any) { alert(e.message); }
  };

  const myId = Number(localStorage.getItem('userId') || '0');

  return (
    <div className="p-4 sm:p-6 max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold text-gray-900">Пользователи</h1>
        <button
          onClick={fetchUsers}
          className="text-sm font-medium px-3 py-2 rounded-xl border border-gray-200 text-gray-600 hover:border-[#7C5CBF] hover:text-[#7C5CBF] transition-colors"
        >
          Обновить
        </button>
      </div>

      {loading && (
        <div className="flex justify-center py-20">
          <div className="w-7 h-7 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: `${P} transparent ${P} ${P}` }} />
        </div>
      )}

      {error && (
        <div className="bg-orange-50 border border-orange-100 rounded-2xl p-5 text-center">
          <p className="text-orange-600 text-sm font-medium">{error}</p>
        </div>
      )}

      {!loading && !error && (
        <>
          {/* Desktop: table */}
          <div className="hidden sm:block bg-white rounded-2xl border border-gray-100 overflow-hidden">
            {users.length === 0 ? (
              <p className="text-center text-gray-400 text-sm py-10">Нет пользователей</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100">
                    <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Логин</th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">Роль</th>
                    <th className="px-4 py-3"></th>
                  </tr>
                </thead>
                <tbody>
                  {users.map(u => (
                    <tr key={u.id} className="border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-gray-800 font-medium">
                        {u.login}
                        {u.id === myId && <span className="ml-2 text-xs text-gray-400">(вы)</span>}
                      </td>
                      <td className="px-4 py-3">
                        {u.id === myId ? (
                          <span className="text-xs bg-purple-100 text-purple-700 px-2 py-1 rounded-full font-semibold">{roleLabel(u.role)}</span>
                        ) : (
                          <select
                            value={u.role}
                            onChange={e => changeRole(u.id, e.target.value)}
                            className="text-xs border border-gray-200 rounded-lg px-2 py-1 outline-none bg-white focus:border-[#7C5CBF]"
                          >
                            {SYSTEM_ROLES.map(r => <option key={r} value={r}>{roleLabel(r)}</option>)}
                          </select>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {u.id !== myId && (
                          <div className="flex flex-col items-end gap-1">
                            {resetUserId === u.id ? (
                              <div className="flex items-center gap-1">
                                <input
                                  type="password"
                                  value={resetPassword}
                                  onChange={e => setResetPassword(e.target.value)}
                                  placeholder="Новый пароль"
                                  className="text-xs border border-gray-200 rounded-lg px-2 py-1 outline-none w-32 focus:border-[#7C5CBF]"
                                />
                                <button
                                  onClick={() => resetUserPassword(u.id)}
                                  className="text-xs text-white px-2 py-1 rounded-lg font-medium"
                                  style={{ background: P }}
                                >
                                  ОК
                                </button>
                                <button
                                  onClick={() => { setResetUserId(null); setResetPassword(''); }}
                                  className="text-xs text-gray-400 hover:text-gray-600 font-medium"
                                >
                                  ✕
                                </button>
                              </div>
                            ) : (
                              <button
                                onClick={() => { setResetUserId(u.id); setResetPassword(''); }}
                                className="text-xs text-[#7C5CBF] hover:text-[#5a3f9a] font-medium transition-colors"
                              >
                                Сбросить пароль
                              </button>
                            )}
                            <button
                              onClick={() => deleteUser(u.id, u.login)}
                              className="text-xs text-red-400 hover:text-red-600 font-medium transition-colors"
                            >
                              Удалить
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {/* Mobile: cards */}
          <div className="sm:hidden space-y-3">
            {users.length === 0 ? (
              <p className="text-center text-gray-400 text-sm py-10">Нет пользователей</p>
            ) : users.map(u => (
              <div key={u.id} className="bg-white rounded-2xl border border-gray-100 p-4">
                <div className="flex items-center justify-between mb-3">
                  <p className="text-sm font-medium text-gray-800">
                    {u.login}
                    {u.id === myId && <span className="ml-2 text-xs text-gray-400">(вы)</span>}
                  </p>
                  {u.id === myId ? (
                    <span className="text-xs bg-purple-100 text-purple-700 px-2 py-1 rounded-full font-semibold">{roleLabel(u.role)}</span>
                  ) : (
                    <select
                      value={u.role}
                      onChange={e => changeRole(u.id, e.target.value)}
                      className="text-xs border border-gray-200 rounded-lg px-2 py-1 outline-none bg-white focus:border-[#7C5CBF]"
                    >
                      {SYSTEM_ROLES.map(r => <option key={r} value={r}>{roleLabel(r)}</option>)}
                    </select>
                  )}
                </div>

                {u.id !== myId && (
                  <div className="border-t border-gray-50 pt-3 space-y-2">
                    {resetUserId === u.id ? (
                      <div className="space-y-2">
                        <input
                          type="password"
                          value={resetPassword}
                          onChange={e => setResetPassword(e.target.value)}
                          placeholder="Новый пароль (мин. 8, A-z, 0-9)"
                          className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 outline-none focus:border-[#7C5CBF]"
                        />
                        <div className="flex gap-2">
                          <button
                            onClick={() => resetUserPassword(u.id)}
                            className="flex-1 py-2 rounded-lg text-white text-sm font-medium"
                            style={{ background: P }}
                          >
                            Сохранить пароль
                          </button>
                          <button
                            onClick={() => { setResetUserId(null); setResetPassword(''); }}
                            className="px-4 py-2 rounded-lg text-sm text-gray-500 border border-gray-200"
                          >
                            Отмена
                          </button>
                        </div>
                      </div>
                    ) : (
                      <div className="flex gap-2">
                        <button
                          onClick={() => { setResetUserId(u.id); setResetPassword(''); }}
                          className="flex-1 py-2 rounded-lg text-sm font-medium border transition-colors"
                          style={{ color: P, borderColor: P }}
                        >
                          Сбросить пароль
                        </button>
                        <button
                          onClick={() => deleteUser(u.id, u.login)}
                          className="px-4 py-2 rounded-lg text-sm font-medium text-red-400 border border-red-200 hover:bg-red-50 transition-colors"
                        >
                          Удалить
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
