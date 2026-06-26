import { devLog, devDebug, devWarn } from '../utils/devLogger';

export class TelemetryService {
  track(event, data) {
    // Send to analytics service
    devLog("Telemetry:", event, data)
  }
}