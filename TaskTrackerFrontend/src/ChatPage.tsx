import { useEffect, useRef, useState } from 'react';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';
const PL = '#EDE9F7';

type Thread = {
  id: number;
  scope: 'org' | 'project' | 'dm';
  peer_login?: string;
  title?: string;
  last_message?: string;
  unread_count?: number;
};
type Msg = {
  id: number;
  user_id: number;
  user_login: string;
  body: string;
  created_at: string;
  edited_at?: string;
  deleted_at?: string;
};

function getInitials(str: string) {
  return str.slice(0, 2).toUpperCase();
}

function avatarColor(str: string) {
  const colors = ['#7C5CBF', '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899'];
  let hash = 0;
  for (let i = 0; i < str.length; i++) hash = str.charCodeAt(i) + ((hash << 5) - hash);
  return colors[Math.abs(hash) % colors.length];
}

export default function ChatPage() {
  const currentUserId = Number(localStorage.getItem('userId') || '0');
  const selectedOrgId = Number(localStorage.getItem('selectedOrgId') || '0');
  const selectedProjectId = Number(localStorage.getItem('selectedProjectId') || '0');

  const [mode, setMode] = useState<'dm' | 'project' | 'org'>('dm');
  const [threads, setThreads] = useState<Thread[]>([]);
  const [activeThreadId, setActiveThreadId] = useState<number>(0);
  const [messages, setMessages] = useState<Msg[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [typingUsers, setTypingUsers] = useState<string[]>([]);
  const [editingMessageId, setEditingMessageId] = useState<number>(0);
  const [editingBody, setEditingBody] = useState('');
  const [body, setBody] = useState('');
  const [dmEmail, setDmEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [showDmInput, setShowDmInput] = useState(false);
  const listRef = useRef<HTMLDivElement | null>(null);
  const lastMsgIdRef = useRef<number>(0);
  const stickToBottomRef = useRef(true);
  const jumpToBottomRef = useRef(false);
  const isTypingRef = useRef(false);
  const typingTimerRef = useRef<number>(0);

  const tabs = [
    { key: 'dm' as const, label: 'Личные' },
    { key: 'project' as const, label: 'Проекты' },
    { key: 'org' as const, label: 'Организации' },
  ];

  const activeThread = threads.find(t => t.id === activeThreadId);
  const activeLabel = activeThread
    ? (activeThread.scope === 'dm' ? activeThread.peer_login : activeThread.title) || `#${activeThread.id}`
    : '';

  const loadThreads = async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      let url = `${API_URL}/chat/threads?scope=${mode}`;
      if (mode === 'org') url += `&org_id=${selectedOrgId}`;
      if (mode === 'project') url += `&project_id=${selectedProjectId}`;
      const res = await apiFetch(url);
      const data = await res.json().catch(() => []);
      const list = Array.isArray(data) ? data : [];
      setThreads(prev => {
        if (prev.length === list.length && prev.every((t, i) => t.id === list[i].id && t.last_message === list[i].last_message && t.unread_count === list[i].unread_count)) {
          return prev;
        }
        return list;
      });
      if (list[0]?.id && !list.some((t: Thread) => t.id === activeThreadId)) {
        setActiveThreadId(list[0].id);
      }
    } finally { if (!silent) setLoading(false); }
  };

  const ensureScopeThread = async () => {
    if (mode === 'dm') return;
    const payload = mode === 'org' ? { scope: 'org', org_id: selectedOrgId } : { scope: 'project', project_id: selectedProjectId };
    const res = await apiFetch(`${API_URL}/chat/threads`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok && data?.id) setActiveThreadId(data.id);
  };

  const loadMessages = async (beforeId?: number, afterId?: number) => {
    if (!activeThreadId) { setMessages([]); setHasMore(false); return; }
    const url = new URL(`${API_URL}/chat/threads/${activeThreadId}/messages`);
    if (afterId) {
      url.searchParams.set('after_id', String(afterId));
    } else {
      url.searchParams.set('limit', '40');
      if (beforeId) url.searchParams.set('before_id', String(beforeId));
    }
    const res = await apiFetch(url.toString());
    const data = await res.json().catch(() => []);
    const list = (Array.isArray(data) ? data : []) as Msg[];
    if (afterId) {
      if (list.length > 0) setMessages(prev => [...prev, ...list]);
    } else {
      setHasMore(list.length >= 40);
      if (beforeId) {
        const el = listRef.current;
        const prevHeight = el?.scrollHeight ?? 0;
        const prevTop = el?.scrollTop ?? 0;
        setMessages(prev => [...list, ...prev]);
        requestAnimationFrame(() => {
          if (el) el.scrollTop = el.scrollHeight - prevHeight + prevTop;
        });
      } else {
        setMessages(list);
      }
    }
  };

  const markRead = async (threadId: number) => {
    if (!threadId) return;
    await apiFetch(`${API_URL}/chat/threads/${threadId}/read`, { method: 'POST' });
  };

  useEffect(() => { loadThreads(); }, [mode]);
  useEffect(() => { ensureScopeThread(); }, [mode, selectedOrgId, selectedProjectId]);
  useEffect(() => {
    setEditingMessageId(0); setEditingBody('');
    lastMsgIdRef.current = 0;
    stickToBottomRef.current = true;
    jumpToBottomRef.current = true;
    loadMessages(); markRead(activeThreadId);
  }, [activeThreadId]);

  useEffect(() => {
    if (!messages.length) return;
    const el = listRef.current;
    const lastId = messages[messages.length - 1].id;
    const isNewTail = lastId !== lastMsgIdRef.current;
    lastMsgIdRef.current = lastId;
    if (!el) return;

    if (jumpToBottomRef.current) {
      jumpToBottomRef.current = false;
      el.scrollTop = el.scrollHeight;
      return;
    }
    if (isNewTail && stickToBottomRef.current) {
      el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
    }
  }, [messages]);

  const handleMessagesScroll = () => {
    const el = listRef.current;
    if (!el) return;
    stickToBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
  };

  useEffect(() => {
    const id = window.setInterval(() => {
      if (document.hidden) return;
      loadMessages(undefined, lastMsgIdRef.current || undefined);
    }, 3000);
    return () => window.clearInterval(id);
  }, [activeThreadId]);

  useEffect(() => {
    const id = window.setInterval(() => {
      if (document.hidden) return;
      loadThreads(true);
    }, 5000);
    return () => window.clearInterval(id);
  }, [mode, selectedOrgId, selectedProjectId]);

  useEffect(() => {
    if (!activeThreadId) return;
    const id = window.setInterval(async () => {
      if (document.hidden) return;
      const res = await apiFetch(`${API_URL}/chat/threads/${activeThreadId}/typing`);
      const data = await res.json().catch(() => []);
      const list = Array.isArray(data) ? data : [];
      setTypingUsers(prev => (prev.length === list.length && prev.every((v, i) => v === list[i])) ? prev : list);
    }, 1500);
    return () => window.clearInterval(id);
  }, [activeThreadId]);

  const updateTyping = async (isTyping: boolean) => {
    if (!activeThreadId) return;
    await apiFetch(`${API_URL}/chat/threads/${activeThreadId}/typing`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_typing: isTyping }),
    });
  };

  const updateTypingDebounced = (isTyping: boolean) => {
    if (isTyping) {
      if (!isTypingRef.current) {
        isTypingRef.current = true;
        updateTyping(true);
      }
      window.clearTimeout(typingTimerRef.current);
      typingTimerRef.current = window.setTimeout(() => {
        isTypingRef.current = false;
        updateTyping(false);
      }, 2000);
    } else {
      if (isTypingRef.current) {
        isTypingRef.current = false;
        window.clearTimeout(typingTimerRef.current);
        updateTyping(false);
      }
    }
  };

  const createDM = async () => {
    if (!dmEmail.trim()) return;
    const res = await apiFetch(`${API_URL}/chat/threads`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scope: 'dm', login: dmEmail.trim() }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) { alert(data?.error || 'Не удалось открыть чат'); return; }
    setDmEmail(''); setShowDmInput(false);
    await loadThreads();
    if (data?.id) setActiveThreadId(data.id);
  };

  const sendMessage = async () => {
    if (!activeThreadId || !body.trim()) return;
    const res = await apiFetch(`${API_URL}/chat/threads/${activeThreadId}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body: body.trim() }),
    });
    if (!res.ok) return;
    setBody('');
    updateTypingDebounced(false);
    stickToBottomRef.current = true;
    await loadMessages(undefined, lastMsgIdRef.current || undefined);
    await markRead(activeThreadId);
  };

  const saveEdit = async () => {
    if (!editingMessageId || !editingBody.trim()) return;
    const res = await apiFetch(`${API_URL}/chat/messages/${editingMessageId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body: editingBody.trim() }),
    });
    if (!res.ok) return;
    const newBody = editingBody.trim();
    setEditingMessageId(0); setEditingBody('');
    setMessages(prev => prev.map(m => m.id === editingMessageId ? { ...m, body: newBody, edited_at: new Date().toISOString() } : m));
  };

  const deleteMessage = async (id: number) => {
    if (!confirm('Удалить сообщение?')) return;
    const res = await apiFetch(`${API_URL}/chat/messages/${id}`, { method: 'DELETE' });
    if (!res.ok) return;
    setMessages(prev => prev.map(m => m.id === id ? { ...m, deleted_at: new Date().toISOString() } : m));
  };

  return (
    <div className="flex h-full" style={{ minHeight: 'calc(100vh - 56px)' }}>
      {/* Left panel */}
      <div className="w-[260px] bg-white border-r border-gray-100 flex flex-col shrink-0">
        {/* Search */}
        <div className="p-3 border-b border-gray-100">
          <div className="relative">
            <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            <input placeholder="Поиск" className="w-full bg-gray-50 border border-gray-100 rounded-lg pl-8 pr-3 py-2 text-xs outline-none" />
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-100">
          {tabs.map(t => (
            <button
              key={t.key}
              onClick={() => setMode(t.key)}
              className="flex-1 py-2.5 text-xs font-medium transition-colors"
              style={mode === t.key ? { color: P, borderBottom: `2px solid ${P}` } : { color: '#9CA3AF' }}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* DM create */}
        {mode === 'dm' && (
          <div className="p-2 border-b border-gray-50">
            {showDmInput ? (
              <div className="flex gap-1">
                <input
                  value={dmEmail}
                  onChange={e => setDmEmail(e.target.value)}
                  placeholder="логин..."
                  className="flex-1 text-xs px-2 py-1.5 border border-gray-200 rounded-lg outline-none"
                  onKeyDown={e => e.key === 'Enter' && createDM()}
                />
                <button onClick={createDM} className="text-xs font-bold text-white px-2 py-1.5 rounded-lg" style={{ background: P }}>OK</button>
                <button onClick={() => setShowDmInput(false)} className="text-xs text-gray-400 px-1">✕</button>
              </div>
            ) : (
              <button onClick={() => setShowDmInput(true)} className="w-full text-xs text-gray-400 hover:text-[#7C5CBF] py-1 transition-colors">+ Новый чат</button>
            )}
          </div>
        )}

        {/* Thread list */}
        <div className="flex-1 overflow-y-auto">
          {loading && <p className="text-xs text-gray-400 p-3">Загрузка...</p>}
          {threads.map(t => {
            const label = t.scope === 'dm' ? (t.peer_login || `DM #${t.id}`) : (t.title || `#${t.id}`);
            const active = t.id === activeThreadId;
            return (
              <button
                key={t.id}
                onClick={() => setActiveThreadId(t.id)}
                className="w-full flex items-center gap-3 px-3 py-3 text-left transition-colors hover:bg-gray-50"
                style={active ? { background: PL } : {}}
              >
                <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-bold shrink-0" style={{ background: avatarColor(label) }}>
                  {getInitials(label)}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium text-gray-800 truncate">{label}</p>
                    {!!t.unread_count && t.unread_count > 0 && (
                      <span className="text-[10px] font-bold text-white rounded-full px-1.5 py-0.5 shrink-0 ml-1" style={{ background: P }}>{t.unread_count}</span>
                    )}
                  </div>
                  {t.last_message && <p className="text-xs text-gray-400 truncate mt-0.5">{t.last_message}</p>}
                </div>
              </button>
            );
          })}
          {threads.length === 0 && !loading && <p className="text-xs text-gray-400 p-3">Нет чатов</p>}
        </div>
      </div>

      {/* Right panel: chat */}
      <div className="flex-1 flex flex-col">
        {/* Chat header */}
        {activeLabel ? (
          <div className="flex items-center gap-3 px-5 py-3 bg-white border-b border-gray-100 shrink-0">
            <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-bold" style={{ background: avatarColor(activeLabel) }}>
              {getInitials(activeLabel)}
            </div>
            <div>
              <p className="font-semibold text-gray-900 text-sm">{activeLabel}</p>
              {typingUsers.length > 0 && <p className="text-xs text-gray-400">Печатает...</p>}
            </div>
          </div>
        ) : (
          <div className="px-5 py-3 bg-white border-b border-gray-100 shrink-0 h-[56px]" />
        )}

        {/* Load more */}
        {hasMore && messages.length > 0 && (
          <button onClick={() => loadMessages(messages[0].id)} className="py-2 text-xs font-bold text-center" style={{ color: P }}>
            Загрузить старые сообщения
          </button>
        )}

        {/* Messages */}
        <div ref={listRef} onScroll={handleMessagesScroll} className="flex-1 overflow-y-auto px-5 py-4 space-y-3">
          {messages.map(m => {
            const mine = m.user_id === currentUserId;
            return (
              <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[70%] rounded-2xl px-4 py-2.5 ${mine ? 'text-white' : 'bg-white border border-gray-100 text-gray-800'}`} style={mine ? { background: P } : {}}>
                  {!mine && <p className="text-[10px] font-medium mb-1 opacity-60">{m.user_login}</p>}
                  {editingMessageId === m.id ? (
                    <div className="space-y-2">
                      <textarea value={editingBody} onChange={e => setEditingBody(e.target.value)} className="w-full text-sm rounded-lg border p-2 text-gray-900 outline-none" rows={2} />
                      <div className="flex gap-2 justify-end">
                        <button onClick={() => { setEditingMessageId(0); setEditingBody(''); }} className="text-xs opacity-70">Отмена</button>
                        <button onClick={saveEdit} className="text-xs font-bold underline">Сохранить</button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <p className={`text-sm whitespace-pre-wrap ${m.deleted_at ? 'italic opacity-60' : ''}`}>
                        {m.deleted_at ? 'Сообщение удалено' : m.body}
                      </p>
                      {mine && !m.deleted_at && (
                        <div className="flex gap-2 mt-1 justify-end text-[10px] opacity-70">
                          {m.edited_at && <span>изм.</span>}
                          <button onClick={() => { setEditingMessageId(m.id); setEditingBody(m.body); }} className="underline">ред.</button>
                          <button onClick={() => deleteMessage(m.id)} className="underline">удал.</button>
                        </div>
                      )}
                    </>
                  )}
                </div>
              </div>
            );
          })}
          {messages.length === 0 && activeThreadId && (
            <p className="text-center text-sm text-gray-400 mt-8">Нет сообщений</p>
          )}
        </div>

        {/* Input */}
        <div className="px-5 py-3 bg-white border-t border-gray-100 flex items-center gap-3">
          <input
            value={body}
            onChange={e => { setBody(e.target.value); updateTypingDebounced(e.target.value.trim().length > 0); }}
            placeholder={activeThreadId ? 'Начните писать...' : 'Выберите чат'}
            className="flex-1 bg-gray-50 border border-gray-100 rounded-xl px-4 py-2.5 text-sm outline-none focus:border-[#7C5CBF]"
            disabled={!activeThreadId}
            onKeyDown={e => { if (e.key === 'Enter') sendMessage(); }}
            onBlur={() => updateTypingDebounced(false)}
          />
          <button
            onClick={sendMessage}
            disabled={!activeThreadId || !body.trim()}
            className="w-9 h-9 rounded-full flex items-center justify-center text-white shrink-0 disabled:opacity-40 transition-opacity"
            style={{ background: P }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
