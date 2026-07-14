/**
 * SafeGAI Web Stack
 *
 * CloudFront distribution for the cloud frontend.
 * S3 origin for static React build assets.
 * API Gateway origin for backend requests.
 *
 * No live video relay.
 * No direct camera credentials.
 * Event thumbnail and metadata only.
 */

import * as cdk from 'aws-cdk-lib';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { EnvironmentConfig } from '../config/dev';

export interface WebStackProps extends cdk.StackProps {
  readonly config: EnvironmentConfig;
  readonly frontendBucket: s3.Bucket;
  readonly apiEndpoint: string;
}

export class WebStack extends cdk.Stack {
  public readonly distributionDomainName: string;

  constructor(scope: Construct, id: string, props: WebStackProps) {
    super(scope, id, props);

    const { config, frontendBucket, apiEndpoint } = props;

    if (!config.cloudFrontEnabled) {
      this.distributionDomainName = 'disabled';
      return;
    }

    // Origin Access Identity for S3
    const oai = new cloudfront.OriginAccessIdentity(this, 'OAI', {
      comment: `SafeGAI ${config.envName} frontend OAI`,
    });

    frontendBucket.grantRead(oai);

    // Extract API Gateway domain from endpoint URL
    const apiDomain = apiEndpoint.replace('https://', '').replace(/\/.*$/, '');

    // CloudFront Distribution
    const distribution = new cloudfront.Distribution(this, 'Distribution', {
      comment: `SafeGAI ${config.envName} Cloud Frontend`,
      defaultRootObject: 'index.html',
      defaultBehavior: {
        origin: new origins.S3Origin(frontendBucket, {
          originAccessIdentity: oai,
        }),
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
      },
      additionalBehaviors: {
        '/api/*': {
          origin: new origins.HttpOrigin(apiDomain, {
            protocolPolicy: cloudfront.OriginProtocolPolicy.HTTPS_ONLY,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.HTTPS_ONLY,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
          originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
        },
      },
      errorResponses: [
        {
          httpStatus: 404,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.seconds(0),
        },
        {
          httpStatus: 403,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.seconds(0),
        },
      ],
    });

    this.distributionDomainName = distribution.distributionDomainName;

    // Outputs
    new cdk.CfnOutput(this, 'DistributionDomainName', {
      value: distribution.distributionDomainName,
      exportName: `${config.envName}-DistributionDomainName`,
    });

    new cdk.CfnOutput(this, 'DistributionId', {
      value: distribution.distributionId,
      exportName: `${config.envName}-DistributionId`,
    });

    // Tags
    cdk.Tags.of(this).add('Project', 'SafeGAI');
    cdk.Tags.of(this).add('Environment', config.envName);
    cdk.Tags.of(this).add('Module', 'web');
  }
}
