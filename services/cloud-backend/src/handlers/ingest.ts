/**
 * SafeGAI Ingest Handler - Lambda triggered by IoT Rule
 *
 * Processes event metadata from MQTT topic:
 *   safegai/v1/{tenant}/{site}/{gateway}/events
 *
 * Responsibilities:
 * - Schema validation (event envelope)
 * - Gateway/certificate/topic identity verification
 * - Idempotent event write (conditional on eventId)
 * - Gateway last-seen/health update
 * - Notification policy evaluation with cooldown
 * - SNS publish for alerts
 * - Structured logging (no credentials, tokens, or PII)
 *
 * Image handling:
 * - Images are received separately via IoT Rule on images/{eventId} topic
 * - Stored directly to S3 as raw JPEG binary (max 96KB)
 * - This handler only records the imageKey reference
 * - Base64-embedded images are REJECTED
 *
 * Security:
 * - Gateway identity bound to X.509 certificate
 * - Topic policy restricts publish scope
 * - No machine control commands
 * - No long-lived credentials
 */

import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DynamoDBDocumentClient,
  PutCommand,
  UpdateCommand,
  GetCommand,
  QueryCommand,
} from '@aws-sdk/lib-dynamodb';
import { SNSClient, PublishCommand } from '@aws-sdk/client-sns';
import { SafeGAIEvent } from '../domain/event';
import { validateEventSchema, validateGatewayIdentity, parseTopic, IngestContext } from '../domain/validation';
import {
  evaluateNotification,
  buildNotificationMessage,
  DEFAULT_NOTIFICATION_POLICY,
  CooldownState,
} from '../domain/notification';

// Environment variables
const EVENTS_TABLE = process.env['EVENTS_TABLE'] || 'Events';
const GATEWAYS_TABLE = process.env['GATEWAYS_TABLE'] || 'Gateways';
const NOTIFICATION_TOPIC_ARN = process.env['NOTIFICATION_TOPIC_ARN'] || '';
const REGION = process.env['AWS_REGION'] || 'ap-northeast-2';

// AWS SDK clients (initialized outside handler for connection reuse)
const dynamoClient = new DynamoDBClient({ region: REGION });
const docClient = DynamoDBDocumentClient.from(dynamoClient, {
  marshallOptions: { removeUndefinedValues: true },
});
const snsClient = new SNSClient({ region: REGION });

/** IoT Rule trigger event structure */
interface IoTRuleEvent {
  /** Raw MQTT payload (event JSON) */
  readonly payload: unknown;
  /** MQTT topic */
  readonly topic: string;
  /** Certificate ID from mTLS */
  readonly certificateId: string;
  /** Principal ARN */
  readonly principal: string;
}

/** Handler response */
interface IngestResult {
  readonly statusCode: number;
  readonly eventId?: string;
  readonly action: 'stored' | 'duplicate' | 'rejected';
  readonly reason?: string;
}

/** Structured log entry - never contains credentials, tokens, or PII */
function structuredLog(level: string, message: string, context: Record<string, unknown>): void {
  const logEntry = {
    timestamp: new Date().toISOString(),
    level,
    service: 'ingest-handler',
    message,
    ...context,
  };
  // Ensure no sensitive fields leak
  delete logEntry['certificateId'];
  delete logEntry['principal'];
  delete logEntry['token'];
  delete logEntry['password'];
  delete logEntry['secret'];

  if (level === 'error') {
    console.error(JSON.stringify(logEntry));
  } else {
    console.log(JSON.stringify(logEntry));
  }
}

/**
 * Main Lambda handler for IoT Rule triggered event ingestion.
 */
