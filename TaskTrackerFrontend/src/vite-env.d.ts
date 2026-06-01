/// <reference types="vite/client" />

// Дополняем типы только для явной переменной, чтобы IDE подсказывала.
interface ImportMetaEnv {
  readonly VITE_API_URL?: string;
}
