import { useState, useEffect, useRef } from 'react';
import { API_URL } from './config';
import { apiFetch } from './api';
import BugsModal, { BugsModalHandle } from './BugsModal';
import BugDetailEditor from './BugDetailEditor';

const P = '#7C5CBF';

type ColKey = 'todo' | 'in_progress' | 'done';

const STATUS_TO_COL: Record<string, ColKey> = {
  'New': 'todo', 'Open': 'todo', 'Reopened': 'todo',
  'In Progress': 'in_progress',
  'Fixed': 'done', 'Ready for Retest': 'done', 'Verified': 'done',
  'Rejected': 'done', "Can't Reproduce": 'done',
};

const COL_TO_STATUS: Record<ColKey, string> = {
  todo: 'New',
  in_progress: 'In Progress',
  done: 'Verified',
};

const COLUMNS: { key: ColKey; label: string; accent: string; bg: string; border: string }[] = [
  { key: 'todo', label: 'К выполнению', accent: '#6B7280', bg: '#F9FAFB', border: '#E5E7EB' },
  { key: 'in_progress', label: 'В работе', accent: '#2563EB', bg: '#EFF6FF', border: '#BFDBFE' },
  { key: 'done', label: 'Готово', accent: '#16A34A', bg: '#F0FDF4', border: '#BBF7D0' },
];

function getCol(status?: string | null): ColKey {
  if (!status) return 'todo';
  return STATUS_TO_COL[status] ?? 'todo';
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '';
  const today = new Date();
  if (d.toDateString() === today.toDateString()) return 'Сегодня';
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' });
}

