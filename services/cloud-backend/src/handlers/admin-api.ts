/**
 * SafeGAI Admin API Handler - Lambda triggered by HTTP API Gateway
 *
 * Cognito JWT authorization with tenant/site claims.
 * Provides read-only status, event management, and presigned image URLs.
 *
 * NO machine control endpoints exist in this handler.
 * NO direct camera credentials or live video relay.
 * NO safety I/O mapping changes.
 *
 * Endpoints:
 * - GET /sites/{siteId}/gateways
 * - GET /sites/{siteId}/gateways/{gatewayId}/status
 * - GET /sites/{siteId}/events
 * - GET /sites/{siteId}/events/{eventId}
 * - POST /sites/{siteId}/events/{eventId}/ack
 * - POST /sites/{siteId}/events/{eventId}/resolve
 * - POST /sites/{siteId}/events/{eventId}/classify
 * - GET /sites/{siteId}/events/{eventId}/image
 */

import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DynamoDBDocumentClient,
  QueryCommand,
  GetCommand,
  UpdateCommand,
} from '@aws-sdk/lib-dynamodb';
import { S3Client, GetObjectCommand } from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';

// Environment variables
const EVENTS_TABLE = process.env['EVENTS_TABLE'] || 'Events';
const GATEWAYS_TABLE = process.env['GATEWAYS_TABLE'] || 'Gateways';
const EVIDENCE_BUCKET = process.env['EVIDENCE_BUCKET'] || '';
const REGION = process.env['AWS_REGION'] || 'ap-northeast-2';

// AWS SDK clients
const dynamoClient = new DynamoDBClient({ region: REGION });
const docClient = DynamoDBDocumentClient.from(dynamoClient, {
  marshallOptions: { removeUndefinedValues: true },
});
const s3Client = new S3Client({ region: REGION });

/** API Gateway HTTP API v2 event */
interface APIGatewayV2Event {
  readonly routeKey: string;
  readonly rawPath: string;
  readonly pathParameters?: Record<string, string>;
  readonly queryStringParameters?: Record<string, string>;
  readonly headers: Record<string, string>;
  readonly body?: string;
  readonly requestContext: {
    readonly authorizer?: {
      readonly jwt?: {
        readonly claims: Record<string, string>;
      };
    };
    readonly http: {
      readonly method: string;
      readonly path: string;
    };
  };
}

/** API Gateway response */
interface APIGatewayResponse {
  readonly statusCode: number;
  readonly headers: Record<string, string>;
  readonly body: string;
}

/** JWT claims from Cognito */
interface UserClaims {
  readonly sub: string;
  readonly email: string;
  readonly tenantId: string;
  readonly siteIds: string[];
  readonly groups: string[];
}

// --- Helper functions ---

function jsonResponse(statusCode: number, body: unknown): APIGatewayResponse {
  return {
    statusCode,
    headers: {
      'Content-Type': 'application/json',
      'X-Content-Type-Options': 'nosniff',
      'Cache-Control': 'no-store',
    },
    body: JSON.stringify(body),
  };
}

function extractClaims(event: APIGatewayV2Event): UserClaims | null {
  const jwt = event.requestContext.authorizer?.jwt;
  if (!jwt) return null;

  const claims = jwt.claims;
  return {
    sub: claims['sub'] || '',
    email: claims['email'] || '',
    tenantId: claims['custom:tenantId'] || '',
    siteIds: (claims['custom:siteIds'] || '').split(',').filter(Boolean),
    groups: (claims['cognito:groups'] || '').split(',').filter(Boolean),
  };
}

function authorizeSite(claims: UserClaims, siteId: string): boolean {
  return claims.siteIds.includes(siteId) || claims.siteIds.includes('*');
}

// --- Route handlers ---

async function listGateways(
  tenantId: string,
  siteId: string,
): Promise<APIGatewayResponse> {
  const result = await docClient.send(new QueryCommand({
    TableName: GATEWAYS_TABLE,
    KeyConditionExpression: 'pk = :tenant AND begins_with(sk, :site)',
    ExpressionAttributeValues: {
      ':tenant': tenantId,
      ':site': `${siteId}#`,
    },
  }));

  return jsonResponse(200, {
    items: result.Items || [],
    count: result.Count || 0,
  });
}

async function getGatewayStatus(
  tenantId: string,
  siteId: string,
  gatewayId: string,
): Promise<APIGatewayResponse> {
  const result = await docClient.send(new GetCommand({
    TableName: GATEWAYS_TABLE,
    Key: { pk: tenantId, sk: `${siteId}#${gatewayId}` },
  }));

  if (!result.Item) {
    return jsonResponse(404, { error: 'Gateway not found' });
  }

  return jsonResponse(200, { gateway: result.Item });
}

