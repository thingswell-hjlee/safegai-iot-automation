/**
 * Local Adapter Interface
 *
 * Defines all Gateway API calls for the SafeGAI Hybrid App.
 * Implementations:
 * - mockAdapter.ts: returns simulated data for development and testing
 * - (future) httpAdapter.ts: connects to real Gateway REST/WebSocket API
 *
 * All methods return Promises to support async API patterns.
 */

import type {
  ZoneStatus,
  EquipmentStatus,
  CameraStatus,
  SystemStatus,
  SafetyEventItem,
  SafetyDecisionEvent,
} from '../types/api';
import type { Role } from '../types/roles';

// ---------------------------------------------------------------------------
// Request/Response Types
// ---------------------------------------------------------------------------

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  role: Role;
  token: string;
  expiresAt: string;
  error?: string;
}

export interface AckEventRequest {
  eventId: string;
  acknowledgedBy: string;
  note?: string;
}

export interface ResolveEventRequest {
  eventId: string;
  resolvedBy: string;
  classification: string;
  note?: string;
}

export interface WorkWindowRequest {
  zoneId: string;
  requestedBy: string;
  reason: string;
  durationMinutes: number;
}

export interface WorkWindowResponse {
  windowId: string;
  zoneId: string;
  startedAt: string;
  endsAt: string;
  active: boolean;
}

export interface DiagnosticsBundle {
  gatewayId: string;
  collectedAt: string;
  cpuPercent: number;
  memoryPercent: number;
  ssdUsedPercent: number;
  ssdSmartHealthOk: boolean;
  temperature: number;
  nicStatus: Array<{ name: string; up: boolean; speed: string }>;
  serviceStatuses: Array<{ name: string; active: boolean; uptime: string }>;
  recentErrors: string[];
}

export interface IOTestRequest {
  outputChannel: number;
  durationMs: number;
  confirmedBy: string;
}

export interface IOTestResult {
  channel: number;
  success: boolean;
  measuredDurationMs: number;
  error?: string;
}

// ---------------------------------------------------------------------------
// Adapter Interface
// ---------------------------------------------------------------------------

export interface LocalAdapter {
  // Authentication
  login(request: LoginRequest): Promise<LoginResponse>;
  logout(): Promise<void>;

  // Status
  getZoneStatuses(): Promise<ZoneStatus[]>;
  getEquipmentStatuses(): Promise<EquipmentStatus[]>;
  getCameraStatuses(): Promise<CameraStatus[]>;
  getSystemStatus(): Promise<SystemStatus>;

  // Events
  getRecentEvents(limit: number): Promise<SafetyEventItem[]>;
  getActiveSafetyDecisions(): Promise<SafetyDecisionEvent[]>;

  // Operator actions (requires OPERATOR or MAINTAINER role)
  acknowledgeEvent(request: AckEventRequest): Promise<void>;
  resolveEvent(request: ResolveEventRequest): Promise<void>;
  requestWorkWindow(request: WorkWindowRequest): Promise<WorkWindowResponse>;
  endWorkWindow(windowId: string): Promise<void>;

  // Maintainer actions (requires MAINTAINER role)
  getDiagnostics(): Promise<DiagnosticsBundle>;
  testIO(request: IOTestRequest): Promise<IOTestResult>;
  exportBackup(): Promise<Blob>;
  restoreBackup(file: File): Promise<{ success: boolean; error?: string }>;

  // WebSocket subscription (future: real-time updates)
  subscribe(onEvent: (event: SafetyDecisionEvent) => void): () => void;
}
