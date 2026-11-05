import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import AppLayout from './components/Layout';
import StudentsPage from './pages/students/StudentsPage';
import CoursesPage from './pages/courses/CoursesPage';
import GradesPage from './pages/grades/GradesPage';
import ChatPage from './pages/chat/ChatPage';
import LoginPage from './pages/login/LoginPage';
import TokensPage from './pages/tokens/TokensPage';
import { AuthProvider } from './auth/AuthContext';
import { useAuth } from './auth/context';

function AppRoutes() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="app-loading">
        <Spin size="large" />
      </div>
    );
  }

  if (!user) {
    return <LoginPage />;
  }

  return (
    <Routes>
      <Route path="/" element={<AppLayout />}>
        <Route index element={<Navigate to="/students" replace />} />
        <Route path="students" element={<StudentsPage />} />
        <Route path="courses" element={<CoursesPage />} />
        <Route path="grades" element={<GradesPage />} />
        <Route path="chat" element={<ChatPage />} />
        <Route path="tokens" element={<TokensPage />} />
        <Route path="*" element={<Navigate to="/students" replace />} />
      </Route>
    </Routes>
  );
}

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;
