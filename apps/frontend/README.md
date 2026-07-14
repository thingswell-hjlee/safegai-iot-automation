# SafeGAI Hybrid App - M01 Mock

## Purpose

React/TypeScript frontend scaffold for the SafeGAI AI Fisheye Zone Safety system.
Provides role-based views (User, Operator, Maintainer) with a mock API adapter
returning simulated zone states, events, and equipment status.

## Architecture

```
src/
  index.tsx              Entry point
  App.tsx                Role-based routing and auth state
  types/
    api.ts               TypeScript interfaces matching contracts
    roles.ts             Role type and permission matrix
  adapters/
    localAdapter.ts      Interface defining all Gateway API calls
    mockAdapter.ts       Mock implementation with simulated data
  components/
    StatusPanel.tsx      Zone occupancy with safety colors
    EventList.tsx        Recent safety events with severity
    EquipmentStatus.tsx  Equipment states with interlock info
  pages/
    UserView.tsx         Read-only: status, video, warnings, guide
    OperatorView.tsx     Event ACK/resolve/classify, work windows
    MaintainerView.tsx   Diagnostics, camera config, I/O test, backup
  mocks/
    mockApi.ts           Re-exports adapter with convenience functions
```

## Role Permissions

| Feature                 | User | Operator | Maintainer |
|-------------------------|:----:|:--------:|:----------:|
| View safety status      |  Y   |    Y     |     Y      |
| View video              |  Y   |    Y     |     Y      |
| View active alerts      |  Y   |    Y     |     Y      |
| Event ACK/resolve       |  N   |    Y     |     Y      |
| Work window management  |  N   |    Y     |     Y      |
| I/O test                |  N   |    N     |     Y      |
| Camera configuration    |  N   |    N     |     Y      |
| Diagnostics (full)      |  N   |    N     |     Y      |
| Backup/restore          |  N   |    N     |     Y      |

## Safety UX Rules

1. **UNKNOWN and STALE must NEVER appear as safe or vacant.**
   - Always rendered with warning/danger styling (yellow or red).
   - Text explicitly states "Cannot Confirm" or "Data Outdated".

2. **Color coding uses color + icon + text** (not color alone).
   - Red: OCCUPIED + RUNNING, STOP_REQUEST_REQUIRED
   - Yellow/Amber: UNKNOWN, STALE, VACANT_PENDING
   - Green: VACANT_CONFIRMED only

3. **Active critical warnings remain visible** regardless of current view.

4. **User mode restrictions:**
   - No ACK button
   - No classify controls
   - No I/O test
   - No settings access
   - No safety mapping

## Mock Login

| Username prefix  | Role assigned   |
|------------------|-----------------|
| `user-*`         | USER            |
| `operator-*`     | OPERATOR        |
| `maintainer-*`   | MAINTAINER      |

Password must be 4+ characters.

## Development

```bash
# Install dependencies (when npm is available)
npm install

# Start development server
npm run dev

# Type check
npm run type-check

# Build for production
npm run build
```

## Contract Alignment

Types in `src/types/api.ts` match the JSON Schemas in:
- `contracts/events/event-envelope-v1.schema.json`
- `contracts/events/occupancy-state-v1.schema.json`
- `contracts/events/equipment-state-v1.schema.json`
- `contracts/events/safety-decision-v1.schema.json`

## Limitations

- This is a mock scaffold; no real API connection exists yet.
- Video placeholders are shown; no actual RTSP stream playback.
- Authentication is simulated; no real Argon2id password hashing.
- No npm dependencies installed (INTEGRATIONS_ONLY network constraint).
- TypeScript source only; compilation requires `npm install` first.
