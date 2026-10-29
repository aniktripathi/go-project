import { useState } from 'react';

export const PRESETS = [
  { value: 'this_month', label: 'This month' },
  { value: 'last_month', label: 'Last month' },
  { value: 'last_3_months', label: 'Last 3 months' },
  { value: 'custom', label: 'Custom range' },
];

// Turns the filter value into query parameters for the API.
export function rangeParams(v) {
  return v.range === 'custom' ? { range: 'custom', from: v.from, to: v.to } : { range: v.range };
}

// value: { range, from, to }. Custom dates are applied with the "Apply" button.
export default function DateRangeFilter({ value, onChange }) {
  const [draft, setDraft] = useState({ from: value.from || '', to: value.to || '' });
  const [error, setError] = useState('');

  function choose(range) {
    setError('');
    if (range !== 'custom') onChange({ range, from: '', to: '' });
    else onChange({ ...value, range: 'custom' });
  }

  function apply() {
    if (!draft.from || !draft.to) return setError('Choose both dates.');
    if (draft.from > draft.to) return setError('The start date must not be after the end date.');
    setError('');
    onChange({ range: 'custom', from: draft.from, to: draft.to });
  }

  return (
    <div className="filter-bar no-print">
      <div className="segmented" role="group" aria-label="Date range">
        {PRESETS.map((p) => (
          <button key={p.value} type="button" className={value.range === p.value ? 'active' : ''} onClick={() => choose(p.value)}>
            {p.label}
          </button>
        ))}
      </div>
      {value.range === 'custom' && (
        <div className="custom-range">
          <label>
            From <input type="date" value={draft.from} onChange={(e) => setDraft({ ...draft, from: e.target.value })} />
          </label>
          <label>
            To <input type="date" value={draft.to} onChange={(e) => setDraft({ ...draft, to: e.target.value })} />
          </label>
          <button className="btn btn-primary btn-small" type="button" onClick={apply}>
            Apply
          </button>
          {error && <span className="error" role="alert">{error}</span>}
        </div>
      )}
    </div>
  );
}
