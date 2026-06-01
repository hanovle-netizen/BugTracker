import { useState, useEffect, useImperativeHandle, forwardRef } from 'react';
import { API_URL } from './config';
import { apiFetch } from './api';

interface BugsModalProps {
  task: any;
  onClose: () => void;
  setIsEditorOpen: (open: boolean) => void;
  setSelectedBugId: (id: string | undefined) => void;
  onBugsLoaded: (bugs: any[]) => void;
}

export interface BugsModalHandle {
  pushBugs: (bugs: any[]) => void;
}

const BugsModal = forwardRef<BugsModalHandle, BugsModalProps>(
  function BugsModal({ task, onClose, setIsEditorOpen, setSelectedBugId, onBugsLoaded }, ref) {
    const [bugs, setBugs] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const currentUserId = Number(localStorage.getItem('userId') || '0');
    const [pendingDeleteBugId, setPendingDeleteBugId] = useState<number | null>(null);

    const [filterStatus, setFilterStatus] = useState('');
    const [filterSeverity, setFilterSeverity] = useState('');
    const [filterPriority, setFilterPriority] = useState('');
    const [searchId, setSearchId] = useState('');

    useImperativeHandle(ref, () => ({
      pushBugs: (newBugs: any[]) => setBugs(newBugs),
    }));

    const fetchBugs = async () => {
      if (!task?.id) return;
      setLoading(true);
      try {
        const response = await apiFetch(`${API_URL}/bugs/${task.id}`);
        const data = await response.json();
        const serverBugs = Array.isArray(data) ? data : (data ? [data] : []);
        setBugs(serverBugs);
        onBugsLoaded(serverBugs);
      } catch (err) { console.error(err); }
      finally { setLoading(false); }
    };

    useEffect(() => {
      fetchBugs();
    }, [task?.id]); // eslint-disable-line react-hooks/exhaustive-deps

    const handleDeleteBug = async (bugId: number) => {
      if (!bugId || !currentUserId) return;
      if (!confirm('Удалить баг?')) return;

      setPendingDeleteBugId(bugId);
      try {
        const res = await apiFetch(`${API_URL}/bugs/${bugId}`, { method: 'DELETE' });

        if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
        await fetchBugs();
      } catch (err) {
        console.error(err);
        alert('Не удалось удалить баг');
      } finally {
        setPendingDeleteBugId(null);
      }
    };

    const visibleBugs = bugs.filter(bug => {
      if (searchId && String(bug.id) !== searchId.trim()) return false;
      if (filterStatus && bug.status !== filterStatus) return false;
      if (filterSeverity && bug.severity !== filterSeverity) return false;
      if (filterPriority && bug.priority !== filterPriority) return false;
      return true;
    });

    if (!task) return null;

    return (
      <div className="fixed inset-0 z-[150] flex items-center justify-center bg-black/50 backdrop-blur-md p-4">
        <div className="bg-white w-full max-w-lg rounded-2xl shadow-2xl flex flex-col max-h-[85vh]">
          <div className="p-6 border-b flex justify-between items-center bg-gray-50 rounded-t-2xl">
            <div>
              <h2 className="text-xl font-black text-gray-900 line-clamp-1">{task.title}</h2>
              <p className="text-[10px] text-[#7C5CBF] font-bold uppercase tracking-widest mt-1">Репозиторий багов</p>
            </div>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-900 text-3xl">&times;</button>
          </div>

          <div className="px-4 pt-4 pb-2 border-b bg-white space-y-2">
            <input
              type="text"
              placeholder="Поиск по ID..."
              value={searchId}
              onChange={e => setSearchId(e.target.value)}
              className="w-full px-3 py-2 rounded-xl border border-slate-200 text-sm outline-none"
            />
            <div className="flex gap-2">
              <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)} className="flex-1 px-2 py-2 rounded-xl border border-slate-200 text-xs outline-none">
                <option value="">Все статусы</option>
                {['New','Open','In Progress','Fixed','Ready for Retest','Verified','Reopened','Rejected',"Can't Reproduce"].map(s => <option key={s} value={s}>{s}</option>)}
              </select>
              <select value={filterSeverity} onChange={e => setFilterSeverity(e.target.value)} className="flex-1 px-2 py-2 rounded-xl border border-slate-200 text-xs outline-none">
                <option value="">Все severity</option>
                {['Blocker','Critical','Major','Minor'].map(s => <option key={s} value={s}>{s}</option>)}
              </select>
              <select value={filterPriority} onChange={e => setFilterPriority(e.target.value)} className="flex-1 px-2 py-2 rounded-xl border border-slate-200 text-xs outline-none">
                <option value="">Все priority</option>
                {['High','Medium','Low'].map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {loading ? (
              <div className="text-center py-10 text-gray-400">Загрузка багов...</div>
            ) : visibleBugs.length > 0 ? (
              visibleBugs.map((bug: any) => (
                <div key={bug.id} onClick={() => { setSelectedBugId(bug.id); setIsEditorOpen(true); }}
                  className="p-4 bg-red-50 border border-red-100 rounded-xl flex items-center justify-between cursor-pointer hover:border-red-400 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="w-2 h-2 bg-red-500 rounded-full shrink-0"></div>
                    <div>
                      <span className="text-sm font-bold text-gray-700">Bug #{bug.id_pk ?? bug.id}</span>
                      {bug.severity && <span className="ml-2 text-[10px] font-bold text-orange-500">{bug.severity}</span>}
                      {bug.priority && <span className="ml-1 text-[10px] font-bold text-slate-400">{bug.priority}</span>}
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-[10px] font-bold text-red-400 uppercase">{bug.status}</span>
                    {bug.created_by === currentUserId && (
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); handleDeleteBug(bug.id_pk ?? bug.id); }}
                        disabled={pendingDeleteBugId === (bug.id_pk ?? bug.id)}
                        className="text-[10px] font-bold text-red-600 bg-red-50 border border-red-100 px-2 py-1 rounded-lg hover:bg-red-100 disabled:opacity-60"
                      >
                        {pendingDeleteBugId === bug.id ? 'Удаляю…' : 'Удалить'}
                      </button>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-10 text-gray-400 italic">
                {bugs.length > 0 ? 'Нет багов по фильтру.' : 'Багов пока нет.'}
              </div>
            )}
          </div>

          <div className="p-4 border-t bg-white rounded-b-2xl">
            <button onClick={() => { setSelectedBugId(undefined); setIsEditorOpen(true); }}
              className="w-full py-3 border-2 border-dashed border-blue-200 text-blue-600 font-bold rounded-xl hover:bg-blue-50 transition-all">
              + Создать баг
            </button>
          </div>
        </div>
      </div>
    );
  }
);

export default BugsModal;
