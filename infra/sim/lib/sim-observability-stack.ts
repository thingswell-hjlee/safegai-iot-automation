import * as cdk from 'aws-cdk-lib';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimObservabilityStackProps extends cdk.StackProps {
  readonly config: SimConfig;
  readonly gatewayInstance: ec2.Instance;
}

/**
 * Observability stack: CloudWatch dashboards, alarms, and EventBridge auto-stop rule.
 * Auto-stop prevents runaway costs by stopping the EC2 instance after configurable hours.
 */
export class SimObservabilityStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: SimObservabilityStackProps) {
    super(scope, id, props);

    // CloudWatch Dashboard
    const dashboard = new cloudwatch.Dashboard(this, 'SimDashboard', {
      dashboardName: `SafeGAI-Sim-${props.config.environment}`,
    });

    // CPU utilization widget
    dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: 'Gateway EC2 CPU',
        left: [
          new cloudwatch.Metric({
            namespace: 'AWS/EC2',
            metricName: 'CPUUtilization',
            dimensionsMap: { InstanceId: props.gatewayInstance.instanceId },
            statistic: 'Average',
            period: cdk.Duration.minutes(1),
          }),
        ],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Gateway Network',
        left: [
          new cloudwatch.Metric({
            namespace: 'AWS/EC2',
            metricName: 'NetworkIn',
            dimensionsMap: { InstanceId: props.gatewayInstance.instanceId },
            statistic: 'Sum',
            period: cdk.Duration.minutes(5),
          }),
          new cloudwatch.Metric({
            namespace: 'AWS/EC2',
            metricName: 'NetworkOut',
            dimensionsMap: { InstanceId: props.gatewayInstance.instanceId },
            statistic: 'Sum',
            period: cdk.Duration.minutes(5),
          }),
        ],
        width: 12,
      }),
    );

    // Auto-stop Lambda
    const autoStopLambda = new lambda.Function(this, 'AutoStopFunction', {
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'index.handler',
      code: lambda.Code.fromInline(`
        const { EC2Client, StopInstancesCommand } = require('@aws-sdk/client-ec2');
        exports.handler = async () => {
          const client = new EC2Client({});
          const instanceId = process.env.INSTANCE_ID;
          console.log('Auto-stopping instance:', instanceId);
          await client.send(new StopInstancesCommand({ InstanceIds: [instanceId] }));
          return { statusCode: 200, body: 'Instance stopped' };
        };
      `),
      environment: {
        INSTANCE_ID: props.gatewayInstance.instanceId,
      },
      timeout: cdk.Duration.seconds(30),
    });

    // Grant EC2 stop permission
    autoStopLambda.addToRolePolicy(new iam.PolicyStatement({
      actions: ['ec2:StopInstances'],
      resources: [
        `arn:aws:ec2:${props.config.region}:${this.account}:instance/${props.gatewayInstance.instanceId}`,
      ],
    }));

    // EventBridge rule: Stop instance after N hours (schedule-based)
    new events.Rule(this, 'AutoStopRule', {
      ruleName: `safegai-sim-auto-stop-${props.config.environment}`,
      description: `Auto-stop sim instance after ${props.config.autoStopHours} hours`,
      schedule: events.Schedule.rate(cdk.Duration.hours(props.config.autoStopHours)),
      targets: [new targets.LambdaFunction(autoStopLambda)],
    });

    // CPU alarm (for awareness, not auto-remediation)
    new cloudwatch.Alarm(this, 'HighCpuAlarm', {
      alarmName: `safegai-sim-high-cpu-${props.config.environment}`,
      metric: new cloudwatch.Metric({
        namespace: 'AWS/EC2',
        metricName: 'CPUUtilization',
        dimensionsMap: { InstanceId: props.gatewayInstance.instanceId },
        statistic: 'Average',
        period: cdk.Duration.minutes(5),
      }),
      threshold: 80,
      evaluationPeriods: 3,
      comparisonOperator: cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
    });

    // Outputs
    new cdk.CfnOutput(this, 'DashboardUrl', {
      value: `https://${props.config.region}.console.aws.amazon.com/cloudwatch/home?region=${props.config.region}#dashboards:name=SafeGAI-Sim-${props.config.environment}`,
    });
  }
}
