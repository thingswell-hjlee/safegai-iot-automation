/**
 * Mock Adapter
 *
 * Implements LocalAdapter with simulated data for development and testing.
 * Provides realistic zone states, equipment status, and safety events
 * matching the contract schemas.
 *
 * SAFETY: Mock data includes UNKNOWN/STALE scenarios to verify
 * that the UI never displays them as safe or vacant.
 */

import type { LocalAdapter } from './localAdapter';
import type {
  LoginRequest,
  LoginResponse,
  AckEventRequest,
  ResolveEventRequest,
  WorkWindowRequest,
  WorkWindowResponse,
  DiagnosticsBundle,
  IOTestRequest,
  IOTestResult,
} from './localAdapter';
import {
  OccupancyState,
  EquipmentState,
  SafetyDecision,
  DataQuality,
  AckStatus,
} from '../types/api';
import type {
  ZoneStatus,
  EquipmentStatus,
  CameraStatus,
  SystemStatus,
  SafetyEventItem,
  SafetyDecisionEvent,
} from '../types/api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function uuid(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

function isoNow(): string {
  return new Date().toISOString();
}

function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

// ---------------------------------------------------------------------------
// Mock Data
// ---------------------------------------------------------------------------

const MOCK_ZONES: ZoneStatus[] = [
  {
    zoneId: 'zone-01',
    zoneName: 'Press Machine Area',
    occupancy: OccupancyState.OCCUPIED,
    safetyDecision: SafetyDecision.STOP_REQUEST_REQUIRED,
    lastUpdated: minutesAgo(0),
    cameraIds: ['cam-01'],
    activeWarnings: ['R-01: Person detected in running equipment zone'],
  },
  {
    zoneId: 'zone-02',
    zoneName: 'Assembly Line North',
    occupancy: OccupancyState.VACANT_CONFIRMED,
    safetyDecision: SafetyDecision.SAFE,
    lastUpdated: minutesAgo(1),
    cameraIds: ['cam-02'],
    activeWarnings: [],
  },
  {
    zoneId: 'zone-03',
    zoneName: 'Conveyor Belt Area',
    occupancy: OccupancyState.UNKNOWN,
    safetyDecision: SafetyDecision.SAFETY_CONFIRMATION_UNAVAILABLE,
    lastUpdated: minutesAgo(5),
    cameraIds: ['cam-03'],
    activeWarnings: ['R-03: Camera safety confirmation unavailable'],
  },
  {
    zoneId: 'zone-04',
    zoneName: 'Loading Dock',
    occupancy: OccupancyState.STALE,
    safetyDecision: SafetyDecision.SAFETY_CONFIRMATION_UNAVAILABLE,
    lastUpdated: minutesAgo(12),
    cameraIds: ['cam-04'],
    activeWarnings: ['R-03: Stale occupancy data - safety confirmation unavailable'],
  },
  {
    zoneId: 'zone-05',
    zoneName: 'Welding Station',
    occupancy: OccupancyState.VACANT_PENDING,
    safetyDecision: SafetyDecision.WARNING,
    lastUpdated: minutesAgo(0),
    cameraIds: ['cam-01'],
    activeWarnings: ['Vacancy confirmation in progress - not yet confirmed'],
  },
];

const MOCK_EQUIPMENT: EquipmentStatus[] = [
  {
    equipmentId: 'eq-press-01',
    equipmentName: 'Hydraulic Press #1',
    state: EquipmentState.RUNNING,
    lastUpdated: minutesAgo(0),
    zoneId: 'zone-01',
    restartInterlockActive: false,
  },
  {
    equipmentId: 'eq-conv-01',
    equipmentName: 'Conveyor Belt Main',
    state: EquipmentState.STOPPED,
    lastUpdated: minutesAgo(3),
    zoneId: 'zone-03',
    restartInterlockActive: true,
  },
  {
    equipmentId: 'eq-weld-01',
    equipmentName: 'Welding Robot Arm',
    state: EquipmentState.RESTART_REQUESTED,
    lastUpdated: minutesAgo(1),
    zoneId: 'zone-05',
    restartInterlockActive: true,
  },
  {
    equipmentId: 'eq-asm-01',
    equipmentName: 'Assembly Robot #2',
    state: EquipmentState.RUNNING,
    lastUpdated: minutesAgo(0),
    zoneId: 'zone-02',
    restartInterlockActive: false,
  },
];

