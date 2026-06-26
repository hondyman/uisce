export const getLineageNodeStyle = (_nodeType: string, isCenter: boolean, direction?: string) => {
  if (isCenter) {
    return {
      background: 'var(--node-bg-center)',
      borderColor: 'var(--node-border-center)',
      color: 'var(--node-text-center)',
      boxShadow: '0 0 15px rgba(76, 81, 191, 0.3)',
    };
  }

  let borderColor = 'var(--node-border-default)';
  if (direction === 'upstream') {
    borderColor = '#10b981'; // Green for upstream
  } else if (direction === 'downstream') {
    borderColor = '#3b82f6'; // Blue for downstream
  }

  return {
    background: 'var(--background-primary)',
    borderColor: borderColor,
    color: 'var(--text-primary)',
  };
};

