/**
 * React hooks for gateway data fetching using TanStack Query patterns.
 * These hooks provide real-time gateway data with caching and refetching.
 */

import type { SafetyEvent, EquipmentStatus, ZoneStatus, HealthStatus } from '../adapters/gatewayAdapter';

// Query key factory for consistent cache keys
export const gatewayKeys = {
  all: ['gateway'] as const,
  health: () => [...gatewayKeys.all, 'health'] as const,
  events: (params?: { limit?: number }) => [...gatewayKeys.all, 'events', params] as const,
  equipment: () => [...gatewayKeys.all, 'equipment'] as const,
  zones: () => [...gatewayKeys.all, 'zones'] as const,
  audit: (params?: { from?: string; to?: string }) => [...gatewayKeys.all, 'audit', params] as const,
};

// Hook configurations (for use with TanStack Query)
export const gatewayQueries = {
  health: {
    queryKey: gatewayKeys.health(),
    refetchInterval: 5000, // Poll health every 5s
    staleTime: 3000,
  },

  events: (params?: { limit?: number }) => ({
    queryKey: gatewayKeys.events(params),
    refetchInterval: 2000, // Real-time event polling
    staleTime: 1000,
  }),

  equipment: {
    queryKey: gatewayKeys.equipment(),
    refetchInterval: 3000,
    staleTime: 2000,
  },

  zones: {
    queryKey: gatewayKeys.zones(),
    refetchInterval: 2000,
    staleTime: 1000,
  },
};

// Type exports for consumers
export type { SafetyEvent, EquipmentStatus, ZoneStatus, HealthStatus };
