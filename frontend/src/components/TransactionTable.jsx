import { Link } from 'react-router-dom';
import { formatDate, formatMoney } from '../utils/format';

// Shows transactions. Pass onDelete to show Edit / Delete actions.
export default function TransactionTable({ rows, onDelete, compact }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Description</th>
            <th>Category</th>
            {!compact && <th>Method</th>}
            <th className="num">Amount</th>
            {onDelete && <th className="no-print">Actions</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((t) => (
            <tr key={t.id}>
              <td>{formatDate(t.transaction_date)}</td>
              <td>{t.description || <span className="muted">—</span>}</td>
              <td>
                <span className={`pill pill-${t.type}`}>{t.category}</span>
              </td>
              {!compact && <td>{t.payment_method || <span className="muted">—</span>}</td>}
              <td className={`num ${t.type === 'income' ? 'pos' : 'neg'}`}>
                {t.type === 'income' ? '+' : '−'}
                {formatMoney(t.amount)}
              </td>
              {onDelete && (
                <td className="no-print actions">
                  <Link className="btn btn-small" to={`/transactions/${t.id}/edit`}>
                    Edit
                  </Link>
                  <button className="btn btn-small btn-danger-outline" onClick={() => onDelete(t)}>
                    Delete
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
