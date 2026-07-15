/**
 * StatusPanel Component
 *
 * Displays zone occupancy status with safety-appropriate colors and icons.
 *
 * COLOR CODING (per ROLE_MODE_SPEC.md and PRODUCT_MVP_SPEC.md):
 * - RED (danger): OCCUPIED + RUNNING equipment = immediate risk
 * - YELLOW/AMBER (warning): UNKNOWN, STALE, or VACANT_PENDING = uncertain state
 * - GREEN (safe): VACANT_CONFIRMED only = confirmed empty zone
 *
 * SAFETY UX RULES:
 * - UNKNOWN and STALE must NEVER appear as safe/vacant
 * - Use color + icon + text (not color alone)
 * - Active critical warnings remain visible regardless of view
 */

import React from 'react';
import { OccupancyState, SafetyDecision } from '../types/api';
import type { ZoneStatus } from '../types/api';

// ---------------------------------------------------------------------------
// Safety Color/Icon Mapping
// ---------------------------------------------------------------------------

interface StatusStyle {
  bgColor: string;
  textColor: string;
  borderColor: string;
  icon: string;
  label: string;
}

/**
 * Determine visual style based on occupancy and safety decision.
 * NEVER display UNKNOWN/STALE as safe.
 */
function getZoneStyle(zone: ZoneStatus): StatusStyle {
  const { occupancy, safetyDecision } = zone;

  // CRITICAL: STOP_REQUEST_REQUIRED always shows as danger
  if (safetyDecision === SafetyDecision.STOP_REQUEST_REQUIRED) {
    return {
      bgColor: '#fee2e2',
      textColor: '#991b1b',
      borderColor: '#ef4444',
      icon: '\u26D4', // no entry
      label: 'DANGER - Stop Requested',
    };
  }

  // UNKNOWN and STALE: always warning/danger, NEVER safe
  if (occupancy === OccupancyState.UNKNOWN || occupancy === OccupancyState.STALE) {
    return {
      bgColor: '#fef3c7',
      textColor: '#92400e',
      borderColor: '#f59e0b',
      icon: '\u26A0\uFE0F', // warning sign
      label: occupancy === OccupancyState.STALE
        ? 'STALE - Data Outdated'
        : 'UNKNOWN - Cannot Confirm',
    };
  }

  // OCCUPIED with running equipment: red danger
  if (occupancy === OccupancyState.OCCUPIED) {
    if (safetyDecision === SafetyDecision.WARNING || safetyDecision === SafetyDecision.STOP_REQUEST_REQUIRED) {
      return {
        bgColor: '#fee2e2',
        textColor: '#991b1b',
        borderColor: '#ef4444',
        icon: '\u{1F6A8}', // rotating light
        label: 'OCCUPIED - Active Warning',
      };
    }
    return {
      bgColor: '#fef3c7',
      textColor: '#92400e',
      borderColor: '#f59e0b',
      icon: '\u{1F6B6}', // walking person
      label: 'OCCUPIED - Person Present',
    };
  }

  // VACANT_PENDING: caution (not yet confirmed)
  if (occupancy === OccupancyState.VACANT_PENDING) {
    return {
      bgColor: '#fef3c7',
      textColor: '#92400e',
      borderColor: '#f59e0b',
      icon: '\u23F3', // hourglass
      label: 'PENDING - Confirming Vacancy',
    };
  }

  // VACANT_CONFIRMED: the ONLY safe/green state
  if (occupancy === OccupancyState.VACANT_CONFIRMED && safetyDecision === SafetyDecision.SAFE) {
    return {
      bgColor: '#dcfce7',
      textColor: '#166534',
      borderColor: '#22c55e',
      icon: '\u2705', // check mark
      label: 'SAFE - Vacancy Confirmed',
    };
  }

  // Default: warning state for any unhandled combination
  return {
    bgColor: '#fef3c7',
    textColor: '#92400e',
    borderColor: '#f59e0b',
    icon: '\u2753', // question mark
    label: 'CAUTION - Review Required',
  };
}

// ---------------------------------------------------------------------------
// Component Props
// ---------------------------------------------------------------------------

interface StatusPanelProps {
  zones: ZoneStatus[];
  loading?: boolean;
  error?: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const StatusPanel: React.FC<StatusPanelProps> = ({ zones, loading, error }) => {
  if (loading) {
    return (
      <div className="status-panel status-panel--loading" role="status" aria-busy="true">
        <span className="status-panel__loader">Loading zone statuses...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="status-panel status-panel--error" role="alert">
        <span className="status-panel__error-icon">{'\u26A0\uFE0F'}</span>
        <span className="status-panel__error-text">{error}</span>
      </div>
    );
  }

  // Separate critical zones (active warnings) to show at top
  const criticalZones = zones.filter(
    (z) =>
      z.safetyDecision === SafetyDecision.STOP_REQUEST_REQUIRED ||
      z.occupancy === OccupancyState.UNKNOWN ||
      z.occupancy === OccupancyState.STALE
  );
  const otherZones = zones.filter(
    (z) =>
      z.safetyDecision !== SafetyDecision.STOP_REQUEST_REQUIRED &&
      z.occupancy !== OccupancyState.UNKNOWN &&
      z.occupancy !== OccupancyState.STALE
  );

  return (
    <div className="status-panel" role="region" aria-label="Zone Safety Status">
      <h2 className="status-panel__title">Zone Safety Status</h2>

      {/* Critical warnings always visible at top */}
      {criticalZones.length > 0 && (
        <div className="status-panel__critical" role="alert" aria-live="assertive">
          <h3 className="status-panel__critical-title">
            {'\u26A0\uFE0F'} Active Warnings ({criticalZones.length})
          </h3>
          {criticalZones.map((zone) => (
            <ZoneCard key={zone.zoneId} zone={zone} />
          ))}
        </div>
      )}

      {/* Normal zones */}
      <div className="status-panel__zones">
        {otherZones.map((zone) => (
          <ZoneCard key={zone.zoneId} zone={zone} />
        ))}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Zone Card Sub-component
// ---------------------------------------------------------------------------

interface ZoneCardProps {
  zone: ZoneStatus;
}

const ZoneCard: React.FC<ZoneCardProps> = ({ zone }) => {
  const style = getZoneStyle(zone);

  return (
    <div
      className="zone-card"
      style={{
        backgroundColor: style.bgColor,
        borderLeft: `4px solid ${style.borderColor}`,
        color: style.textColor,
      }}
      role="article"
      aria-label={`Zone ${zone.zoneName}: ${style.label}`}
    >
      <div className="zone-card__header">
        <span className="zone-card__icon" aria-hidden="true">{style.icon}</span>
        <span className="zone-card__name">{zone.zoneName}</span>
      </div>
      <div className="zone-card__status">
        <span className="zone-card__label">{style.label}</span>
      </div>
      <div className="zone-card__details">
        <span className="zone-card__occupancy">Occupancy: {zone.occupancy}</span>
        <span className="zone-card__decision">Decision: {zone.safetyDecision}</span>
      </div>
      {zone.activeWarnings.length > 0 && (
        <ul className="zone-card__warnings" role="list" aria-label="Active warnings">
          {zone.activeWarnings.map((warning, i) => (
            <li key={i} className="zone-card__warning-item">
              {'\u26A0\uFE0F'} {warning}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default StatusPanel;
