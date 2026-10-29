import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Bar, BarChart, CartesianGrid, Cell, Legend, Line, LineChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { api } from '../services/api';
import { useFetch } from '../hooks/useFetch';
import DateRangeFilter, { rangeParams } from '../components/DateRangeFilter';
import TransactionTable from '../components/TransactionTable';
import { EmptyState, ErrorBanner, Loader, SummaryCard } from '../components/Common';
import { formatDate, formatMoney, formatPercent, monthLabel } from '../utils/format';

export const COLORS = ['#2563eb', '#16a34a', '#f59e0b', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#65a30d', '#64748b'];
const money = (v) => formatMoney(v);

export function BudgetBars({ budgets }) {
  if (!budgets.length) return <p className="muted">No budgets set for this month. <Link to="/budgets">Create one</Link>.</p>;
  return (
    <div className="budget-list">
      {budgets.map((b) => (
        <div key={b.id} className="budget-row">
          <div className="budget-top">
            <strong>{b.category}</strong>
            <span className={`badge badge-${b.status}`}>{b.status === 'exceeded' ? 'Over budget' : b.status === 'warning' ? 'Near limit' : 'On track'}</span>
          </div>
          <div className="progress" role="progressbar" aria-valuenow={Math.min(b.utilization, 100)} aria-valuemin="0" aria-valuemax="100" aria-label={`${b.category} budget used`}>
            <div className={`progress-bar bar-${b.status}`} style={{ width: `${Math.min(b.utilization, 100)}%` }} />
          </div>
          <div className="budget-meta">
            {formatMoney(b.spent)} of {formatMoney(b.amount)} ({formatPercent(b.utilization)}) ·{' '}
            {b.remaining < 0 ? <span className="neg">{formatMoney(-b.remaining)} over</span> : <span>{formatMoney(b.remaining)} left</span>}
          </div>
        </div>
      ))}
    </div>
  );
}

function Prediction() {
  const { data, loading } = useFetch(() => api.get('/dashboard/prediction'), []);
  if (loading) return null;
  return (
    <div className="card">
      <h2>Expense prediction <span className="badge badge-info">Optional</span></h2>
      {data?.available ? (
        <>
          <div className="big-number">{formatMoney(data.predicted_total)}</div>
          <p className="muted">Estimated total expenses for {data.target_month}. {data.disclaimer}</p>
          {data.method && <p className="muted small">Method: {data.method} · based on {data.months_used} months</p>}
        </>
      ) : (
        <p className="muted">{data?.reason || 'Prediction is not available right now.'} The rest of the app works normally.</p>
      )}
    </div>
  );
}

export default function Dashboard() {
  const [filter, setFilter] = useState({ range: 'this_month', from: '', to: '' });
  const summary = useFetch(() => api.get('/dashboard/summary', rangeParams(filter)), [JSON.stringify(filter)]);
  const trends = useFetch(() => api.get('/dashboard/trends', { months: 6 }), []);

  const s = summary.data;
  const hasData = s && (Number(s.total_income) > 0 || Number(s.total_expenses) > 0);
  const trendData = (trends.data || []).map((m) => ({ name: monthLabel(m.label), Income: Number(m.income), Expenses: Number(m.expenses), Savings: Number(m.net_savings) }));
  const pieData = (s?.category_breakdown || []).map((c) => ({ name: c.category, value: Number(c.total) }));

  return (
    <>
      <div className="page-head"><h1>Dashboard</h1></div>
      <DateRangeFilter value={filter} onChange={setFilter} />
      <ErrorBanner message={summary.error} onRetry={summary.reload} />
      {summary.loading && !s ? <Loader /> : s && (
        <>
          <p className="muted small">Period: {formatDate(s.range.from)} – {formatDate(s.range.to)}</p>
          <div className="grid cards-4">
            <SummaryCard label="Total income" value={formatMoney(s.total_income)} tone="green" />
            <SummaryCard label="Total expenses" value={formatMoney(s.total_expenses)} tone="red"
              hint={s.previous_period.expense_change_pct != null ? `${s.previous_period.expense_change_pct > 0 ? '▲' : '▼'} ${Math.abs(s.previous_period.expense_change_pct)}% vs previous period` : 'No previous period data'} />
            <SummaryCard label="Net savings" value={formatMoney(s.net_savings)} tone={Number(s.net_savings) < 0 ? 'red' : 'blue'} hint="Income − expenses (not a bank balance)" />
            <SummaryCard label="Savings rate" value={formatPercent(s.savings_percentage)} hint="Share of income saved" />
          </div>
          <div className="grid cards-3 gap-top">
            <SummaryCard label="Average daily spending" value={formatMoney(s.average_daily_expenditure)} />
            <SummaryCard label="Highest spending category" value={s.highest_spending_category ? s.highest_spending_category.category : '—'}
              hint={s.highest_spending_category ? `${formatMoney(s.highest_spending_category.total)} (${formatPercent(s.highest_spending_category.percentage)})` : 'No expenses in this period'} />
            <SummaryCard label="Monthly expense growth" value={s.monthly_expense_growth_pct != null ? `${s.monthly_expense_growth_pct > 0 ? '+' : ''}${s.monthly_expense_growth_pct}%` : '—'} hint="Latest month vs the month before" />
          </div>

          {!hasData && (
            <EmptyState title="No records in this period" text="Add income or expenses to see charts and insights." action={<Link className="btn btn-primary" to="/transactions/new">Add a transaction</Link>} />
          )}

          <div className="grid cards-2 gap-top">
            <div className="card">
              <h2>Income vs expenses (last 6 months)</h2>
              {trends.loading ? <Loader /> : (
                <div className="chart"><ResponsiveContainer width="100%" height="100%">
                  <BarChart data={trendData}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="name" /><YAxis tickFormatter={(v) => `₹${v / 1000}k`} /><Tooltip formatter={money} /><Legend />
                    <Bar dataKey="Income" fill="#16a34a" /><Bar dataKey="Expenses" fill="#dc2626" /></BarChart>
                </ResponsiveContainer></div>
              )}
            </div>
            <div className="card">
              <h2>Spending by category</h2>
              {pieData.length === 0 ? <p className="muted">No expenses in this period.</p> : (
                <div className="chart"><ResponsiveContainer width="100%" height="100%">
                  <PieChart><Pie data={pieData} dataKey="value" nameKey="name" outerRadius="75%" label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}>
                    {pieData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}</Pie><Tooltip formatter={money} /></PieChart>
                </ResponsiveContainer></div>
              )}
            </div>
          </div>

          <div className="grid cards-2 gap-top">
            <div className="card">
              <h2>Monthly spending trend</h2>
              {trends.loading ? <Loader /> : (
                <div className="chart"><ResponsiveContainer width="100%" height="100%">
                  <LineChart data={trendData}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="name" /><YAxis tickFormatter={(v) => `₹${v / 1000}k`} /><Tooltip formatter={money} /><Legend />
                    <Line type="monotone" dataKey="Expenses" stroke="#dc2626" strokeWidth={2} /><Line type="monotone" dataKey="Savings" stroke="#2563eb" strokeWidth={2} /></LineChart>
                </ResponsiveContainer></div>
              )}
            </div>
            <div className="card">
              <h2>Budget utilization <span className="muted small">({s.budget_utilization.month}/{s.budget_utilization.year})</span></h2>
              <BudgetBars budgets={s.budget_utilization.budgets} />
            </div>
          </div>

          <div className="grid cards-2 gap-top">
            <div className="card flush">
              <div className="card-head"><h2>Recent transactions</h2><Link to="/transactions">View all</Link></div>
              {s.recent_transactions.length === 0 ? <p className="muted pad">Nothing recorded in this period.</p> : <TransactionTable rows={s.recent_transactions} compact />}
            </div>
            <Prediction />
          </div>
        </>
      )}
    </>
  );
}
