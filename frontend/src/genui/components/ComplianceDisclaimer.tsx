import React from 'react';
import { AlertTriangle, Info, ShieldAlert } from 'lucide-react';
import { motion } from 'framer-motion';

interface ComplianceDisclaimerProps {
  topic: string;
  severity: "info" | "warning" | "critical";
}

export function ComplianceDisclaimer({ topic, severity }: ComplianceDisclaimerProps) {
  const getStyles = () => {
    switch (severity) {
      case 'critical':
        return { bg: 'bg-red-50', border: 'border-red-200', text: 'text-red-800', icon: ShieldAlert };
      case 'warning':
        return { bg: 'bg-amber-50', border: 'border-amber-200', text: 'text-amber-800', icon: AlertTriangle };
      default:
        return { bg: 'bg-blue-50', border: 'border-blue-200', text: 'text-blue-800', icon: Info };
    }
  };

  const styles = getStyles();
  const Icon = styles.icon;

  const getDisclaimerText = (topic: string) => {
    // In a real app, this would come from a CMS or metadata service
    if (topic.includes("Futures")) return "Futures trading involves substantial risk of loss and is not suitable for every investor.";
    if (topic.includes("Muni")) return "Municipal bonds are subject to interest rate risk, credit risk, and market risk.";
    return `Standard disclaimer for ${topic}: Past performance is not indicative of future results.`;
  };

  return (
    <motion.div 
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      className={`flex items-start gap-3 p-4 rounded-lg border ${styles.bg} ${styles.border} my-2`}
    >
      <Icon className={`w-5 h-5 mt-0.5 ${styles.text}`} />
      <div>
        <h4 className={`text-sm font-semibold ${styles.text} mb-1`}>Compliance Notice: {topic}</h4>
        <p className={`text-sm ${styles.text} opacity-90`}>
          {getDisclaimerText(topic)}
        </p>
      </div>
    </motion.div>
  );
}
