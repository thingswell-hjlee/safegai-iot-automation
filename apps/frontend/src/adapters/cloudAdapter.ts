/**
 * Cloud Adapter - AWS API connection for dashboard when accessing via CloudFront.
 * Uses Cognito authentication and API Gateway endpoints.
 */

export interface CloudConfig {
  apiEndpoint: string;
  userPoolId: string;
  userPoolClientId: string;
  region: string;
}

export interface AuthTokens {
  idToken: string;
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
}

export interface CloudEvent {
  PK: string;
  SK: string;
  EventType: string;
  GatewayId: string;
  Timestamp: string;
  Payload: Record<string, unknown>;
}

export interface GatewayInfo {
  gatewayId: string;
  status: string;
  lastSeen: string;
  version: string;
}

export class CloudAdapter {
  private config: CloudConfig;
  private tokens: AuthTokens | null = null;
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(config: CloudConfig) {
    this.config = config;
  }

  // --- Authentication ---

  async signIn(email: string, password: string): Promise<void> {
    // In production, this would use AWS Cognito SRP auth flow.
    // Simplified for the adapter interface definition.
    const response = await fetch(`${this.config.apiEndpoint}/auth/signin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      throw new Error(`Authentication failed: ${response.status}`);
    }

    const data = await response.json();
    this.tokens = {
      idToken: data.idToken,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken,
      expiresAt: Date.now() + data.expiresIn * 1000,
    };

    this.scheduleTokenRefresh();
  }

  signOut(): void {
    this.tokens = null;
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  isAuthenticated(): boolean {
    return this.tokens !== null && this.tokens.expiresAt > Date.now();
  }

  // --- API Methods ---

  async getEvents(params?: {
    gatewayId?: string;
    eventType?: string;
    from?: string;
    to?: string;
    limit?: number;
  }): Promise<CloudEvent[]> {
    const query = new URLSearchParams();
    if (params?.gatewayId) query.set('gatewayId', params.gatewayId);
    if (params?.eventType) query.set('eventType', params.eventType);
    if (params?.from) query.set('from', params.from);
    if (params?.to) query.set('to', params.to);
    if (params?.limit) query.set('limit', String(params.limit));
    const qs = query.toString();
    return this.fetchAPI<CloudEvent[]>(`/api/events${qs ? `?${qs}` : ''}`);
  }

  async getGateways(): Promise<GatewayInfo[]> {
    return this.fetchAPI<GatewayInfo[]>('/api/gateways');
  }

  async getGatewayHealth(gatewayId: string): Promise<Record<string, unknown>> {
    return this.fetchAPI<Record<string, unknown>>(`/api/gateways/${gatewayId}/health`);
  }

  async getHealthCheck(): Promise<{ status: string; service: string }> {
    // Health endpoint does not require auth
    const response = await fetch(`${this.config.apiEndpoint}/api/health`);
    if (!response.ok) {
      throw new Error(`API health check failed: ${response.status}`);
    }
    return response.json();
  }

  // --- Private ---

  private async fetchAPI<T>(path: string, init?: RequestInit): Promise<T> {
    if (!this.tokens) {
      throw new Error('Not authenticated. Call signIn() first.');
    }

    const response = await fetch(`${this.config.apiEndpoint}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': this.tokens.idToken,
        ...init?.headers,
      },
    });

    if (response.status === 401) {
      // Token expired, try refresh
      await this.refreshTokens();
      return this.fetchAPI<T>(path, init);
    }

    if (!response.ok) {
      throw new Error(`Cloud API error: ${response.status} ${response.statusText}`);
    }

    return response.json() as Promise<T>;
  }

  private async refreshTokens(): Promise<void> {
    if (!this.tokens?.refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await fetch(`${this.config.apiEndpoint}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: this.tokens.refreshToken }),
    });

    if (!response.ok) {
      this.tokens = null;
      throw new Error('Token refresh failed');
    }

    const data = await response.json();
    this.tokens = {
      ...this.tokens,
      idToken: data.idToken,
      accessToken: data.accessToken,
      expiresAt: Date.now() + data.expiresIn * 1000,
    };
  }

  private scheduleTokenRefresh(): void {
    if (!this.tokens) return;
    const msUntilExpiry = this.tokens.expiresAt - Date.now() - 60000; // Refresh 1 min early
    if (msUntilExpiry > 0) {
      this.refreshTimer = setTimeout(() => this.refreshTokens(), msUntilExpiry);
    }
  }
}
