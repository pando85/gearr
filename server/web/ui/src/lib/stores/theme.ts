import { writable } from 'svelte/store';

export type ThemeName = 'light' | 'dark';
export type ThemeSetting = ThemeName | 'auto';

function createThemeStore() {
  const getStoredTheme = (): ThemeSetting => {
    if (typeof window === 'undefined') return 'auto';
    const stored = localStorage.getItem('user-prefers-color-scheme');
    return (stored as ThemeSetting) || 'auto';
  };

  const getBrowserTheme = (): ThemeName => {
    if (typeof window === 'undefined') return 'light';
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  };

  const resolveTheme = (setting: ThemeSetting): ThemeName => {
    if (setting === 'auto') {
      return getBrowserTheme();
    }
    return setting;
  };

  const { subscribe, set } = writable<ThemeSetting>(getStoredTheme());

  return {
    subscribe,
    setTheme: (setting: ThemeSetting) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem('user-prefers-color-scheme', setting);
      }
      set(setting);
    },
    get resolved(): ThemeName {
      let setting: ThemeSetting = 'auto';
      const unsubscribe = subscribe((s) => (setting = s));
      unsubscribe();
      return resolveTheme(setting);
    },
    resolve: resolveTheme,
  };
}

export const themeStore = createThemeStore();

export function applyTheme(theme: ThemeName) {
  if (typeof window === 'undefined') return;

  const root = document.documentElement;
  if (theme === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
}
