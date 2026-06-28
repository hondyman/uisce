import { Box, Container } from '@mui/material';
import { ConsoleSidebar } from './ConsoleSidebar';
import { ConsoleTopBar } from './ConsoleTopBar';
import ImpersonationBanner from '../components/ImpersonationBanner';

export interface ConsoleLayoutProps {
  children: React.ReactNode;
}

export function ConsoleLayout({ children }: ConsoleLayoutProps) {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <ConsoleSidebar />
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <ConsoleTopBar />
        {/* Impersonation banner renders here so it spans the full content width
            and stays visible regardless of the active page or scroll position. */}
        <ImpersonationBanner />
        <Container
          maxWidth="xl"
          sx={{
            flexGrow: 1,
            py: 3,
            overflow: 'auto',
          }}
        >
          {children}
        </Container>
      </Box>
    </Box>
  );
}
