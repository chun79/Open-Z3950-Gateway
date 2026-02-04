import { BrowserRouter, Routes, Route, NavLink, Navigate } from 'react-router-dom'
import Search from './pages/Search'
import BookDetail from './pages/BookDetail'
import Requests from './pages/Requests'
import Browse from './pages/Browse'
import Login from './pages/Login'
import Settings from './pages/Settings'
import AdminDashboard from './pages/AdminDashboard'
import MyLibrary from './pages/MyLibrary'
import MARCEditor from './pages/MARCEditor'
import { AuthProvider, useAuth } from './context/AuthContext'
import { I18nProvider, useI18n } from './context/I18nContext'
import { ThemeProvider, useTheme } from './context/ThemeContext'

function ProtectedRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function AdminRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated, user } = useAuth();
  const { t } = useI18n();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (user?.role !== 'admin') {
    return <div>{t('common.access_denied')}</div>;
  }
  return children;
}

function Navigation() {
  const { isAuthenticated, logout, user } = useAuth();
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  
  return (
    <nav>
      <ul>
        <li><strong>📚 {t('nav.brand')}</strong></li>
      </ul>
      <ul>
        {isAuthenticated ? (
          <>
            <li>
              <NavLink to="/" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                {t('nav.search')}
              </NavLink>
            </li>
            <li>
              <NavLink to="/browse" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                {t('nav.browse')}
              </NavLink>
            </li>
            <li>
              <NavLink to="/my-library" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                My Card
              </NavLink>
            </li>
            {user?.role === 'admin' && (
              <>
                <li>
                  <NavLink to="/admin" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                    Dashboard
                  </NavLink>
                </li>
                <li>
                  <NavLink to="/settings" role="button" className={({ isActive }) => isActive ? '' : 'outline'}>
                    {t('nav.settings')}
                  </NavLink>
                </li>
              </>
            )}
            <li>
              <button 
                className="outline secondary" 
                style={{ padding: '5px 10px', fontSize: '0.8em', marginRight: '5px' }}
                onClick={toggleTheme}
                title="Toggle Theme"
              >
                {theme === 'light' ? '🌙' : '☀️'}
              </button>
              <button 
                className="outline secondary" 
                style={{ padding: '5px 10px', fontSize: '0.8em', marginRight: '10px' }}
                onClick={() => setLocale(locale === 'en' ? 'zh' : 'en')}
              >
                {locale === 'en' ? '中文' : 'English'}
              </button>
              <button onClick={logout} className="outline secondary">{t('nav.logout')}</button>
            </li>
          </>
        ) : (
          <li>
            <NavLink to="/login" role="button" className="outline">{t('nav.login')}</NavLink>
          </li>
        )}
      </ul>
    </nav>
  )
}

function App() {
  return (
    <ThemeProvider>
      <I18nProvider>
        <AuthProvider>
          <BrowserRouter>
            <div className="container">
              <Navigation />
              <main>
                <Routes>
                  <Route path="/login" element={<Login />} />
                  <Route path="/" element={<ProtectedRoute><Search /></ProtectedRoute>} />
                  <Route path="/book/:db/:id" element={<ProtectedRoute><BookDetail /></ProtectedRoute>} />
                  <Route path="/browse" element={<ProtectedRoute><Browse /></ProtectedRoute>} />
                  <Route path="/requests" element={<ProtectedRoute><Requests /></ProtectedRoute>} />
                  <Route path="/my-library" element={<ProtectedRoute><MyLibrary /></ProtectedRoute>} />
                  <Route path="/edit/:db/:id" element={<ProtectedRoute><MARCEditor /></ProtectedRoute>} />
                  <Route path="/settings" element={<AdminRoute><Settings /></AdminRoute>} />
                  <Route path="/admin" element={<AdminRoute><AdminDashboard /></AdminRoute>} />
                </Routes>
              </main>
            </div>
          </BrowserRouter>
        </AuthProvider>
      </I18nProvider>
    </ThemeProvider>
  )
}

export default App