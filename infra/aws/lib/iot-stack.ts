/**
 * SafeGAI IoT Stack
 *
 * AWS IoT Core: Thing Type, Certificate Policy, Topic Rules.
 *
 * Topic structure:
 *   safegai/v1/{tenant}/{site}/{gateway}/status
 *   safegai/v1/{tenant}/{site}/{gateway}/events
 *   safegai/v1/{tenant}/{site}/{gateway}/images/{eventId}
 *   safegai/v1/{tenant}/{site}/{gateway}/acks
 *
 * Rules:
 *   - Event metadata -> ingest-handler Lambda
 *   - Image binary -> S3 direct put
 *
 * Security:
 *   - X.509 certificate per gateway
 *   - Topic policy scoped to tenant/site/gateway
 *   - NO cloud-to-device safety command topic
 *   - NO machine control topics
 *   - NO actuator command topics
 */

import * as cdk from 'aws-cdk-lib';
import * as iot from 'aws-cdk-lib/aws-iot';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as sns from 'aws-cdk-lib/aws-sns';
import { Construct } from 'constructs';
import { EnvironmentConfig } from '../config/dev';

export interface IoTStackProps extends cdk.StackProps {
  readonly config: EnvironmentConfig;
}

export class IoTStack extends cdk.Stack {
  public readonly notificationTopic: sns.Topic;

