/**
 * OperatorView Page
 *
 * Event queue with ACK/resolve/classify, work window controls,
 * system status summary, and camera/I/O overview.
 *
 * OPERATOR permissions:
 * - Event ACK, resolve, classify
 * - Work window request/end
 * - System status summary (not full diagnostics)
 * - Reports
 *
 * OPERATOR restrictions (cannot do):
 * - Safety I/O mapping change
 * - Stop request logic change
 * - Hardware profile change
 * - Remote equipment control
 * - I/O test
 * - Backup/restore
 */

import React, { useEffect, useState, useCallback } from 'react';
import type {
  ZoneStatus,
  EquipmentStatus as EquipmentStatusType,
  SystemStatus,
  SafetyEventItem,
} from '../types/api';
import type { LocalAdapter, WorkWindowResponse } from '../adapters/localAdapter';
import { StatusPanel } from '../components/StatusPanel';
import { EventList } from '../components/EventList';
import { EquipmentStatusPanel } from '../components/EquipmentStatus';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface OperatorViewProps {
  adapter: LocalAdapter;
  username: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const OperatorView: React.FC<OperatorViewProps> = ({ adapter, username }) => {
  const [zones, setZones] = useState<ZoneStatus[]>([]);
  const [equipment, setEquipment] = useState<EquipmentStatusType[]>([]);
  const [events, setEvents] = useState<SafetyEventItem[]>([]);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [activeWorkWindow, setActiveWorkWindow] = useState<WorkWindowResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [workWindowZone, setWorkWindowZone] = useState('');
  const [workWindowReason, setWorkWindowReason] = useState('');
  const [workWindowDuration, setWorkWindowDuration] = useState(30);

  const fetchData = useCallback(async () => {
    try {
      const [z, eq, ev, sys] = await Promise.all([
        adapter.getZoneStatuses(),
        adapter.getEquipmentStatuses(),
        adapter.getRecentEvents(20),
        adapter.getSystemStatus(),
      ]);
      setZones(z);
      setEquipment(eq);
      setEvents(ev);
      setSystemStatus(sys);
      setLoading(false);
    } catch (_err) {
      setError('Failed to load data.');
      setLoading(false);
    }
  }, [adapter]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // Event handlers
  const handleAcknowledge = useCallback(
    async (eventId: string) => {
      await adapter.acknowledgeEvent({ eventId, acknowledgedBy: username });
      fetchData();
    },
    [adapter, username, fetchData]
  );

  const handleResolve = useCallback(
    async (eventId: string) => {
      await adapter.resolveEvent({
        eventId,
        resolvedBy: username,
        classification: 'resolved_by_operator',
      });
      fetchData();
    },
    [adapter, username, fetchData]
  );

  const handleRequestWorkWindow = useCallback(async () => {
    if (!workWindowZone || !workWindowReason) return;
    const response = await adapter.requestWorkWindow({
      zoneId: workWindowZone,
      requestedBy: username,
      reason: workWindowReason,
      durationMinutes: workWindowDuration,
    });
    setActiveWorkWindow(response);
  }, [adapter, username, workWindowZone, workWindowReason, workWindowDuration]);

  const handleEndWorkWindow = useCallback(async () => {
    if (!activeWorkWindow) return;
    await adapter.endWorkWindow(activeWorkWindow.windowId);
    setActiveWorkWindow(null);
  }, [adapter, activeWorkWindow]);

  return (
    <div className="operator-view" role="main" aria-label="Safety Operations - Operator View">
      {/* System Status Bar */}
      {systemStatus && (
        <div className="operator-view__status-bar" role="status">
          <span className="operator-view__status-item">
            Gateway: {systemStatus.gatewayOnline ? '\u2705 Online' : '\u274C Offline'}
          </span>
          <span className="operator-view__status-item">
            CPU: {systemStatus.cpuPercent}%
          </span>
          <span className="operator-view__status-item">
            RAM: {systemStatus.memoryPercent}%
          </span>
          <span className="operator-view__status-item">
            SSD: {systemStatus.ssdUsedPercent}%
          </span>
          <span className="operator-view__status-item">
            AWS: {systemStatus.awsConnected ? '\u2705' : '\u274C'}
          </span>
          <span className="operator-view__status-item">
            Alerts: {systemStatus.activeAlerts}
          </span>
        </div>
      )}

      {/* Zone Status */}
      <StatusPanel zones={zones} loading={loading} error={error} />

      {/* Event Queue - with ACK/Resolve buttons for OPERATOR */}
      <EventList
        events={events}
        role="OPERATOR"
        onAcknowledge={handleAcknowledge}
        onResolve={handleResolve}
        loading={loading}
      />

      {/* Equipment Status */}
      <EquipmentStatusPanel equipment={equipment} loading={loading} />

      {/* Work Window Controls */}
      <section className="operator-view__work-window" aria-label="Work Window Management">
        <h2 className="operator-view__section-title">Work Window Management</h2>

        {activeWorkWindow ? (
          <div className="operator-view__work-window-active" role="status">
            <h3>{'\u{1F6E0}\uFE0F'} Active Work Window</h3>
            <p>Zone: {activeWorkWindow.zoneId}</p>
            <p>Started: {new Date(activeWorkWindow.startedAt).toLocaleString()}</p>
            <p>Ends: {new Date(activeWorkWindow.endsAt).toLocaleString()}</p>
            <button
              className="operator-view__btn operator-view__btn--end"
              onClick={handleEndWorkWindow}
              aria-label="End active work window"
            >
              End Work Window
            </button>
          </div>
        ) : (
          <div className="operator-view__work-window-form">
            <div className="operator-view__form-group">
              <label htmlFor="ww-zone">Zone:</label>
              <select
                id="ww-zone"
                value={workWindowZone}
                onChange={(e) => setWorkWindowZone(e.target.value)}
              >
                <option value="">Select zone...</option>
                {zones.map((z) => (
                  <option key={z.zoneId} value={z.zoneId}>
                    {z.zoneName}
                  </option>
                ))}
              </select>
            </div>
            <div className="operator-view__form-group">
              <label htmlFor="ww-reason">Reason:</label>
              <input
                id="ww-reason"
                type="text"
                value={workWindowReason}
                onChange={(e) => setWorkWindowReason(e.target.value)}
                placeholder="Maintenance reason..."
              />
            </div>
            <div className="operator-view__form-group">
              <label htmlFor="ww-duration">Duration (minutes):</label>
              <input
                id="ww-duration"
                type="number"
                value={workWindowDuration}
                onChange={(e) => setWorkWindowDuration(Number(e.target.value))}
                min={5}
                max={480}
              />
            </div>
            <button
              className="operator-view__btn operator-view__btn--request"
              onClick={handleRequestWorkWindow}
              disabled={!workWindowZone || !workWindowReason}
              aria-label="Request work window"
            >
              Request Work Window
            </button>
          </div>
        )}
      </section>
    </div>
  );
};

export default OperatorView;