export async function handler(iotEvent: IoTRuleEvent): Promise<IngestResult> {
  const startTime = Date.now();

  structuredLog('info', 'Ingest handler invoked', {
    topic: iotEvent.topic,
    hasPayload: !!iotEvent.payload,
  });

  // 1. Parse topic to extract identity context
  const topicContext = parseTopic(iotEvent.topic);
  if (!topicContext) {
    structuredLog('error', 'Invalid topic format', { topic: iotEvent.topic });
    return { statusCode: 400, action: 'rejected', reason: 'Invalid topic format' };
  }

  const ingestContext: IngestContext = {
    ...topicContext,
    certificateId: iotEvent.certificateId,
  };

  // 2. Schema validation
  const schemaResult = validateEventSchema(iotEvent.payload);
  if (!schemaResult.valid) {
    structuredLog('error', 'Schema validation failed', {
      errors: schemaResult.errors.map((e) => ({ field: e.field, message: e.message })),
      tenantId: ingestContext.topicTenantId,
      siteId: ingestContext.topicSiteId,
      gatewayId: ingestContext.topicGatewayId,
    });
    return {
      statusCode: 400,
      action: 'rejected',
      reason: `Schema validation failed: ${schemaResult.errors.map((e) => e.message).join('; ')}`,
    };
  }

  const event = iotEvent.payload as unknown as SafeGAIEvent;

  // 3. Gateway identity verification (certificate/topic scope)
  const registeredCertId = await getRegisteredCertificateId(event.gatewayId);
  if (!registeredCertId) {
    structuredLog('error', 'Gateway not registered', {
      gatewayId: event.gatewayId,
      tenantId: event.tenantId,
      siteId: event.siteId,
    });
    return { statusCode: 403, action: 'rejected', reason: 'Gateway not registered' };
  }

  const identityResult = validateGatewayIdentity(event, ingestContext, registeredCertId);
  if (!identityResult.valid) {
    structuredLog('error', 'Gateway identity verification failed', {
      errors: identityResult.errors.map((e) => ({ field: e.field, message: e.message })),
      gatewayId: event.gatewayId,
    });
    return { statusCode: 403, action: 'rejected', reason: 'Gateway identity verification failed' };
  }

  // 4. Idempotent event write (conditional on eventId)
  const stored = await storeEvent(event);
  if (!stored) {
    structuredLog('info', 'Duplicate event detected', {
      eventId: event.eventId,
      gatewayId: event.gatewayId,
    });
    return { statusCode: 200, eventId: event.eventId, action: 'duplicate' };
  }

  // 5. Update gateway last-seen/health
  await updateGatewayLastSeen(event.tenantId, event.siteId, event.gatewayId);

  // 6. Notification policy evaluation with cooldown
  await evaluateAndNotify(event);

  const duration = Date.now() - startTime;
  structuredLog('info', 'Event ingested successfully', {
    eventId: event.eventId,
    gatewayId: event.gatewayId,
    severity: event.severity,
    durationMs: duration,
  });

  return { statusCode: 200, eventId: event.eventId, action: 'stored' };
}

/**
 * Retrieves the registered certificate ID for a gateway.
 * Uses GSI1 (gatewayId) for lookup.
 */
