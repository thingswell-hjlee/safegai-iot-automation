/**
 * SafeGAI Data Stack
 *
 * DynamoDB tables:
 *   - Gateways (PK=tenantId, SK=siteId#gatewayId, GSI1=gatewayId)
 *   - Events (PK=tenantId#siteId, SK=detectedAt#eventId, GSI1=gatewayId+detectedAt#eventId)
 *
 * S3 buckets:
 *   - Evidence: event thumbnails (raw JPEG binary, max 96KB per image)
 *   - Frontend: static web assets
 *
 * Event images are stored as binary objects, NOT base64 in JSON.
 */

import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { EnvironmentConfig } from '../config/dev';

export interface DataStackProps extends cdk.StackProps {
  readonly config: EnvironmentConfig;
}

export class DataStack extends cdk.Stack {
  public readonly gatewaysTable: dynamodb.Table;
  public readonly eventsTable: dynamodb.Table;
  public readonly evidenceBucket: s3.Bucket;
  public readonly frontendBucket: s3.Bucket;

  constructor(scope: Construct, id: string, props: DataStackProps) {
    super(scope, id, props);

    const { config } = props;

    // --- DynamoDB: Gateways Table ---
    this.gatewaysTable = new dynamodb.Table(this, 'GatewaysTable', {
      tableName: `safegai-${config.envName}-gateways`,
      partitionKey: { name: 'pk', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'sk', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      pointInTimeRecovery: true,
      removalPolicy: config.envName === 'dev'
        ? cdk.RemovalPolicy.DESTROY
        : cdk.RemovalPolicy.RETAIN,
    });

    // GSI1: gatewayId for certificate/topic-bound ingest lookup
    this.gatewaysTable.addGlobalSecondaryIndex({
      indexName: 'GSI1',
      partitionKey: { name: 'gsi1pk', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // --- DynamoDB: Events Table ---
    this.eventsTable = new dynamodb.Table(this, 'EventsTable', {
      tableName: `safegai-${config.envName}-events`,
      partitionKey: { name: 'pk', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'sk', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      pointInTimeRecovery: true,
      timeToLiveAttribute: 'ttl',
      removalPolicy: config.envName === 'dev'
        ? cdk.RemovalPolicy.DESTROY
        : cdk.RemovalPolicy.RETAIN,
    });

    // GSI1: gatewayId + detectedAt#eventId for gateway-specific queries
    this.eventsTable.addGlobalSecondaryIndex({
      indexName: 'GSI1',
      partitionKey: { name: 'gsi1pk', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'gsi1sk', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // --- S3: Evidence Bucket (thumbnails) ---
    this.evidenceBucket = new s3.Bucket(this, 'EvidenceBucket', {
      bucketName: `safegai-${config.envName}-evidence`,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      versioned: false,
      lifecycleRules: [
        {
          id: 'expire-thumbnails',
          expiration: cdk.Duration.days(config.thumbnailLifecycleDays),
        },
      ],
      removalPolicy: config.envName === 'dev'
        ? cdk.RemovalPolicy.DESTROY
        : cdk.RemovalPolicy.RETAIN,
      autoDeleteObjects: config.envName === 'dev',
    });

    // --- S3: Frontend Bucket ---
    this.frontendBucket = new s3.Bucket(this, 'FrontendBucket', {
      bucketName: `safegai-${config.envName}-frontend`,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      websiteIndexDocument: 'index.html',
      websiteErrorDocument: 'index.html',
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
    });

    // Outputs
    new cdk.CfnOutput(this, 'GatewaysTableName', {
      value: this.gatewaysTable.tableName,
      exportName: `${config.envName}-GatewaysTableName`,
    });

    new cdk.CfnOutput(this, 'EventsTableName', {
      value: this.eventsTable.tableName,
      exportName: `${config.envName}-EventsTableName`,
    });

    new cdk.CfnOutput(this, 'EvidenceBucketName', {
      value: this.evidenceBucket.bucketName,
      exportName: `${config.envName}-EvidenceBucketName`,
    });

    // Tags
    cdk.Tags.of(this).add('Project', 'SafeGAI');
    cdk.Tags.of(this).add('Environment', config.envName);
    cdk.Tags.of(this).add('Module', 'data');
  }
}
