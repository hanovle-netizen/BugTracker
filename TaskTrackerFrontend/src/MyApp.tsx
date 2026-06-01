import { Navigate, BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './Layout';
import AuthPage from './AuthPage';
import MainPage from './MainPage';
import AnalyticsPage from './AnalyticsPage';
import AdminPage from './AdminPage';
import ChatPage from './ChatPage';
import ProfilePage from './ProfilePage';
import NewTaskPage from './NewTaskPage';
import UsersPage from './UsersPage';

function isAuth() {
  return localStorage.getItem('isAuthenticated') === 'true' && !!localStorage.getItem('jwtToken');
}

function Protected({ el }: { el: React.ReactNode }) {
  if (!isAuth()) return <Navigate to="/login" replace />;
  return <Layout>{el}</Layout>;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<AuthPage />} />
        <Route path="/" element={<Protected el={<MainPage />} />} />
        <Route path="/new" element={<Protected el={<NewTaskPage />} />} />
        <Route path="/analytics" element={<Protected el={<AnalyticsPage />} />} />
        <Route path="/admin" element={<Protected el={<AdminPage />} />} />
        <Route path="/chat" element={<Protected el={<ChatPage />} />} />
        <Route path="/profile" element={<Protected el={<ProfilePage />} />} />
        <Route path="/users" element={<Protected el={<UsersPage />} />} />
      </Routes>
    </BrowserRouter>
  );
}
