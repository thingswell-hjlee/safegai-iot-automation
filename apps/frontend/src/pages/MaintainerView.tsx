/**
 * MaintainerView Page
 *
 * Diagnostics, camera config, I/O test, backup controls.
 * Only accessible in MAINTAINER role via Local Maintenance Network.
 *
 * MAINTAINER permissions:
 * - All OPERATOR permissions plus:
 * - Camera discovery/register/test
 * - Zone mapping
 * - DI/DO mapping (with approval)
 * - Output test in TEST state
 * - Full logs and diagnostics
 * - Hardware detail
 * - Backup/restore
 * - Update/rollback
 *
 * SAFETY RULES:
 * - I/O test requires TEST state entry
 * - TEST state requires operator confirmation
 * - Safety I/O mapping changes require signed config and T2 approval
 * - Active critical warnings always visible
 */

import React, { useEffect, useState, useCallback } from 'react';
import type {
  ZoneStatus,
  CameraStatus,
  SystemStatus,
  SafetyEventItem,
} from '../types/api';
import type {
  LocalAdapter,
  DiagnosticsBundle,
  IOTestResult,
} from '../adapters/localAdapter';
import { StatusPanel } from '../components/StatusPanel';
import { EventList } from '../components/EventList';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface MaintainerViewProps {
  adapter: LocalAdapter;
  username: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const MaintainerView: React.FC<MaintainerViewProps> = ({ adapter, username }) => {
  const [zones, setZones] = useState<ZoneStatus[]>([]);
  const [cameras, setCameras] = useState<CameraStatus[]>([]);
  const [events, setEvents] = useState<SafetyEventItem[]>([]);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsBundle | null>(null);
  const [ioTestResult, setIoTestResult] = useState<IOTestResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [ioChannel, setIoChannel] = useState(1);
  const [ioDuration, setIoDuration] = useState(500);
  const [ioTestConfirmed, setIoTestConfirmed] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [z, cam, ev, sys, diag] = await Promise.all([
        adapter.getZoneStatuses(),
        adapter.getCameraStatuses(),
        adapter.getRecentEvents(20),
        adapter.getSystemStatus(),
        adapter.getDiagnostics(),
      ]);
      setZones(z);
      setCameras(cam);
      setEvents(ev);
      setSystemStatus(sys);
      setDiagnostics(diag);
      setLoading(false);
    } catch (_err) {
      setError('Failed to load diagnostics data.');
      setLoading(false);
    }
  }, [adapter]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, [fetchData]);

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
        classification: 'resolved_by_maintainer',
      });
      fetchData();
    },
    [adapter, username, fetchData]
  );

  const handleIOTest = useCallback(async () => {
    if (!ioTestConfirmed) return;
    const result = await adapter.testIO({
      outputChannel: ioChannel,
      durationMs: ioDuration,
      confirmedBy: username,
    });
    setIoTestResult(result);
  }, [adapter, ioChannel, ioDuration, ioTestConfirmed, username]);

  const handleExportBackup = useCallback(async () => {
    const blob = await adapter.exportBackup();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `safegai-backup-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [adapter]);

  return (
    <div className="maintainer-view" role="main" aria-label="Maintenance - Maintainer View">
      {/* Critical Warnings - always visible */}
      <StatusPanel zones={zones} loading={loading} error={error} />

      {/* Events with full ACK/resolve */}
      <EventList
        events={events}
        role="MAINTAINER"
        onAcknowledge={handleAcknowledge}
        onResolve={handleResolve}
        loading={loading}
      />

      {/* System Diagnostics */}
      {diagnostics && (
        <section className="maintainer-view__diagnostics" aria-label="System Diagnostics">
          <h2 className="maintainer-view__section-title">System Diagnostics</h2>
          <div className="maintainer-view__diag-grid">
            <div className="maintainer-view__diag-item">
              <span className="maintainer-view__diag-label">CPU</span>
              <span className="maintainer-view__diag-value">{diagnostics.cpuPercent}%</span>
            </div>
            <div className="maintainer-view__diag-item">
              <span className="maintainer-view__diag-label">Memory</span>
              <span className="maintainer-view__diag-value">{diagnostics.memoryPercent}%</span>
            </div>
            <div className="maintainer-view__diag-item">
              <span className="maintainer-view__diag-label">SSD Used</span>
              <span className="maintainer-view__diag-value">{diagnostics.ssdUsedPercent}%</span>
            </div>
            <div className="maintainer-view__diag-item">
              <span className="maintainer-view__diag-label">SSD Health</span>
              <span className="maintainer-view__diag-value">
                {diagnostics.ssdSmartHealthOk ? '\u2705 OK' : '\u274C FAILING'}
              </span>
            </div>
            <div className="maintainer-view__diag-item">
              <span className="maintainer-view__diag-label">Temperature</span>
              <span className="maintainer-view__diag-value">{diagnostics.temperature}C</span>
            </div>
          </div>

          {/* NIC Status */}
          <h3>Network Interfaces</h3>
          <ul className="maintainer-view__nic-list">
            {diagnostics.nicStatus.map((nic) => (
              <li key={nic.name}>
                {nic.up ? '\u2705' : '\u274C'} {nic.name} - {nic.speed} {nic.up ? 'UP' : 'DOWN'}
              </li>
            ))}
          </ul>

          {/* Services */}
          <h3>Services</h3>
          <ul className="maintainer-view__service-list">
            {diagnostics.serviceStatuses.map((svc) => (
              <li key={svc.name}>
                {svc.active ? '\u2705' : '\u274C'} {svc.name} - {svc.uptime}
              </li>
            ))}
          </ul>

          {/* Recent Errors */}
          {diagnostics.recentErrors.length > 0 && (
            <>
              <h3>Recent Errors</h3>
              <ul className="maintainer-view__error-list">
                {diagnostics.recentErrors.map((err, i) => (
                  <li key={i} className="maintainer-view__error-item">{err}</li>
                ))}
              </ul>
            </>
          )}
        </section>
      )}

      {/* Camera Status */}
      <section className="maintainer-view__cameras" aria-label="Camera Configuration">
        <h2 className="maintainer-view__section-title">Camera Configuration</h2>
        <div className="maintainer-view__camera-grid">
          {cameras.map((cam) => (
            <div
              key={cam.cameraId}
              className={`maintainer-view__camera-card ${!cam.connected ? 'maintainer-view__camera-card--offline' : ''}`}
            >
              <div className="maintainer-view__camera-header">
                <span>{cam.connected ? '\u{1F7E2}' : '\u{1F534}'}</span>
                <span>{cam.cameraName}</span>
              </div>
              <div className="maintainer-view__camera-details">
                <p>ID: {cam.cameraId}</p>
                <p>Resolution: {cam.resolution}</p>
                <p>FPS: {cam.fps}</p>
                <p>Zones: {cam.zoneIds.join(', ')}</p>
                <p>Stream: {cam.streamUrl}</p>
                <p>Last Frame: {new Date(cam.lastFrameAt).toLocaleString()}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* I/O Test */}
      <section className="maintainer-view__io-test" aria-label="I/O Output Test">
        <h2 className="maintainer-view__section-title">I/O Output Test</h2>
        <div className="maintainer-view__io-warning" role="alert">
          <span>{'\u26A0\uFE0F'}</span>
          <p>
            <strong>WARNING:</strong> I/O test requires TEST state entry.
            Ensure production equipment is physically separated or approved
            test relay is in use. Real stop request output requires separate
            2-step confirmation.
          </p>
        </div>
        <div className="maintainer-view__io-form">
          <div className="maintainer-view__form-group">
            <label htmlFor="io-channel">Output Channel (1-8):</label>
            <input
              id="io-channel"
              type="number"
              value={ioChannel}
              onChange={(e) => setIoChannel(Number(e.target.value))}
              min={1}
              max={8}
            />
          </div>
          <div className="maintainer-view__form-group">
            <label htmlFor="io-duration">Duration (ms):</label>
            <input
              id="io-duration"
              type="number"
              value={ioDuration}
              onChange={(e) => setIoDuration(Number(e.target.value))}
              min={100}
              max={5000}
            />
          </div>
          <div className="maintainer-view__form-group">
            <label htmlFor="io-confirm">
              <input
                id="io-confirm"
                type="checkbox"
                checked={ioTestConfirmed}
                onChange={(e) => setIoTestConfirmed(e.target.checked)}
              />
              {' '}I confirm this system is in TEST state and production equipment is isolated.
            </label>
          </div>
          <button
            className="maintainer-view__btn maintainer-view__btn--test"
            onClick={handleIOTest}
            disabled={!ioTestConfirmed}
            aria-label="Execute I/O test"
          >
            Execute I/O Test
          </button>
          {ioTestResult && (
            <div className="maintainer-view__io-result" role="status">
              <p>
                Channel {ioTestResult.channel}:{' '}
                {ioTestResult.success ? '\u2705 Success' : '\u274C Failed'}
              </p>
              <p>Duration: {ioTestResult.measuredDurationMs.toFixed(1)}ms</p>
              {ioTestResult.error && <p>Error: {ioTestResult.error}</p>}
            </div>
          )}
        </div>
      </section>

      {/* Backup Controls */}
      <section className="maintainer-view__backup" aria-label="Backup and Restore">
        <h2 className="maintainer-view__section-title">Backup and Restore</h2>
        <div className="maintainer-view__backup-controls">
          <button
            className="maintainer-view__btn maintainer-view__btn--backup"
            onClick={handleExportBackup}
            aria-label="Export configuration backup"
          >
            Export Backup
          </button>
          <div className="maintainer-view__restore">
            <label htmlFor="restore-file">Restore from backup:</label>
            <input
              id="restore-file"
              type="file"
              accept=".json"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (file) {
                  await adapter.restoreBackup(file);
                }
              }}
            />
          </div>
        </div>
      </section>

      {/* System Status */}
      {systemStatus && (
        <section className="maintainer-view__system" aria-label="System Status">
          <h2 className="maintainer-view__section-title">System Status</h2>
          <div className="maintainer-view__system-details">
            <p>Gateway ID: {systemStatus.gatewayId}</p>
            <p>Online: {systemStatus.gatewayOnline ? 'Yes' : 'No'}</p>
            <p>Uptime: {systemStatus.uptime}</p>
            <p>AWS Connected: {systemStatus.awsConnected ? 'Yes' : 'No'}</p>
            <p>Last AWS Sync: {new Date(systemStatus.lastAwsSyncAt).toLocaleString()}</p>
            <p>Pending Events: {systemStatus.pendingEvents}</p>
          </div>
        </section>
      )}
    </div>
  );
};

export default MaintainerView;
