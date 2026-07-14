/**
 * Mock API Module
 *
 * Re-exports the mock adapter and provides convenience functions
 * for testing and development. Returns simulated zone states,
 * events, and equipment states matching contract schemas.
 *
 * SAFETY: Includes UNKNOWN and STALE scenarios to verify
 * that the UI handles them correctly (never as safe/vacant).
 */

import { mockAdapter, MockAdapter } from '../adapters/mockAdapter';
import { OccupancyState, EquipmentState, SafetyDecision, DataQuality } from '../types/api';
import type {
  ZoneStatus,
  EquipmentStatus,
  CameraStatus,
  SystemStatus,
  SafetyEventItem,
  SafetyDecisionEvent,
} from '../types/api';

// Re-export the singleton adapter
export { mockAdapter, MockAdapter };

// ---------------------------------------------------------------------------
// Convenience Functions for Development/Testing
// ---------------------------------------------------------------------------

/**
 * Get all zone statuses with simulated data.
 */
export async function getZoneStatuses(): Promise<ZoneStatus[]> {
  return mockAdapter.getZoneStatuses();
}

/**
 * Get all equipment statuses with simulated data.
 */
export async function getEquipmentStatuses(): Promise<EquipmentStatus[]> {
  return mockAdapter.getEquipmentStatuses();
}

/**
 * Get camera statuses including offline cameras.
 */
export async function getCameraStatuses(): Promise<CameraStatus[]> {
  return mockAdapter.getCameraStatuses();
}

/**
 * Get system status for the gateway.
 */
export async function getSystemStatus(): Promise<SystemStatus> {
  return mockAdapter.getSystemStatus();
}

/**
 * Get recent safety events.
 */
export async function getRecentEvents(limit: number = 10): Promise<SafetyEventItem[]> {
  return mockAdapter.getRecentEvents(limit);
}

/**
 * Get active safety decisions requiring attention.
 */
export async function getActiveSafetyDecisions(): Promise<SafetyDecisionEvent[]> {
  return mockAdapter.getActiveSafetyDecisions();
}

// ---------------------------------------------------------------------------
// Test Data Generators
// ---------------------------------------------------------------------------

/**
 * Generate a zone status in a specific occupancy state for testing.
 */
export function createTestZone(
  zoneId: string,
  occupancy: OccupancyState,
  decision: SafetyDecision
): ZoneStatus {
  return {
    zoneId,
    zoneName: `Test Zone ${zoneId}`,
    occupancy,
    safetyDecision: decision,
    lastUpdated: new Date().toISOString(),
    cameraIds: ['cam-test-01'],
    activeWarnings:
      decision === SafetyDecision.STOP_REQUEST_REQUIRED
        ? ['Test: Stop request required']
        : occupancy === OccupancyState.UNKNOWN || occupancy === OccupancyState.STALE
          ? ['Test: Safety confirmation unavailable']
          : [],
  };
}

/**
 * Generate equipment status for testing.
 */
export function createTestEquipment(
  equipmentId: string,
  state: EquipmentState,
  interlock: boolean
): EquipmentStatus {
  return {
    equipmentId,
    equipmentName: `Test Equipment ${equipmentId}`,
    state,
    lastUpdated: new Date().toISOString(),
    zoneId: 'zone-test-01',
    restartInterlockActive: interlock,
  };
}

/**
 * Generate a safety decision event for testing.
 */
export function createTestSafetyDecision(
  decision: SafetyDecision,
  occupancy: OccupancyState,
  equipment: EquipmentState
): SafetyDecisionEvent {
  return {
    schemaVersion: '1.0.0',
    eventId: crypto.randomUUID?.() || 'test-uuid',
    correlationId: crypto.randomUUID?.() || 'test-corr-uuid',
    tenantId: 'tenant-test',
    siteId: 'site-test',
    gatewayId: 'gw-test',
    deviceId: 'cam-test-01',
    zoneId: 'zone-test-01',
    observedAt: new Date().toISOString(),
    receivedAt: new Date().toISOString(),
    sequenceNo: 1,
    source: 'zone-state-engine',
    quality: DataQuality.GOOD,
    decision,
    ruleId: 'R-TEST',
    occupancyState: occupancy,
    equipmentState: equipment,
    reason: `Test decision: ${decision}`,
    actions: [],
  };
}
