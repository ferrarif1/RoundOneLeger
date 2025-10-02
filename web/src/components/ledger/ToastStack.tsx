interface ToastStackProps {
  status?: string | null;
  error?: string | null;
}

export const ToastStack = ({ status, error }: ToastStackProps) => {
  if (!status && !error) {
    return null;
  }
  const message = error ?? status ?? '';
  const isError = Boolean(error);
  const baseClass = isError
    ? 'bg-red-50 text-red-600 border border-red-100'
    : 'bg-white/90 text-[var(--text)] border border-black/10';

  return (
    <div className={`rounded-full px-4 py-2 text-sm shadow-[var(--shadow-sm)] ${baseClass}`}>{message}</div>
  );
};