  constructor(scope: Construct, id: string, props: IoTStackProps) {
    super(scope, id, props);

    const { config } = props;

    // SNS Topic for event notifications
    this.notificationTopic = new sns.Topic(this, 'EventNotificationTopic', {
      topicName: `safegai-${config.envName}-event-notifications`,
      displayName: `SafeGAI ${config.envName} Event Notifications`,
    });

    // IoT Thing Type for SafeGAI gateways
    new iot.CfnThingType(this, 'GatewayThingType', {
      thingTypeName: `safegai-gateway-${config.envName}`,
      thingTypeProperties: {
        thingTypeDescription: 'SafeGAI Edge Gateway',
        searchableAttributes: ['tenantId', 'siteId', 'hardwareProfileId'],
      },
    });

    /**
     * IoT Policy Template
     * Each gateway gets a unique policy scoped to its tenant/site/gateway path.
     * NO subscribe/publish to actuator or safety-command topics.
     *
     * Actual per-device policies are created during provisioning.
     * This is the template definition for documentation/reference.
     */
    new iot.CfnPolicy(this, 'GatewayPolicyTemplate', {
      policyName: `safegai-gateway-policy-template-${config.envName}`,
      policyDocument: {
        Version: '2012-10-17',
        Statement: [
          {
            Effect: 'Allow',
            Action: ['iot:Connect'],
            Resource: [`arn:aws:iot:${config.region}:*:client/\${iot:Connection.Thing.ThingName}`],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Publish'],
            Resource: [
              // Publish to own status/events/images topics only
              `arn:aws:iot:${config.region}:*:topic/safegai/v1/\${iot:Connection.Thing.Attributes[tenantId]}/\${iot:Connection.Thing.Attributes[siteId]}/\${iot:Connection.Thing.ThingName}/status`,
              `arn:aws:iot:${config.region}:*:topic/safegai/v1/\${iot:Connection.Thing.Attributes[tenantId]}/\${iot:Connection.Thing.Attributes[siteId]}/\${iot:Connection.Thing.ThingName}/events`,
              `arn:aws:iot:${config.region}:*:topic/safegai/v1/\${iot:Connection.Thing.Attributes[tenantId]}/\${iot:Connection.Thing.Attributes[siteId]}/\${iot:Connection.Thing.ThingName}/images/*`,
            ],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Subscribe'],
            Resource: [
              // Subscribe to own acks topic only
              `arn:aws:iot:${config.region}:*:topicfilter/safegai/v1/\${iot:Connection.Thing.Attributes[tenantId]}/\${iot:Connection.Thing.Attributes[siteId]}/\${iot:Connection.Thing.ThingName}/acks`,
            ],
          },
          {
            Effect: 'Allow',
            Action: ['iot:Receive'],
            Resource: [
              `arn:aws:iot:${config.region}:*:topic/safegai/v1/\${iot:Connection.Thing.Attributes[tenantId]}/\${iot:Connection.Thing.Attributes[siteId]}/\${iot:Connection.Thing.ThingName}/acks`,
            ],
          },
          {
            Effect: 'Allow',
            Action: [
              'iot:UpdateThingShadow',
              'iot:GetThingShadow',
            ],
            Resource: [
              `arn:aws:iot:${config.region}:*:thing/\${iot:Connection.Thing.ThingName}/shadow/name/health`,
              `arn:aws:iot:${config.region}:*:thing/\${iot:Connection.Thing.ThingName}/shadow/name/settings`,
            ],
          },
          // EXPLICIT DENY: No actuator, safety-command, or machine-control topics
          {
            Effect: 'Deny',
            Action: ['iot:Publish', 'iot:Subscribe', 'iot:Receive'],
            Resource: [
              `arn:aws:iot:${config.region}:*:topic/safegai/*/control/*`,
              `arn:aws:iot:${config.region}:*:topic/safegai/*/actuator/*`,
              `arn:aws:iot:${config.region}:*:topic/safegai/*/command/*`,
              `arn:aws:iot:${config.region}:*:topicfilter/safegai/*/control/*`,
              `arn:aws:iot:${config.region}:*:topicfilter/safegai/*/actuator/*`,
              `arn:aws:iot:${config.region}:*:topicfilter/safegai/*/command/*`,
            ],
          },
        ],
      },
    });

    // IoT Rule: Event metadata -> Lambda
    const ingestRuleRole = new iam.Role(this, 'IngestRuleRole', {
      assumedBy: new iam.ServicePrincipal('iot.amazonaws.com'),
      description: 'Role for IoT Rule to invoke ingest-handler Lambda',
    });

    new iot.CfnTopicRule(this, 'EventIngestRule', {
      ruleName: `safegai_${config.envName}_event_ingest`,
      topicRulePayload: {
        description: 'Routes event metadata to ingest-handler Lambda',
        sql: `SELECT *, topic() as topic, principal() as principal, certificates.certificateId as certificateId FROM 'safegai/v1/+/+/+/events'`,
        awsIotSqlVersion: '2016-03-23',
        ruleDisabled: false,
        actions: [
          {
            lambda: {
              // Lambda ARN will be set via cross-stack reference
              functionArn: `arn:aws:lambda:${config.region}:\${AWS::AccountId}:function:safegai-${config.envName}-ingest-handler`,
            },
          },
        ],
        errorAction: {
          cloudwatchLogs: {
            logGroupName: `/safegai/${config.envName}/iot-rule-errors`,
            roleArn: ingestRuleRole.roleArn,
          },
        },
      },
    });

    // IoT Rule: Image binary -> S3 direct
    const imageRuleRole = new iam.Role(this, 'ImageRuleRole', {
      assumedBy: new iam.ServicePrincipal('iot.amazonaws.com'),
      description: 'Role for IoT Rule to store images directly in S3',
    });

    // S3 put permission will be granted by data-stack bucket policy
    imageRuleRole.addToPolicy(new iam.PolicyStatement({
      actions: ['s3:PutObject'],
      resources: [`arn:aws:s3:::safegai-${config.envName}-evidence/*`],
    }));

    new iot.CfnTopicRule(this, 'ImageStoreRule', {
      ruleName: `safegai_${config.envName}_image_store`,
      topicRulePayload: {
        description: 'Stores event thumbnail JPEG directly to S3 as binary (max 96KB)',
        sql: `SELECT * FROM 'safegai/v1/+/+/+/images/+'`,
        awsIotSqlVersion: '2016-03-23',
        ruleDisabled: false,
        actions: [
          {
            s3: {
              bucketName: `safegai-${config.envName}-evidence`,
              // Key format: {tenant}/{site}/{yyyy}/{mm}/{dd}/{eventId}.jpg
              key: '${topic(3)}/${topic(4)}/${parse_time("yyyy", timestamp())}/${parse_time("MM", timestamp())}/${parse_time("dd", timestamp())}/${topic(7)}.jpg',
              roleArn: imageRuleRole.roleArn,
              cannedAcl: 'private',
            },
          },
        ],
        errorAction: {
          cloudwatchLogs: {
            logGroupName: `/safegai/${config.envName}/iot-rule-errors`,
            roleArn: imageRuleRole.roleArn,
          },
        },
      },
    });

    // Error log group for IoT Rules
    ingestRuleRole.addToPolicy(new iam.PolicyStatement({
      actions: ['logs:CreateLogStream', 'logs:PutLogEvents'],
      resources: [`arn:aws:logs:${config.region}:*:log-group:/safegai/${config.envName}/iot-rule-errors:*`],
    }));

    // Tags
    cdk.Tags.of(this).add('Project', 'SafeGAI');
    cdk.Tags.of(this).add('Environment', config.envName);
    cdk.Tags.of(this).add('Module', 'iot');
  }
}
