import * as cdk from 'aws-cdk-lib';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimIdentityStackProps extends cdk.StackProps {
  readonly config: SimConfig;
}

/**
 * Identity stack: IAM roles and instance profile for the gateway EC2 instance.
 * Follows least-privilege principle.
 */
export class SimIdentityStack extends cdk.Stack {
  public readonly instanceRole: iam.Role;
  public readonly instanceProfile: iam.CfnInstanceProfile;

  constructor(scope: Construct, id: string, props: SimIdentityStackProps) {
    super(scope, id, props);

    // IAM role for the gateway EC2 instance
    this.instanceRole = new iam.Role(this, 'GatewayInstanceRole', {
      assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com'),
      description: 'Role for SafeGAI simulation gateway EC2 instance',
      managedPolicies: [
        // SSM for remote management (no SSH needed)
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMManagedInstanceCore'),
      ],
    });

    // IoT Core permissions for MQTT publish/subscribe
    this.instanceRole.addToPolicy(new iam.PolicyStatement({
      sid: 'IoTCoreAccess',
      effect: iam.Effect.ALLOW,
      actions: [
        'iot:Connect',
        'iot:Publish',
        'iot:Subscribe',
        'iot:Receive',
      ],
      resources: [
        `arn:aws:iot:${props.config.region}:${this.account}:topic/safegai/*`,
        `arn:aws:iot:${props.config.region}:${this.account}:topicfilter/safegai/*`,
        `arn:aws:iot:${props.config.region}:${this.account}:client/safegai-*`,
      ],
    }));

    // S3 access for artifact download and event storage
    this.instanceRole.addToPolicy(new iam.PolicyStatement({
      sid: 'S3ArtifactAccess',
      effect: iam.Effect.ALLOW,
      actions: [
        's3:GetObject',
        's3:PutObject',
        's3:ListBucket',
      ],
      resources: [
        `arn:aws:s3:::safegai-*`,
        `arn:aws:s3:::safegai-*/*`,
      ],
    }));

    // CloudWatch Logs for centralized logging
    this.instanceRole.addToPolicy(new iam.PolicyStatement({
      sid: 'CloudWatchLogs',
      effect: iam.Effect.ALLOW,
      actions: [
        'logs:CreateLogGroup',
        'logs:CreateLogStream',
        'logs:PutLogEvents',
        'logs:DescribeLogStreams',
      ],
      resources: [
        `arn:aws:logs:${props.config.region}:${this.account}:log-group:/safegai/*`,
      ],
    }));

    // CloudWatch metrics
    this.instanceRole.addToPolicy(new iam.PolicyStatement({
      sid: 'CloudWatchMetrics',
      effect: iam.Effect.ALLOW,
      actions: [
        'cloudwatch:PutMetricData',
      ],
      resources: ['*'],
      conditions: {
        StringEquals: {
          'cloudwatch:namespace': 'SafeGAI/Sim',
        },
      },
    }));

    // Instance profile
    this.instanceProfile = new iam.CfnInstanceProfile(this, 'GatewayInstanceProfile', {
      roles: [this.instanceRole.roleName],
      instanceProfileName: `safegai-sim-gateway-${props.config.environment}`,
    });

    // Outputs
    new cdk.CfnOutput(this, 'InstanceRoleArn', { value: this.instanceRole.roleArn });
  }
}
