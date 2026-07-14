#!/usr/bin/env node
/**
 * SafeGAI CDK Application Entry Point
 *
 * Region: ap-northeast-2
 * Environments: dev, pilot
 * Deploy via GitHub Actions OIDC - no long-lived access keys.
 *
 * Stack dependencies:
 *   Foundation -> IoT -> Data -> API -> Web
 */

import * as cdk from 'aws-cdk-lib';
import { FoundationStack } from '../lib/foundation-stack';
import { IoTStack } from '../lib/iot-stack';
import { DataStack } from '../lib/data-stack';
import { ApiStack } from '../lib/api-stack';
import { WebStack } from '../lib/web-stack';
import { devConfig } from '../config/dev';
import { pilotConfig } from '../config/pilot';
import { EnvironmentConfig } from '../config/dev';

const app = new cdk.App();

const envName = app.node.tryGetContext('env') || 'dev';
const config: EnvironmentConfig = envName === 'pilot' ? pilotConfig : devConfig;

const env: cdk.Environment = {
  region: config.region,
  account: config.account || process.env['CDK_DEFAULT_ACCOUNT'],
};

const prefix = `SafeGAI-${config.envName}`;

// Foundation: VPC, log groups, budget alarms
const foundation = new FoundationStack(app, `${prefix}-Foundation`, {
  env,
  config,
});

// IoT Core: Thing, Certificate, Policy, Rules
const iot = new IoTStack(app, `${prefix}-IoT`, {
  env,
  config,
});
iot.addDependency(foundation);

// Data: DynamoDB tables, S3 buckets
const data = new DataStack(app, `${prefix}-Data`, {
  env,
  config,
});
data.addDependency(foundation);

// API: Lambda functions, API Gateway, Cognito
const api = new ApiStack(app, `${prefix}-API`, {
  env,
  config,
  eventsTable: data.eventsTable,
  gatewaysTable: data.gatewaysTable,
  evidenceBucket: data.evidenceBucket,
  notificationTopic: iot.notificationTopic,
});
api.addDependency(data);
api.addDependency(iot);

// Web: CloudFront + S3 frontend
const web = new WebStack(app, `${prefix}-Web`, {
  env,
  config,
  frontendBucket: data.frontendBucket,
  apiEndpoint: api.apiEndpoint,
});
web.addDependency(api);

app.synth();
