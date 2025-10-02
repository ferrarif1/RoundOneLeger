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
  return (
    <div
      className={`rounded-full px-4 py-2 text-sm shadow-[var(--shadow-sm)] ${
        isError ? 'bg-red-50 text-red-600' : 'bg-emerald-50 text-emerald-700'
      }`}
    >
      {message}
    </div>
  );
};
