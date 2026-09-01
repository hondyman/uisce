// Dashboard Page - Admin overview and quick stats

import React from "react";
import { Link } from "react-router-dom";
import { useTenants, useAPIKeys } from "../hooks/useAdmin";
import {
  Box,
  Typography,
  Grid,
  Card,
  CardContent,
  Link as MuiLink,
  CircularProgress,
  useTheme,
} from "@mui/material";

export const DashboardPage: React.FC = () => {
  const theme = useTheme();
  const { tenants, total: totalTenants, loading: tenantsLoading } = useTenants(1000, 0);
  const { keys, total: totalKeys, loading: keysLoading } = useAPIKeys(1000, 0);

  const activeTenants = tenants.filter(t => !t.is_suspended).length;
  const suspendedTenants = tenants.filter(t => t.is_suspended).length;
  const activeKeys = keys.filter(k => !k.is_revoked).length;

  return (
    <Box>
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" component="h1" sx={{ fontWeight: 700, color: 'grey.900', m: 0 }}>
          Dashboard
        </Typography>
        <Typography variant="body1" sx={{ mt: 1, color: 'grey.500' }}>
          Overview of your SemLayer platform
        </Typography>
      </Box>

      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            title="Total Tenants"
            value={totalTenants}
            change="+2 this month"
            icon="🏢"
            color="blue"
            link="/admin/tenants"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            title="Active Tenants"
            value={activeTenants}
            change={`${suspendedTenants} suspended`}
            icon="✅"
            color="green"
            link="/admin/tenants"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            title="Total API Keys"
            value={totalKeys}
            change={`${activeKeys} active`}
            icon="🔑"
            color="purple"
            link="/admin/api-keys"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            title="System Status"
            value="Operational"
            change="All systems healthy"
            icon="🟢"
            color="emerald"
            link="/admin/usage"
          />
        </Grid>
      </Grid>

      <Box sx={{ mb: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" sx={{ color: 'grey.900', m: 0 }}>
            Recent Activity
          </Typography>
          <MuiLink component={Link} to="/admin/usage" sx={{ color: 'primary.main', textDecoration: 'none', fontWeight: 500, '&:hover': { color: 'primary.dark' } }}>
            View all →
          </MuiLink>
        </Box>

        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 6 }}>
            <ActivityCard
              title="Recent Tenants"
              count={Math.min(3, tenants.length)}
              items={tenants.slice(0, 3).map(t => ({
                name: t.name,
                meta: `Plan: ${t.plan}`,
              }))}
              loading={tenantsLoading}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <ActivityCard
              title="Recent API Keys"
              count={Math.min(3, keys.length)}
              items={keys.slice(0, 3).map(k => ({
                name: k.name,
                meta: k.roles?.join(", ") || "USER",
              }))}
              loading={keysLoading}
            />
          </Grid>
        </Grid>
      </Box>

      <Box sx={{ mb: 4 }}>
        <Typography variant="h6" sx={{ color: 'grey.900', mb: 2 }}>
          Quick Actions
        </Typography>
        <Grid container spacing={3}>
          <Grid size={{ xs: 12, sm: 6, md: 3 }}>
            <ActionCard
              icon="➕"
              title="Create Tenant"
              description="Set up a new tenant account"
              link="/admin/tenants"
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6, md: 3 }}>
            <ActionCard
              icon="🔑"
              title="Generate API Key"
              description="Create a new API key for authentication"
              link="/admin/api-keys"
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6, md: 3 }}>
            <ActionCard
              icon="📊"
              title="View Usage Analytics"
              description="See detailed usage statistics"
              link="/admin/usage"
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6, md: 3 }}>
            <ActionCard
              icon="📋"
              title="Documentation"
              description="Read API and platform docs (external)"
              href="https://docs.semlayer.io"
            />
          </Grid>
        </Grid>
      </Box>

      <Card sx={{ p: 3 }}>
        <Typography variant="h6" sx={{ color: 'grey.900', mb: 2 }}>
          Platform Information
        </Typography>
        <Grid container spacing={4}>
          <Grid size={{ xs: 6, sm: 3 }}>
            <InfoItem label="Version" value="1.0.0-beta" />
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <InfoItem label="Environment" value="Production" />
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <InfoItem label="Region" value="Multi-region" />
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <InfoItem label="API Endpoint" value="api.semlayer.io" />
          </Grid>
        </Grid>
      </Card>
    </Box>
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
  const theme = useTheme();
  const borderColor = {
    blue: theme.palette.primary.main,
    green: '#52c41a',
    purple: '#764ba2',
    emerald: '#1abc9c',
  }[color];

  const card = (
    <Card
      sx={{
        display: 'flex',
        gap: 2,
        p: 2,
        boxShadow: 2,
        borderRadius: 2,
        borderLeft: `4px solid ${borderColor}`,
        transition: 'all 0.2s ease',
        height: '100%',
        '&:hover': {
          transform: 'translateY(-4px)',
          boxShadow: 4,
        },
      }}
    >
      <Box sx={{ fontSize: '2rem', flexShrink: 0 }}>{icon}</Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
        <Typography variant="caption" sx={{ color: 'grey.500', textTransform: 'uppercase', letterSpacing: '0.5px', fontWeight: 600 }}>
          {title}
        </Typography>
        <Typography variant="h5" sx={{ fontWeight: 700, color: 'grey.900', lineHeight: 1 }}>
          {value}
        </Typography>
        <Typography variant="caption" sx={{ color: 'grey.600' }}>
          {change}
        </Typography>
      </Box>
    </Card>
  );

  return link ? (
    <MuiLink href={link} underline="none" color="inherit" sx={{ display: 'block', textDecoration: 'none' }}>
      {card}
    </MuiLink>
  ) : card;
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
}) => {
  const theme = useTheme();

  return (
    <Card sx={{ p: 2, boxShadow: 2, borderRadius: 2, height: '100%' }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 600, color: 'grey.900', mb: 2 }}>
        {title} ({count})
      </Typography>
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress size={24} />
        </Box>
      ) : items.length === 0 ? (
        <Typography sx={{ textAlign: 'center', color: 'grey.500', py: 4 }}>
          No items yet
        </Typography>
      ) : (
        <Box component="ul" sx={{ listStyle: 'none', m: 0, p: 0, display: 'flex', flexDirection: 'column', gap: 1 }}>
          {items.map((item, idx) => (
            <Box
              component="li"
              key={idx}
              sx={{
                p: 1,
                bgcolor: 'grey.50',
                borderRadius: 1,
                borderLeft: `2px solid ${theme.palette.primary.main}`,
              }}
            >
              <Typography variant="body2" sx={{ fontWeight: 500, color: 'grey.900' }}>
                {item.name}
              </Typography>
              <Typography variant="caption" sx={{ color: 'grey.500' }}>
                {item.meta}
              </Typography>
            </Box>
          ))}
        </Box>
      )}
    </Card>
  );
};

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
  const theme = useTheme();

  const card = (
    <Card
      sx={{
        p: 2,
        display: 'flex',
        alignItems: 'flex-start',
        gap: 2,
        boxShadow: 2,
        borderRadius: 2,
        border: '1px solid',
        borderColor: 'grey.200',
        transition: 'all 0.2s ease',
        height: '100%',
        '&:hover': {
          borderColor: 'primary.main',
          bgcolor: 'action.hover',
          boxShadow: 3,
        },
      }}
    >
      <Box sx={{ fontSize: '1.8rem', flexShrink: 0 }}>{icon}</Box>
      <Box sx={{ flex: 1 }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 600, color: 'grey.900', mb: 0.5 }}>
          {title}
        </Typography>
        <Typography variant="body2" sx={{ color: 'grey.500', m: 0 }}>
          {description}
        </Typography>
      </Box>
      <Box sx={{ color: 'grey.300', fontSize: '1rem' }}>→</Box>
    </Card>
  );

  if (href) {
    return (
      <MuiLink href={href} target="_blank" rel="noopener noreferrer" underline="none" color="inherit" sx={{ display: 'block' }}>
        {card}
      </MuiLink>
    );
  }

  if (link) {
    return (
      <MuiLink href={link} underline="none" color="inherit" sx={{ display: 'block' }}>
        {card}
      </MuiLink>
    );
  }

  return card;
};

interface InfoItemProps {
  label: string;
  value: string;
}

const InfoItem: React.FC<InfoItemProps> = ({ label, value }) => (
  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
    <Typography variant="caption" sx={{ fontWeight: 600, color: 'grey.500', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
      {label}
    </Typography>
    <Typography variant="body1" sx={{ fontWeight: 500, color: 'grey.900' }}>
      {value}
    </Typography>
  </Box>
);

export default DashboardPage;
