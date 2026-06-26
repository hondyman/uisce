// React default import removed — using automatic JSX runtime

interface CoreCustomIndicatorProps {
  isCore?: boolean;
}

const CoreCustomIndicator: React.FC<CoreCustomIndicatorProps> = ({ isCore }) => {
  if (!isCore) {
    return null;
  }

  return (
    <span title="Core Semantic Asset" className="core-custom-indicator">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="currentColor"
        stroke="currentColor"
        strokeWidth="1"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
      </svg>
    </span>
  );
};

export default CoreCustomIndicator;