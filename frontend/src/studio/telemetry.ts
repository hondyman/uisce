import { devLog } from '../utils/devLogger';

export class TelemetryService {
  track(event: any, data: any) {
    devLog("Telemetry:", event, data)
  }
}