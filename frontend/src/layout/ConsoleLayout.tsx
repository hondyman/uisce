import { Box, Container } from '@mui/material';
import { ConsoleSidebar } from './ConsoleSidebar';
import { ConsoleTopBar } from './ConsoleTopBar';

export interface ConsoleLayoutProps {
  children: React.ReactNode;
}

export function ConsoleLayout({ children }: ConsoleLayoutProps) {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <ConsoleSidebar />
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <ConsoleTopBar />
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
