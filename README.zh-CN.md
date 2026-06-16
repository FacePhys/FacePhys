# FacePhys SDK

[English](README.md) | [简体中文](README.zh-CN.md)

FacePhys 提供基于人脸视频的非接触式生理信号检测能力。本仓库用于对外展示
FacePhys 视频检测 API 的公开 SDK 示例和接入文档。

## SDK

| 语言 | 文件 | 运行环境 |
| --- | --- | --- |
| JavaScript | [`sdks/javascript/facephys-sdk.js`](sdks/javascript/facephys-sdk.js) | 支持 `fetch` 和 Web Crypto 的现代浏览器或运行时 |
| Python | [`sdks/python/facephys_sdk.py`](sdks/python/facephys_sdk.py) | Python 3.8+，依赖 `requests` |
| Java | [`sdks/java/FacePhysClient.java`](sdks/java/FacePhysClient.java) | Java 11+ |
| iOS | [`sdks/ios/RPPGIntegration.swift`](sdks/ios/RPPGIntegration.swift) | iOS 15+，Swift 5.9+（端侧 rPPG） |
| Android | [`sdks/android/RppgIntegration.kt`](sdks/android/RppgIntegration.kt) | Android API 24+（端侧 rPPG） |

## 快速开始

```python
from facephys_sdk import FacePhysClient

client = FacePhysClient(
    key_id="your-key-id",
    secret_key="your-secret-key",
    base_url="https://www.facephys.com",
)

result = client.process_video("/path/to/video.mp4")
cardiac = result["data"].get("cardiac", result["data"])

print("心率:", cardiac["hr"])
print("信号质量:", cardiac.get("sqi"))
```

SDK 会封装标准接入流程：

1. 向 FacePhys 请求预签名上传地址。
2. 将视频直传到对象存储。
3. 提交已上传的视频对象进行生理信号检测。
4. 返回分组检测结果，例如心率、HRV、血压、血氧、情绪、行为、外观和计费字段。

完整接入方式、鉴权说明、各语言示例和视频采集要求，请查看
[`docs/sdk-usage.zh-CN.md`](docs/sdk-usage.zh-CN.md)。完整返回字段定义请查看
[`docs/response-fields.zh-CN.md`](docs/response-fields.zh-CN.md)。

## 端侧移动 SDK（iOS / Android）

除云端视频 API 外，FacePhys 还提供原生端侧 rPPG SDK，直接基于摄像头测量心率、
信号质量、呼吸率和 HRV。以下仅展示**接入（调用）流程**。SDK 主体、加密模型资产
与模型解密套件单独交付。完整调用示例：

- iOS：[`sdks/ios/RPPGIntegration.swift`](sdks/ios/RPPGIntegration.swift)
- Android：[`sdks/android/RppgIntegration.kt`](sdks/android/RppgIntegration.kt)

> `appKey` 须在运行时从你自己的服务端动态下发，**切勿硬编码**。

### iOS

```swift
import RPPGSDK

let manager = RPPGManager()
manager.delegate = self

// 1. 解密随包交付的加密模型资产
let modelData = try ModelDecryptor.decryptBundleResource(named: "step_mini", withExtension: "tflite.enc")
let stateData = try ModelDecryptor.decryptBundleResource(named: "state",     withExtension: "gz.enc")

// 2. 初始化引擎
let config = RPPGConfig.production(
    appKey:     appKeyFromYourServer,
    modelBytes: modelData,
    stateBytes: stateData
)
manager.initialize(config: config) { result in /* 处理 result.success / result.error */ }

// 3. 逐帧送入（如来自 AVCaptureVideoDataOutput）
manager.processFrame(pixelBuffer, faceRect: faceRect, timestamp: ts)

// 4. 通过 RPPGDelegate 在主线程接收结果
func rppgDidCalculateResult(hr: Int, sqi: Float, breathingRate: Int?) { /* HR 约 30s 起，每 ~5s */ }
func rppgDidUpdateHrv(rrBpm: Float?, sdnnMs: Float?, rmssdMs: Float?, ibiCount: Int) { /* HRV 约 60s 起 */ }

// 5. 切换被测者时重置 / 结束时释放
manager.reset()
manager.release()
```

### Android

```kotlin
import com.stepmini.rppg.RppgConfig
import com.stepmini.rppg.RppgEngineImpl

private val engine = RppgEngineImpl()

// 1. 解密随包交付的加密模型资产（IO 线程）
val modelBytes = ModelDecryptor.decryptAsset(context, "step_mini.tflite.enc")
val stateBytes = ModelDecryptor.decryptAsset(context, "state_mini.bin.enc")

// 2. 初始化引擎
val result = engine.initializeAsync(
    RppgConfig(
        context    = context,
        modelBytes = modelBytes,
        stateBytes = stateBytes,
        appKey     = appKeyFromYourServer,
        sdkInitUrl = "https://www.facephys.com/endsdk/sdk/init"
    )
)
// 处理 result.success / result.code / result.message

// 3. 逐帧送入（如来自 CameraX ImageAnalysis）
val frameResult = engine.processFrame(rgb9x9, dtSec)

// 4. 数据积累足够后（约 30s）查询指标
if (frameResult.hasEnoughWindow) {
    val metrics = engine.getMetricsAsync()
    val hr  = metrics.hr    // Float?（null 表示数据不足）
    val sqi = metrics.sqi   // Float?（0~1）
}

// 5. 结束时释放
engine.release()
```

## 安全说明

不要在公开前端代码中暴露长期有效的 API Secret。生产环境浏览器接入建议通过自有后端
调用 FacePhys，或为受控客户端签发短期、低额度的子密钥。


[![Star History Chart](https://api.star-history.com/chart?repos=FacePhys/FacePhys&type=date&legend=top-left)](https://www.star-history.com/?repos=FacePhys%2FFacePhys&type=date&legend=bottom-right)