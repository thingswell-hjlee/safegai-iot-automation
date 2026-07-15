/**
 * Gateway Adapter - Direct HTTP + WebSocket connection to local gateway.
 * Used when the dashboard connects directly to the edge gateway on the LAN.
 */

export interface GatewayConfig {
  baseUrl: string;
  wsUrl: string;
  timeout?: number;
}

export interface HealthStatus {
  status: string;
  version: string;
  uptime: string;
}

export interface SafetyEvent {
  id: string;
  eventType: string;
  zoneId: string;
  severity: string;
  timestamp: string;
  details: Record<string, unknown>;
}

export interface EquipmentStatus {
  equipmentId: string;
  state: string;
  timestamp: string;
  source: string;
}

export interface ZoneStatus {
  zoneId: string;
  occupancyState: string;
  personCount: number;
  lastUpdated: string;
}

export interface OutputCommandRequest {
  commandType: string;
  target: string;
  parameters?: Record<string, string>;
  correlationId?: string;
}

export interface OutputCommandResult {
  commandId: string;
  success: boolean;
  executedAt: string;
  errorMsg?: string;
}

const DEFAULT_TIMEOUT = 10000;

export class GatewayAdapter {
  private config: GatewayConfig;
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private eventListeners: Map<string, Set<(data: unknown) => void>> = new Map();

  constructor(config: GatewayConfig) {
    this.config = {
      ...config,
      timeout: config.timeout ?? DEFAULT_TIMEOUT,
    };
  }

  // --- HTTP Methods ---

  async getHealth(): Promise<HealthStatus> {
    return this.fetchJSON<HealthStatus>('/health/ready');
  }

  async getEvents(params?: { limit?: number; offset?: number }): Promise<SafetyEvent[]> {
    const query = new URLSearchParams();
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.offset) query.set('offset', String(params.offset));
    const qs = query.toString();
    return this.fetchJSON<SafetyEvent[]>(`/api/events${qs ? `?${qs}` : ''}`);
  }

  async getEquipmentStatus(): Promise<EquipmentStatus[]> {
    return this.fetchJSON<EquipmentStatus[]>('/api/equipment/status');
  }

  async getZoneStatus(): Promise<ZoneStatus[]> {
    return this.fetchJSON<ZoneStatus[]>('/api/zones/status');
  }

  async executeCommand(cmd: OutputCommandRequest): Promise<OutputCommandResult> {
    return this.fetchJSON<OutputCommandResult>('/api/commands/execute', {
      method: 'POST',
      body: JSON.stringify(cmd),
    });
  }

  async getAuditLog(params?: { from?: string; to?: string }): Promise<unknown[]> {
    const query = new URLSearchParams();
    if (params?.from) query.set('from', params.from);
    if (params?.to) query.set('to', params.to);
    const qs = query.toString();
    return this.fetchJSON<unknown[]>(`/api/audit${qs ? `?${qs}` : ''}`);
  }

  // --- WebSocket Methods ---

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(this.config.wsUrl);

    this.ws.onopen = () => {
      this.emit('connection', { status: 'connected' });
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string);
        this.emit(msg.type ?? 'message', msg);
      } catch {
        this.emit('message', { raw: event.data });
      }
    };

    this.ws.onclose = () => {
      this.emit('connection', { status: 'disconnected' });
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.emit('connection', { status: 'error' });
    };
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  on(event: string, listener: (data: unknown) => void): () => void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, new Set());
    }
    this.eventListeners.get(event)!.add(listener);
    return () => {
      this.eventListeners.get(event)?.delete(listener);
    };
  }

  // --- Private ---

  private async fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await fetch(`${this.config.baseUrl}${path}`, {
        ...init,
        headers: {
          'Content-Type': 'application/json',
          ...init?.headers,
        },
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`Gateway API error: ${response.status} ${response.statusText}`);
      }

      return response.json() as Promise<T>;
    } finally {
      clearTimeout(timeout);
    }
  }

  private emit(event: string, data: unknown): void {
    this.eventListeners.get(event)?.forEach((listener) => listener(data));
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 5000);
  }
}
