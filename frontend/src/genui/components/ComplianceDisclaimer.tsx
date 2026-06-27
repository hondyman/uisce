import React from 'react';
import { Alert, AlertTitle } from '@mui/material';
import { motion } from 'framer-motion';

interface ComplianceDisclaimerProps {
  topic: string;
  severity: "info" | "warning" | "error"; // note: converted critical to error for standard Alert severity
}

export function ComplianceDisclaimer({ topic, severity }: ComplianceDisclaimerProps) {
  const getSeverity = () => {
    if (severity === 'critical') return 'error';
    return severity;
  };

  const getDisclaimerText = (topic: string) => {
    if (topic.includes("Futures")) return "Futures trading involves substantial risk of loss and is not suitable for every investor.";
    if (topic.includes("Muni")) return "Municipal bonds are subject to interest rate risk, credit risk, and market risk.";
    return `Standard disclaimer for ${topic}: Past performance is not indicative of future results.`;
  };

  const alertSeverity = getSeverity() as 'info' | 'warning' | 'error';

  return (
    <motion.div 
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
    >
      <Alert 
        severity={alertSeverity} 
        sx={{ my: 2, borderRadius: 2 }}
      >
        <AlertTitle sx={{ fontWeight: 'bold' }}>Compliance Notice: {topic}</AlertTitle>
        {getDisclaimerText(topic)}
      </Alert>
    </motion.div>
  );
}

