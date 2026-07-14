/**
 * SafeGAI Notification Policy and Cooldown
 *
 * Evaluates whether an event should trigger SNS notification.
 * Implements cooldown to prevent notification fatigue.
 */

import { EventSeverity } from './event';

/** Notification channel types */
export type NotificationChannel = 'email' | 'sms';

/** Notification policy rule */
export interface NotificationRule {
  /** Minimum severity to trigger this rule */
  readonly minSeverity: EventSeverity;
  /** Channels to notify */
  readonly channels: readonly NotificationChannel[];
  /** Cooldown period in seconds before next notification of same type */
  readonly cooldownSec: number;
  /** Whether to aggregate during cooldown */
  readonly aggregate: boolean;
}

/** Default notification policy */
export interface NotificationPolicy {
  readonly version: string;
  readonly rules: readonly NotificationRule[];
  /** Global cooldown per gateway in seconds */
  readonly globalCooldownSec: number;
  /** Maximum notifications per hour per tenant */
  readonly maxPerHourPerTenant: number;
}

/** Cooldown state for a specific notification context */
export interface CooldownState {
  readonly lastNotifiedAt: string;
  readonly severity: EventSeverity;
  readonly channel: NotificationChannel;
  readonly gatewayId: string;
  readonly tenantId: string;
  readonly siteId: string;
  /** Count of suppressed notifications during cooldown */
  readonly suppressedCount: number;
}

/** Notification evaluation result */
export interface NotificationEvaluation {
  readonly shouldNotify: boolean;
  readonly channels: readonly NotificationChannel[];
  readonly reason: string;
  /** If suppressed, the cooldown remaining in seconds */
  readonly cooldownRemainingSec?: number;
  readonly suppressedCount?: number;
}

/** SNS message payload for event notification */
export interface NotificationMessage {
  readonly type: 'safety_event';
  readonly eventId: string;
  readonly tenantId: string;
  readonly siteId: string;
  readonly gatewayId: string;
  readonly severity: EventSeverity;
  readonly description: string;
  readonly detectedAt: string;
  readonly channels: readonly NotificationChannel[];
}

/** Severity ordering for comparison */
const SEVERITY_ORDER: Record<EventSeverity, number> = {
  critical: 5,
  high: 4,
  medium: 3,
  low: 2,
  info: 1,
};

/** Default notification policy for MVP */
export const DEFAULT_NOTIFICATION_POLICY: NotificationPolicy = {
  version: '1.0.0',
  rules: [
    {
      minSeverity: 'critical',
      channels: ['email', 'sms'],
      cooldownSec: 60,
      aggregate: false,
    },
    {
      minSeverity: 'high',
      channels: ['email'],
      cooldownSec: 300,
      aggregate: true,
    },
    {
      minSeverity: 'medium',
      channels: ['email'],
      cooldownSec: 900,
      aggregate: true,
    },
  ],
  globalCooldownSec: 30,
  maxPerHourPerTenant: 100,
};

/**
 * Evaluates whether a notification should be sent for the given event.
 */
export function evaluateNotification(
  severity: EventSeverity,
  gatewayId: string,
  tenantId: string,
  siteId: string,
  policy: NotificationPolicy,
  cooldownStates: readonly CooldownState[],
  currentTime: Date,
): NotificationEvaluation {
  // Find matching rules (highest priority first)
  const matchingRules = policy.rules.filter(
    (rule) => SEVERITY_ORDER[severity] >= SEVERITY_ORDER[rule.minSeverity],
  );

  if (matchingRules.length === 0) {
    return {
      shouldNotify: false,
      channels: [],
      reason: `Severity "${severity}" does not meet any notification threshold`,
    };
  }

  // Use the most specific (highest severity threshold) matching rule
  const rule = matchingRules[0];

  // Check global cooldown for this gateway
  const gatewayStates = cooldownStates.filter(
    (s) => s.gatewayId === gatewayId && s.tenantId === tenantId && s.siteId === siteId,
  );

  if (gatewayStates.length > 0) {
    const lastNotification = gatewayStates.reduce((latest, state) => {
      const stateTime = new Date(state.lastNotifiedAt).getTime();
      const latestTime = new Date(latest.lastNotifiedAt).getTime();
      return stateTime > latestTime ? state : latest;
    });

    const elapsedSec = (currentTime.getTime() - new Date(lastNotification.lastNotifiedAt).getTime()) / 1000;

    // Global cooldown check
    if (elapsedSec < policy.globalCooldownSec) {
      return {
        shouldNotify: false,
        channels: [],
        reason: `Global cooldown active (${Math.ceil(policy.globalCooldownSec - elapsedSec)}s remaining)`,
        cooldownRemainingSec: Math.ceil(policy.globalCooldownSec - elapsedSec),
        suppressedCount: lastNotification.suppressedCount + 1,
      };
    }

    // Rule-specific cooldown check
    const ruleStates = gatewayStates.filter(
      (s) => SEVERITY_ORDER[s.severity] >= SEVERITY_ORDER[rule.minSeverity],
    );

    if (ruleStates.length > 0) {
      const lastRuleNotification = ruleStates.reduce((latest, state) => {
        const stateTime = new Date(state.lastNotifiedAt).getTime();
        const latestTime = new Date(latest.lastNotifiedAt).getTime();
        return stateTime > latestTime ? state : latest;
      });

      const ruleElapsedSec = (currentTime.getTime() - new Date(lastRuleNotification.lastNotifiedAt).getTime()) / 1000;

      if (ruleElapsedSec < rule.cooldownSec) {
        return {
          shouldNotify: false,
          channels: [],
          reason: `Rule cooldown active for severity >= ${rule.minSeverity} (${Math.ceil(rule.cooldownSec - ruleElapsedSec)}s remaining)`,
          cooldownRemainingSec: Math.ceil(rule.cooldownSec - ruleElapsedSec),
          suppressedCount: lastRuleNotification.suppressedCount + 1,
        };
      }
    }
  }

  // Check hourly rate limit per tenant
  const oneHourAgo = new Date(currentTime.getTime() - 3600 * 1000);
  const hourlyCount = cooldownStates.filter(
    (s) => s.tenantId === tenantId && new Date(s.lastNotifiedAt) > oneHourAgo,
  ).length;

  if (hourlyCount >= policy.maxPerHourPerTenant) {
    return {
      shouldNotify: false,
      channels: [],
      reason: `Hourly rate limit reached (${policy.maxPerHourPerTenant}/hour for tenant)`,
    };
  }

  return {
    shouldNotify: true,
    channels: [...rule.channels],
    reason: `Severity "${severity}" matches rule (min: ${rule.minSeverity})`,
  };
}

/**
 * Builds SNS notification message payload.
 * Never includes credentials, tokens, PII, or sensitive data in the message.
 */
export function buildNotificationMessage(
  eventId: string,
  tenantId: string,
  siteId: string,
  gatewayId: string,
  severity: EventSeverity,
  description: string,
  detectedAt: string,
  channels: readonly NotificationChannel[],
): NotificationMessage {
  return {
    type: 'safety_event',
    eventId,
    tenantId,
    siteId,
    gatewayId,
    severity,
    description: description || `Safety event detected (severity: ${severity})`,
    detectedAt,
    channels,
  };
}
