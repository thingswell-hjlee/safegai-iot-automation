/**
 * Zustand-style safety event state store.
 * Manages safety events, zone states, and alarm status.
 */

export interface SafetyEvent {
  id: string;
  eventType: string;
  zoneId: string;
  severity: 'INFO' | 'WARNING' | 'CRITICAL' | 'ALARM';
  timestamp: string;
  acknowledged: boolean;
  details: Record<string, unknown>;
}

export interface ZoneState {
  zoneId: string;
  occupancyState: string;
  personCount: number;
  lastUpdated: string;
  cameraId: string;
}

export interface AlarmState {
  active: boolean;
  count: number;
  highestSeverity: string | null;
  lastAlarmAt: string | null;
}

export interface SafetyState {
  events: SafetyEvent[];
  zones: ZoneState[];
  alarm: AlarmState;
  maxEvents: number;
}

export interface SafetyActions {
  addEvent: (event: SafetyEvent) => void;
  acknowledgeEvent: (eventId: string) => void;
  updateZone: (zone: ZoneState) => void;
  setAlarm: (alarm: AlarmState) => void;
  clearEvents: () => void;
}

const initialState: SafetyState = {
  events: [],
  zones: [],
  alarm: { active: false, count: 0, highestSeverity: null, lastAlarmAt: null },
  maxEvents: 500,
};

/**
 * Creates a safety store (Zustand-compatible pattern).
 * Usage: const useSafetyStore = create(createSafetyStore);
 */
export function createSafetyStore(
  set: (fn: (state: SafetyState) => Partial<SafetyState>) => void,
): SafetyState & SafetyActions {
  return {
    ...initialState,

    addEvent: (event) =>
      set((state) => {
        const events = [event, ...state.events].slice(0, state.maxEvents);
        return { events };
      }),

    acknowledgeEvent: (eventId) =>
      set((state) => ({
        events: state.events.map((e) =>
          e.id === eventId ? { ...e, acknowledged: true } : e,
        ),
      })),

    updateZone: (zone) =>
      set((state) => {
        const idx = state.zones.findIndex((z) => z.zoneId === zone.zoneId);
        const zones = [...state.zones];
        if (idx >= 0) {
          zones[idx] = zone;
        } else {
          zones.push(zone);
        }
        return { zones };
      }),

    setAlarm: (alarm) => set(() => ({ alarm })),

    clearEvents: () => set(() => ({ events: [] })),
  };
}
