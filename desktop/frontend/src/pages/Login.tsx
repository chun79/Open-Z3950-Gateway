import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useI18n } from '../context/I18nContext';

export default function Login() {
  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    const endpoint = isLogin ? '/api/auth/login' : '/api/auth/register';

    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || 'Operation failed');
      }

      if (isLogin) {
        login(data.token, data.user);
        navigate('/');
      } else {
        // After register, switch to login
        setIsLogin(true);
        setSuccess(t('login.success'));
        setUsername('');
        setPassword('');
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const toggleMode = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsLogin(!isLogin);
    setError('');
    setSuccess('');
  }

  return (
    <article style={{ maxWidth: '400px', margin: '2rem auto' }}>
      <header><strong>{isLogin ? t('login.title') : t('login.register_title')}</strong></header>
      <form onSubmit={handleSubmit}>
        <label>
          {t('login.username')}
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </label>
        <label>
          {t('login.password')}
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        
        {error && <p className="pico-color-red-500" style={{ marginBottom: '10px' }}>❌ {error}</p>}
        {success && <p className="pico-color-green-500" style={{ marginBottom: '10px' }}>✅ {success}</p>}

        <button type="submit" aria-busy={loading}>
          {isLogin ? t('login.submit') : t('login.register_submit')}
        </button>
      </form>
      <footer>
        <small>
          {isLogin ? t('login.no_account') : t('login.has_account')}{' '}
          <a href="#" onClick={toggleMode}>
            {isLogin ? t('login.link_register') : t('login.link_login')}
          </a>
        </small>
      </footer>
    </article>
  );
}
