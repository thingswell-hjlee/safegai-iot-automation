/**
 * React hooks for cloud data fetching using TanStack Query patterns.
 * These hooks provide cloud API data with authentication handling.
 */

import type { CloudEvent, GatewayInfo } from '../adapters/cloudAdapter';

// Query key factory for cloud queries
export const cloudKeys = {
  all: ['cloud'] as const,
  health: () => [...cloudKeys.all, 'health'] as const,
  events: (params?: { gatewayId?: string; eventType?: string; limit?: number }) =>
    [...cloudKeys.all, 'events', params] as const,
  gateways: () => [...cloudKeys.all, 'gateways'] as const,
  gatewayHealth: (id: string) => [...cloudKeys.all, 'gateways', id, 'health'] as const,
};

// Hook configurations
export const cloudQueries = {
  health: {
    queryKey: cloudKeys.health(),
    refetchInterval: 30000, // Less frequent for cloud
    staleTime: 10000,
  },

  events: (params?: { gatewayId?: string; eventType?: string; limit?: number }) => ({
    queryKey: cloudKeys.events(params),
    refetchInterval: 10000,
    staleTime: 5000,
  }),

  gateways: {
    queryKey: cloudKeys.gateways(),
    refetchInterval: 30000,
    staleTime: 15000,
  },
};

// Type exports
export type { CloudEvent, GatewayInfo };
