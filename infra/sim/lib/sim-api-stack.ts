import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimApiStackProps extends cdk.StackProps {
  readonly config: SimConfig;
  readonly eventsTable: dynamodb.ITable;
  readonly dataBucket: s3.IBucket;
}

/**
 * API stack: API Gateway + Lambda + Cognito for cloud dashboard access.
 */
export class SimApiStack extends cdk.Stack {
  public readonly apiEndpoint: string;

  constructor(scope: Construct, id: string, props: SimApiStackProps) {
    super(scope, id, props);

    // Cognito User Pool for authentication
    const userPool = new cognito.UserPool(this, 'SimUserPool', {
      userPoolName: `safegai-sim-users-${props.config.environment}`,
      selfSignUpEnabled: false,
      signInAliases: { email: true },
      autoVerify: { email: true },
      passwordPolicy: {
        minLength: 12,
        requireLowercase: true,
        requireUppercase: true,
        requireDigits: true,
        requireSymbols: true,
      },
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    const userPoolClient = new cognito.UserPoolClient(this, 'SimAppClient', {
      userPool,
      userPoolClientName: `safegai-sim-app-${props.config.environment}`,
      authFlows: {
        userSrp: true,
      },
      generateSecret: false,
    });

    // Lambda function for API handlers
    const apiHandler = new lambda.Function(this, 'ApiHandler', {
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'index.handler',
      code: lambda.Code.fromInline(`
        exports.handler = async (event) => {
          const path = event.path || '/';
          const method = event.httpMethod || 'GET';

          // Route to appropriate handler
          if (path.startsWith('/api/events')) {
            return { statusCode: 200, body: JSON.stringify({ events: [], message: 'Query events via DynamoDB' }) };
          }
          if (path.startsWith('/api/health')) {
            return { statusCode: 200, body: JSON.stringify({ status: 'healthy', service: 'safegai-sim-api' }) };
          }
          return { statusCode: 200, body: JSON.stringify({ message: 'SafeGAI Simulation API', path, method }) };
        };
      `),
      environment: {
        EVENTS_TABLE: props.eventsTable.tableName,
        DATA_BUCKET: props.dataBucket.bucketName,
        ENVIRONMENT: props.config.environment,
      },
      timeout: cdk.Duration.seconds(30),
      memorySize: 256,
    });

    // Grant Lambda access to DynamoDB and S3
    props.eventsTable.grantReadData(apiHandler);
    props.dataBucket.grantRead(apiHandler);

    // API Gateway
    const api = new apigateway.RestApi(this, 'SimApi', {
      restApiName: `safegai-sim-api-${props.config.environment}`,
      description: 'SafeGAI Simulation Cloud API',
      deployOptions: {
        stageName: 'v1',
        throttlingRateLimit: 100,
        throttlingBurstLimit: 200,
      },
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
        allowHeaders: ['Content-Type', 'Authorization'],
      },
    });

    // Cognito authorizer
    const authorizer = new apigateway.CognitoUserPoolsAuthorizer(this, 'ApiAuthorizer', {
      cognitoUserPools: [userPool],
    });

    // API routes
    const apiResource = api.root.addResource('api');

    const eventsResource = apiResource.addResource('events');
    eventsResource.addMethod('GET', new apigateway.LambdaIntegration(apiHandler), {
      authorizer,
      authorizationType: apigateway.AuthorizationType.COGNITO,
    });

    const healthResource = apiResource.addResource('health');
    healthResource.addMethod('GET', new apigateway.LambdaIntegration(apiHandler));

    this.apiEndpoint = api.url;

    // Outputs
    new cdk.CfnOutput(this, 'ApiUrl', { value: api.url });
    new cdk.CfnOutput(this, 'UserPoolId', { value: userPool.userPoolId });
    new cdk.CfnOutput(this, 'UserPoolClientId', { value: userPoolClient.userPoolClientId });
  }
}
