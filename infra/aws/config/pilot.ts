/**
 * SafeGAI Pilot Environment Configuration
 *
 * Region: ap-northeast-2
 * Pilot deployment requires version tag + manual approval.
 * No long-lived AWS access keys.
 */

import { EnvironmentConfig } from './dev';

export const pilotConfig: EnvironmentConfig = {
  envName: 'pilot',
  region: 'ap-northeast-2',
  domainPrefix: 'safegai-pilot',
  budgetLimit: 200, // USD
  budgetEmail: 'pilot-alerts@safegai.io',
  retentionDays: 365,
  lambdaMemoryMB: 512,
  lambdaTimeoutSec: 30,
  eventTtlDays: 365,
  thumbnailLifecycleDays: 365,
  notificationEnabled: true,
  cloudFrontEnabled: true,
};
