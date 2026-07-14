/**
 * SafeGAI API Stack
 *
 * API Gateway HTTP API with Cognito JWT authorizer.
 * Lambda functions for ingest and admin-api.
 *
 * NO machine control endpoints.
 * Cognito JWT claims verified at Lambda level (not just API Gateway).
 */

import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import { EnvironmentConfig } from '../config/dev';

export interface ApiStackProps extends cdk.StackProps {
  readonly config: EnvironmentConfig;
  readonly eventsTable: dynamodb.Table;
  readonly gatewaysTable: dynamodb.Table;
  readonly evidenceBucket: s3.Bucket;
  readonly notificationTopic: sns.Topic;
}

export class ApiStack extends cdk.Stack {
  public readonly apiEndpoint: string;
  public readonly userPool: cognito.UserPool;

  constructor(scope: Construct, id: string, props: ApiStackProps) {
    super(scope, id, props);

    const { config, eventsTable, gatewaysTable, evidenceBucket, notificationTopic } = props;

    // --- Cognito User Pool ---
    this.userPool = new cognito.UserPool(this, 'UserPool', {
      userPoolName: `safegai-${config.envName}-users`,
      selfSignUpEnabled: false, // Admin-created accounts only
      signInAliases: { email: true },
      standardAttributes: {
        email: { required: true, mutable: false },
      },
      customAttributes: {
        tenantId: new cognito.StringAttribute({ minLen: 1, maxLen: 64, mutable: false }),
        siteIds: new cognito.StringAttribute({ minLen: 1, maxLen: 512, mutable: true }),
      },
      passwordPolicy: {
        minLength: 12,
        requireLowercase: true,
        requireUppercase: true,
        requireDigits: true,
        requireSymbols: true,
      },
      accountRecovery: cognito.AccountRecovery.EMAIL_ONLY,
      removalPolicy: config.envName === 'dev'
        ? cdk.RemovalPolicy.DESTROY
        : cdk.RemovalPolicy.RETAIN,
    });

    // Cognito groups: operator, maintainer
    new cognito.CfnUserPoolGroup(this, 'OperatorGroup', {
      userPoolId: this.userPool.userPoolId,
      groupName: 'operator',
      description: 'Tenant/Site scoped read + acknowledge/resolve/classify',
    });

    new cognito.CfnUserPoolGroup(this, 'MaintainerGroup', {
      userPoolId: this.userPool.userPoolId,
      groupName: 'maintainer',
      description: 'Operator permissions + diagnostics/hardware view. No safety I/O changes.',
    });

    // User Pool Client
    const userPoolClient = this.userPool.addClient('WebClient', {
      userPoolClientName: `safegai-${config.envName}-web-client`,
      authFlows: {
        userSrp: true,
      },
      generateSecret: false,
      accessTokenValidity: cdk.Duration.hours(1),
      idTokenValidity: cdk.Duration.hours(1),
      refreshTokenValidity: cdk.Duration.days(30),
    });

    // --- Lambda: Ingest Handler ---
    const ingestHandler = new lambda.Function(this, 'IngestHandler', {
      functionName: `safegai-${config.envName}-ingest-handler`,
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'handlers/ingest.handler',
      code: lambda.Code.fromAsset('../services/cloud-backend/dist'),
      memorySize: config.lambdaMemoryMB,
      timeout: cdk.Duration.seconds(config.lambdaTimeoutSec),
      environment: {
        EVENTS_TABLE: eventsTable.tableName,
        GATEWAYS_TABLE: gatewaysTable.tableName,
        NOTIFICATION_TOPIC_ARN: notificationTopic.topicArn,
        NODE_OPTIONS: '--enable-source-maps',
      },
      tracing: lambda.Tracing.ACTIVE,
    });

    // Permissions for ingest handler
    eventsTable.grantReadWriteData(ingestHandler);
    gatewaysTable.grantReadWriteData(ingestHandler);
    notificationTopic.grantPublish(ingestHandler);

    // Allow IoT Rule to invoke ingest handler
    ingestHandler.addPermission('IoTRuleInvoke', {
      principal: new iam.ServicePrincipal('iot.amazonaws.com'),
      action: 'lambda:InvokeFunction',
    });

    // --- Lambda: Admin API Handler ---
    const adminApiHandler = new lambda.Function(this, 'AdminApiHandler', {
      functionName: `safegai-${config.envName}-admin-api-handler`,
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'handlers/admin-api.handler',
      code: lambda.Code.fromAsset('../services/cloud-backend/dist'),
      memorySize: config.lambdaMemoryMB,
      timeout: cdk.Duration.seconds(config.lambdaTimeoutSec),
      environment: {
        EVENTS_TABLE: eventsTable.tableName,
        GATEWAYS_TABLE: gatewaysTable.tableName,
        EVIDENCE_BUCKET: evidenceBucket.bucketName,
        NODE_OPTIONS: '--enable-source-maps',
      },
      tracing: lambda.Tracing.ACTIVE,
    });

    // Permissions for admin API handler
    eventsTable.grantReadWriteData(adminApiHandler);
    gatewaysTable.grantReadData(adminApiHandler);
    evidenceBucket.grantRead(adminApiHandler);

    // --- API Gateway HTTP API ---
    const httpApi = new apigatewayv2.CfnApi(this, 'HttpApi', {
      name: `safegai-${config.envName}-api`,
      protocolType: 'HTTP',
      corsConfiguration: {
        allowHeaders: ['Authorization', 'Content-Type'],
        allowMethods: ['GET', 'POST', 'OPTIONS'],
        allowOrigins: config.envName === 'dev' ? ['*'] : [`https://${config.domainPrefix}.safegai.io`],
        maxAge: 3600,
      },
    });

    this.apiEndpoint = `https://${httpApi.ref}.execute-api.${config.region}.amazonaws.com`;

    // Cognito JWT Authorizer
    const authorizer = new apigatewayv2.CfnAuthorizer(this, 'JwtAuthorizer', {
      apiId: httpApi.ref,
      authorizerType: 'JWT',
      name: 'CognitoJWT',
      identitySource: '$request.header.Authorization',
      jwtConfiguration: {
        audience: [userPoolClient.userPoolClientId],
        issuer: `https://cognito-idp.${config.region}.amazonaws.com/${this.userPool.userPoolId}`,
      },
    });

    // Lambda integration for admin API
    const integration = new apigatewayv2.CfnIntegration(this, 'AdminApiIntegration', {
      apiId: httpApi.ref,
      integrationType: 'AWS_PROXY',
      integrationUri: adminApiHandler.functionArn,
      payloadFormatVersion: '2.0',
    });

    // Routes - NO machine control endpoints
    const routes = [
      { method: 'GET', path: '/sites/{siteId}/gateways' },
      { method: 'GET', path: '/sites/{siteId}/gateways/{gatewayId}/status' },
      { method: 'GET', path: '/sites/{siteId}/events' },
      { method: 'GET', path: '/sites/{siteId}/events/{eventId}' },
      { method: 'POST', path: '/sites/{siteId}/events/{eventId}/ack' },
      { method: 'POST', path: '/sites/{siteId}/events/{eventId}/resolve' },
      { method: 'POST', path: '/sites/{siteId}/events/{eventId}/classify' },
      { method: 'GET', path: '/sites/{siteId}/events/{eventId}/image' },
    ];

    for (const route of routes) {
      new apigatewayv2.CfnRoute(this, `Route-${route.method}-${route.path.replace(/[{}\/]/g, '')}`, {
        apiId: httpApi.ref,
        routeKey: `${route.method} ${route.path}`,
        authorizationType: 'JWT',
        authorizerId: authorizer.ref,
        target: `integrations/${integration.ref}`,
      });
    }

    // Default stage
    new apigatewayv2.CfnStage(this, 'DefaultStage', {
      apiId: httpApi.ref,
      stageName: '$default',
      autoDeploy: true,
    });

    // Allow API Gateway to invoke Lambda
    adminApiHandler.addPermission('ApiGatewayInvoke', {
      principal: new iam.ServicePrincipal('apigateway.amazonaws.com'),
      sourceArn: `arn:aws:execute-api:${config.region}:${this.account}:${httpApi.ref}/*/*`,
    });

    // Outputs
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: this.apiEndpoint,
      exportName: `${config.envName}-ApiEndpoint`,
    });

    new cdk.CfnOutput(this, 'UserPoolId', {
      value: this.userPool.userPoolId,
      exportName: `${config.envName}-UserPoolId`,
    });

    new cdk.CfnOutput(this, 'UserPoolClientId', {
      value: userPoolClient.userPoolClientId,
      exportName: `${config.envName}-UserPoolClientId`,
    });

    // Tags
    cdk.Tags.of(this).add('Project', 'SafeGAI');
    cdk.Tags.of(this).add('Environment', config.envName);
    cdk.Tags.of(this).add('Module', 'api');
  }
}
