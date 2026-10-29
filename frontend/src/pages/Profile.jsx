import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../context/ToastContext';
import { ErrorBanner, Field } from '../components/Common';
import { formatDate } from '../utils/format';

export default function Profile() {
  const { user, setUser, logout } = useAuth();
  const navigate = useNavigate();
  const notify = useToast();
  const [name, setName] = useState(user?.name || '');
  const [error, setError] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [busy, setBusy] = useState(false);

  async function save(e) {
    e.preventDefault();
    setError('');
    setFieldError('');
    if (name.trim().length < 2) return setFieldError('Name must be at least 2 characters.');
    setBusy(true);
    try {
      const res = await api.put('/auth/me', { name: name.trim() });
      setUser(res.data);
      notify('Profile updated.');
    } catch (err) {
      setError(err.message);
      setFieldError(err.details?.name || '');
    } finally {
      setBusy(false);
    }
  }

  async function handleLogout() {
    await logout();
    navigate('/login');
  }

  return (
    <>
      <div className="page-head"><h1>Profile</h1></div>
      <div className="card form-card">
        <dl className="profile-info">
          <dt>E-mail</dt><dd>{user?.email}</dd>
          <dt>Member since</dt><dd>{formatDate(user?.created_at)}</dd>
        </dl>
        <form onSubmit={save} noValidate>
          <ErrorBanner message={error} />
          <Field label="Display name" htmlFor="profile-name" error={fieldError}>
            <input id="profile-name" value={name} onChange={(e) => setName(e.target.value)} />
          </Field>
          <div className="form-actions">
            <button className="btn btn-primary" disabled={busy}>{busy ? 'Saving…' : 'Save name'}</button>
            <button type="button" className="btn btn-danger-outline" onClick={handleLogout}>Log out</button>
          </div>
        </form>
      </div>
    </>
  );
}
