// React default import removed — using automatic JSX runtime
import * as TablerIcons from '@tabler/icons-react';

const HeaderBranding: React.FC = () => (
  <div className="header-branding">
    <TablerIcons.IconBrain size={20} className="brand-icon" />
    <div className="brand-text">
  <h1>Model Builder</h1>
      <span className="subtitle">Data Modeling & Business Logic</span>
    </div>
  </div>
);

export default HeaderBranding;
