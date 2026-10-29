import { describe, it, expect } from 'vitest';
import { formatMoney, formatDate, validateAmount, monthLabel } from './format';

describe('format helpers', () => {
  it('formats rupees with Indian digit grouping', () => {
    expect(formatMoney(1234.5)).toMatch(/1,234\.50/);
    expect(formatMoney(123456.7)).toMatch(/1,23,456\.70/);
    expect(formatMoney('oops')).toMatch(/0\.00/);
  });
  it('formats dates without timezone shifts', () => {
    expect(formatDate('2026-01-05')).toBe('05 Jan 2026');
    expect(formatDate('2026-12-31T00:00:00Z')).toBe('31 Dec 2026');
    expect(formatDate('')).toBe('');
  });
  it('labels months', () => {
    expect(monthLabel('2026-03')).toBe('Mar 26');
  });
  it('validates amounts like the backend', () => {
    expect(validateAmount('10')).toBe('');
    expect(validateAmount('10.5')).toBe('');
    expect(validateAmount('.5')).toBe('');
    expect(validateAmount('')).not.toBe('');
    expect(validateAmount('-3')).not.toBe('');
    expect(validateAmount('0')).not.toBe('');
    expect(validateAmount('1.234')).not.toBe('');
    expect(validateAmount('abc')).not.toBe('');
  });
});
