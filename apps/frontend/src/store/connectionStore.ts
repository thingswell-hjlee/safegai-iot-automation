/**
 * Zustand-style connection state store for managing gateway/cloud connections.
 * Tracks connection status, selected adapters, and notification state.
 */

export type ConnectionMode = 'local' | 'cloud' | 'hybrid';
export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error';

export interface ConnectionState {
  mode: ConnectionMode;
  gatewayStatus: ConnectionStatus;
  cloudStatus: ConnectionStatus;
  gatewayUrl: string;
  cloudApiUrl: string;
  lastGatewayHeartbeat: string | null;
  lastCloudSync: string | null;
  pendingSync: number;
  error: string | null;
}

export interface ConnectionActions {
  setMode: (mode: ConnectionMode) => void;
  setGatewayStatus: (status: ConnectionStatus) => void;
  setCloudStatus: (status: ConnectionStatus) => void;
  setGatewayUrl: (url: string) => void;
  setCloudApiUrl: (url: string) => void;
  updateHeartbeat: () => void;
  updateCloudSync: () => void;
  setPendingSync: (count: number) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState: ConnectionState = {
  mode: 'local',
  gatewayStatus: 'disconnected',
  cloudStatus: 'disconnected',
  gatewayUrl: 'http://localhost:8080',
  cloudApiUrl: '',
  lastGatewayHeartbeat: null,
  lastCloudSync: null,
  pendingSync: 0,
  error: null,
};

/**
 * Creates a connection store (Zustand-compatible pattern).
 * Usage: const useConnectionStore = create(createConnectionStore);
 */
export function createConnectionStore(
  set: (partial: Partial<ConnectionState>) => void,
): ConnectionState & ConnectionActions {
  return {
    ...initialState,

    setMode: (mode) => set({ mode }),
    setGatewayStatus: (status) => set({ gatewayStatus: status }),
    setCloudStatus: (status) => set({ cloudStatus: status }),
    setGatewayUrl: (url) => set({ gatewayUrl: url }),
    setCloudApiUrl: (url) => set({ cloudApiUrl: url }),
    updateHeartbeat: () => set({ lastGatewayHeartbeat: new Date().toISOString() }),
    updateCloudSync: () => set({ lastCloudSync: new Date().toISOString() }),
    setPendingSync: (count) => set({ pendingSync: count }),
    setError: (error) => set({ error }),
    reset: () => set(initialState),
  };
}