const MOCK_CAMERAS: CameraStatus[] = [
  {
    cameraId: 'cam-01',
    cameraName: 'Fisheye Main (Press/Weld)',
    connected: true,
    streamUrl: 'rtsp://gateway:8554/cam-01/substream',
    lastFrameAt: minutesAgo(0),
    zoneIds: ['zone-01', 'zone-05'],
    resolution: '2048x2048',
    fps: 15,
  },
  {
    cameraId: 'cam-02',
    cameraName: 'Assembly North PTZ',
    connected: true,
    streamUrl: 'rtsp://gateway:8554/cam-02/substream',
    lastFrameAt: minutesAgo(0),
    zoneIds: ['zone-02'],
    resolution: '1920x1080',
    fps: 25,
  },
  {
    cameraId: 'cam-03',
    cameraName: 'Conveyor Overview',
    connected: false,
    streamUrl: 'rtsp://gateway:8554/cam-03/substream',
    lastFrameAt: minutesAgo(5),
    zoneIds: ['zone-03'],
    resolution: '1920x1080',
    fps: 0,
  },
  {
    cameraId: 'cam-04',
    cameraName: 'Loading Dock Wide',
    connected: false,
    streamUrl: 'rtsp://gateway:8554/cam-04/substream',
    lastFrameAt: minutesAgo(12),
    zoneIds: ['zone-04'],
    resolution: '1280x720',
    fps: 0,
  },
];

const MOCK_SYSTEM_STATUS: SystemStatus = {
  gatewayId: 'gw-factory-01',
  gatewayOnline: true,
  cpuPercent: 42,
  memoryPercent: 58,
  ssdUsedPercent: 23,
  ssdHealthOk: true,
  awsConnected: true,
  lastAwsSyncAt: minutesAgo(2),
  uptime: '14d 6h 32m',
  activeAlerts: 3,
  pendingEvents: 2,
};

const MOCK_EVENTS: SafetyEventItem[] = [
  {
    eventId: uuid(),
    timestamp: minutesAgo(0),
    zoneId: 'zone-01',
    zoneName: 'Press Machine Area',
    type: 'safety_decision',
    severity: 'critical',
    summary: 'STOP REQUEST: Person in running equipment zone',
    detail: 'Rule R-01 triggered. Person detected in zone-01 while Hydraulic Press #1 is RUNNING. Stop request sent to Safety Relay.',
    ackStatus: AckStatus.PENDING,
  },
  {
    eventId: uuid(),
    timestamp: minutesAgo(1),
    zoneId: 'zone-05',
    zoneName: 'Welding Station',
    type: 'equipment',
    severity: 'warning',
    summary: 'Restart interlock active - vacancy not confirmed',
    detail: 'Rule R-02 triggered. Restart requested for Welding Robot Arm but zone-05 is VACANT_PENDING (not VACANT_CONFIRMED). Restart blocked.',
    ackStatus: AckStatus.PENDING,
  },
  {
    eventId: uuid(),
    timestamp: minutesAgo(5),
    zoneId: 'zone-03',
    zoneName: 'Conveyor Belt Area',
    type: 'occupancy',
    severity: 'warning',
    summary: 'Camera offline - safety confirmation unavailable',
    detail: 'Rule R-03 triggered. Camera cam-03 is disconnected. Zone-03 occupancy state is UNKNOWN. Restart not permitted.',
    ackStatus: AckStatus.ACKNOWLEDGED,
    acknowledgedBy: 'operator-kim',
    acknowledgedAt: minutesAgo(3),
  },
  {
    eventId: uuid(),
    timestamp: minutesAgo(12),
    zoneId: 'zone-04',
    zoneName: 'Loading Dock',
    type: 'occupancy',
    severity: 'warning',
    summary: 'Stale occupancy data - camera timeout',
    detail: 'Rule R-03 triggered. Camera cam-04 last frame 12 minutes ago. Zone-04 state is STALE. Safety confirmation unavailable.',
    ackStatus: AckStatus.RESOLVED,
    acknowledgedBy: 'operator-park',
    acknowledgedAt: minutesAgo(10),
    resolvedBy: 'operator-park',
    resolvedAt: minutesAgo(8),
    classification: 'network_issue',
  },
  {
    eventId: uuid(),
    timestamp: minutesAgo(30),
    zoneId: 'zone-02',
    zoneName: 'Assembly Line North',
    type: 'safety_decision',
    severity: 'info',
    summary: 'Zone confirmed vacant - operations normal',
    detail: 'Zone-02 vacancy confirmed after 3 consecutive checks. Assembly Robot #2 operating normally.',
    ackStatus: AckStatus.RESOLVED,
    acknowledgedBy: 'system',
    acknowledgedAt: minutesAgo(30),
    resolvedBy: 'system',
    resolvedAt: minutesAgo(30),
    classification: 'normal_operation',
  },
];

// ---------------------------------------------------------------------------
// Mock Adapter Implementation
// ---------------------------------------------------------------------------

export class MockAdapter implements LocalAdapter {
  private events: SafetyEventItem[] = [...MOCK_EVENTS];
  private listeners: Array<(event: SafetyDecisionEvent) => void> = [];

  async login(request: LoginRequest): Promise<LoginResponse> {
    // Simulate role assignment based on username prefix
    const roleMap: Record<string, LoginResponse['role']> = {
      user: 'USER',
      operator: 'OPERATOR',
      maintainer: 'MAINTAINER',
      admin: 'MAINTAINER',
    };

    const prefix = request.username.split('-')[0] || request.username;
    const role = roleMap[prefix] || 'USER';

    if (request.password.length < 4) {
      return {
        success: false,
        role: 'USER',
        token: '',
        expiresAt: '',
        error: 'Invalid credentials',
      };
    }

    return {
      success: true,
      role,
      token: `mock-token-${uuid()}`,
      expiresAt: new Date(Date.now() + 3_600_000).toISOString(),
    };
  }