async function listEvents(
  tenantId: string,
  siteId: string,
  queryParams?: Record<string, string>,
): Promise<APIGatewayResponse> {
  const limit = Math.min(parseInt(queryParams?.['limit'] || '20', 10), 100);
  const nextToken = queryParams?.['nextToken'];

  const params: Record<string, unknown> = {
    TableName: EVENTS_TABLE,
    KeyConditionExpression: 'pk = :pk',
    ExpressionAttributeValues: { ':pk': `${tenantId}#${siteId}` },
    Limit: limit,
    ScanIndexForward: false, // Most recent first
  };

  if (nextToken) {
    try {
      params['ExclusiveStartKey'] = JSON.parse(
        Buffer.from(nextToken, 'base64').toString('utf-8'),
      );
    } catch {
      return jsonResponse(400, { error: 'Invalid nextToken' });
    }
  }

  const result = await docClient.send(new QueryCommand(params as Parameters<typeof docClient.send>[0] extends { input: infer I } ? I : never));

  let responseNextToken: string | undefined;
  if (result.LastEvaluatedKey) {
    responseNextToken = Buffer.from(
      JSON.stringify(result.LastEvaluatedKey),
    ).toString('base64');
  }

  return jsonResponse(200, {
    items: result.Items || [],
    count: result.Count || 0,
    nextToken: responseNextToken,
  });
}

async function getEventDetail(
  tenantId: string,
  siteId: string,
  eventId: string,
): Promise<APIGatewayResponse> {
  // Query by PK with filter on eventId since SK includes detectedAt
  const result = await docClient.send(new QueryCommand({
    TableName: EVENTS_TABLE,
    KeyConditionExpression: 'pk = :pk',
    FilterExpression: 'eventId = :eid',
    ExpressionAttributeValues: {
      ':pk': `${tenantId}#${siteId}`,
      ':eid': eventId,
    },
    Limit: 1,
  }));

  if (!result.Items || result.Items.length === 0) {
    return jsonResponse(404, { error: 'Event not found' });
  }

  return jsonResponse(200, { event: result.Items[0] });
}

async function acknowledgeEvent(
  tenantId: string,
  siteId: string,
  eventId: string,
  userId: string,
): Promise<APIGatewayResponse> {
  // Find event first to get the SK
  const eventResult = await findEventBySiteAndId(tenantId, siteId, eventId);
  if (!eventResult) {
    return jsonResponse(404, { error: 'Event not found' });
  }

  const now = new Date().toISOString();
  await docClient.send(new UpdateCommand({
    TableName: EVENTS_TABLE,
    Key: { pk: eventResult['pk'], sk: eventResult['sk'] },
    UpdateExpression: 'SET ackStatus = :status, ackBy = :by, ackAt = :at',
    ExpressionAttributeValues: {
      ':status': 'acknowledged',
      ':by': userId,
      ':at': now,
    },
    ConditionExpression: 'attribute_exists(pk)',
  }));

  return jsonResponse(200, { eventId, ackStatus: 'acknowledged', ackBy: userId, ackAt: now });
}

async function resolveEvent(
  tenantId: string,
  siteId: string,
  eventId: string,
  userId: string,
  body: string | undefined,
): Promise<APIGatewayResponse> {
  const eventResult = await findEventBySiteAndId(tenantId, siteId, eventId);
  if (!eventResult) {
    return jsonResponse(404, { error: 'Event not found' });
  }

  let note = '';
  if (body) {
    try {
      const parsed = JSON.parse(body);
      note = parsed.note || '';
    } catch {
      // No note provided
    }
  }

  const now = new Date().toISOString();
  await docClient.send(new UpdateCommand({
    TableName: EVENTS_TABLE,
    Key: { pk: eventResult['pk'], sk: eventResult['sk'] },
    UpdateExpression: 'SET ackStatus = :status, resolvedBy = :by, resolvedAt = :at, resolutionNote = :note',
    ExpressionAttributeValues: {
      ':status': 'resolved',
      ':by': userId,
      ':at': now,
      ':note': note,
    },
    ConditionExpression: 'attribute_exists(pk)',
  }));

  return jsonResponse(200, { eventId, ackStatus: 'resolved', resolvedBy: userId, resolvedAt: now });
}

async function classifyEvent(
  tenantId: string,
  siteId: string,
  eventId: string,
  userId: string,
  body: string | undefined,
): Promise<APIGatewayResponse> {
  if (!body) {
    return jsonResponse(400, { error: 'Request body required with classification field' });
  }

  let classification: string;
  try {
    const parsed = JSON.parse(body);
    classification = parsed.classification;
  } catch {
    return jsonResponse(400, { error: 'Invalid JSON body' });
  }

  const validClassifications = ['true_positive', 'false_positive', 'needs_review', 'unclassified'];
  if (!validClassifications.includes(classification)) {
    return jsonResponse(400, {
      error: `classification must be one of: ${validClassifications.join(', ')}`,
    });
  }

  const eventResult = await findEventBySiteAndId(tenantId, siteId, eventId);
  if (!eventResult) {
    return jsonResponse(404, { error: 'Event not found' });
  }

  const now = new Date().toISOString();
  await docClient.send(new UpdateCommand({
    TableName: EVENTS_TABLE,
    Key: { pk: eventResult['pk'], sk: eventResult['sk'] },
    UpdateExpression: 'SET classification = :cls, classifiedBy = :by, classifiedAt = :at',
    ExpressionAttributeValues: {
      ':cls': classification,
      ':by': userId,
      ':at': now,
    },
    ConditionExpression: 'attribute_exists(pk)',
  }));

  return jsonResponse(200, { eventId, classification, classifiedBy: userId, classifiedAt: now });
}

