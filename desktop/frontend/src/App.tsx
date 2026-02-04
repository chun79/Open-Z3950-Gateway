import React, { useState } from 'react';
import { Routes, Route, NavLink, HashRouter } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { Search as SearchIcon, Library, Settings as SettingsIcon, Moon, Sun, Languages, BookOpen } from 'lucide-react';
import Search from './pages/Search';
import Settings from './pages/Settings';
import Bookshelf from './pages/Bookshelf';
import Requests from './pages/Requests';
import BookDetail from './pages/BookDetail';
import { I18nProvider, useI18n } from './context/I18nContext';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import { ErrorBoundary } from './components/ErrorBoundary';

// Minimal Auth Context Mock for Desktop (Single User)
const AuthContext = React.createContext({ isAuthenticated: true, user: { role: 'admin' } });

function Navigation() {
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  
  const navLinkStyle = ({ isActive }: { isActive: boolean }) => ({
    padding: '6px 12px',
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    textDecoration: 'none',
    color: isActive ? 'var(--pico-primary)' : 'var(--pico-color)',
    fontWeight: isActive ? 'bold' : 'normal'
  });
  
  return (
    <nav className="container-fluid" style={{padding: '10px 20px', borderBottom: '1px solid var(--pico-muted-border-color)'}}>
      <ul>
        <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <BookOpen size={24} />
          <strong>Desktop Gateway</strong>
        </li>
      </ul>
      <ul>
        <li>
          <NavLink to="/" style={navLinkStyle}>
            <SearchIcon size={18} />
            {t('nav.search')}
          </NavLink>
        </li>
        <li>
          <NavLink to="/bookshelf" style={navLinkStyle}>
            <Library size={18} />
            Bookshelf
          </NavLink>
        </li>
        <li>
          <NavLink to="/requests" style={navLinkStyle}>
            <BookOpen size={18} />
            {t('nav.requests')}
          </NavLink>
        </li>
        <li>
          <NavLink to="/settings" style={navLinkStyle}>
            <SettingsIcon size={18} />
            {t('nav.settings')}
          </NavLink>
        </li>
        <li>
          <div role="group" style={{ marginBottom: 0, marginLeft: '10px' }}>
            <button 
              className="outline secondary" 
              style={{ padding: '6px', border: 'none' }}
              onClick={toggleTheme}
              title="Toggle Theme"
            >
              {theme === 'light' ? <Moon size={18} /> : <Sun size={18} />}
            </button>
            <button 
              className="outline secondary" 
              style={{ padding: '6px', border: 'none' }}
              onClick={() => setLocale(locale === 'en' ? 'zh' : 'en')}
              title="Switch Language"
            >
              <Languages size={18} />
            </button>
          </div>
        </li>
      </ul>
    </nav>
  )
}

function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <I18nProvider>
          <AuthContext.Provider value={{ isAuthenticated: true, user: { role: 'admin' } }}>
            <HashRouter>
              <Toaster position="bottom-right" />
              <Navigation />
              <main className="container" style={{marginTop: '20px'}}>
                <Routes>
                  <Route path="/" element={<Search />} />
                  <Route path="/bookshelf" element={<Bookshelf />} />
                  <Route path="/requests" element={<Requests />} />
                  <Route path="/settings" element={<Settings />} />
                  <Route path="/book/:db/:id" element={<BookDetail />} />
                </Routes>
              </main>
            </HashRouter>
          </AuthContext.Provider>
        </I18nProvider>
      </ThemeProvider>
    </ErrorBoundary>
  )
}

export default App