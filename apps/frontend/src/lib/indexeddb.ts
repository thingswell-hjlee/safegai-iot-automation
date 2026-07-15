/**
 * IndexedDB wrapper for offline-first event caching.
 * Stores events locally when gateway/cloud connection is unavailable.
 */

export interface CachedEvent {
  id: string;
  type: string;
  payload: unknown;
  timestamp: string;
  synced: boolean;
}

const DB_NAME = 'safegai-cache';
const DB_VERSION = 1;
const EVENTS_STORE = 'events';
const CONFIG_STORE = 'config';

export class IndexedDBStore {
  private db: IDBDatabase | null = null;

  async open(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Events store
        if (!db.objectStoreNames.contains(EVENTS_STORE)) {
          const eventsStore = db.createObjectStore(EVENTS_STORE, { keyPath: 'id' });
          eventsStore.createIndex('timestamp', 'timestamp', { unique: false });
          eventsStore.createIndex('synced', 'synced', { unique: false });
          eventsStore.createIndex('type', 'type', { unique: false });
        }

        // Config store
        if (!db.objectStoreNames.contains(CONFIG_STORE)) {
          db.createObjectStore(CONFIG_STORE, { keyPath: 'key' });
        }
      };

      request.onsuccess = (event) => {
        this.db = (event.target as IDBOpenDBRequest).result;
        resolve();
      };

      request.onerror = () => {
        reject(new Error('Failed to open IndexedDB'));
      };
    });
  }

  async addEvent(event: CachedEvent): Promise<void> {
    return this.transaction(EVENTS_STORE, 'readwrite', (store) => {
      store.put(event);
    });
  }

  async getEvents(limit = 100): Promise<CachedEvent[]> {
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(EVENTS_STORE, 'readonly');
      const store = tx.objectStore(EVENTS_STORE);
      const index = store.index('timestamp');
      const request = index.openCursor(null, 'prev');
      const results: CachedEvent[] = [];

      request.onsuccess = (event) => {
        const cursor = (event.target as IDBRequest<IDBCursorWithValue>).result;
        if (cursor && results.length < limit) {
          results.push(cursor.value);
          cursor.continue();
        } else {
          resolve(results);
        }
      };

      request.onerror = () => reject(new Error('Failed to query events'));
    });
  }

  async getUnsyncedEvents(): Promise<CachedEvent[]> {
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(EVENTS_STORE, 'readonly');
      const store = tx.objectStore(EVENTS_STORE);
      const index = store.index('synced');
      const request = index.getAll(IDBKeyRange.only(false));

      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(new Error('Failed to query unsynced events'));
    });
  }

  async markSynced(eventId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(EVENTS_STORE, 'readwrite');
      const store = tx.objectStore(EVENTS_STORE);
      const getReq = store.get(eventId);

      getReq.onsuccess = () => {
        const event = getReq.result;
        if (event) {
          event.synced = true;
          store.put(event);
        }
        resolve();
      };

      getReq.onerror = () => reject(new Error('Failed to mark event synced'));
    });
  }

  async setConfig(key: string, value: unknown): Promise<void> {
    return this.transaction(CONFIG_STORE, 'readwrite', (store) => {
      store.put({ key, value, updatedAt: new Date().toISOString() });
    });
  }

  async getConfig(key: string): Promise<unknown | null> {
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(CONFIG_STORE, 'readonly');
      const store = tx.objectStore(CONFIG_STORE);
      const request = store.get(key);

      request.onsuccess = () => {
        resolve(request.result?.value ?? null);
      };

      request.onerror = () => reject(new Error('Failed to get config'));
    });
  }

  async clearOldEvents(olderThanMs: number): Promise<number> {
    const cutoff = new Date(Date.now() - olderThanMs).toISOString();
    let deleted = 0;

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(EVENTS_STORE, 'readwrite');
      const store = tx.objectStore(EVENTS_STORE);
      const index = store.index('timestamp');
      const range = IDBKeyRange.upperBound(cutoff);
      const request = index.openCursor(range);

      request.onsuccess = (event) => {
        const cursor = (event.target as IDBRequest<IDBCursorWithValue>).result;
        if (cursor) {
          cursor.delete();
          deleted++;
          cursor.continue();
        } else {
          resolve(deleted);
        }
      };

      request.onerror = () => reject(new Error('Failed to clear old events'));
    });
  }

  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
  }

  // --- Private ---

  private transaction(
    storeName: string,
    mode: IDBTransactionMode,
    operation: (store: IDBObjectStore) => void,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error('Database not open'));
        return;
      }

      const tx = this.db.transaction(storeName, mode);
      const store = tx.objectStore(storeName);
      operation(store);

      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(new Error(`Transaction failed on ${storeName}`));
    });
  }
}
