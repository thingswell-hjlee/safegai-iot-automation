/**
 * SafeGAI API Types
 *
 * TypeScript interfaces matching the contracts defined in:
 * - contracts/events/event-envelope-v1.schema.json
 * - contracts/events/occupancy-state-v1.schema.json
 * - contracts/events/equipment-state-v1.schema.json
 * - contracts/events/safety-decision-v1.schema.json
 *
 * SAFETY RULES:
 * - UNKNOWN and STALE must NEVER be treated as vacant/safe.
 * - Only VACANT_CONFIRMED satisfies vacancy.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

/**
 * Zone occupancy states.
 * UNKNOWN and STALE are NEVER treated as vacancy.
 * Only VACANT_CONFIRMED satisfies the "empty zone" requirement.
 */
export enum OccupancyState {
  OCCUPIED = 'OCCUPIED',
  VACANT_PENDING = 'VACANT_PENDING',
  VACANT_CONFIRMED = 'VACANT_CONFIRMED',
  UNKNOWN = 'UNKNOWN',
  STALE = 'STALE',
}

/**
 * Equipment operational states from DI/PLC source.
 */
export enum EquipmentState {
  RUNNING = 'RUNNING',
  STOPPED = 'STOPPED',
  RESTART_REQUESTED = 'RESTART_REQUESTED',
  UNKNOWN = 'UNKNOWN',
}

/**
 * Safety rule evaluation outcome.
 * STOP_REQUEST_REQUIRED triggers actuation via PLC or Safety Relay.
 */
export enum SafetyDecision {
  SAFE = 'SAFE',
  WARNING = 'WARNING',
  STOP_REQUEST_REQUIRED = 'STOP_REQUEST_REQUIRED',
  RESTART_INTERLOCK = 'RESTART_INTERLOCK',
  SAFETY_CONFIRMATION_UNAVAILABLE = 'SAFETY_CONFIRMATION_UNAVAILABLE',
  MAINTENANCE_MONITORING = 'MAINTENANCE_MONITORING',
}

/**
 * Data quality indicator for the event payload.
 */
export enum DataQuality {
  GOOD = 'GOOD',
  UNCERTAIN = 'UNCERTAIN',
  BAD = 'BAD',
  STALE = 'STALE',
}

/**
 * Event acknowledgement status.
 */
export enum AckStatus {
  PENDING = 'PENDING',
  ACKNOWLEDGED = 'ACKNOWLEDGED',
  RESOLVED = 'RESOLVED',
}

// ---------------------------------------------------------------------------
// Event Envelope
// ---------------------------------------------------------------------------

/**
 * Common envelope wrapping all SafeGAI domain events.
 * Matches contracts/events/event-envelope-v1.schema.json
 */
export interface EventEnvelope {
  schemaVersion: string;
  eventId: string;
  correlationId: string;
  tenantId: string;
  siteId: string;
  gatewayId: string;
  deviceId: string;
  zoneId: string;
  observedAt: string;
  receivedAt: string;
  sequenceNo: number;
  source: string;
  quality: DataQuality;
}

// ---------------------------------------------------------------------------
// Domain Events
// ---------------------------------------------------------------------------

export interface OccupancyStateEvent extends EventEnvelope {
  previousState: OccupancyState;
  currentState: OccupancyState;
  transitionReason: string;
  dwellSeconds?: number;
  confirmationCount?: number;
}

export interface EquipmentStateEvent extends EventEnvelope {
  previousState: EquipmentState;
  currentState: EquipmentState;
  equipmentId: string;
  transitionReason: string;
  requestedBy?: string;
}

export interface SafetyDecisionEvent extends EventEnvelope {
  decision: SafetyDecision;
  ruleId: string;
  occupancyState: OccupancyState;
  equipmentState: EquipmentState;
  reason: string;
  actions?: string[];
}

// ---------------------------------------------------------------------------
// Status Interfaces
// ---------------------------------------------------------------------------

/**
 * Zone status combining occupancy and safety information.
 */
export interface ZoneStatus {
  zoneId: string;
  zoneName: string;
  occupancy: OccupancyState;
  safetyDecision: SafetyDecision;
  lastUpdated: string;
  cameraIds: string[];
  activeWarnings: string[];
}

/**
 * Equipment status combining state and metadata.
 */
export interface EquipmentStatus {
  equipmentId: string;
  equipmentName: string;
  state: EquipmentState;
  lastUpdated: string;
  zoneId: string;
  restartInterlockActive: boolean;
}

/**
 * Camera health and connection status.
 */
export interface CameraStatus {
  cameraId: string;
  cameraName: string;
  connected: boolean;
  streamUrl: string;
  lastFrameAt: string;
  zoneIds: string[];
  resolution: string;
  fps: number;
}

/**
 * Overall system status for the gateway.
 */
export interface SystemStatus {
  gatewayId: string;
  gatewayOnline: boolean;
  cpuPercent: number;
  memoryPercent: number;
  ssdUsedPercent: number;
  ssdHealthOk: boolean;
  awsConnected: boolean;
  lastAwsSyncAt: string;
  uptime: string;
  activeAlerts: number;
  pendingEvents: number;
}

// ---------------------------------------------------------------------------
// Event List Item (UI display model)
// ---------------------------------------------------------------------------

export interface SafetyEventItem {
  eventId: string;
  timestamp: string;
  zoneId: string;
  zoneName: string;
  type: 'occupancy' | 'equipment' | 'safety_decision';
  severity: 'critical' | 'warning' | 'info';
  summary: string;
  detail: string;
  ackStatus: AckStatus;
  acknowledgedBy?: string;
  acknowledgedAt?: string;
  resolvedBy?: string;
  resolvedAt?: string;
  classification?: string;
}
