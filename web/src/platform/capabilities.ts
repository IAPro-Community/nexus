export interface PlatformCapabilities {
  native: boolean;
  filePicker: boolean;
  folderPicker: boolean;
  notifications: boolean;
  tray: boolean;
  nativeMenus: boolean;
  deepLinks: boolean;
  autoStart: boolean;
  windowManagement: boolean;
  nativeVoiceCapture?: boolean;
}

export interface NativeVoiceRecording {
  dataBase64: string;
  mimeType: string;
  durationMs: number;
}

export interface FilePickerOptions {
  title?: string;
  defaultPath?: string;
  filters?: Array<{
    name: string;
    extensions: string[];
  }>;
}

export interface NotificationOptions {
  title: string;
  body: string;
  icon?: string;
  silent?: boolean;
}

export interface DesktopBootstrapInfo {
  serverUrl: string;
  sessionToken: string;
  csrfToken: string;
}
