import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { api } from '../services/api';
import { useCategories } from '../hooks/useCategories';
import { useToast } from '../context/ToastContext';
import { ErrorBanner, Field, Loader } from '../components/Common';
import { todayISO, validateAmount } from '../utils/format';

// Used for both "Add Transaction" (/transactions/new) and "Edit Transaction" (/transactions/:id/edit).
export default function TransactionForm() {
  const { id } = useParams();
  const editing = Boolean(id);
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const notify = useToast();
  const cats = useCategories();

  const initialType = params.get('type') === 'income' ? 'income' : 'expense';
  const [form, setForm] = useState({ type: initialType, category: '', amount: '', transaction_date: todayISO(), payment_method: 'UPI', description: '' });
  const [errors, setErrors] = useState({});
  const [formError, setFormError] = useState('');
  const [loading, setLoading] = useState(editing);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!editing) return;
    api
      .get(`/transactions/${id}`)
      .then((res) => {
        const t = res.data;
        setForm({ type: t.type, category: t.category, amount: String(t.amount), transaction_date: t.transaction_date, payment_method: t.payment_method || 'UPI', description: t.description || '' });
      })
      .catch((err) => setFormError(err.message))
      .finally(() => setLoading(false));
  }, [editing, id]);

  const categories = form.type === 'income' ? cats.income_categories : cats.expense_categories;

  // Pick the first category when the list is ready or the type changes.
  useEffect(() => {
    if (categories.length && !categories.includes(form.category)) {
      setForm((f) => ({ ...f, category: editing && f.category ? f.category : categories[0] }));
    }
  }, [categories, form.category, editing]);

  const bind = (key) => ({ id: key, value: form[key], onChange: (e) => setForm({ ...form, [key]: e.target.value }) });

  async function submit(e) {
    e.preventDefault();
    const next = {};
    if (!categories.includes(form.category)) next.category = 'Choose a category.';
    const amountMsg = validateAmount(form.amount);
    if (amountMsg) next.amount = amountMsg;
    if (!form.transaction_date) next.transaction_date = 'Choose a date.';
    if (form.description.length > 255) next.description = 'Description must be at most 255 characters.';
    setErrors(next);
    setFormError('');
    if (Object.keys(next).length) return;

    const body = {
      type: form.type,
      category: form.category,
      amount: form.amount.trim(),
      transaction_date: form.transaction_date,
      description: form.description.trim(),
      payment_method: form.type === 'expense' ? form.payment_method : '',
    };
    setBusy(true);
    try {
      if (editing) await api.put(`/transactions/${id}`, body);
      else await api.post('/transactions', body);
      notify(editing ? 'Transaction updated.' : 'Transaction added.');
      navigate(form.type === 'income' ? '/income' : '/expenses');
    } catch (err) {
      setFormError(err.message);
      setErrors(err.details || {});
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <Loader />;

  return (
    <>
      <div className="page-head">
        <h1>{editing ? 'Edit transaction' : 'Add transaction'}</h1>
        <Link className="btn" to="/transactions">Back to list</Link>
      </div>
      <form className="card form-card" onSubmit={submit} noValidate>
        <ErrorBanner message={formError} />
        <div className="segmented" role="group" aria-label="Transaction type">
          {['expense', 'income'].map((t) => (
            <button key={t} type="button" className={form.type === t ? 'active' : ''} onClick={() => setForm({ ...form, type: t, category: '' })}>
              {t === 'expense' ? 'Expense' : 'Income'}
            </button>
          ))}
        </div>

        <div className="form-grid">
          <Field label="Category" htmlFor="category" error={errors.category}>
            <select {...bind('category')}>
              {categories.map((c) => <option key={c}>{c}</option>)}
            </select>
          </Field>
          <Field label="Amount (₹)" htmlFor="amount" error={errors.amount}>
            <input {...bind('amount')} inputMode="decimal" placeholder="0.00" autoComplete="off" />
          </Field>
          <Field label="Date" htmlFor="transaction_date" error={errors.transaction_date}>
            <input type="date" {...bind('transaction_date')} />
          </Field>
          {form.type === 'expense' && (
            <Field label="Payment method" htmlFor="payment_method" error={errors.payment_method}>
              <select {...bind('payment_method')}>
                {cats.payment_methods.map((m) => <option key={m}>{m}</option>)}
              </select>
            </Field>
          )}
        </div>
        <Field label="Description (optional)" htmlFor="description" error={errors.description}>
          <textarea rows="3" maxLength={255} {...bind('description')} />
        </Field>
        <div className="form-actions">
          <button className="btn btn-primary" disabled={busy}>{busy ? 'Saving…' : editing ? 'Save changes' : 'Add transaction'}</button>
          <Link className="btn" to="/transactions">Cancel</Link>
        </div>
      </form>
    </>
  );
}
