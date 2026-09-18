export namespace desktop {
  export class BootstrapInfo {
    serverUrl: string;
    sessionToken: string;
    csrfToken: string;

    static createFrom(source: any = {}) {
      return new BootstrapInfo(source);
    }

    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source);
      this.serverUrl = source['serverUrl'];
      this.sessionToken = source['sessionToken'];
      this.csrfToken = source['csrfToken'];
    }
  }
  export class Capabilities {
    native: boolean;
    filePicker: boolean;
    folderPicker: boolean;
    notifications: boolean;
    tray: boolean;
    nativeMenus: boolean;
    deepLinks: boolean;
    autoStart: boolean;
    windowManagement: boolean;

    static createFrom(source: any = {}) {
      return new Capabilities(source);
    }

    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source);
      this.native = source['native'];
      this.filePicker = source['filePicker'];
      this.folderPicker = source['folderPicker'];
      this.notifications = source['notifications'];
      this.tray = source['tray'];
      this.nativeMenus = source['nativeMenus'];
      this.deepLinks = source['deepLinks'];
      this.autoStart = source['autoStart'];
      this.windowManagement = source['windowManagement'];
    }
  }
  export class FileFilter {
    name: string;
    extensions: string[];

    static createFrom(source: any = {}) {
      return new FileFilter(source);
    }

    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source);
      this.name = source['name'];
      this.extensions = source['extensions'];
    }
  }
  export class FilePickerOptions {
    title: string;
    defaultPath: string;
    filters: FileFilter[];

    static createFrom(source: any = {}) {
      return new FilePickerOptions(source);
    }

    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source);
      this.title = source['title'];
      this.defaultPath = source['defaultPath'];
      this.filters = this.convertValues(source['filters'], FileFilter);
    }

    convertValues(a: any, classs: any, asMap: boolean = false): any {
      if (!a) {
        return a;
      }
      if (a.slice && a.map) {
        return (a as any[]).map((elem) => this.convertValues(elem, classs));
      } else if ('object' === typeof a) {
        if (asMap) {
          for (const key of Object.keys(a)) {
            a[key] = new classs(a[key]);
          }
          return a;
        }
        return new classs(a);
      }
      return a;
    }
  }
  export class NotificationOptions {
    title: string;
    body: string;
    icon: string;
    silent: boolean;

    static createFrom(source: any = {}) {
      return new NotificationOptions(source);
    }

    constructor(source: any = {}) {
      if ('string' === typeof source) source = JSON.parse(source);
      this.title = source['title'];
      this.body = source['body'];
      this.icon = source['icon'];
      this.silent = source['silent'];
    }
  }
}
