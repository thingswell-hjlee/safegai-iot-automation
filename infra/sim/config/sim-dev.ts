// Configuration for the SafeGAI simulation development environment.
export interface SimConfig {
  readonly environment: string;
  readonly region: string;
  readonly vpcCidr: string;
  readonly instanceType: string;
  readonly gatewayVersion: string;
  readonly autoStopHours: number;
  readonly costTagProject: string;
  readonly costTagOwner: string;
  readonly domainPrefix: string;
  readonly retentionDays: number;
}

export const simDevConfig: SimConfig = {
  environment: 'sim-dev',
  region: 'ap-northeast-2',
  vpcCidr: '10.100.0.0/16',
  instanceType: 't3.medium',
  gatewayVersion: '0.1.0',
  autoStopHours: 8,
  costTagProject: 'safegai-sim',
  costTagOwner: 'thingswell',
  domainPrefix: 'safegai-sim-dev',
  retentionDays: 7,
};
