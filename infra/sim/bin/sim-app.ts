#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { SimNetworkStack } from '../lib/sim-network-stack';
import { SimIdentityStack } from '../lib/sim-identity-stack';
import { SimDataStack } from '../lib/sim-data-stack';
import { SimIotStack } from '../lib/sim-iot-stack';
import { SimGatewayStack } from '../lib/sim-gateway-stack';
import { SimApiStack } from '../lib/sim-api-stack';
import { SimFrontendStack } from '../lib/sim-frontend-stack';
import { SimObservabilityStack } from '../lib/sim-observability-stack';
import { simDevConfig } from '../config/sim-dev';

const app = new cdk.App();

const env: cdk.Environment = {
  account: process.env.CDK_DEFAULT_ACCOUNT,
  region: simDevConfig.region,
};

const commonTags = {
  Project: simDevConfig.costTagProject,
  Owner: simDevConfig.costTagOwner,
  Environment: simDevConfig.environment,
  ManagedBy: 'cdk',
};

// Apply tags to all stacks
Object.entries(commonTags).forEach(([key, value]) => {
  cdk.Tags.of(app).add(key, value);
});

const networkStack = new SimNetworkStack(app, 'SafeGAI-Sim-Network', {
  env,
  config: simDevConfig,
});

const identityStack = new SimIdentityStack(app, 'SafeGAI-Sim-Identity', {
  env,
  config: simDevConfig,
});

const dataStack = new SimDataStack(app, 'SafeGAI-Sim-Data', {
  env,
  config: simDevConfig,
});

const iotStack = new SimIotStack(app, 'SafeGAI-Sim-IoT', {
  env,
  config: simDevConfig,
  dataBucket: dataStack.dataBucket,
  eventsTable: dataStack.eventsTable,
});

const gatewayStack = new SimGatewayStack(app, 'SafeGAI-Sim-Gateway', {
  env,
  config: simDevConfig,
  vpc: networkStack.vpc,
  instanceRole: identityStack.instanceRole,
  securityGroup: networkStack.gatewaySecurityGroup,
});

const apiStack = new SimApiStack(app, 'SafeGAI-Sim-API', {
  env,
  config: simDevConfig,
  eventsTable: dataStack.eventsTable,
  dataBucket: dataStack.dataBucket,
});

new SimFrontendStack(app, 'SafeGAI-Sim-Frontend', {
  env,
  config: simDevConfig,
  apiEndpoint: apiStack.apiEndpoint,
});

new SimObservabilityStack(app, 'SafeGAI-Sim-Observability', {
  env,
  config: simDevConfig,
  gatewayInstance: gatewayStack.instance,
});
