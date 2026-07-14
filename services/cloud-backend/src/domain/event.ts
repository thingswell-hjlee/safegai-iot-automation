/**
 * SafeGAI Event Domain Types
 *
 * Matches contracts defined in contracts/mqtt/topics.md and AWS_MVP_SPEC.
 * Event images are SEPARATE binary JPEG payloads (max 96KB), NOT base64 in JSON.
 */

/** Severity levels for safety events */
export type EventSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info';

/** Occupancy states - VACANT_CONFIRMED is the ONLY valid vacancy state */
export type OccupancyState =
  | 'OCCUPIED'
  | 'VACANT_CONFIRMED'
  | 'UNKNOWN'
  | 'STALE';

/** Equipment operational states */
export type EquipmentState = 'RUNNING' | 'STOPPED' | 'FAULT' | 'UNKNOWN';

/** Event acknowledgment status */
export type AckStatus = 'pending' | 'acknowledged' | 'resolved';

/** Event classification by operator */
export type EventClassification =
  | 'true_positive'
  | 'false_positive'
  | 'needs_review'
  | 'unclassified';

/**
 * Action taken by the gateway in response to the event.
 * NOTE: Actions are PLC/Safety Relay stop-requests ONLY.
 * No direct machine power switching.
 */
export interface EventAction {
  /** Action type - stop_request to PLC/Safety Relay only */
  readonly type: 'stop_request' | 'alarm_activate' | 'notification';
  /** Target device or relay */
  readonly target: string;
  /** ISO 8601 timestamp of action execution */
  readonly executedAt: string;
  /** Whether the action was confirmed by the target */
  readonly confirmed: boolean;
}

/**
 * Event envelope published on safegai/v1/{tenant}/{site}/{gateway}/events
 *
 * Image is published separately on safegai/v1/{tenant}/{site}/{gateway}/images/{eventId}
 * as raw JPEG binary (max 96KB). It is NEVER embedded as base64 in this JSON.
 */
export interface SafeGAIEvent {
  /** Unique event identifier (UUID v4) */
  readonly eventId: string;
  /** Idempotency key for conditional write */
  readonly idempotencyKey: string;
  /** ISO 8601 detection timestamp */
  readonly detectedAt: string;
  /** Tenant identifier */
  readonly tenantId: string;
  /** Site identifier */
  readonly siteId: string;
  /** Gateway that produced this event */
  readonly gatewayId: string;
  /** Camera that captured the event */
  readonly cameraId: string;
  /** Zone where event occurred */
  readonly zoneId: string;
  /** Event severity */
  readonly severity: EventSeverity;
  /** Occupancy state at time of event */
  readonly occupancy: OccupancyState;
  /** Equipment state at time of event */
  readonly equipmentState: EquipmentState;
  /** Actions taken by gateway */
  readonly actions: readonly EventAction[];
  /**
   * S3 key for thumbnail image.
   * Format: {tenant}/{site}/{yyyy}/{mm}/{dd}/{eventId}.jpg
   * Image is stored separately as binary JPEG, NOT base64 in this envelope.
   */
  readonly imageKey?: string;
  /** Human-readable event description */
  readonly description?: string;
  /** Schema version */
  readonly schemaVersion: '1.0';
}

/** DynamoDB Events table item */
export interface EventItem {
  /** PK: tenantId#siteId */
  readonly pk: string;
  /** SK: detectedAt#eventId */
  readonly sk: string;
  readonly eventId: string;
  readonly idempotencyKey: string;
  readonly tenantId: string;
  readonly siteId: string;
  readonly gatewayId: string;
  readonly cameraId: string;
  readonly zoneId: string;
  readonly detectedAt: string;
  readonly severity: EventSeverity;
  readonly occupancy: OccupancyState;
  readonly equipmentState: EquipmentState;
  readonly actions: readonly EventAction[];
  readonly imageKey?: string;
  readonly description?: string;
  readonly schemaVersion: '1.0';
  /** Acknowledgment status */
  readonly ackStatus: AckStatus;
  readonly ackBy?: string;
  readonly ackAt?: string;
  /** Resolution */
  readonly resolvedBy?: string;
  readonly resolvedAt?: string;
  readonly resolutionNote?: string;
  /** Classification */
  readonly classification: EventClassification;
  readonly classifiedBy?: string;
  readonly classifiedAt?: string;
  /** TTL for DynamoDB (epoch seconds, default 365 days) */
  readonly ttl: number;
  /** GSI1 PK: gatewayId */
  readonly gsi1pk: string;
  /** GSI1 SK: detectedAt#eventId */
  readonly gsi1sk: string;
}

/** Paginated event list response */
export interface EventListResponse {
  readonly items: readonly EventItem[];
  readonly nextToken?: string;
  readonly count: number;
}

/** Event detail response */
export interface EventDetailResponse {
  readonly event: EventItem;
}

/** Presigned URL response for event image */
export interface EventImageResponse {
  /** Presigned S3 GET URL (expires in 15 minutes) */
  readonly url: string;
  readonly expiresAt: string;
}
