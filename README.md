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
language examples, video requirements, and response field descriptions.

## Security

Do not expose long-lived API secrets in public frontend code. For production
browser integrations, call FacePhys from your own backend or issue short-lived,
low-quota keys for controlled client-side use.
