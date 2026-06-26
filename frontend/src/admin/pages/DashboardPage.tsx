// Dashboard Page - Admin overview and quick stats

import React from "react";
import { Link } from "react-router-dom";
import { useTenants, useAPIKeys } from "../hooks/useAdmin";
import "./DashboardPage.css";

export const DashboardPage: React.FC = () => {
  const { tenants, total: totalTenants, loading: tenantsLoading } = useTenants(1000, 0);
  const { keys, total: totalKeys, loading: keysLoading } = useAPIKeys(1000, 0);

  const activeTenants = tenants.filter(t => !t.is_suspended).length;
  const suspendedTenants = tenants.filter(t => t.is_suspended).length;
  const activeKeys = keys.filter(k => !k.is_revoked).length;

  return (
    <div className="dashboard-page">
      <div className="page-header">
        <h1>Dashboard</h1>
        <p className="subtitle">Overview of your SemLayer platform</p>
      </div>

      {/* Quick Stats */}
      <div className="stats-grid">
        <StatCard
          title="Total Tenants"
          value={totalTenants}
          change="+2 this month"
          icon="🏢"
          color="blue"
          link="/admin/tenants"
        />
        <StatCard
          title="Active Tenants"
          value={activeTenants}
          change={`${suspendedTenants} suspended`}
          icon="✅"
          color="green"
          link="/admin/tenants"
        />
        <StatCard
          title="Total API Keys"
          value={totalKeys}
          change={`${activeKeys} active`}
          icon="🔑"
          color="purple"
          link="/admin/api-keys"
        />
        <StatCard
          title="System Status"
          value="Operational"
          change="All systems healthy"
          icon="🟢"
          color="emerald"
          link="/admin/usage"
        />
      </div>

      {/* Recent Activity Section */}
      <div className="activity-section">
        <div className="section-header">
          <h2>Recent Activity</h2>
          <Link to="/admin/usage" className="link-btn">
            View all →
          </Link>
        </div>

        <div className="activity-cards">
          <ActivityCard
            title="Recent Tenants"
            count={Math.min(3, tenants.length)}
            items={tenants.slice(0, 3).map(t => ({
              name: t.name,
              meta: `Plan: ${t.plan}`,
            }))}
            loading={tenantsLoading}
          />
          <ActivityCard
            title="Recent API Keys"
            count={Math.min(3, keys.length)}
            items={keys.slice(0, 3).map(k => ({
              name: k.name,
              meta: k.roles?.join(", ") || "USER",
            }))}
            loading={keysLoading}
          />
        </div>
      </div>

      {/* Quick Actions */}
      <div className="quick-actions">
        <h2>Quick Actions</h2>
        <div className="actions-grid">
          <ActionCard
            icon="➕"
            title="Create Tenant"
            description="Set up a new tenant account"
            link="/admin/tenants"
          />
          <ActionCard
            icon="🔑"
            title="Generate API Key"
            description="Create a new API key for authentication"
            link="/admin/api-keys"
          />
          <ActionCard
            icon="📊"
            title="View Usage Analytics"
            description="See detailed usage statistics"
            link="/admin/usage"
          />
          <ActionCard
            icon="📋"
            title="Documentation"
            description="Read API and platform docs (external)"
            href="https://docs.semlayer.io"
          />
        </div>
      </div>

      {/* Platform Info */}
      <div className="platform-info">
        <h2>Platform Information</h2>
        <div className="info-grid">
          <InfoItem label="Version" value="1.0.0-beta" />
          <InfoItem label="Environment" value="Production" />
          <InfoItem label="Region" value="Multi-region" />
          <InfoItem label="API Endpoint" value="api.semlayer.io" />
        </div>
      </div>
    </div>
  );
};

interface StatCardProps {
  title: string;
  value: number | string;
  change: string;
  icon: string;
  color: "blue" | "green" | "purple" | "emerald";
  link?: string;
}

const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  change,
  icon,
  color,
  link,
}) => {
  const Card = (
    <div className={`stat-card stat-card-${color}`}>
      <div className="stat-icon">{icon}</div>
      <div className="stat-content">
        <div className="stat-title">{title}</div>
        <div className="stat-value">{value}</div>
        <div className="stat-change">{change}</div>
      </div>
    </div>
  );

  return link ? (
    <a href={link} className="stat-card-link">
      {Card}
    </a>
  ) : (
    Card
  );
};

interface ActivityItem {
  name: string;
  meta: string;
}

interface ActivityCardProps {
  title: string;
  count: number;
  items: ActivityItem[];
  loading: boolean;
}

const ActivityCard: React.FC<ActivityCardProps> = ({
  title,
  count,
  items,
  loading,
}) => (
  <div className="activity-card">
    <div className="activity-title">
      {title} ({count})
    </div>
    {loading ? (
      <div className="activity-loading">Loading...</div>
    ) : items.length === 0 ? (
      <div className="activity-empty">No items yet</div>
    ) : (
      <ul className="activity-list">
        {items.map((item, idx) => (
          <li key={idx} className="activity-item">
            <div className="item-name">{item.name}</div>
            <div className="item-meta">{item.meta}</div>
          </li>
        ))}
      </ul>
    )}
  </div>
);

interface ActionCardProps {
  icon: string;
  title: string;
  description: string;
  link?: string;
  href?: string;
}

const ActionCard: React.FC<ActionCardProps> = ({
  icon,
  title,
  description,
  link,
  href,
}) => {
  const Card = (
    <div className="action-card">
      <div className="action-icon">{icon}</div>
      <div className="action-content">
        <div className="action-title">{title}</div>
        <p className="action-description">{description}</p>
      </div>
      <div className="action-arrow">→</div>
    </div>
  );

  if (href) {
    return (
      <a href={href} target="_blank" rel="noopener noreferrer" className="action-link">
        {Card}
      </a>
    );
  }

  if (link) {
    return (
      <a href={link} className="action-link">
        {Card}
      </a>
    );
  }

  return Card;
};

interface InfoItemProps {
  label: string;
  value: string;
}

const InfoItem: React.FC<InfoItemProps> = ({ label, value }) => (
  <div className="info-item">
    <div className="info-label">{label}</div>
    <div className="info-value">{value}</div>
  </div>
);

export default DashboardPage;
