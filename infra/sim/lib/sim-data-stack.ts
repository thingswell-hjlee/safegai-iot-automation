import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimDataStackProps extends cdk.StackProps {
  readonly config: SimConfig;
}

/**
 * Data stack: DynamoDB tables and S3 buckets for event storage and artifacts.
 */
export class SimDataStack extends cdk.Stack {
  public readonly eventsTable: dynamodb.Table;
  public readonly dataBucket: s3.Bucket;

  constructor(scope: Construct, id: string, props: SimDataStackProps) {
    super(scope, id, props);

    // DynamoDB table for safety events
    this.eventsTable = new dynamodb.Table(this, 'EventsTable', {
      tableName: `safegai-events-${props.config.environment}`,
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      timeToLiveAttribute: 'TTL',
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      pointInTimeRecovery: true,
    });

    // GSI for querying by gateway
    this.eventsTable.addGlobalSecondaryIndex({
      indexName: 'GSI-Gateway',
      partitionKey: { name: 'GatewayId', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'Timestamp', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // GSI for querying by event type
    this.eventsTable.addGlobalSecondaryIndex({
      indexName: 'GSI-EventType',
      partitionKey: { name: 'EventType', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'Timestamp', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // S3 bucket for event data, snapshots, and artifacts
    this.dataBucket = new s3.Bucket(this, 'DataBucket', {
      bucketName: `safegai-data-${props.config.environment}-${this.account}`,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
      encryption: s3.BucketEncryption.S3_MANAGED,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      lifecycleRules: [
        {
          id: 'expire-old-events',
          prefix: 'events/',
          expiration: cdk.Duration.days(props.config.retentionDays),
        },
        {
          id: 'expire-old-snapshots',
          prefix: 'snapshots/',
          expiration: cdk.Duration.days(3),
        },
      ],
      versioned: false,
    });

    // Outputs
    new cdk.CfnOutput(this, 'EventsTableName', { value: this.eventsTable.tableName });
    new cdk.CfnOutput(this, 'DataBucketName', { value: this.dataBucket.bucketName });
  }
}
