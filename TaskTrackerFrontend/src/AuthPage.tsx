import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { API_URL } from './config';

export default function AuthPage() {
  const [isLogin, setIsLogin] = useState(true);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const navigate = useNavigate();

  const switchMode = () => {
    setIsLogin(prev => !prev);
    setLogin(''); setPassword(''); setFormError(null); setPending(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setPending(true);

    const url = isLogin ? `${API_URL}/login` : `${API_URL}/users`;

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(isLogin ? { login, password } : { login, password, role: 'qa' }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Ошибка при выполнении запроса');
      }

      const userId = data.id || data.Id;
      const token = data.token || data.jwt || data.access_token;

      if (userId) {
        if (!token) {
          throw new Error('Бэкенд не вернул JWT token');
        }
        localStorage.setItem('isAuthenticated', 'true');
        localStorage.setItem('userId', userId.toString());
        localStorage.setItem('userLogin', login);
        localStorage.setItem('jwtToken', token);
        localStorage.setItem('userRole', data.role || 'qa');
        navigate('/');
        window.location.reload();
      } else {
        throw new Error("Сервер не вернул ID пользователя");
      }
    } catch (err: any) {
      setFormError(err.message);
      console.error('Auth error:', err);
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="max-w-md w-full space-y-8 bg-white p-8 rounded-2xl shadow-xl border border-gray-100">
        <div>
          <h2 className="text-center text-3xl font-extrabold text-gray-900">
            {isLogin ? 'Войти в аккаунт' : 'Создать аккаунт'}
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Или{' '}
            <button
              onClick={switchMode}
              className="font-medium text-[#7C5CBF] hover:text-[#6548A8] underline"
            >
              {isLogin ? 'зарегистрироваться' : 'войти в существующий'}
            </button>
          </p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div className="rounded-md shadow-sm space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Логин</label>
              <input
                type="text"
                required
                className="appearance-none relative block w-full px-3 py-3 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="username"
                value={login}
                onChange={(e) => setLogin(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Пароль</label>
              <input
                type="password"
                required
                className="appearance-none relative block w-full px-3 py-3 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>

          {formError && (
            <div className="text-red-500 text-sm text-center bg-red-50 p-2 rounded-lg">
              {formError}
            </div>
          )}
          <button
            type="submit"
            disabled={pending}
            className="w-full flex justify-center py-3 px-4 rounded-xl text-sm font-semibold text-white disabled:opacity-60 transition-opacity"
            style={{ background: '#7C5CBF' }}
          >
            {pending ? 'Загрузка...' : (isLogin ? 'Войти' : 'Зарегистрироваться')}
          </button>
        </form>
      </div>
    </div>
  );
}
