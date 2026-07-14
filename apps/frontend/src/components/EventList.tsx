/**
 * EventList Component
 *
 * Displays recent safety events with severity indicators.
 * Shows event queue for operators/maintainers and read-only for users.
 *
 * SAFETY UX RULES:
 * - Critical events always visible
 * - Use color + icon + text (not color alone)
 * - ACK buttons only shown for OPERATOR/MAINTAINER roles
 */

import React from 'react';
import { AckStatus } from '../types/api';
import type { SafetyEventItem } from '../types/api';
import type { Role } from '../types/roles';
import { canAcknowledge } from '../types/roles';

// ---------------------------------------------------------------------------
// Severity Styling
// ---------------------------------------------------------------------------

interface SeverityStyle {
  icon: string;
  bgColor: string;
  textColor: string;
  borderColor: string;
}

function getSeverityStyle(severity: SafetyEventItem['severity']): SeverityStyle {
  switch (severity) {
    case 'critical':
      return {
        icon: '\u{1F6A8}', // rotating light
        bgColor: '#fee2e2',
        textColor: '#991b1b',
        borderColor: '#ef4444',
      };
    case 'warning':
      return {
        icon: '\u26A0\uFE0F', // warning sign
        bgColor: '#fef3c7',
        textColor: '#92400e',
        borderColor: '#f59e0b',
      };
    case 'info':
      return {
        icon: '\u2139\uFE0F', // info
        bgColor: '#dbeafe',
        textColor: '#1e40af',
        borderColor: '#3b82f6',
      };
  }
}

function getAckStatusLabel(status: AckStatus): { icon: string; label: string } {
  switch (status) {
    case AckStatus.PENDING:
      return { icon: '\u{1F534}', label: 'Pending' }; // red circle
    case AckStatus.ACKNOWLEDGED:
      return { icon: '\u{1F7E1}', label: 'Acknowledged' }; // yellow circle
    case AckStatus.RESOLVED:
      return { icon: '\u{1F7E2}', label: 'Resolved' }; // green circle
  }
}

// ---------------------------------------------------------------------------
// Component Props
// ---------------------------------------------------------------------------

interface EventListProps {
  events: SafetyEventItem[];
  role: Role;
  onAcknowledge?: (eventId: string) => void;
  onResolve?: (eventId: string) => void;
  loading?: boolean;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const EventList: React.FC<EventListProps> = ({
  events,
  role,
  onAcknowledge,
  onResolve,
  loading,
}) => {
  if (loading) {
    return (
      <div className="event-list event-list--loading" role="status" aria-busy="true">
        Loading events...
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <div className="event-list event-list--empty" role="status">
        <span className="event-list__empty-icon">{'\u2705'}</span>
        <span className="event-list__empty-text">No recent safety events</span>
      </div>
    );
  }

  // Sort events: pending first, then by severity, then by timestamp
  const sortedEvents = [...events].sort((a, b) => {
    // Pending events first
    if (a.ackStatus === AckStatus.PENDING && b.ackStatus !== AckStatus.PENDING) return -1;
    if (b.ackStatus === AckStatus.PENDING && a.ackStatus !== AckStatus.PENDING) return 1;
    // Then by severity
    const severityOrder = { critical: 0, warning: 1, info: 2 };
    if (severityOrder[a.severity] !== severityOrder[b.severity]) {
      return severityOrder[a.severity] - severityOrder[b.severity];
    }
    // Then by timestamp (newest first)
    return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
  });

  return (
    <div className="event-list" role="region" aria-label="Safety Events">
      <h2 className="event-list__title">
        Safety Events
        {events.filter((e) => e.ackStatus === AckStatus.PENDING).length > 0 && (
          <span className="event-list__pending-badge">
            {events.filter((e) => e.ackStatus === AckStatus.PENDING).length} pending
          </span>
        )}
      </h2>
      <ul className="event-list__items" role="list">
        {sortedEvents.map((event) => (
          <EventListItem
            key={event.eventId}
            event={event}
            role={role}
            onAcknowledge={onAcknowledge}
            onResolve={onResolve}
          />
        ))}
      </ul>
    </div>
  );
};

// ---------------------------------------------------------------------------
// Event Item Sub-component
// ---------------------------------------------------------------------------

interface EventListItemProps {
  event: SafetyEventItem;
  role: Role;
  onAcknowledge?: (eventId: string) => void;
  onResolve?: (eventId: string) => void;
}

const EventListItem: React.FC<EventListItemProps> = ({
  event,
  role,
  onAcknowledge,
  onResolve,
}) => {
  const severityStyle = getSeverityStyle(event.severity);
  const ackLabel = getAckStatusLabel(event.ackStatus);

  const showAckButton =
    canAcknowledge(role) &&
    event.ackStatus === AckStatus.PENDING &&
    onAcknowledge;

  const showResolveButton =
    canAcknowledge(role) &&
    event.ackStatus === AckStatus.ACKNOWLEDGED &&
    onResolve;

  return (
    <li
      className="event-item"
      style={{
        backgroundColor: severityStyle.bgColor,
        borderLeft: `4px solid ${severityStyle.borderColor}`,
        color: severityStyle.textColor,
      }}
      role="article"
      aria-label={`${event.severity} event: ${event.summary}`}
    >
      <div className="event-item__header">
        <span className="event-item__severity-icon" aria-hidden="true">
          {severityStyle.icon}
        </span>
        <span className="event-item__summary">{event.summary}</span>
        <span className="event-item__status">
          {ackLabel.icon} {ackLabel.label}
        </span>
      </div>
      <div className="event-item__meta">
        <span className="event-item__zone">{event.zoneName}</span>
        <span className="event-item__time">
          {new Date(event.timestamp).toLocaleString()}
        </span>
      </div>
      <p className="event-item__detail">{event.detail}</p>
      {event.classification && (
        <span className="event-item__classification">
          Classification: {event.classification}
        </span>
      )}

      {/* ACK/Resolve buttons: NEVER shown for USER role */}
      {(showAckButton || showResolveButton) && (
        <div className="event-item__actions">
          {showAckButton && (
            <button
              className="event-item__btn event-item__btn--ack"
              onClick={() => onAcknowledge(event.eventId)}
              aria-label={`Acknowledge event: ${event.summary}`}
            >
              Acknowledge
            </button>
          )}
          {showResolveButton && (
            <button
              className="event-item__btn event-item__btn--resolve"
              onClick={() => onResolve(event.eventId)}
              aria-label={`Resolve event: ${event.summary}`}
            >
              Resolve
            </button>
          )}
        </div>
      )}
    </li>
  );
};

export default EventList;
