import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from './config';
import { apiFetch } from './api';

const P = '#7C5CBF';

export default function NewTaskPage() {
  const navigate = useNavigate();
  const [taskType, setTaskType] = useState('Задача');
  const [title, setTitle] = useState('');
  const [desc, setDesc] = useState('');
  const [photos, setPhotos] = useState<File[]>([]);
  const [previews, setPreviews] = useState<string[]>([]);
  const [pending, setPending] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const selectedProjectId = Number(localStorage.getItem('selectedProjectId') || '0');

  const handlePhotoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (!files.length) return;
    setPhotos(prev => [...prev, ...files]);
    files.forEach(file => {
      const reader = new FileReader();
      reader.onload = ev => setPreviews(prev => [...prev, ev.target?.result as string]);
      reader.readAsDataURL(file);
    });
    if (fileRef.current) fileRef.current.value = '';
  };

  const removePhoto = (i: number) => {
    setPhotos(prev => prev.filter((_, j) => j !== i));
    setPreviews(prev => prev.filter((_, j) => j !== i));
  };

  const handleCreate = async () => {
    const taskTitle = title.trim() || `${taskType}: ${desc.trim().slice(0, 40)}`;
    if (!taskTitle || !selectedProjectId) {
      alert(!selectedProjectId ? 'Выберите проект в разделе «Мои задачи»' : 'Введите описание');
      return;
    }
    setPending(true);
    try {
      const res = await apiFetch(`${API_URL}/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: taskTitle, description: desc.trim(), project_id: selectedProjectId }),
      });
      if (!res.ok) throw new Error('Не удалось создать задачу');
      const data = await res.json();
      const taskId = data.id;

      for (const photo of photos) {
        const form = new FormData();
        form.append('photo', photo);
        await apiFetch(`${API_URL}/tasks/${taskId}/photos`, { method: 'POST', body: form });
      }

      navigate('/');
    } catch (err: any) {
      alert(err.message);
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-full p-6">
      <div className="bg-white rounded-2xl p-8 w-full max-w-sm shadow-sm border border-gray-100">
        <h1 className="text-xl font-bold text-center text-gray-900 mb-6">Создать задачу</h1>

        <div className="mb-4">
          <p className="text-sm text-gray-500 mb-2">Выберите тип задачи</p>
          <div className="relative">
            <select
              value={taskType}
              onChange={e => setTaskType(e.target.value)}
              className="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm text-gray-700 bg-white appearance-none outline-none focus:border-[#7C5CBF]"
            >
              <option>Баг</option>
              <option>Задача</option>
              <option>Улучшение</option>
            </select>
            <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none">▾</span>
          </div>
        </div>

        <div className="mb-4">
          <input
            placeholder="Название задачи (необязательно)"
            value={title}
            onChange={e => setTitle(e.target.value)}
            className="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm text-gray-700 outline-none focus:border-[#7C5CBF]"
          />
        </div>

        <div className="mb-4">
          <textarea
            placeholder="Описание..."
            value={desc}
            onChange={e => setDesc(e.target.value)}
            rows={4}
            className="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm text-gray-700 resize-none outline-none focus:border-[#7C5CBF]"
          />
        </div>

        {/* Photos */}
        <div className="mb-6 space-y-2">
          {previews.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {previews.map((src, i) => (
                <div key={i} className="relative w-20 h-20 rounded-xl overflow-hidden border border-gray-200">
                  <img src={src} className="w-full h-full object-cover" />
                  <button
                    onClick={() => removePhoto(i)}
                    className="absolute top-1 right-1 bg-black/50 text-white rounded-full w-5 h-5 flex items-center justify-center text-[10px] hover:bg-black/70"
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          )}
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            className="w-full border-2 border-dashed border-gray-200 rounded-xl py-3 flex items-center justify-center gap-2 text-gray-400 hover:border-[#7C5CBF] hover:text-[#7C5CBF] transition-colors"
          >
            <span className="text-xl">📎</span>
            <span className="text-xs font-medium">{previews.length > 0 ? 'Добавить ещё' : 'Прикрепить фото'}</span>
          </button>
          <input
            ref={fileRef}
            type="file"
            accept="image/*"
            multiple
            className="hidden"
            onChange={handlePhotoChange}
          />
        </div>

        <button
          onClick={handleCreate}
          disabled={pending}
          className="w-full py-3 rounded-xl text-white font-semibold text-sm transition-opacity disabled:opacity-60"
          style={{ background: P }}
        >
          {pending ? 'Создание...' : 'Создать задачу'}
        </button>
      </div>
    </div>
  );
}
