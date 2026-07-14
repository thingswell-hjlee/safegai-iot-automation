/**
 * SafeGAI Role Definitions and Permission Matrix
 *
 * Based on docs/ROLE_MODE_SPEC.md
 *
 * SAFETY UX RULES:
 * - USER mode must NOT expose: ACK, classify, I/O test, settings, safety mapping
 * - Hiding a control is not authorization. Backend APIs must enforce the same role.
 */

// ---------------------------------------------------------------------------
// Role Types
// ---------------------------------------------------------------------------

export type Role = 'USER' | 'OPERATOR' | 'MAINTAINER';

// ---------------------------------------------------------------------------
// Permission Keys
// ---------------------------------------------------------------------------

export type Permission =
  | 'view_safety_status'
  | 'view_video'
  | 'view_active_alerts'
  | 'ack_events'
  | 'resolve_events'
  | 'classify_events'
  | 'view_reports_full'
  | 'view_reports_limited'
  | 'request_work_window'
  | 'end_work_window'
  | 'register_camera'
  | 'configure_zone_mapping'
  | 'configure_dio_mapping'
  | 'io_test'
  | 'view_diagnostics_full'
  | 'view_diagnostics_summary'
  | 'backup_restore'
  | 'update_rollback'
  | 'view_hardware_detail'
  | 'view_hardware_summary';

// ---------------------------------------------------------------------------
// Permission Matrix
// ---------------------------------------------------------------------------

/**
 * Permission matrix mapping roles to allowed operations.
 * This is an immutable compile-time reference. Runtime authorization
 * must also be enforced by the backend API.
 */
export const PERMISSION_MATRIX: Record<Role, ReadonlySet<Permission>> = {
  USER: new Set<Permission>([
    'view_safety_status',
    'view_video',
    'view_active_alerts',
    'view_reports_limited',
  ]),
  OPERATOR: new Set<Permission>([
    'view_safety_status',
    'view_video',
    'view_active_alerts',
    'ack_events',
    'resolve_events',
    'classify_events',
    'view_reports_full',
    'request_work_window',
    'end_work_window',
    'view_diagnostics_summary',
    'view_hardware_summary',
  ]),
  MAINTAINER: new Set<Permission>([
    'view_safety_status',
    'view_video',
    'view_active_alerts',
    'ack_events',
    'resolve_events',
    'classify_events',
    'view_reports_full',
    'request_work_window',
    'end_work_window',
    'register_camera',
    'configure_zone_mapping',
    'configure_dio_mapping',
    'io_test',
    'view_diagnostics_full',
    'backup_restore',
    'update_rollback',
    'view_hardware_detail',
  ]),
};

// ---------------------------------------------------------------------------
// Permission Helpers
// ---------------------------------------------------------------------------

/**
 * Check if a given role has a specific permission.
 */
export function hasPermission(role: Role, permission: Permission): boolean {
  return PERMISSION_MATRIX[role].has(permission);
}

/**
 * Get all permissions for a given role.
 */
export function getPermissions(role: Role): ReadonlySet<Permission> {
  return PERMISSION_MATRIX[role];
}

/**
 * Check if a role can perform event acknowledgement operations.
 * USER mode must NEVER have ACK capability.
 */
export function canAcknowledge(role: Role): boolean {
  return role === 'OPERATOR' || role === 'MAINTAINER';
}

/**
 * Check if a role can access maintenance/diagnostic features.
 */
export function canMaintain(role: Role): boolean {
  return role === 'MAINTAINER';
}
