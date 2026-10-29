import { useEffect, useRef } from 'react';
import { formatMoney } from '../utils/format';

export function Loader({ text = 'Loading…' }) {
  return (
    <div className="loader" role="status">
      <span className="spinner" aria-hidden="true" /> {text}
    </div>
  );
}

export function ErrorBanner({ message, onRetry }) {
  if (!message) return null;
  return (
    <div className="banner banner-error" role="alert">
      <span>{message}</span>
      {onRetry && (
        <button className="btn btn-small" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  );
}

export function EmptyState({ title, text, action }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      {text && <p>{text}</p>}
      {action}
    </div>
  );
}

export function SummaryCard({ label, value, hint, tone }) {
  return (
    <div className={`card summary ${tone ? `tone-${tone}` : ''}`}>
      <div className="summary-label">{label}</div>
      <div className="summary-value">{value}</div>
      {hint && <div className="summary-hint">{hint}</div>}
    </div>
  );
}

export function Money({ value, signed }) {
  const n = Number(value) || 0;
  return <span className={signed ? (n < 0 ? 'neg' : 'pos') : ''}>{formatMoney(n)}</span>;
}

export function ConfirmDialog({ open, title, message, confirmText = 'Delete', busy, onConfirm, onCancel }) {
  const cancelRef = useRef(null);
  useEffect(() => {
    if (!open) return undefined;
    cancelRef.current?.focus();
    const onKey = (e) => e.key === 'Escape' && onCancel();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onCancel]);

  if (!open) return null;
  return (
    <div className="modal-backdrop" onClick={onCancel}>
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title" onClick={(e) => e.stopPropagation()}>
        <h3 id="confirm-title">{title}</h3>
        <p>{message}</p>
        <div className="modal-actions">
          <button ref={cancelRef} className="btn" onClick={onCancel} disabled={busy}>
            Cancel
          </button>
          <button className="btn btn-danger" onClick={onConfirm} disabled={busy}>
            {busy ? 'Working…' : confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}

export function Field({ label, htmlFor, error, children, hint }) {
  return (
    <div className={`field ${error ? 'has-error' : ''}`}>
      <label htmlFor={htmlFor}>{label}</label>
      {children}
      {hint && !error && <small className="hint">{hint}</small>}
      {error && (
        <small className="error" role="alert">
          {error}
        </small>
      )}
    </div>
  );
}

export function Pagination({ meta, onPage }) {
  if (!meta || meta.total_pages <= 1) return null;
  return (
    <div className="pagination">
      <button className="btn btn-small" disabled={meta.page <= 1} onClick={() => onPage(meta.page - 1)}>
        ‹ Previous
      </button>
      <span>
        Page {meta.page} of {meta.total_pages} · {meta.total} records
      </span>
      <button className="btn btn-small" disabled={meta.page >= meta.total_pages} onClick={() => onPage(meta.page + 1)}>
        Next ›
      </button>
    </div>
  );
}