async function getRegisteredCertificateId(gatewayId: string): Promise<string | null> {
  try {
    const result = await docClient.send(new QueryCommand({
      TableName: GATEWAYS_TABLE,
      IndexName: 'GSI1',
      KeyConditionExpression: 'gsi1pk = :gw',
      ExpressionAttributeValues: { ':gw': gatewayId },
      ProjectionExpression: 'certificateId',
      Limit: 1,
    }));

    if (result.Items && result.Items.length > 0) {
      return result.Items[0]['certificateId'] as string;
    }
    return null;
  } catch (error) {
    structuredLog('error', 'Failed to query gateway certificate', {
      gatewayId,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
    return null;
  }
}

/**
 * Stores event with conditional write for idempotency.
 * Returns true if stored, false if duplicate.
 */
async function storeEvent(event: SafeGAIEvent): Promise<boolean> {
  const now = new Date();
  const ttl = Math.floor(now.getTime() / 1000) + 365 * 24 * 3600; // 365 days

  try {
    await docClient.send(new PutCommand({
      TableName: EVENTS_TABLE,
      Item: {
        pk: `${event.tenantId}#${event.siteId}`,
        sk: `${event.detectedAt}#${event.eventId}`,
        eventId: event.eventId,
        idempotencyKey: event.idempotencyKey,
        tenantId: event.tenantId,
        siteId: event.siteId,
        gatewayId: event.gatewayId,
        cameraId: event.cameraId,
        zoneId: event.zoneId,
        detectedAt: event.detectedAt,
        severity: event.severity,
        occupancy: event.occupancy,
        equipmentState: event.equipmentState,
        actions: event.actions,
        imageKey: event.imageKey,
        description: event.description,
        schemaVersion: event.schemaVersion,
        ackStatus: 'pending',
        classification: 'unclassified',
        ttl,
        gsi1pk: event.gatewayId,
        gsi1sk: `${event.detectedAt}#${event.eventId}`,
      },
      // Conditional write: only succeed if eventId does not already exist
      ConditionExpression: 'attribute_not_exists(pk) AND attribute_not_exists(sk)',
    }));
    return true;
  } catch (error) {
    if (error instanceof Error && error.name === 'ConditionalCheckFailedException') {
      return false; // Duplicate - idempotency check passed
    }
    structuredLog('error', 'Failed to store event', {
      eventId: event.eventId,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
    throw error;
  }
}

/**
 * Updates gateway last-seen timestamp and online status.
 */
async function updateGatewayLastSeen(
  tenantId: string,
  siteId: string,
  gatewayId: string,
): Promise<void> {
  const now = new Date().toISOString();

  try {
    await docClient.send(new UpdateCommand({
      TableName: GATEWAYS_TABLE,
      Key: {
        pk: tenantId,
        sk: `${siteId}#${gatewayId}`,
      },
      UpdateExpression: 'SET lastSeen = :ls, #status = :st, updatedAt = :ua',
      ExpressionAttributeNames: { '#status': 'status' },
      ExpressionAttributeValues: {
        ':ls': now,
        ':st': 'online',
        ':ua': now,
      },
    }));
  } catch (error) {
    structuredLog('error', 'Failed to update gateway last-seen', {
      gatewayId,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
    // Non-fatal: event is already stored
  }
}

/**
 * Evaluates notification policy and publishes to SNS if needed.
 */
async function evaluateAndNotify(event: SafeGAIEvent): Promise<void> {
  // Retrieve recent cooldown states for this gateway
  const cooldownStates = await getRecentCooldownStates(event.tenantId, event.siteId, event.gatewayId);

  const evaluation = evaluateNotification(
    event.severity,
    event.gatewayId,
    event.tenantId,
    event.siteId,
    DEFAULT_NOTIFICATION_POLICY,
    cooldownStates,
    new Date(),
  );

  if (!evaluation.shouldNotify) {
    structuredLog('info', 'Notification suppressed', {
      eventId: event.eventId,
      reason: evaluation.reason,
      cooldownRemainingSec: evaluation.cooldownRemainingSec,
    });
    return;
  }

  // Build and publish notification
  const message = buildNotificationMessage(
    event.eventId,
    event.tenantId,
    event.siteId,
    event.gatewayId,
    event.severity,
    event.description || `Safety event detected (severity: ${event.severity})`,
    event.detectedAt,
    evaluation.channels,
  );

  try {
    if (NOTIFICATION_TOPIC_ARN) {
      await snsClient.send(new PublishCommand({
        TopicArn: NOTIFICATION_TOPIC_ARN,
        Subject: `[SafeGAI ${event.severity.toUpperCase()}] Safety event at ${event.siteId}`,
        Message: JSON.stringify(message),
        MessageAttributes: {
          severity: { DataType: 'String', StringValue: event.severity },
          tenantId: { DataType: 'String', StringValue: event.tenantId },
          siteId: { DataType: 'String', StringValue: event.siteId },
        },
      }));

      structuredLog('info', 'Notification published', {
        eventId: event.eventId,
        severity: event.severity,
        channels: evaluation.channels,
      });
    }
  } catch (error) {
    structuredLog('error', 'Failed to publish notification', {
      eventId: event.eventId,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
    // Non-fatal: event is already stored
  }
}

/**
 * Retrieves recent cooldown states for notification evaluation.
 * Uses a simple query on recent events to approximate cooldown state.
 */
async function getRecentCooldownStates(
  tenantId: string,
  siteId: string,
  gatewayId: string,
): Promise<CooldownState[]> {
  try {
    const oneHourAgo = new Date(Date.now() - 3600 * 1000).toISOString();

    const result = await docClient.send(new QueryCommand({
      TableName: EVENTS_TABLE,
      IndexName: 'GSI1',
      KeyConditionExpression: 'gsi1pk = :gw AND gsi1sk > :since',
      ExpressionAttributeValues: {
        ':gw': gatewayId,
        ':since': oneHourAgo,
      },
      ProjectionExpression: 'detectedAt, severity, tenantId, siteId, gatewayId',
      Limit: 50,
      ScanIndexForward: false,
    }));

    if (!result.Items || result.Items.length === 0) {
      return [];
    }

    // Convert recent events to cooldown states
    return result.Items.map((item) => ({
      lastNotifiedAt: item['detectedAt'] as string,
      severity: item['severity'] as 'critical' | 'high' | 'medium' | 'low' | 'info',
      channel: 'email' as const,
      gatewayId,
      tenantId,
      siteId,
      suppressedCount: 0,
    }));
  } catch (error) {
    structuredLog('error', 'Failed to query cooldown states', {
      gatewayId,
      error: error instanceof Error ? error.message : 'Unknown error',
    });
    return [];
  }
}
