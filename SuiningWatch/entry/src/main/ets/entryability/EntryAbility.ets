import { UIAbility } from '@kit.AbilityKit';
import { window } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';

export default class EntryAbility extends UIAbility {
  onWindowStageCreate(stage: window.WindowStage): void {
    try {
      stage.loadContent('pages/Index', (error) => {
        if (error.code) {
          hilog.error(0x0001, 'SuiningWatch', 'loadContent failed: %{public}d', error.code);
          return;
        }
        hilog.info(0x0001, 'SuiningWatch', 'Index page loaded');
      });
    } catch (error) {
      hilog.error(0x0001, 'SuiningWatch', 'Unable to initialize window');
    }
  }
}
