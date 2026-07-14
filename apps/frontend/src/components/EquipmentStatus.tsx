/**
 * EquipmentStatus Component
 *
 * Displays equipment running states with appropriate visual indicators.
 *
 * SAFETY UX RULES:
 * - Use color + icon + text (not color alone)
 * - RESTART_REQUESTED with interlock active must show as blocked
 * - UNKNOWN state must show as warning (never safe)
 */

import React from 'react';
import { EquipmentState } from '../types/api';
import type { EquipmentStatus as EquipmentStatusType } from '../types/api';

// ---------------------------------------------------------------------------
// State Styling
// ---------------------------------------------------------------------------

interface StateStyle {
  icon: string;
  bgColor: string;
  textColor: string;
  borderColor: string;
  label: string;
}

function getEquipmentStyle(equipment: EquipmentStatusType): StateStyle {
  const { state, restartInterlockActive } = equipment;

  switch (state) {
    case EquipmentState.RUNNING:
      return {
        icon: '\u2699\uFE0F', // gear
        bgColor: '#dbeafe',
        textColor: '#1e40af',
        borderColor: '#3b82f6',
        label: 'RUNNING',
      };

    case EquipmentState.STOPPED:
      if (restartInterlockActive) {
        return {
          icon: '\u{1F512}', // lock
          bgColor: '#fef3c7',
          textColor: '#92400e',
          borderColor: '#f59e0b',
          label: 'STOPPED - Restart Blocked',
        };
      }
      return {
        icon: '\u23F9\uFE0F', // stop
        bgColor: '#f3f4f6',
        textColor: '#374151',
        borderColor: '#9ca3af',
        label: 'STOPPED',
      };

    case EquipmentState.RESTART_REQUESTED:
      if (restartInterlockActive) {
        return {
          icon: '\u{1F6AB}', // prohibited
          bgColor: '#fee2e2',
          textColor: '#991b1b',
          borderColor: '#ef4444',
          label: 'RESTART BLOCKED - Interlock Active',
        };
      }
      return {
        icon: '\u{1F504}', // counterclockwise arrows
        bgColor: '#fef3c7',
        textColor: '#92400e',
        borderColor: '#f59e0b',
        label: 'RESTART REQUESTED',
      };

    case EquipmentState.UNKNOWN:
      return {
        icon: '\u2753', // question mark
        bgColor: '#fef3c7',
        textColor: '#92400e',
        borderColor: '#f59e0b',
        label: 'UNKNOWN - State Unconfirmed',
      };
  }
}

// ---------------------------------------------------------------------------
// Component Props
// ---------------------------------------------------------------------------

interface EquipmentStatusProps {
  equipment: EquipmentStatusType[];
  loading?: boolean;
  error?: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const EquipmentStatusPanel: React.FC<EquipmentStatusProps> = ({
  equipment,
  loading,
  error,
}) => {
  if (loading) {
    return (
      <div className="equipment-panel equipment-panel--loading" role="status" aria-busy="true">
        Loading equipment status...
      </div>
    );
  }

  if (error) {
    return (
      <div className="equipment-panel equipment-panel--error" role="alert">
        <span>{'\u26A0\uFE0F'}</span> {error}
      </div>
    );
  }

  // Show interlocked/blocked equipment first
  const sorted = [...equipment].sort((a, b) => {
    if (a.restartInterlockActive && !b.restartInterlockActive) return -1;
    if (!a.restartInterlockActive && b.restartInterlockActive) return 1;
    return 0;
  });

  return (
    <div className="equipment-panel" role="region" aria-label="Equipment Status">
      <h2 className="equipment-panel__title">Equipment Status</h2>
      <div className="equipment-panel__grid">
        {sorted.map((eq) => (
          <EquipmentCard key={eq.equipmentId} equipment={eq} />
        ))}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Equipment Card Sub-component
// ---------------------------------------------------------------------------

interface EquipmentCardProps {
  equipment: EquipmentStatusType;
}

const EquipmentCard: React.FC<EquipmentCardProps> = ({ equipment }) => {
  const style = getEquipmentStyle(equipment);

  return (
    <div
      className="equipment-card"
      style={{
        backgroundColor: style.bgColor,
        borderLeft: `4px solid ${style.borderColor}`,
        color: style.textColor,
      }}
      role="article"
      aria-label={`${equipment.equipmentName}: ${style.label}`}
    >
      <div className="equipment-card__header">
        <span className="equipment-card__icon" aria-hidden="true">{style.icon}</span>
        <span className="equipment-card__name">{equipment.equipmentName}</span>
      </div>
      <div className="equipment-card__state">
        <span className="equipment-card__label">{style.label}</span>
      </div>
      <div className="equipment-card__meta">
        <span className="equipment-card__zone">Zone: {equipment.zoneId}</span>
        <span className="equipment-card__updated">
          Updated: {new Date(equipment.lastUpdated).toLocaleTimeString()}
        </span>
      </div>
      {equipment.restartInterlockActive && (
        <div className="equipment-card__interlock" role="alert">
          {'\u{1F512}'} Restart interlock active - vacancy not confirmed
        </div>
      )}
    </div>
  );
};

export default EquipmentStatusPanel;
