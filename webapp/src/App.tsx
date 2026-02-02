import { BrowserRouter, Routes, Route, NavLink, Navigate } from 'react-router-dom'
import Search from './pages/Search'
import Requests from './pages/Requests'
import Browse from './pages/Browse'
import Login from './pages/Login'
import Settings from './pages/Settings'
import { AuthProvider, useAuth } from './context/AuthContext'

function ProtectedRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function AdminRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated, user } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (user?.role !== 'admin') {
    return <div>Access Denied: Admins only</div>;
  }
  return children;
}

function Navigation() {
  const { isAuthenticated, logout, user } = useAuth();
  return (
    <nav>
      <ul>
        <li><strong>📚 Open Z39.50 Gateway</strong></li>
      </ul>
      <ul>
        {isAuthenticated ? (
          <>
            <li>
              <NavLink to="/" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                Search
              </NavLink>
            </li>
            <li>
              <NavLink to="/browse" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                Browse
              </NavLink>
            </li>
            <li>
              <NavLink to="/requests" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                ILL Requests
              </NavLink>
            </li>
            {user?.role === 'admin' && (
              <li>
                <NavLink to="/settings" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                  Settings
                </NavLink>
              </li>
            )}
            <li>
              <span style={{ marginRight: '10px' }}>👤 {user?.username}</span>
              <button onClick={logout} className="outline secondary">Logout</button>
            </li>
          </>
        ) : (
          <li>
            <NavLink to="/login" role="button" className="outline">Login</NavLink>
          </li>
        )}
      </ul>
    </nav>
  )
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <div className="container">
          <Navigation />
          <main>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/" element={<ProtectedRoute><Search /></ProtectedRoute>} />
              <Route path="/browse" element={<ProtectedRoute><Browse /></ProtectedRoute>} />
              <Route path="/requests" element={<ProtectedRoute><Requests /></ProtectedRoute>} />
              <Route path="/settings" element={<AdminRoute><Settings /></AdminRoute>} />
            </Routes>
          </main>
        </div>
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App