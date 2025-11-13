// Favorite database history management using localStorage

export interface FavoriteDatabase {
  path: string;
  name: string;
  addedAt: string;
  lastConnected?: string;
}

const STORAGE_KEY = 'rocksdb_favorite_databases';
const MAX_FAVORITES = 20;

export const dbHistory = {
  // Get all favorite databases
  getFavorites(): FavoriteDatabase[] {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) return [];
      return JSON.parse(stored);
    } catch (error) {
      console.error('Failed to load favorites:', error);
      return [];
    }
  },

  // Add a database to favorites
  addFavorite(path: string): void {
    try {
      const favorites = this.getFavorites();

      // Check if already exists
      const existingIndex = favorites.findIndex(db => db.path === path);

      if (existingIndex >= 0) {
        // Update last connected time
        favorites[existingIndex].lastConnected = new Date().toISOString();
      } else {
        // Add new favorite
        const name = path.split('/').pop() || path;
        const newFavorite: FavoriteDatabase = {
          path,
          name,
          addedAt: new Date().toISOString(),
          lastConnected: new Date().toISOString(),
        };

        favorites.unshift(newFavorite);

        // Keep only MAX_FAVORITES
        if (favorites.length > MAX_FAVORITES) {
          favorites.splice(MAX_FAVORITES);
        }
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(favorites));
    } catch (error) {
      console.error('Failed to add favorite:', error);
    }
  },

  // Remove a database from favorites
  removeFavorite(path: string): void {
    try {
      const favorites = this.getFavorites();
      const filtered = favorites.filter(db => db.path !== path);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(filtered));
    } catch (error) {
      console.error('Failed to remove favorite:', error);
    }
  },

  // Clear all favorites
  clearAll(): void {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error('Failed to clear favorites:', error);
    }
  },

  // Check if a path is favorited
  isFavorite(path: string): boolean {
    const favorites = this.getFavorites();
    return favorites.some(db => db.path === path);
  },
};
