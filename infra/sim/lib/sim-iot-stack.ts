import * as cdk from 'aws-cdk-lib';
import * as iot from 'aws-cdk-lib/aws-iot';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimIotStackProps extends cdk.StackProps {
  readonly config: SimConfig;
  readonly dataBucket: s3.IBucket;
  readonly eventsTable: dynamodb.ITable;
}

/**
 * IoT stack: AWS IoT Core thing, policy, and topic rules for event routing.
 */
export class SimIotStack extends cdk.Stack {
  public readonly thingName: string;

  constructor(scope: Construct, id: string, props: SimIotStackProps) {
    super(scope, id, props);

    this.thingName = `safegai-gateway-${props.config.environment}`;

    // IoT Thing
    new iot.CfnThing(this, 'GatewayThing', {
      thingName: this.thingName,
      attributePayload: {
        attributes: {
          environment: props.config.environment,
          version: props.config.gatewayVersion,
          type: 'gateway',
        },
      },
    });

    // IoT Policy for the gateway
    new iot.CfnPolicy(this, 'GatewayIoTPolicy', {
      policyName: `safegai-gateway-policy-${props.config.environment}`,
      policyDocument: {
        Version: '2012-10-17',
        Statement: [
          {
            Effect: 'Allow',
            Action: ['iot:Connect'],
            Resource: [`arn:aws:iot:${props.config.region}:${this.account}:client/safegai-*`],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Publish'],
            Resource: [
              `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/events/*`,
              `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/telemetry/*`,
              `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/health/*`,
            ],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Subscribe'],
            Resource: [
              `arn:aws:iot:${props.config.region}:${this.account}:topicfilter/safegai/commands/*`,
              `arn:aws:iot:${props.config.region}:${this.account}:topicfilter/safegai/config/*`,
            ],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Receive'],
            Resource: [
              `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/commands/*`,
              `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/config/*`,
            ],
          },
        ],
      },
    });

    // IAM role for IoT topic rules
    const topicRuleRole = new iam.Role(this, 'TopicRuleRole', {
      assumedBy: new iam.ServicePrincipal('iot.amazonaws.com'),
      description: 'Role for IoT topic rules to write to DynamoDB and S3',
    });

    props.eventsTable.grantWriteData(topicRuleRole);
    props.dataBucket.grantWrite(topicRuleRole);

    // IoT Topic Rule: Route safety events to DynamoDB
    new iot.CfnTopicRule(this, 'SafetyEventsRule', {
      ruleName: `safegai_safety_events_${props.config.environment.replace('-', '_')}`,
      topicRulePayload: {
        description: 'Route safety events from gateway to DynamoDB',
        sql: "SELECT *, topic(3) as gatewayId, timestamp() as ingestTimestamp FROM 'safegai/events/+'",
        actions: [
          {
            dynamoDBv2: {
              putItem: {
                tableName: props.eventsTable.tableName,
              },
              roleArn: topicRuleRole.roleArn,
            },
          },
        ],
        ruleDisabled: false,
        awsIotSqlVersion: '2016-03-23',
      },
    });

    // IoT Topic Rule: Route telemetry to S3
    new iot.CfnTopicRule(this, 'TelemetryRule', {
      ruleName: `safegai_telemetry_${props.config.environment.replace('-', '_')}`,
      topicRulePayload: {
        description: 'Archive telemetry data to S3',
        sql: "SELECT * FROM 'safegai/telemetry/+'",
        actions: [
          {
            s3: {
              bucketName: props.dataBucket.bucketName,
              key: 'telemetry/${topic(3)}/${parse_time("yyyy/MM/dd/HH", timestamp())}/${newuuid()}.json',
              roleArn: topicRuleRole.roleArn,
            },
          },
        ],
        ruleDisabled: false,
        awsIotSqlVersion: '2016-03-23',
      },
    });

    // Outputs
    new cdk.CfnOutput(this, 'ThingName', { value: this.thingName });
  }
}
