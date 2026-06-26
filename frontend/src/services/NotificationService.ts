import { devLog, devDebug, devWarn } from '../utils/devLogger';
let notifier: { (msg: string, opts?: { variant?: 'default' | 'error' | 'success' | 'warning' | 'info' }): void } | null = null;

export const NotificationService = {
  setNotifier(n: typeof notifier) {
    notifier = n;
  },
  clear() {
    notifier = null;
  },
  info(msg: string) {
    if (notifier) notifier(msg, { variant: 'info' });
    else devLog(msg);
  },
  success(msg: string) {
    if (notifier) notifier(msg, { variant: 'success' });
    else devDebug(msg);
  },
  warn(msg: string) {
    if (notifier) notifier(msg, { variant: 'warning' });
    else devWarn(msg);
  },
  error(msg: string) {
    if (notifier) notifier(msg, { variant: 'error' });
    else console.error(msg);
  },
};

export default NotificationService;