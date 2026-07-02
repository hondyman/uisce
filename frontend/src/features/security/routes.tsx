import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { SecurityDashboardPage } from './pages/SecurityDashboardPage';
import { RoleManagementPage } from './pages/RoleManagementPage';
import { UserManagementPage } from './pages/UserManagementPage';
import { AuditLogsPage } from './pages/AuditLogsPage';
import { ComplianceReportsPage } from './pages/ComplianceReportsPage';
import { AccessRulesDashboard } from './pages/AccessRulesDashboard';
import { AccessRuleWizardPage } from './pages/AccessRuleWizardPage';
import { AccessRuleDetailPage } from './pages/AccessRuleDetailPage';
import { IDPMappingsPage } from './pages/IDPMappingsPage';

export const SecurityRoutes: React.FC = () => {
  return (
    <Routes>
      <Route path="/" element={<SecurityDashboardPage />} />
      <Route path="/roles" element={<RoleManagementPage />} />
      <Route path="/users" element={<UserManagementPage />} />
      <Route path="/audit" element={<AuditLogsPage />} />
      <Route path="/reports" element={<ComplianceReportsPage />} />
      <Route path="/access-rules" element={<AccessRulesDashboard />} />
      <Route path="/access-rules/wizard" element={<AccessRuleWizardPage />} />
      <Route path="/access-rules/:ruleId" element={<AccessRuleDetailPage />} />
      <Route path="/idp-mappings" element={<IDPMappingsPage />} />
    </Routes>
  );
};

