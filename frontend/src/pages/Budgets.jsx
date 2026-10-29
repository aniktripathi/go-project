import { useState } from 'react';
import { api } from '../services/api';
import { useFetch } from '../hooks/useFetch';
import { useCategories } from '../hooks/useCategories';
import { useToast } from '../context/ToastContext';
import { ConfirmDialog, EmptyState, ErrorBanner, Field, Loader } from '../components/Common';
import { BudgetBars } from './Dashboard';
import { MONTH_NAMES, formatMoney, formatPercent, validateAmount } from '../utils/format';

function BudgetForm({ initial, month, year, categories, onSaved, onCancel }) {
  const notify = useToast();
  const [form, setForm] = useState({ category: initial?.category || categories[0] || '', amount: initial ? String(initial.amount) : '' });
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(e) {
    e.preventDefault();
    const next = {};
    if (!form.category) next.category = 'Choose a category.';
    const msg = validateAmount(form.amount);
    if (msg) next.amount = msg;
    setErrors(next);
    setFormError('');
    if (Object.keys(next).length) return;
    setBusy(true);
    const body = { category: form.category, amount: form.amount.trim(), month, year };
    try {
      if (initial) await api.put(`/budgets/${initial.id}`, body);
      else await api.post('/budgets', body);
      notify(initial ? 'Budget updated.' : 'Budget created.');
      onSaved();
    } catch (err) {
      setFormError(err.message);
      setErrors(err.details || {});
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card form-card" onSubmit={submit} noValidate>
      <h2>{initial ? 'Edit budget' : 'New budget'} · {MONTH_NAMES[month - 1]} {year}</h2>
      <ErrorBanner message={formError} />
      <div className="form-grid">
        <Field label="Category" htmlFor="b-category" error={errors.category}>
          <select id="b-category" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}>
            {categories.map((c) => <option key={c}>{c}</option>)}
          </select>
        </Field>
        <Field label="Monthly limit (₹)" htmlFor="b-amount" error={errors.amount}>
          <input id="b-amount" inputMode="decimal" placeholder="0.00" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} />
        </Field>
      </div>
      <div className="form-actions">
        <button className="btn btn-primary" disabled={busy}>{busy ? 'Saving…' : 'Save budget'}</button>
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}

export default function Budgets() {
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());
  const [editing, setEditing] = useState(null); // null | 'new' | budget object
  const [toDelete, setToDelete] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const cats = useCategories();
  const notify = useToast();
  const { data, loading, error, reload } = useFetch(() => api.get('/budgets', { month, year }), [month, year]);

  const alerts = (data || []).filter((b) => b.status !== 'ok');
  const used = new Set((data || []).map((b) => b.category));
  const available = cats.expense_categories.filter((c) => !used.has(c));

  async function confirmDelete() {
    setDeleting(true);
    try {
      await api.del(`/budgets/${toDelete.id}`);
      notify('Budget deleted.');
      setToDelete(null);
      reload();
    } catch (err) {
      notify(err.message, 'error');
    } finally {
      setDeleting(false);
    }
  }

  return (
    <>
      <div className="page-head">
        <h1>Budgets</h1>
        <button className="btn btn-primary" disabled={available.length === 0} onClick={() => setEditing('new')}>+ New budget</button>
      </div>

      <div className="card filters">
        <label className="inline-label">Month
          <select value={month} onChange={(e) => { setMonth(Number(e.target.value)); setEditing(null); }}>
            {MONTH_NAMES.map((m, i) => <option key={m} value={i + 1}>{m}</option>)}
          </select>
        </label>
        <label className="inline-label">Year
          <input type="number" min="2000" max="2100" value={year} onChange={(e) => { const y = Number(e.target.value); if (y >= 2000 && y <= 2100) { setYear(y); setEditing(null); } }} />
        </label>
      </div>

      {alerts.map((b) => (
        <div key={b.id} className={`banner ${b.status === 'exceeded' ? 'banner-error' : 'banner-warn'}`} role="alert">
          {b.status === 'exceeded'
            ? `${b.category}: you are ${formatMoney(-b.remaining)} over your ${formatMoney(b.amount)} budget (${formatPercent(b.utilization)} used).`
            : `${b.category}: ${formatPercent(b.utilization)} of your budget used — only ${formatMoney(b.remaining)} left.`}
        </div>
      ))}

      <ErrorBanner message={error} onRetry={reload} />

      {editing && (
        <BudgetForm
          key={editing === 'new' ? 'new' : editing.id}
          initial={editing === 'new' ? null : editing}
          month={month}
          year={year}
          categories={editing === 'new' ? available : cats.expense_categories.filter((c) => c === editing.category || !used.has(c))}
          onSaved={() => { setEditing(null); reload(); }}
          onCancel={() => setEditing(null)}
        />
      )}

      {loading && !data ? <Loader /> : data && data.length === 0 ? (
        <EmptyState title="No budgets for this month" text="Set a monthly limit for a category to track your spending against it." />
      ) : data && (
        <div className="card">
          <BudgetBars budgets={data} />
          <div className="table-wrap">
            <table>
              <thead><tr><th>Category</th><th className="num">Budget</th><th className="num">Spent</th><th className="num">Remaining</th><th className="num">Used</th><th className="no-print">Actions</th></tr></thead>
              <tbody>
                {data.map((b) => (
                  <tr key={b.id}>
                    <td>{b.category}</td>
                    <td className="num">{formatMoney(b.amount)}</td>
                    <td className="num">{formatMoney(b.spent)}</td>
                    <td className={`num ${b.remaining < 0 ? 'neg' : ''}`}>{formatMoney(b.remaining)}</td>
                    <td className="num">{formatPercent(b.utilization)}</td>
                    <td className="actions no-print">
                      <button className="btn btn-small" onClick={() => setEditing(b)}>Edit</button>
                      <button className="btn btn-small btn-danger-outline" onClick={() => setToDelete(b)}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={Boolean(toDelete)}
        title="Delete this budget?"
        message={toDelete ? `The ${toDelete.category} budget for ${MONTH_NAMES[month - 1]} ${year} will be removed. Your transactions are not affected.` : ''}
        busy={deleting}
        onConfirm={confirmDelete}
        onCancel={() => setToDelete(null)}
      />
    </>
  );
}
