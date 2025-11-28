import { useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
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

  return (
    <div className="eidos-ledger-root">
      <div className="flex min-h-screen gap-6 bg-[var(--bg)] text-[var(--text)]">
        {token && <Sidebar collapsed={navCollapsed} onToggle={() => setNavCollapsed((prev) => !prev)} />}
        <div className="flex-1 flex flex-col eidos-ledger-wrapper">
          {token && <TopBar />}
          <main className="flex-1 overflow-y-auto bg-white p-6 md:p-8 rounded-lg border border-[var(--line)] shadow-none">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route
                path="/dashboard"
                element={
                  <ProtectedRoute>
                    <div className="mx-auto max-w-5xl space-y-6">
                      <Dashboard />
                    </div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/assets"
                element={
                  <ProtectedRoute>
                    <div className="mx-auto w-full max-w-[1680px]">
                      <Assets />
                    </div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/ip-allowlist"
                element={
                  <ProtectedRoute>
                    <div className="mx-auto max-w-5xl">
                      <IPAllowlist />
                    </div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/audit-logs"
                element={
                  <ProtectedRoute>
                    <div className="mx-auto max-w-5xl">
                      <AuditLogs />
                    </div>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/users"
                element={
                  <ProtectedRoute>
                    <div className="mx-auto max-w-5xl">
                      <Users />
                    </div>
                  </ProtectedRoute>
                }
              />
              <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
            </Routes>
          </main>
        </div>
      </div>
    </div>
  );
};

export default App;
