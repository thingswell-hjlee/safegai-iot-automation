import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import { SimConfig } from '../config/sim-dev';
import * as fs from 'fs';
import * as path from 'path';

export interface SimGatewayStackProps extends cdk.StackProps {
  readonly config: SimConfig;
  readonly vpc: ec2.IVpc;
  readonly instanceRole: iam.Role;
  readonly securityGroup: ec2.ISecurityGroup;
}

/**
 * Gateway stack: EC2 t3.medium instance running the SafeGAI edge gateway
 * and all simulators. Uses SSM for management (no SSH key pair needed).
 */
export class SimGatewayStack extends cdk.Stack {
  public readonly instance: ec2.Instance;

  constructor(scope: Construct, id: string, props: SimGatewayStackProps) {
    super(scope, id, props);

    // Read user-data script
    const userDataScript = fs.readFileSync(
      path.join(__dirname, '..', 'scripts', 'user-data.sh'),
      'utf-8',
    );

    // Create user data with environment variables
    const userData = ec2.UserData.forLinux();
    userData.addCommands(
      `export SAFEGAI_VERSION="${props.config.gatewayVersion}"`,
      `export SAFEGAI_PROFILE="aws-sim"`,
      `export SAFEGAI_REGION="${props.config.region}"`,
      userDataScript,
    );

    // EC2 instance for the gateway
    this.instance = new ec2.Instance(this, 'GatewayInstance', {
      vpc: props.vpc,
      vpcSubnets: { subnetType: ec2.SubnetType.PUBLIC },
      instanceType: new ec2.InstanceType(props.config.instanceType),
      machineImage: ec2.MachineImage.latestAmazonLinux2023({
        cpuType: ec2.AmazonLinuxCpuType.X86_64,
      }),
      role: props.instanceRole,
      securityGroup: props.securityGroup,
      userData,
      blockDevices: [
        {
          deviceName: '/dev/xvda',
          volume: ec2.BlockDeviceVolume.ebs(30, {
            volumeType: ec2.EbsDeviceVolumeType.GP3,
            encrypted: true,
          }),
        },
      ],
      requireImdsv2: true,
    });

    // Outputs
    new cdk.CfnOutput(this, 'InstanceId', { value: this.instance.instanceId });
    new cdk.CfnOutput(this, 'PrivateIp', {
      value: this.instance.instancePrivateIp,
    });
  }
}
