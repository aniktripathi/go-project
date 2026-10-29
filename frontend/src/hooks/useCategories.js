import { useEffect, useState } from 'react';
import { api } from '../services/api';

// The category lists come from the backend so both sides always agree. Cached after the first load.
let cache = null;

export function useCategories() {
  const [cats, setCats] = useState(cache);
  useEffect(() => {
    if (cache) return;
    api.get('/meta/categories').then((res) => {
      cache = res.data;
      setCats(res.data);
    }).catch(() => {});
  }, []);
  return cats || { income_categories: [], expense_categories: [], payment_methods: [] };
}