export default function MainPage() {
  const [tasks, setTasks] = useState<any[]>([]);
  const [projects, setProjects] = useState<any[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<number>(
    () => Number(localStorage.getItem('selectedProjectId') || '0')
  );
  const [isLoading, setIsLoading] = useState(true);
  const [selectedTask, setSelectedTask] = useState<any>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [selectedBugId, setSelectedBugId] = useState<string | undefined>();
  const [dragId, setDragId] = useState<number | null>(null);
  const [dragOver, setDragOver] = useState<ColKey | null>(null);
  const currentUserId = Number(localStorage.getItem('userId') || '0');
  const bugsModalRef = useRef<BugsModalHandle>(null);

  const loadProjects = async () => {
    try {
      const res = await apiFetch(`${API_URL}/orgs`);
      const orgsData = await res.json().catch(() => []);
      const orgList: any[] = Array.isArray(orgsData) ? orgsData : [];

      const allProjects: any[] = [];
      await Promise.all(orgList.map(async (org: any) => {
        const r = await apiFetch(`${API_URL}/projects?org_id=${org.id}`);
        const ps = await r.json().catch(() => []);
        if (Array.isArray(ps)) allProjects.push(...ps);
      }));
      setProjects(allProjects);

      const stored = Number(localStorage.getItem('selectedProjectId') || '0');
      const firstId = allProjects[0]?.id || 0;
      const next = stored && allProjects.some((p: any) => p.id === stored) ? stored : firstId;
      if (next) {
        setSelectedProjectId(next);
        localStorage.setItem('selectedProjectId', String(next));
      }
    } catch (e) { console.error(e); }
  };

  const fetchTasks = async (projectId = selectedProjectId) => {
    if (!projectId) { setTasks([]); setIsLoading(false); return; }
    setIsLoading(true);
    try {
      const res = await apiFetch(`${API_URL}/tasks?project_id=${projectId}`);
      const data = await res.json().catch(() => []);
      setTasks(Array.isArray(data) ? data : []);
    } catch (e) { console.error(e); }
    finally { setIsLoading(false); }
  };

  useEffect(() => { loadProjects(); }, []);
  useEffect(() => { fetchTasks(); }, [selectedProjectId]);

  const moveTask = async (taskId: number, col: ColKey) => {
    const newStatus = COL_TO_STATUS[col];
    setTasks(prev => prev.map(t => t.id === taskId ? { ...t, status: newStatus } : t));
    try {
      await apiFetch(`${API_URL}/tasks/${taskId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus }),
      });
    } catch { fetchTasks(); }
  };

  const handleDeleteTask = async (taskId: number) => {
    if (!confirm('Удалить задачу?')) return;
    try {
      await apiFetch(`${API_URL}/tasks/${taskId}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner_id: currentUserId }),
      });
      if (selectedTask?.id === taskId) { setSelectedTask(null); setIsEditorOpen(false); }
      await fetchTasks();
    } catch { alert('Не удалось удалить задачу'); }
  };

  const handleBugSavedInState = (updatedBugs: any[]) => {
    setTasks(prev => prev.map(t => t.id === selectedTask?.id ? { ...t, bugs: updatedBugs } : t));
    bugsModalRef.current?.pushBugs(updatedBugs);
  };

  if (!isLoading && projects.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full py-32 px-4 text-center">
        <div className="text-6xl mb-5">📚</div>
        <h2 className="text-xl font-bold text-gray-900 mb-2">Нет проектов</h2>
        <p className="text-gray-400 max-w-xs text-sm">Администратор создаст группу, проект и добавит вас в него.</p>
      </div>
    );
  }

  return (
    <div className="p-5 flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-3 mb-4 shrink-0">
        <h1 className="text-lg font-bold text-gray-900">Мои задачи</h1>
        <select
          value={selectedProjectId || ''}
          onChange={e => {
            const v = Number(e.target.value);
            setSelectedProjectId(v);
            localStorage.setItem('selectedProjectId', String(v));
          }}
          className="ml-auto px-3 py-2 rounded-xl border border-gray-200 text-sm text-gray-700 bg-white outline-none focus:border-[#7C5CBF] max-w-[200px] truncate"
        >
          <option value="">Выберите проект</option>
          {projects.map((p: any) => <option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
      </div>

      {/* Kanban */}
      {isLoading ? (
        <div className="flex justify-center py-20">
          <div className="w-7 h-7 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: `${P} transparent ${P} ${P}` }} />
        </div>
      ) : (
        <div className="flex gap-3 flex-1 min-h-0">
          {COLUMNS.map(col => {
            const colTasks = tasks.filter(t => getCol(t.status) === col.key);
            const isOver = dragOver === col.key;
            return (
              <div
                key={col.key}
                className="flex flex-col flex-1 rounded-2xl min-w-0 transition-all"
                style={{ background: isOver ? col.border : col.bg, border: `2px solid ${isOver ? col.accent : 'transparent'}` }}
                onDragOver={e => { e.preventDefault(); setDragOver(col.key); }}
                onDragLeave={() => setDragOver(null)}
                onDrop={e => {
                  e.preventDefault();
                  if (dragId !== null) moveTask(dragId, col.key);
                  setDragId(null);
                  setDragOver(null);
                }}
              >
                {/* Column header */}
                <div className="flex items-center gap-2 px-4 py-3 shrink-0">
                  <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: col.accent }} />
                  <span className="text-xs font-bold uppercase tracking-wide" style={{ color: col.accent }}>{col.label}</span>
                  <span className="ml-auto text-xs text-gray-400 font-semibold bg-white rounded-full px-2 py-0.5 border border-gray-100">{colTasks.length}</span>
                </div>

                {/* Cards */}
                <div className="flex-1 overflow-y-auto px-3 pb-3 space-y-2">
                  {colTasks.map(task => (
                    <div
                      key={task.id}
                      draggable
                      onDragStart={() => setDragId(task.id)}
                      onDragEnd={() => { setDragId(null); setDragOver(null); }}
                      onClick={() => setSelectedTask(task)}
                      className="group bg-white rounded-xl border border-gray-100 px-3 py-3 cursor-grab active:cursor-grabbing hover:border-[#C4B0E8] hover:shadow-sm transition-all select-none"
                      style={{ opacity: dragId === task.id ? 0.5 : 1 }}
                    >
                      <p className="text-sm font-medium text-gray-800 leading-snug mb-2">{task.title}</p>
                      <div className="flex items-center gap-2">
                        {task.updated_at || task.created_at
                          ? <span className="text-xs text-gray-400">{formatDate(task.updated_at || task.created_at)}</span>
                          : null}
                        {task.owner_id === currentUserId && (
                          <button
                            onClick={e => { e.stopPropagation(); handleDeleteTask(task.id); }}
                            className="ml-auto opacity-0 group-hover:opacity-100 text-xs text-red-400 hover:text-red-600 transition-all"
                          >
                            Удалить
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                  {colTasks.length === 0 && (
                    <div className="flex items-center justify-center h-16">
                      <p className="text-xs text-gray-300">Пусто</p>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {selectedTask && (
        <div style={isEditorOpen ? { display: 'none' } : undefined}>
          <BugsModal
            ref={bugsModalRef}
            task={tasks.find(t => t.id === selectedTask.id) || selectedTask}
            onClose={() => setSelectedTask(null)}
            setIsEditorOpen={setIsEditorOpen}
            setSelectedBugId={setSelectedBugId}
            onBugsLoaded={handleBugSavedInState}
          />
        </div>
      )}

      {isEditorOpen && selectedTask && (
        <BugDetailEditor
          isOpen={isEditorOpen}
          onClose={() => { setIsEditorOpen(false); setSelectedBugId(undefined); }}
          task={tasks.find(t => t.id === selectedTask.id) || selectedTask}
          currentBug={tasks.find(t => t.id === selectedTask.id)?.bugs?.find((b: any) => b.id === Number(selectedBugId))}
          bugId={selectedBugId}
          onBugSaved={handleBugSavedInState}
        />
      )}
    </div>
  );
}
