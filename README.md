# FacePhys SDKs

[English](README.md) | [简体中文](README.zh-CN.md)

FacePhys provides contactless physiological signal detection from face videos.
This repository contains public SDK examples for integrating the FacePhys video
detection API.

## SDKs

| Language | File | Runtime |
| --- | --- | --- |
| JavaScript | [`sdks/javascript/facephys-sdk.js`](sdks/javascript/facephys-sdk.js) | Modern browser or runtime with `fetch` and Web Crypto |
| Python | [`sdks/python/facephys_sdk.py`](sdks/python/facephys_sdk.py) | Python 3.8+ with `requests` |
| Java | [`sdks/java/FacePhysClient.java`](sdks/java/FacePhysClient.java) | Java 11+ |
| iOS | [`sdks/ios/RPPGIntegration.swift`](sdks/ios/RPPGIntegration.swift) | iOS 15+, Swift 5.9+ (on-device rPPG) |
| Android | [`sdks/android/RppgIntegration.kt`](sdks/android/RppgIntegration.kt) | Android API 24+ (on-device rPPG) |

## Quick Start

```python
from facephys_sdk import FacePhysClient

client = FacePhysClient(
    key_id="your-key-id",
    secret_key="your-secret-key",
    base_url="https://www.facephys.com",
)

result = client.process_video("/path/to/video.mp4")
cardiac = result["data"].get("cardiac", result["data"])

print("Heart rate:", cardiac["hr"])
print("Signal quality:", cardiac.get("sqi"))
```

The SDKs handle the standard integration flow:

1. Request a presigned upload URL from FacePhys.
2. Upload the video directly to object storage.
3. Submit the uploaded object for physiological signal detection.
4. Return grouped detection results such as heart rate, HRV, blood pressure,
   SpO2, emotion, behavior, appearance, and billing fields.

See [`docs/sdk-usage.md`](docs/sdk-usage.md) for setup, authentication,
language examples, and video requirements. For full response field definitions,
see [`docs/response-fields.md`](docs/response-fields.md).

## On-device Mobile SDKs (iOS / Android)

In addition to the cloud video API, FacePhys ships native on-device rPPG SDKs that
measure heart rate, signal quality, breathing rate and HRV directly from the camera.
The snippets below show only the **integration (calling) flow**. The SDK binaries,
encrypted model assets and model-decryption kits are delivered separately. Full
calling examples:

- iOS: [`sdks/ios/RPPGIntegration.swift`](sdks/ios/RPPGIntegration.swift)
- Android: [`sdks/android/RppgIntegration.kt`](sdks/android/RppgIntegration.kt)

> `appKey` must be fetched from your own server at runtime — never hard-code it.

### iOS

```swift
import RPPGSDK

let manager = RPPGManager()
manager.delegate = self

// 1. Decrypt the bundled, encrypted model assets
let modelData = try ModelDecryptor.decryptBundleResource(named: "step_mini", withExtension: "tflite.enc")
let stateData = try ModelDecryptor.decryptBundleResource(named: "state",     withExtension: "gz.enc")

// 2. Initialize the engine
let config = RPPGConfig.production(
    appKey:     appKeyFromYourServer,
    modelBytes: modelData,
    stateBytes: stateData
)
manager.initialize(config: config) { result in /* handle result.success / result.error */ }

// 3. Feed each camera frame (e.g. from AVCaptureVideoDataOutput)
manager.processFrame(pixelBuffer, faceRect: faceRect, timestamp: ts)

// 4. Receive results on the main thread via RPPGDelegate
func rppgDidCalculateResult(hr: Int, sqi: Float, breathingRate: Int?) { /* HR ~30s, every ~5s */ }
func rppgDidUpdateHrv(rrBpm: Float?, sdnnMs: Float?, rmssdMs: Float?, ibiCount: Int) { /* HRV ~60s */ }

// 5. Reset between subjects / release when done
manager.reset()
manager.release()
```

### Android

```kotlin
import com.stepmini.rppg.RppgConfig
import com.stepmini.rppg.RppgEngineImpl

private val engine = RppgEngineImpl()

// 1. Decrypt the bundled, encrypted model assets (IO thread)
val modelBytes = ModelDecryptor.decryptAsset(context, "step_mini.tflite.enc")
val stateBytes = ModelDecryptor.decryptAsset(context, "state_mini.bin.enc")

// 2. Initialize the engine
val result = engine.initializeAsync(
    RppgConfig(
        context    = context,
        modelBytes = modelBytes,
        stateBytes = stateBytes,
        appKey     = appKeyFromYourServer,
        sdkInitUrl = "https://www.facephys.com/endsdk/sdk/init"
    )
)
// handle result.success / result.code / result.message

// 3. Feed each camera frame (e.g. from CameraX ImageAnalysis)
val frameResult = engine.processFrame(rgb9x9, dtSec)

// 4. Query metrics once enough data is collected (~30s)
if (frameResult.hasEnoughWindow) {
    val metrics = engine.getMetricsAsync()
    val hr  = metrics.hr    // Float? (null when data insufficient)
    val sqi = metrics.sqi   // Float? (0..1)
}

// 5. Release when done
engine.release()
```

## Security

Do not expose long-lived API secrets in public frontend code. For production
browser integrations, call FacePhys from your own backend or issue short-lived,
low-quota keys for controlled client-side use.
