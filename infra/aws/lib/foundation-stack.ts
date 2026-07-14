/**
 * SafeGAI Foundation Stack
 *
 * Base infrastructure: log groups, budget alarms.
 * No VPC in MVP (Lambda uses default VPC-less execution).
 */

import * as cdk from 'aws-cdk-lib';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as budgets from 'aws-cdk-lib/aws-budgets';
import { Construct } from 'constructs';
import { EnvironmentConfig } from '../config/dev';

export interface FoundationStackProps extends cdk.StackProps {
  readonly config: EnvironmentConfig;
}

export class FoundationStack extends cdk.Stack {
  public readonly ingestLogGroup: logs.LogGroup;
  public readonly adminApiLogGroup: logs.LogGroup;

  constructor(scope: Construct, id: string, props: FoundationStackProps) {
    super(scope, id, props);

    const { config } = props;

    // CloudWatch Log Groups with retention
    this.ingestLogGroup = new logs.LogGroup(this, 'IngestHandlerLogs', {
      logGroupName: `/safegai/${config.envName}/ingest-handler`,
      retention: config.retentionDays <= 30
        ? logs.RetentionDays.ONE_MONTH
        : logs.RetentionDays.ONE_YEAR,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    this.adminApiLogGroup = new logs.LogGroup(this, 'AdminApiHandlerLogs', {
      logGroupName: `/safegai/${config.envName}/admin-api-handler`,
      retention: config.retentionDays <= 30
        ? logs.RetentionDays.ONE_MONTH
        : logs.RetentionDays.ONE_YEAR,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Budget alarm
    new budgets.CfnBudget(this, 'MonthlyBudget', {
      budget: {
        budgetName: `safegai-${config.envName}-monthly`,
        budgetType: 'COST',
        timeUnit: 'MONTHLY',
        budgetLimit: {
          amount: config.budgetLimit,
          unit: 'USD',
        },
      },
      notificationsWithSubscribers: [
        {
          notification: {
            comparisonOperator: 'GREATER_THAN',
            notificationType: 'ACTUAL',
            threshold: 80,
            thresholdType: 'PERCENTAGE',
          },
          subscribers: [
            {
              address: config.budgetEmail,
              subscriptionType: 'EMAIL',
            },
          ],
        },
        {
          notification: {
            comparisonOperator: 'GREATER_THAN',
            notificationType: 'FORECASTED',
            threshold: 100,
            thresholdType: 'PERCENTAGE',
          },
          subscribers: [
            {
              address: config.budgetEmail,
              subscriptionType: 'EMAIL',
            },
          ],
        },
      ],
    });

    // Tags
    cdk.Tags.of(this).add('Project', 'SafeGAI');
    cdk.Tags.of(this).add('Environment', config.envName);
    cdk.Tags.of(this).add('Module', 'foundation');
  }
}
