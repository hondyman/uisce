import React from 'react';
import { DynamicPageResolver } from '../../engine/DynamicPageResolver';

export const DynamicMDMBreakStudioPage: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  return (
    <div>
      {/* 100% Dynamic MDM Break Queue driven by 'mdm_steward_studio' pageKey */}
      <DynamicPageResolver pageKey="mdm_steward_studio" tenantId={tenantId} />
    </div>
  );
};
