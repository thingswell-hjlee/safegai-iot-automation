# FEAT-011: M09 AWS Cloud

Status: in_progress

## Description
AWS Cloud backend with Lambda handlers, CDK infrastructure definitions, DynamoDB schemas, and IoT topic contracts.
TypeScript implementation with proper types. NO machine control endpoints or topics.

## Key Constraints
- Region: ap-northeast-2
- NO machine control endpoint, API, topic, or shadow field
- Cloud-to-device settings: non-safety allowlist ONLY
- Event images are separate binary objects, NOT base64 in JSON
- Gateway identity bound to certificate/topic
- No long-lived AWS access keys
- INTEGRATIONS_ONLY network - no npm install
