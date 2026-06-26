// Shared Card component
import React from "react";
import "./Card.css";

export function Card({
  title,
  subtitle,
  children,
  className = ""
}: {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={`card ${className}`}>
      {(title || subtitle) && (
        <div className="card-header">
          {title && <div className="card-title">{title}</div>}
          {subtitle && <div className="card-subtitle">{subtitle}</div>}
        </div>
      )}
      <div className="card-body">{children}</div>
    </div>
  );
}
