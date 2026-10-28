// 老人位置坐标（device_location 轨迹、device 最新位置、alarm_precess/alarm_process
// 告警坐标）属 PIPL 敏感个人信息：写入时以 securestore AEAD 密文落库，读取时解密。
// 未配置密钥（config.Validate 已在生产阻断该组合）与历史明文行原样读写，无需回填迁移。
// fence_event.lng/lat 是 double 数值列且 API 按数字消费，保持明文（见审计报告 M-3）。
package handlers

import (
	"strings"

	"suining-care/internal/securestore"
)

const coordinateScope = "location"

// sealCoordinate 加密坐标字符串；空值原样返回。
// crypto/rand.Read 不会失败，Seal 仅在 box 为 nil 时报错，此处已排除。
func sealCoordinate(box *securestore.Box, v string) string {
	if box == nil || v == "" {
		return v
	}
	sealed, _ := box.Seal(v, coordinateScope)
	return sealed
}

// openCoordinate 解密坐标；nil、空值与无密文前缀的历史明文行原样返回。
func openCoordinate(box *securestore.Box, v *string) (*string, error) {
	if v == nil || *v == "" || !strings.HasPrefix(*v, securestore.Prefix) {
		return v, nil
	}
	plain, err := box.Open(*v, coordinateScope)
	if err != nil {
		return nil, err
	}
	return &plain, nil
}

func openCoordinates(box *securestore.Box, lng, lat *string) (*string, *string, error) {
	lng, err := openCoordinate(box, lng)
	if err != nil {
		return nil, nil, err
	}
	lat, err = openCoordinate(box, lat)
	if err != nil {
		return nil, nil, err
	}
	return lng, lat, nil
}

// coordinateArg 供 INSERT 使用：优先取本次上报坐标，否则取设备最近坐标（已解密），
// 两者皆无时写入 NULL，与原 nullableCoordinate 语义一致。
func coordinateArg(box *securestore.Box, v string, fallback *string) any {
	if v != "" {
		return sealCoordinate(box, v)
	}
	if fallback == nil || *fallback == "" {
		return nil
	}
	return sealCoordinate(box, *fallback)
}
