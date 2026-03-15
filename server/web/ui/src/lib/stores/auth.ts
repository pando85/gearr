import { writable } from 'svelte/store';

export interface AuthState {
  token: string;
  isAuthenticated: boolean;
}

function createAuthStore() {
  const { subscribe, set } = writable<AuthState>({
    token: '',
    isAuthenticated: false,
  });

  return {
    subscribe,
    login: (token: string) => set({ token, isAuthenticated: true }),
    logout: () => set({ token: '', isAuthenticated: false }),
    getToken: () => {
      let token = '';
      const unsubscribe = subscribe((state) => (token = state.token));
      unsubscribe();
      return token;
    },
  };
}

export const authStore = createAuthStore();
