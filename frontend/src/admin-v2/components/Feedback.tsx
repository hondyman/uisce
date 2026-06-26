// Spinner component
import React from "react";
import "./Spinner.css";

export function Spinner({ size = "md" }: { size?: "sm" | "md" | "lg" }) {
  return (
    <div className={`spinner spinner-${size}`}>
      <div className="spinner-inner"></div>
    </div>
  );
}

// Error banner component
export function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="error-banner">
      <span className="error-icon">⚠</span>
      <span className="error-message">{message}</span>
    </div>
  );
}

// Success message component
export function SuccessBanner({ message }: { message: string }) {
  return (
    <div className="success-banner">
      <span className="success-icon">✓</span>
      <span className="success-message">{message}</span>
    </div>
  );
}

// Loading skeleton
export function Skeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div className="skeleton">
      {Array.from({ length: lines }).map((_, i) => (
        <div key={i} className="skeleton-line"></div>
      ))}
    </div>
  );
}
