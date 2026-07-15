/**
 * WebSocket client with automatic reconnection and message typing.
 */

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface WebSocketMessage {
  type: string;
  payload: unknown;
  timestamp: string;
}

export interface WebSocketClientConfig {
  url: string;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
}

type MessageHandler = (msg: WebSocketMessage) => void;
type StatusHandler = (status: ConnectionStatus) => void;

export class WebSocketClient {
  private config: WebSocketClientConfig;
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();
  private status: ConnectionStatus = 'disconnected';
  private intentionalClose = false;

  constructor(config: WebSocketClientConfig) {
    this.config = {
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      ...config,
    };
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;
    this.intentionalClose = false;
    this.setStatus('connecting');

    this.ws = new WebSocket(this.config.url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.setStatus('connected');
      this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as WebSocketMessage;
        if (msg.type === 'pong') return; // Heartbeat response
        this.messageHandlers.forEach((handler) => handler(msg));
      } catch {
        // Non-JSON message
        const fallback: WebSocketMessage = {
          type: 'raw',
          payload: event.data,
          timestamp: new Date().toISOString(),
        };
        this.messageHandlers.forEach((handler) => handler(fallback));
      }
    };

    this.ws.onclose = () => {
      this.stopHeartbeat();
      this.setStatus('disconnected');
      if (!this.intentionalClose) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      this.setStatus('error');
    };
  }

  disconnect(): void {
    this.intentionalClose = true;
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
    this.setStatus('disconnected');
  }

  send(type: string, payload: unknown): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not connected');
    }
    const msg: WebSocketMessage = {
      type,
      payload,
      timestamp: new Date().toISOString(),
    };
    this.ws.send(JSON.stringify(msg));
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onStatusChange(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    return () => this.statusHandlers.delete(handler);
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  // --- Private ---

  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.statusHandlers.forEach((handler) => handler(status));
    }
  }

  private scheduleReconnect(): void {
    const maxAttempts = this.config.maxReconnectAttempts!;
    if (this.reconnectAttempts >= maxAttempts) {
      this.setStatus('error');
      return;
    }
    this.reconnectAttempts++;
    const delay = this.config.reconnectInterval! * Math.min(this.reconnectAttempts, 5);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping', timestamp: new Date().toISOString() }));
      }
    }, this.config.heartbeatInterval!);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }
}
