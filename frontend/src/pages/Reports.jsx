import { useState } from 'react';
import { Bar, BarChart, CartesianGrid, Cell, Legend, Line, LineChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { useFetch } from '../hooks/useFetch';
import { useToast } from '../context/ToastContext';
import DateRangeFilter, { rangeParams } from '../components/DateRangeFilter';
import { EmptyState, ErrorBanner, Loader, SummaryCard } from '../components/Common';
import { BudgetBars, COLORS } from './Dashboard';
import { formatDate, formatMoney, formatPercent, monthLabel } from '../utils/format';

export default function Reports() {
  const { user } = useAuth();
  const notify = useToast();
  const [filter, setFilter] = useState({ range: 'last_3_months', from: '', to: '' });
  const [exporting, setExporting] = useState(false);
  const key = JSON.stringify(filter);

  const monthly = useFetch(() => api.get('/reports/monthly', rangeParams(filter)), [key]);
  const cats = useFetch(() => api.get('/reports/categories', rangeParams(filter)), [key]);
  const daily = useFetch(() => api.get('/reports/daily', rangeParams(filter)), [key]);
  const endMonth = monthly.data ? monthly.data.range.to.slice(0, 7).split('-').map(Number) : null; // [year, month]
  const budgets = useFetch(
    () => (endMonth ? api.get('/reports/budgets', { year: endMonth[0], month: endMonth[1] }) : Promise.resolve({ data: null })),
    [endMonth ? endMonth.join('-') : 'none'],
  );

  const error = monthly.error || cats.error || daily.error || budgets.error;
  const loading = monthly.loading || cats.loading || daily.loading;
  const reloadAll = () => { monthly.reload(); cats.reload(); daily.reload(); budgets.reload(); };

  async function exportCsv() {
    setExporting(true);
    try {
      const { blob, filename } = await api.download('/reports/export', rangeParams(filter));
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      notify('CSV file downloaded.');
    } catch (err) {
      notify(err.message, 'error');
    } finally {
      setExporting(false);
    }
  }

  const m = monthly.data;
  const monthData = (m?.months || []).map((x) => ({ name: monthLabel(x.label), Income: Number(x.income), Expenses: Number(x.expenses) }));
  const expCats = cats.data?.expense_categories || [];
  const incCats = cats.data?.income_categories || [];
  const days = daily.data?.days || [];
  const dayData = days.map((d) => ({ name: d.date.slice(5), Expenses: Number(d.expenses) }));
  const empty = m && Number(m.totals.income) === 0 && Number(m.totals.expenses) === 0;

  return (
    <>
      <div className="page-head">
        <h1>Financial reports</h1>
        <div className="head-actions no-print">
          <button className="btn" onClick={() => window.print()}>🖨 Print</button>
          <button className="btn btn-primary" onClick={exportCsv} disabled={exporting}>{exporting ? 'Preparing…' : '⬇ Export CSV'}</button>
        </div>
      </div>
      <DateRangeFilter value={filter} onChange={setFilter} />
      <ErrorBanner message={error} onRetry={reloadAll} />

      {loading && !m ? <Loader /> : m && (
        <div className="report">
          <div className="print-only report-header">
            <h2>Financial report — {user?.name}</h2>
            <p>Period: {formatDate(m.range.from)} to {formatDate(m.range.to)} · Generated {new Date().toLocaleDateString()}</p>
          </div>
          <p className="muted small no-print">Period: {formatDate(m.range.from)} – {formatDate(m.range.to)}</p>

          <div className="grid cards-4">
            <SummaryCard label="Total income" value={formatMoney(m.totals.income)} tone="green" />
            <SummaryCard label="Total expenses" value={formatMoney(m.totals.expenses)} tone="red" />
            <SummaryCard label="Net savings" value={formatMoney(m.totals.net_savings)} tone={Number(m.totals.net_savings) < 0 ? 'red' : 'blue'} />
            <SummaryCard label="Savings rate" value={formatPercent(m.totals.savings_rate)} />
          </div>

          {empty ? (
            <EmptyState title="No data for this period" text="Choose another date range or add some transactions." />
          ) : (
            <>
              <div className="card gap-top">
                <h2>Monthly income vs expenses</h2>
                <div className="chart"><ResponsiveContainer width="100%" height="100%">
                  <BarChart data={monthData}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="name" /><YAxis /><Tooltip formatter={(v) => formatMoney(v)} /><Legend />
                    <Bar dataKey="Income" fill="#16a34a" /><Bar dataKey="Expenses" fill="#dc2626" /></BarChart>
                </ResponsiveContainer></div>
                <div className="table-wrap">
                  <table>
                    <thead><tr><th>Month</th><th className="num">Income</th><th className="num">Expenses</th><th className="num">Net savings</th><th className="num">Savings rate</th></tr></thead>
                    <tbody>{m.months.map((x) => (
                      <tr key={x.label}><td>{monthLabel(x.label)}</td><td className="num">{formatMoney(x.income)}</td><td className="num">{formatMoney(x.expenses)}</td>
                        <td className={`num ${Number(x.net_savings) < 0 ? 'neg' : ''}`}>{formatMoney(x.net_savings)}</td><td className="num">{formatPercent(x.savings_rate)}</td></tr>
                    ))}</tbody>
                  </table>
                </div>
              </div>

              <div className="grid cards-2 gap-top">
                <div className="card">
                  <h2>Category-wise expenditure</h2>
                  {expCats.length === 0 ? <p className="muted">No expenses in this period.</p> : (
                    <>
                      <div className="chart small-chart"><ResponsiveContainer width="100%" height="100%">
                        <PieChart><Pie data={expCats.map((c) => ({ name: c.category, value: Number(c.total) }))} dataKey="value" nameKey="name" outerRadius="75%">
                          {expCats.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}</Pie><Tooltip formatter={(v) => formatMoney(v)} /></PieChart>
                      </ResponsiveContainer></div>
                      <div className="table-wrap"><table>
                        <thead><tr><th>Category</th><th className="num">Total</th><th className="num">Share</th><th className="num">Count</th></tr></thead>
                        <tbody>{expCats.map((c) => <tr key={c.category}><td>{c.category}</td><td className="num">{formatMoney(c.total)}</td><td className="num">{formatPercent(c.percentage)}</td><td className="num">{c.count}</td></tr>)}</tbody>
                      </table></div>
                    </>
                  )}
                </div>
                <div className="card">
                  <h2>Income by category</h2>
                  {incCats.length === 0 ? <p className="muted">No income in this period.</p> : (
                    <div className="table-wrap"><table>
                      <thead><tr><th>Category</th><th className="num">Total</th><th className="num">Share</th><th className="num">Count</th></tr></thead>
                      <tbody>{incCats.map((c) => <tr key={c.category}><td>{c.category}</td><td className="num">{formatMoney(c.total)}</td><td className="num">{formatPercent(c.percentage)}</td><td className="num">{c.count}</td></tr>)}</tbody>
                    </table></div>
                  )}
                </div>
              </div>

              <div className="card gap-top">
                <h2>Daily spending summary</h2>
                {days.length === 0 ? <p className="muted">No daily activity in this period.</p> : (
                  <>
                    <div className="chart small-chart"><ResponsiveContainer width="100%" height="100%">
                      <LineChart data={dayData}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="name" /><YAxis /><Tooltip formatter={(v) => formatMoney(v)} /><Line type="monotone" dataKey="Expenses" stroke="#dc2626" strokeWidth={2} dot={false} /></LineChart>
                    </ResponsiveContainer></div>
                    <div className="table-wrap scroll-y"><table>
                      <thead><tr><th>Date</th><th className="num">Income</th><th className="num">Expenses</th><th className="num">Net</th></tr></thead>
                      <tbody>{days.map((d) => <tr key={d.date}><td>{formatDate(d.date)}</td><td className="num">{formatMoney(d.income)}</td><td className="num">{formatMoney(d.expenses)}</td><td className={`num ${Number(d.net_savings) < 0 ? 'neg' : ''}`}>{formatMoney(d.net_savings)}</td></tr>)}</tbody>
                    </table></div>
                  </>
                )}
              </div>
            </>
          )}

          <div className="card gap-top">
            <h2>Budget performance <span className="muted small">({endMonth ? `${endMonth[1]}/${endMonth[0]}` : ''})</span></h2>
            {budgets.loading ? <Loader /> : budgets.data && budgets.data.budgets.length > 0 ? (
              <>
                <p className="muted">
                  Budgeted {formatMoney(budgets.data.summary.total_budgeted)} · Spent {formatMoney(budgets.data.summary.total_spent)} ·
                  {' '}{formatPercent(budgets.data.summary.utilization)} used · {budgets.data.summary.exceeded} over budget
                </p>
                <BudgetBars budgets={budgets.data.budgets} />
              </>
            ) : <p className="muted">No budgets were set for the last month of this period.</p>}
          </div>
        </div>
      )}
    </>
  );
}
