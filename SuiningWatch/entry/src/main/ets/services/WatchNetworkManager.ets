import { connection } from '@kit.NetworkKit';
import { BusinessError } from '@kit.BasicServicesKit';

export class WatchNetworkManager {
  private netConn: connection.NetConnection | undefined = undefined;
  private registered: boolean = false;
  private retrying: boolean = false;
  private onRestored: (() => Promise<void> | void) | undefined = undefined;

  startMonitoring(onRestored: () => Promise<void> | void): void {
    if (this.netConn) {
      this.onRestored = onRestored;
      return;
    }
    this.onRestored = onRestored;
    const netConn = connection.createNetConnection();
    this.netConn = netConn;
    netConn.on('netAvailable', (_network: connection.NetHandle) => {
      this.triggerRestored();
    });
    netConn.register((error: BusinessError) => {
      if (error) {
        this.netConn = undefined;
        this.registered = false;
        return;
      }
      this.registered = true;
    });
  }

  stopMonitoring(): void {
    const netConn = this.netConn;
    this.netConn = undefined;
    this.onRestored = undefined;
    if (!netConn || !this.registered) {
      this.registered = false;
      return;
    }
    this.registered = false;
    netConn.unregister((_error: BusinessError) => {});
  }

  private async triggerRestored(): Promise<void> {
    if (this.retrying || !this.onRestored) { return; }
    this.retrying = true;
    try {
      await this.onRestored();
    } finally {
      this.retrying = false;
    }
  }
}
