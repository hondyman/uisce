const useClipboard = () => {
  const copyToClipboard = (content: string, type: string, notify: (m: string, t?: 'success'|'error') => void) => {
    try { navigator.clipboard.writeText(content); notify(`${type} copied to clipboard`, 'success'); } catch (_) { /* ignore */ }
  };

  return { copyToClipboard } as const;
};

export default useClipboard;