async function getEventImage(
  tenantId: string,
  siteId: string,
  eventId: string,
): Promise<APIGatewayResponse> {
  const eventResult = await findEventBySiteAndId(tenantId, siteId, eventId);
  if (!eventResult) {
    return jsonResponse(404, { error: 'Event not found' });
  }

  const imageKey = eventResult['imageKey'] as string | undefined;
  if (!imageKey) {
    return jsonResponse(404, { error: 'No image available for this event' });
  }

  // Generate presigned S3 GET URL (15 min expiry)
  const command = new GetObjectCommand({
    Bucket: EVIDENCE_BUCKET,
    Key: imageKey,
  });
  const url = await getSignedUrl(s3Client, command, { expiresIn: 900 });
  const expiresAt = new Date(Date.now() + 900 * 1000).toISOString();

  return jsonResponse(200, { url, expiresAt });
}

/** Find event by siteId and eventId (needs query since SK includes detectedAt) */
async function findEventBySiteAndId(
  tenantId: string,
  siteId: string,
  eventId: string,
): Promise<Record<string, unknown> | null> {
  const result = await docClient.send(new QueryCommand({
    TableName: EVENTS_TABLE,
    KeyConditionExpression: 'pk = :pk',
    FilterExpression: 'eventId = :eid',
    ExpressionAttributeValues: {
      ':pk': `${tenantId}#${siteId}`,
      ':eid': eventId,
    },
    Limit: 1,
  }));

  return result.Items && result.Items.length > 0 ? result.Items[0] : null;
}

// --- Main handler ---

/**
 * Lambda handler for HTTP API Gateway with Cognito JWT authorization.
 * Routes requests to appropriate handlers based on path and method.
 *
 * NO machine control endpoints are implemented.
 */
export async function handler(event: APIGatewayV2Event): Promise<APIGatewayResponse> {
  // Extract and verify JWT claims
  const claims = extractClaims(event);
  if (!claims || !claims.tenantId) {
    return jsonResponse(401, { error: 'Unauthorized: missing or invalid token claims' });
  }

  const method = event.requestContext.http.method;
  const path = event.rawPath;
  const params = event.pathParameters || {};
  const siteId = params['siteId'] || '';

  // Authorize site access
  if (siteId && !authorizeSite(claims, siteId)) {
    return jsonResponse(403, { error: 'Forbidden: no access to this site' });
  }

  try {
    // Route matching
    if (method === 'GET' && path.match(/^\/sites\/[^/]+\/gateways$/)) {
      return await listGateways(claims.tenantId, siteId);
    }

    if (method === 'GET' && path.match(/^\/sites\/[^/]+\/gateways\/[^/]+\/status$/)) {
      const gatewayId = params['gatewayId'] || '';
      return await getGatewayStatus(claims.tenantId, siteId, gatewayId);
    }

    if (method === 'GET' && path.match(/^\/sites\/[^/]+\/events$/) ) {
      return await listEvents(claims.tenantId, siteId, event.queryStringParameters);
    }

    if (method === 'GET' && path.match(/^\/sites\/[^/]+\/events\/[^/]+$/) && !path.includes('/image')) {
      const eventId = params['eventId'] || '';
      return await getEventDetail(claims.tenantId, siteId, eventId);
    }

    if (method === 'POST' && path.match(/^\/sites\/[^/]+\/events\/[^/]+\/ack$/)) {
      const eventId = params['eventId'] || '';
      return await acknowledgeEvent(claims.tenantId, siteId, eventId, claims.sub);
    }

    if (method === 'POST' && path.match(/^\/sites\/[^/]+\/events\/[^/]+\/resolve$/)) {
      const eventId = params['eventId'] || '';
      return await resolveEvent(claims.tenantId, siteId, eventId, claims.sub, event.body);
    }

    if (method === 'POST' && path.match(/^\/sites\/[^/]+\/events\/[^/]+\/classify$/)) {
      const eventId = params['eventId'] || '';
      return await classifyEvent(claims.tenantId, siteId, eventId, claims.sub, event.body);
    }

    if (method === 'GET' && path.match(/^\/sites\/[^/]+\/events\/[^/]+\/image$/)) {
      const eventId = params['eventId'] || '';
      return await getEventImage(claims.tenantId, siteId, eventId);
    }

    // No machine control endpoints exist
    return jsonResponse(404, { error: 'Not found' });
  } catch (error) {
    console.error(JSON.stringify({
      timestamp: new Date().toISOString(),
      level: 'error',
      service: 'admin-api-handler',
      message: 'Unhandled error',
      error: error instanceof Error ? error.message : 'Unknown error',
      path,
      method,
    }));

    return jsonResponse(500, { error: 'Internal server error' });
  }
}
