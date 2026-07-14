/**
 * UserView Page
 *
 * Read-only view for workers/users.
 * Shows: current safety status, video placeholders, warnings, action guide.
 *
 * CRITICAL: This view must NOT expose:
 * - ACK button
 * - Classify button
 * - I/O test
 * - Settings
 * - Safety mapping
 * - Work window controls
 *
 * The user sees large text, clear icons, and immediate action guidance.
 * UNKNOWN and STALE are always shown as danger/warning.
 */

import React, { useEffect, useState } from 'react';
import type { ZoneStatus, EquipmentStatus as EquipmentStatusType, CameraStatus } from '../types/api';
import type { LocalAdapter } from '../adapters/localAdapter';
import { StatusPanel } from '../components/StatusPanel';
import { EquipmentStatusPanel } from '../components/EquipmentStatus';
import { SafetyDecision } from '../types/api';

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface UserViewProps {
  adapter: LocalAdapter;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const UserView: React.FC<UserViewProps> = ({ adapter }) => {
  const [zones, setZones] = useState<ZoneStatus[]>([]);
  const [equipment, setEquipment] = useState<EquipmentStatusType[]>([]);
  const [cameras, setCameras] = useState<CameraStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      try {
        const [z, eq, cam] = await Promise.all([
          adapter.getZoneStatuses(),
          adapter.getEquipmentStatuses(),
          adapter.getCameraStatuses(),
        ]);
        if (!cancelled) {
          setZones(z);
          setEquipment(eq);
          setCameras(cam);
          setLoading(false);
        }
      } catch (err) {
        if (!cancelled) {
          setError('Failed to load safety status. Contact operator.');
          setLoading(false);
        }
      }
    }

    fetchData();
    const interval = setInterval(fetchData, 5000);

    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [adapter]);

  // Determine if there is an active critical warning
  const hasCritical = zones.some(
    (z) => z.safetyDecision === SafetyDecision.STOP_REQUEST_REQUIRED
  );

  return (
    <div className="user-view" role="main" aria-label="Safety Status - User View">
      {/* Global Warning Banner */}
      {hasCritical && (
        <div className="user-view__critical-banner" role="alert" aria-live="assertive">
          <span className="user-view__critical-icon" aria-hidden="true">{'\u{1F6A8}'}</span>
          <div className="user-view__critical-text">
            <strong>DANGER: Active safety stop request</strong>
            <p>Do not enter the marked zone. Wait for operator clearance.</p>
          </div>
        </div>
      )}

      {/* Zone Safety Status */}
      <StatusPanel zones={zones} loading={loading} error={error} />

      {/* Equipment Status */}
      <EquipmentStatusPanel equipment={equipment} loading={loading} />

      {/* Video Placeholders */}
      <section className="user-view__video" aria-label="Camera Feeds">
        <h2 className="user-view__section-title">Camera Feeds</h2>
        <div className="user-view__video-grid">
          {cameras.map((cam) => (
            <div
              key={cam.cameraId}
              className={`user-view__video-cell ${!cam.connected ? 'user-view__video-cell--offline' : ''}`}
              role="img"
              aria-label={`${cam.cameraName} ${cam.connected ? 'live' : 'offline'}`}
            >
              <div className="user-view__video-placeholder">
                {cam.connected ? (
                  <>
                    <span className="user-view__video-live-icon">{'\u{1F4F9}'}</span>
                    <span className="user-view__video-label">{cam.cameraName}</span>
                    <span className="user-view__video-status">LIVE - {cam.resolution} @ {cam.fps}fps</span>
                  </>
                ) : (
                  <>
                    <span className="user-view__video-offline-icon">{'\u{1F6AB}'}</span>
                    <span className="user-view__video-label">{cam.cameraName}</span>
                    <span className="user-view__video-status">OFFLINE - Contact operator</span>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Action Guide */}
      <section className="user-view__guide" aria-label="Action Guide">
        <h2 className="user-view__section-title">Action Guide</h2>
        <div className="user-view__guide-content">
          <div className="user-view__guide-item user-view__guide-item--danger">
            <span className="user-view__guide-icon">{'\u26D4'}</span>
            <div>
              <strong>Red Zone / Stop Request</strong>
              <p>Exit the zone immediately. Do not re-enter until operator confirms safe.</p>
            </div>
          </div>
          <div className="user-view__guide-item user-view__guide-item--warning">
            <span className="user-view__guide-icon">{'\u26A0\uFE0F'}</span>
            <div>
              <strong>Yellow / Unknown Status</strong>
              <p>Camera or sensor is unavailable. Treat as potentially occupied. Inform operator.</p>
            </div>
          </div>
          <div className="user-view__guide-item user-view__guide-item--safe">
            <span className="user-view__guide-icon">{'\u2705'}</span>
            <div>
              <strong>Green / Confirmed Vacant</strong>
              <p>Zone is confirmed safe for equipment operation.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Emergency Contact */}
      <section className="user-view__emergency" aria-label="Emergency Contact">
        <h2 className="user-view__section-title">Emergency Contact</h2>
        <div className="user-view__emergency-content">
          <p><strong>Safety Officer:</strong> Extension 119</p>
          <p><strong>Maintenance:</strong> Extension 120</p>
          <p><strong>External Emergency:</strong> 119 (Fire/Medical)</p>
        </div>
      </section>
    </div>
  );
};

export default UserView;
