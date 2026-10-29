import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../services/api';
import { useFetch } from '../hooks/useFetch';
import { useCategories } from '../hooks/useCategories';
import { useToast } from '../context/ToastContext';
import { ConfirmDialog, EmptyState, ErrorBanner, Loader, Pagination } from './Common';
import TransactionTable from './TransactionTable';
import { formatMoney } from '../utils/format';

const INITIAL = { search: '', category: '', payment_method: '', type: '', from: '', to: '', sort: 'date', order: 'desc', page: 1 };

// One list screen used by the Transactions, Income and Expenses pages.
// fixedType: '' (all), 'income' or 'expense'.
export default function TransactionList({ title, fixedType, endpoint }) {
  const cats = useCategories();
  const notify = useToast();
  const [filters, setFilters] = useState(INITIAL);
  const [searchText, setSearchText] = useState('');
  const [toDelete, setToDelete] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [dateError, setDateError] = useState('');

  // Reset filters when switching between Transactions / Income / Expenses.
  useEffect(() => {
    setFilters(INITIAL);
    setSearchText('');
  }, [endpoint]);

  // Wait 400 ms after typing before searching.
  useEffect(() => {
    const id = setTimeout(() => setFilters((f) => (f.search === searchText ? f : { ...f, search: searchText, page: 1 })), 400);
    return () => clearTimeout(id);
  }, [searchText]);

  const { data, meta, loading, error, reload } = useFetch(
    () => api.get(endpoint, { ...filters, page_size: 10 }),
    [endpoint, JSON.stringify(filters)],
  );

  const categoryOptions = fixedType === 'income' ? cats.income_categories
    : fixedType === 'expense' ? cats.expense_categories
    : [...new Set([...cats.income_categories, ...cats.expense_categories])];

  function set(key, value) {
    setFilters((f) => ({ ...f, [key]: value, page: 1 }));
  }

  function setDates(key, value) {
    const next = { ...filters, [key]: value };
    if (next.from && next.to && next.from > next.to) {
      setDateError('The start date must not be after the end date.');
      return;
    }
    setDateError('');
    set(key, value);
  }

  async function confirmDelete() {
    setDeleting(true);
    try {
      await api.del(`/transactions/${toDelete.id}`);
      notify('Transaction deleted.');
      setToDelete(null);
      reload();
    } catch (err) {
      notify(err.message, 'error');
    } finally {
      setDeleting(false);
    }
  }

  const addType = fixedType || 'expense';
  const hasFilters = Boolean(filters.search || filters.category || filters.payment_method || filters.type || filters.from || filters.to);

  return (
    <>
      <div className="page-head">
        <h1>{title}</h1>
        <Link className="btn btn-primary" to={`/transactions/new?type=${addType}`}>
          + Add {fixedType || 'transaction'}
        </Link>
      </div>

      <div className="card filters">
        <input type="search" placeholder="Search description or category…" aria-label="Search" value={searchText} onChange={(e) => setSearchText(e.target.value)} />
        {!fixedType && (
          <select aria-label="Type" value={filters.type} onChange={(e) => set('type', e.target.value)}>
            <option value="">All types</option>
            <option value="income">Income</option>
            <option value="expense">Expense</option>
          </select>
        )}
        <select aria-label="Category" value={filters.category} onChange={(e) => set('category', e.target.value)}>
          <option value="">All categories</option>
          {categoryOptions.map((c) => (
            <option key={c}>{c}</option>
          ))}
        </select>
        {fixedType !== 'income' && (
          <select aria-label="Payment method" value={filters.payment_method} onChange={(e) => set('payment_method', e.target.value)}>
            <option value="">All payment methods</option>
            {cats.payment_methods.map((m) => (
              <option key={m}>{m}</option>
            ))}
          </select>
        )}
        <label className="inline-label">
          From <input type="date" value={filters.from} onChange={(e) => setDates('from', e.target.value)} />
        </label>
        <label className="inline-label">
          To <input type="date" value={filters.to} onChange={(e) => setDates('to', e.target.value)} />
        </label>
        <select aria-label="Sort by" value={`${filters.sort}:${filters.order}`} onChange={(e) => { const [sort, order] = e.target.value.split(':'); setFilters((f) => ({ ...f, sort, order, page: 1 })); }}>
          <option value="date:desc">Newest first</option>
          <option value="date:asc">Oldest first</option>
          <option value="amount:desc">Highest amount</option>
          <option value="amount:asc">Lowest amount</option>
        </select>
        {hasFilters && (
          <button className="btn btn-small" onClick={() => { setFilters(INITIAL); setSearchText(''); setDateError(''); }}>
            Clear filters
          </button>
        )}
        {dateError && <span className="error" role="alert">{dateError}</span>}
      </div>

      <ErrorBanner message={error} onRetry={reload} />
      {loading && !data ? (
        <Loader />
      ) : data && data.length === 0 ? (
        <EmptyState
          title={hasFilters ? 'No records match your filters' : `No ${fixedType || 'transaction'} records yet`}
          text={hasFilters ? 'Try changing or clearing the filters.' : 'Add your first record to get started.'}
          action={!hasFilters && <Link className="btn btn-primary" to={`/transactions/new?type=${addType}`}>Add one now</Link>}
        />
      ) : (
        data && (
          <div className="card flush">
            <TransactionTable rows={data} onDelete={setToDelete} />
            <Pagination meta={meta} onPage={(p) => setFilters((f) => ({ ...f, page: p }))} />
          </div>
        )
      )}

      <ConfirmDialog
        open={Boolean(toDelete)}
        title="Delete this record?"
        message={toDelete ? `${toDelete.category} · ${formatMoney(toDelete.amount)} on ${toDelete.transaction_date}. This cannot be undone.` : ''}
        busy={deleting}
        onConfirm={confirmDelete}
        onCancel={() => setToDelete(null)}
      />
    </>
  );
}
