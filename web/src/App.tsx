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

  return (
    <div className="flex min-h-screen bg-[var(--bg)] text-[var(--text)]">
      {token && <Sidebar />}
      <div className="flex-1 flex flex-col">
        {token && <TopBar />}
        <main className="flex-1 overflow-y-auto bg-[var(--bg-subtle)]/80 p-6 md:p-10">
          <AnimatePresence mode="wait">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route
                path="/dashboard"
                element={
                  <ProtectedRoute>
                    <motion.div
                      key="dashboard"
                      initial={{ opacity: 0, y: 16 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -16 }}
                      transition={{ duration: 0.3, ease: 'easeOut' }}
                      className="mx-auto max-w-5xl space-y-6"
                    >
                      <Dashboard />
                    </motion.div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/assets"
                element={
                  <ProtectedRoute>
                    <motion.div
                      key="assets"
                      initial={{ opacity: 0, y: 16 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -16 }}
                      transition={{ duration: 0.3, ease: 'easeOut' }}
                      className="mx-auto max-w-5xl"
                    >
                      <Assets />
                    </motion.div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/ip-allowlist"
                element={
                  <ProtectedRoute>
                    <motion.div
                      key="ip-allowlist"
                      initial={{ opacity: 0, y: 16 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -16 }}
                      transition={{ duration: 0.3, ease: 'easeOut' }}
                      className="mx-auto max-w-5xl"
                    >
                      <IPAllowlist />
                    </motion.div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/audit-logs"
                element={
                  <ProtectedRoute>
                    <motion.div
                      key="audit-logs"
                      initial={{ opacity: 0, y: 16 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -16 }}
                      transition={{ duration: 0.3, ease: 'easeOut' }}
                      className="mx-auto max-w-5xl"
                    >
                      <AuditLogs />
                    </motion.div>
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/users"
                element={
                  <ProtectedRoute>
                    <motion.div
                      key="users"
                      initial={{ opacity: 0, y: 16 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, y: -16 }}
                      transition={{ duration: 0.3, ease: 'easeOut' }}
                      className="mx-auto max-w-5xl"
                    >
                      <Users />
                    </motion.div>
                  </ProtectedRoute>
                }
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
