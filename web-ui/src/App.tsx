import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Connect from './pages/Connect';
import Dashboard from './pages/Dashboard';
import { useDbStore } from './stores/dbStore';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const connected = useDbStore((state) => state.connected);

  if (!connected) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Connect />} />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
