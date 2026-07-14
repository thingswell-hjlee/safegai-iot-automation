/**
 * SafeGAI Gateway Domain Types
 *
 * Gateway identity is bound to X.509 certificate and topic scope.
 * Hardware inventory is informational only - never used for logic branching or remote commands.
 */

/** Gateway operational status */
export type GatewayStatus = 'online' | 'offline' | 'degraded';

/** Camera connection summary */
export interface CameraSummary {
  readonly cameraId: string;
  readonly connected: boolean;
  readonly resolution?: string;
  readonly fps?: number;
}

/** I/O module summary */
export interface IOSummary {
  readonly moduleId: string;
  readonly type: 'DI' | 'DO' | 'AI';
  readonly connected: boolean;
  readonly channelCount: number;
}

/** SSD health information */
export interface SSDHealth {
  readonly model: string;
  readonly capacityGiB: number;
  readonly healthPercent: number;
  readonly temperatureCelsius?: number;
}

/** NIC link state */
export interface NICState {
  readonly interfaceName: string;
  readonly driver: string;
  readonly linkUp: boolean;
  readonly speedMbps?: number;
}

/**
 * Hardware inventory - informational only.
 * NOT used for remote command execution or logic branching in cloud.
 */
export interface HardwareInventory {
  readonly hardwareProfileId: string;
  readonly manufacturer: string;
  readonly model: string;
  readonly revision: string;
  readonly serialNumber: string;
  readonly architecture: 'amd64';
  readonly cpuModel: string;
  readonly cpuCoreCount: number;
  readonly memoryMiB: number;
  readonly ssd: SSDHealth;
  readonly nicCount: number;
  readonly nics: readonly NICState[];
  readonly biosVersion: string;
  readonly tpmCapable: boolean;
  readonly watchdogCapable: boolean;
}

/** Gateway health reported via Device Shadow (reported only) */
export interface GatewayHealth {
  readonly online: boolean;
  readonly lastSeen: string;
  readonly version: string;
  readonly hardwareProfileId: string;
  readonly osImageId: string;
  readonly cameras: readonly CameraSummary[];
  readonly ioModules: readonly IOSummary[];
  readonly cpuPercent: number;
  readonly ramPercent: number;
  readonly diskPercent: number;
  readonly temperatureCelsius?: number;
  readonly ssdHealth: SSDHealth;
  readonly nicStates: readonly NICState[];
  readonly outboxDepth: number;
}

/**
 * Non-safety settings allowlist for Device Shadow desired/reported.
 *
 * FORBIDDEN in shadow settings:
 * - Safety rules
 * - I/O mapping
 * - Stop-request behavior
 * - Restart interlock
 * - BIOS/OS image changes
 * - Occupancy override
 */
export interface GatewaySettings {
  /** Heartbeat interval in seconds */
  readonly heartbeatIntervalSec: number;
  /** Whether to upload cloud thumbnails */
  readonly cloudThumbnailEnabled: boolean;
  /** Log level with expiration */
  readonly logLevel: 'debug' | 'info' | 'warn' | 'error';
  readonly logLevelExpiresAt?: string;
  /** Notification policy version identifier */
  readonly notificationPolicyVersion: string;
}

/** DynamoDB Gateways table item */
export interface GatewayItem {
  /** PK: tenantId */
  readonly pk: string;
  /** SK: siteId#gatewayId */
  readonly sk: string;
  readonly tenantId: string;
  readonly siteId: string;
  readonly gatewayId: string;
  /** X.509 certificate ID bound to this gateway */
  readonly certificateId: string;
  readonly status: GatewayStatus;
  readonly lastSeen: string;
  readonly version: string;
  readonly hardwareProfileId: string;
  readonly hardwareInventory: HardwareInventory;
  readonly osImageId: string;
  readonly health: GatewayHealth;
  readonly cameras: readonly CameraSummary[];
  readonly outboxDepth: number;
  readonly settings: GatewaySettings;
  /** GSI1 PK: gatewayId (for certificate/topic-bound ingest lookup) */
  readonly gsi1pk: string;
  readonly createdAt: string;
  readonly updatedAt: string;
}

/** Gateway list response for admin API */
export interface GatewayListResponse {
  readonly items: readonly GatewayItem[];
  readonly count: number;
}

/** Gateway detail/status response */
export interface GatewayStatusResponse {
  readonly gateway: GatewayItem;
}
