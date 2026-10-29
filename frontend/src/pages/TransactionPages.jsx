import TransactionList from '../components/TransactionList';

export function TransactionsPage() {
  return <TransactionList title="All transactions" fixedType="" endpoint="/transactions" />;
}

export function IncomePage() {
  return <TransactionList title="Income" fixedType="income" endpoint="/income" />;
}

export function ExpensesPage() {
  return <TransactionList title="Expenses" fixedType="expense" endpoint="/expenses" />;
}