  async logout(): Promise<void> {
    // No-op for mock
  }

  async getZoneStatuses(): Promise<ZoneStatus[]> {
    return MOCK_ZONES.map((z) => ({ ...z, lastUpdated: isoNow() }));
  }

  async getEquipmentStatuses(): Promise<EquipmentStatus[]> {
    return MOCK_EQUIPMENT.map((e) => ({ ...e, lastUpdated: isoNow() }));
  }

  async getCameraStatuses(): Promise<CameraStatus[]> {
    return [...MOCK_CAMERAS];
  }

  async getSystemStatus(): Promise<SystemStatus> {
    return { ...MOCK_SYSTEM_STATUS, lastAwsSyncAt: isoNow() };
  }

  async getRecentEvents(limit: number): Promise<SafetyEventItem[]> {
    return this.events.slice(0, limit);
  }

  async getActiveSafetyDecisions(): Promise<SafetyDecisionEvent[]> {
    // Return mock active safety decisions for critical zones
    return [
      {
        schemaVersion: '1.0.0',
        eventId: uuid(),
        correlationId: uuid(),
        tenantId: 'tenant-01',
        siteId: 'site-factory-01',
        gatewayId: 'gw-factory-01',
        deviceId: 'cam-01',
        zoneId: 'zone-01',
        observedAt: isoNow(),
        receivedAt: isoNow(),
        sequenceNo: 1042,
        source: 'zone-state-engine',
        quality: DataQuality.GOOD,
        decision: SafetyDecision.STOP_REQUEST_REQUIRED,
        ruleId: 'R-01',
        occupancyState: OccupancyState.OCCUPIED,
        equipmentState: EquipmentState.RUNNING,
        reason: 'Person detected in running equipment zone',
        actions: ['stop_request', 'red_lamp', 'voice_warning', 'event_log'],
      },
    ];
  }

  async acknowledgeEvent(request: AckEventRequest): Promise<void> {
    const event = this.events.find((e) => e.eventId === request.eventId);
    if (event) {
      event.ackStatus = AckStatus.ACKNOWLEDGED;
      event.acknowledgedBy = request.acknowledgedBy;
      event.acknowledgedAt = isoNow();
    }
  }

  async resolveEvent(request: ResolveEventRequest): Promise<void> {
    const event = this.events.find((e) => e.eventId === request.eventId);
    if (event) {
      event.ackStatus = AckStatus.RESOLVED;
      event.resolvedBy = request.resolvedBy;
      event.resolvedAt = isoNow();
      event.classification = request.classification;
    }
  }

  async requestWorkWindow(request: WorkWindowRequest): Promise<WorkWindowResponse> {
    return {
      windowId: uuid(),
      zoneId: request.zoneId,
      startedAt: isoNow(),
      endsAt: new Date(Date.now() + request.durationMinutes * 60_000).toISOString(),
      active: true,
    };
  }

  async endWorkWindow(_windowId: string): Promise<void> {
    // No-op for mock
  }

  async getDiagnostics(): Promise<DiagnosticsBundle> {
    return {
      gatewayId: 'gw-factory-01',
      collectedAt: isoNow(),
      cpuPercent: 42,
      memoryPercent: 58,
      ssdUsedPercent: 23,
      ssdSmartHealthOk: true,
      temperature: 52,
      nicStatus: [
        { name: 'eth0', up: true, speed: '1Gbps' },
        { name: 'eth1', up: true, speed: '1Gbps' },
      ],
      serviceStatuses: [
        { name: 'safegai-gateway', active: true, uptime: '14d 6h 32m' },
        { name: 'mediamtx', active: true, uptime: '14d 6h 30m' },
        { name: 'safegai-frontend', active: true, uptime: '14d 6h 31m' },
      ],
      recentErrors: [
        '[2024-01-15 08:32:12] Camera cam-03 connection timeout after 30s',
        '[2024-01-15 08:32:42] Camera cam-04 stale frame detected (>10s)',
      ],
    };
  }

  async testIO(request: IOTestRequest): Promise<IOTestResult> {
    return {
      channel: request.outputChannel,
      success: true,
      measuredDurationMs: request.durationMs + Math.random() * 5,
    };
  }

  async exportBackup(): Promise<Blob> {
    const data = JSON.stringify({
      exportedAt: isoNow(),
      gatewayId: 'gw-factory-01',
      configVersion: '1.0.0',
    });
    return new Blob([data], { type: 'application/json' });
  }

  async restoreBackup(_file: File): Promise<{ success: boolean; error?: string }> {
    return { success: true };
  }

  subscribe(onEvent: (event: SafetyDecisionEvent) => void): () => void {
    this.listeners.push(onEvent);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== onEvent);
    };
  }
}

/**
 * Singleton instance of the mock adapter for use throughout the app.
 */
export const mockAdapter = new MockAdapter();
