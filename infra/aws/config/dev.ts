/**
 * SafeGAI Development Environment Configuration
 *
 * Region: ap-northeast-2
 * No long-lived AWS access keys.
 * Deploy via GitHub Actions OIDC only.
 */

export interface EnvironmentConfig {
  readonly envName: string;
  readonly region: string;
  readonly account?: string;
  readonly domainPrefix: string;
  readonly budgetLimit: number;
  readonly budgetEmail: string;
  readonly retentionDays: number;
  readonly lambdaMemoryMB: number;
  readonly lambdaTimeoutSec: number;
  readonly eventTtlDays: number;
  readonly thumbnailLifecycleDays: number;
  readonly notificationEnabled: boolean;
  readonly cloudFrontEnabled: boolean;
}

export const devConfig: EnvironmentConfig = {
  envName: 'dev',
  region: 'ap-northeast-2',
  domainPrefix: 'safegai-dev',
  budgetLimit: 50, // USD
  budgetEmail: 'dev-alerts@safegai.io',
  retentionDays: 30,
  lambdaMemoryMB: 256,
  lambdaTimeoutSec: 30,
  eventTtlDays: 90,
  thumbnailLifecycleDays: 90,
  notificationEnabled: true,
  cloudFrontEnabled: false, // Dev uses API Gateway directly
};
