/**
 * SafeGAI Event Schema Validation
 *
 * Validates inbound event envelopes from IoT Rule triggers.
 * Ensures gateway identity matches certificate/topic scope.
 * Rejects any base64-embedded image data.
 */

import { SafeGAIEvent, EventSeverity, OccupancyState, EquipmentState } from './event';

/** Validation error with field-level detail */
export interface ValidationError {
  readonly field: string;
  readonly message: string;
  readonly value?: unknown;
}

/** Validation result */
export interface ValidationResult {
  readonly valid: boolean;
  readonly errors: readonly ValidationError[];
}

/** IoT Rule trigger context with certificate/topic identity */
export interface IngestContext {
  /** Certificate ID from IoT Core mTLS */
  readonly certificateId: string;
  /** Topic from which the message was received */
  readonly topic: string;
  /** Tenant extracted from topic */
  readonly topicTenantId: string;
  /** Site extracted from topic */
  readonly topicSiteId: string;
  /** Gateway extracted from topic */
  readonly topicGatewayId: string;
}

const VALID_SEVERITIES: readonly EventSeverity[] = [
  'critical', 'high', 'medium', 'low', 'info',
];

const VALID_OCCUPANCY_STATES: readonly OccupancyState[] = [
  'OCCUPIED', 'VACANT_CONFIRMED', 'UNKNOWN', 'STALE',
];

const VALID_EQUIPMENT_STATES: readonly EquipmentState[] = [
  'RUNNING', 'STOPPED', 'FAULT', 'UNKNOWN',
];

const UUID_V4_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const ISO_8601_PATTERN = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;

/**
 * Validates event envelope schema.
 * Does NOT accept base64 image data in the JSON body.
 */
export function validateEventSchema(payload: unknown): ValidationResult {
  const errors: ValidationError[] = [];

  if (payload === null || payload === undefined || typeof payload !== 'object') {
    return { valid: false, errors: [{ field: 'root', message: 'Payload must be a non-null object' }] };
  }

  const event = payload as Record<string, unknown>;

  // Required string fields
  const requiredStrings: Array<[string, RegExp | null]> = [
    ['eventId', UUID_V4_PATTERN],
    ['idempotencyKey', null],
    ['detectedAt', ISO_8601_PATTERN],
    ['tenantId', null],
    ['siteId', null],
    ['gatewayId', null],
    ['cameraId', null],
    ['zoneId', null],
    ['schemaVersion', null],
  ];

  for (const [field, pattern] of requiredStrings) {
    const value = event[field];
    if (typeof value !== 'string' || value.length === 0) {
      errors.push({ field, message: `${field} must be a non-empty string`, value });
    } else if (pattern && !pattern.test(value)) {
      errors.push({ field, message: `${field} does not match expected pattern`, value });
    }
  }

  // Schema version check
  if (event['schemaVersion'] !== '1.0') {
    errors.push({ field: 'schemaVersion', message: 'schemaVersion must be "1.0"', value: event['schemaVersion'] });
  }

  // Enum fields
  if (!VALID_SEVERITIES.includes(event['severity'] as EventSeverity)) {
    errors.push({ field: 'severity', message: `severity must be one of: ${VALID_SEVERITIES.join(', ')}`, value: event['severity'] });
  }

  if (!VALID_OCCUPANCY_STATES.includes(event['occupancy'] as OccupancyState)) {
    errors.push({ field: 'occupancy', message: `occupancy must be one of: ${VALID_OCCUPANCY_STATES.join(', ')}`, value: event['occupancy'] });
  }

  if (!VALID_EQUIPMENT_STATES.includes(event['equipmentState'] as EquipmentState)) {
    errors.push({ field: 'equipmentState', message: `equipmentState must be one of: ${VALID_EQUIPMENT_STATES.join(', ')}`, value: event['equipmentState'] });
  }

  // Actions array
  if (!Array.isArray(event['actions'])) {
    errors.push({ field: 'actions', message: 'actions must be an array' });
  } else {
    for (let i = 0; i < (event['actions'] as unknown[]).length; i++) {
      const action = (event['actions'] as unknown[])[i] as Record<string, unknown>;
      if (!action || typeof action !== 'object') {
        errors.push({ field: `actions[${i}]`, message: 'action must be an object' });
        continue;
      }
      const validActionTypes = ['stop_request', 'alarm_activate', 'notification'];
      if (!validActionTypes.includes(action['type'] as string)) {
        errors.push({ field: `actions[${i}].type`, message: `action type must be one of: ${validActionTypes.join(', ')}` });
      }
      if (typeof action['target'] !== 'string' || (action['target'] as string).length === 0) {
        errors.push({ field: `actions[${i}].target`, message: 'action target must be a non-empty string' });
      }
      if (typeof action['executedAt'] !== 'string' || !ISO_8601_PATTERN.test(action['executedAt'] as string)) {
        errors.push({ field: `actions[${i}].executedAt`, message: 'action executedAt must be ISO 8601' });
      }
      if (typeof action['confirmed'] !== 'boolean') {
        errors.push({ field: `actions[${i}].confirmed`, message: 'action confirmed must be boolean' });
      }
    }
  }

  // REJECT base64-embedded image data - images must be separate binary payloads
  if ('imageData' in event || 'imageBase64' in event || 'thumbnailBase64' in event) {
    errors.push({
      field: 'imageData/imageBase64/thumbnailBase64',
      message: 'Image data must NOT be embedded as base64 in JSON. Use separate binary MQTT message on images/{eventId} topic.',
    });
  }

  // imageKey is optional but must be string if present
  if (event['imageKey'] !== undefined && typeof event['imageKey'] !== 'string') {
    errors.push({ field: 'imageKey', message: 'imageKey must be a string if present' });
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validates gateway identity against certificate and topic scope.
 * Ensures the gateway publishing the event is authorized for the claimed tenant/site/gateway.
 */
export function validateGatewayIdentity(
  event: SafeGAIEvent,
  context: IngestContext,
  registeredCertificateId: string,
): ValidationResult {
  const errors: ValidationError[] = [];

  // Certificate must match the registered gateway certificate
  if (context.certificateId !== registeredCertificateId) {
    errors.push({
      field: 'certificateId',
      message: 'Certificate ID does not match registered gateway certificate',
    });
  }

  // Topic-extracted identity must match event claims
  if (event.tenantId !== context.topicTenantId) {
    errors.push({
      field: 'tenantId',
      message: 'Event tenantId does not match topic tenant',
      value: event.tenantId,
    });
  }

  if (event.siteId !== context.topicSiteId) {
    errors.push({
      field: 'siteId',
      message: 'Event siteId does not match topic site',
      value: event.siteId,
    });
  }

  if (event.gatewayId !== context.topicGatewayId) {
    errors.push({
      field: 'gatewayId',
      message: 'Event gatewayId does not match topic gateway',
      value: event.gatewayId,
    });
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Parses MQTT topic to extract tenant, site, and gateway identifiers.
 * Topic format: safegai/v1/{tenant}/{site}/{gateway}/events
 */
export function parseTopic(topic: string): IngestContext | null {
  const parts = topic.split('/');
  // Expected: safegai/v1/{tenant}/{site}/{gateway}/{type}
  if (parts.length < 6 || parts[0] !== 'safegai' || parts[1] !== 'v1') {
    return null;
  }

  return {
    certificateId: '', // Filled from IoT Rule SQL context
    topic,
    topicTenantId: parts[2],
    topicSiteId: parts[3],
    topicGatewayId: parts[4],
  };
}
