import React, { useEffect } from 'react';
import { AbbreviationManager } from '../../components/AbbreviationManagerV2';
import './AbbreviationsPage.css';

export const AbbreviationsPage: React.FC = () => {
  useEffect(() => {
    // Scroll to top on mount
    window.scrollTo(0, 0);
  }, []);

  return (
    <div className="abbreviations-page">
      <div className="abbreviations-header">
        <h1>Abbreviations Manager</h1>
        <p className="subtitle">Manage business abbreviations and their expansions</p>
      </div>
      <div className="abbreviations-content">
        <AbbreviationManager />
      </div>
    </div>
  );
};

export default AbbreviationsPage;
