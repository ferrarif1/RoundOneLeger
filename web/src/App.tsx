import { useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { AnimatePresence, motion } from 'framer-motion';
import Sidebar from './components/Sidebar';
import TopBar from './components/TopBar';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import Assets from './pages/Assets';
import IPAllowlist from './pages/IPAllowlist';
import AuditLogs from './pages/AuditLogs';
import Users from './pages/Users';
import { useSession } from './hooks/useSession';

const ProtectedRoute = ({ children }: { children: JSX.Element }) => {
  const { token } = useSession();
  return token ? children : <Navigate to="/login" replace />;
};

const App = () => {
  const { token } = useSession();
  const [navCollapsed, setNavCollapsed] = useState(false);

  const renderProtectedPage = (key: string, className: string, content: JSX.Element) => (
    <ProtectedRoute>
      <motion.div
        key={key}
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -16 }}
        transition={{ duration: 0.3, ease: 'easeOut' }}
        className={className}
      >
        {content}
      </motion.div>
    </ProtectedRoute>
  );

  return (
    <div className="flex min-h-screen bg-[var(--bg)] text-[var(--text)]">
      {token && <Sidebar collapsed={navCollapsed} onToggle={() => setNavCollapsed((prev) => !prev)} />}
      <div className="flex-1 flex flex-col">
        {token && <TopBar />}
        <main className="flex-1 overflow-y-auto bg-[var(--bg-subtle)]/80 p-6 md:p-10">
          <AnimatePresence mode="wait">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route
                path="/dashboard"
                element={renderProtectedPage('dashboard', 'mx-auto max-w-5xl space-y-6', <Dashboard />)}
              />
              <Route
                path="/assets"
                element={renderProtectedPage('assets', 'mx-auto w-full max-w-[1680px]', <Assets />)}
              />
              <Route
                path="/ip-allowlist"
                element={renderProtectedPage('ip-allowlist', 'mx-auto max-w-5xl', <IPAllowlist />)}
              />
              <Route
                path="/audit-logs"
                element={renderProtectedPage('audit-logs', 'mx-auto max-w-5xl', <AuditLogs />)}
              />
              <Route
                path="/users"
                element={renderProtectedPage('users', 'mx-auto max-w-5xl', <Users />)}
              />
              <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
            </Routes>
          </AnimatePresence>
        </main>
      </div>
    </div>
  );
};

export default App;
