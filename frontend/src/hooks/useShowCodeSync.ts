import { useEffect } from 'react';

const useShowCodeSync = (showCode: any, setShowCode: (s: any) => void) => {
  const formatType = showCode === 'json' || showCode === 'yaml' ? showCode : 'yaml';
  useEffect(() => {
    // keep last good format in sync externally
  }, [formatType]);

  return { formatType, setFormatType: (fmt: 'json' | 'yaml') => setShowCode(fmt) } as const;
};

export default useShowCodeSync;
