import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';

export interface SimNetworkStackProps extends cdk.StackProps {
  readonly config: SimConfig;
}

/**
 * Network stack: VPC with 2 public subnets, no NAT gateway (cost optimization).
 * All resources in public subnets with proper security groups.
 */
export class SimNetworkStack extends cdk.Stack {
  public readonly vpc: ec2.IVpc;
  public readonly gatewaySecurityGroup: ec2.ISecurityGroup;

  constructor(scope: Construct, id: string, props: SimNetworkStackProps) {
    super(scope, id, props);

    // VPC with 2 public subnets only (no NAT for cost savings)
    this.vpc = new ec2.Vpc(this, 'SimVpc', {
      ipAddresses: ec2.IpAddresses.cidr(props.config.vpcCidr),
      maxAzs: 2,
      natGateways: 0,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: 'Public',
          subnetType: ec2.SubnetType.PUBLIC,
        },
      ],
    });

    // Security group for gateway EC2 instance
    this.gatewaySecurityGroup = new ec2.SecurityGroup(this, 'GatewaySG', {
      vpc: this.vpc,
      description: 'SafeGAI simulation gateway security group',
      allowAllOutbound: true,
    });

    // Allow inbound HTTP for gateway API (from VPC only for security)
    this.gatewaySecurityGroup.addIngressRule(
      ec2.Peer.ipv4(props.config.vpcCidr),
      ec2.Port.tcp(8080),
      'Gateway HTTP API from VPC',
    );

    // Allow inbound for simulator health endpoints (from VPC only)
    this.gatewaySecurityGroup.addIngressRule(
      ec2.Peer.ipv4(props.config.vpcCidr),
      ec2.Port.tcpRange(9001, 9010),
      'Simulator HTTP APIs from VPC',
    );

    // Allow SSM access (no SSH needed)
    // SSM endpoint access is via HTTPS outbound which is already allowed

    // Outputs
    new cdk.CfnOutput(this, 'VpcId', { value: this.vpc.vpcId });
    new cdk.CfnOutput(this, 'GatewaySGId', {
      value: this.gatewaySecurityGroup.securityGroupId,
    });
  }
}
