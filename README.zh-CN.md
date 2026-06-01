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

完整接入方式、鉴权说明、各语言示例、视频采集要求和返回结果说明，请查看
[`docs/sdk-usage.zh-CN.md`](docs/sdk-usage.zh-CN.md)。

## 安全说明

不要在公开前端代码中暴露长期有效的 API Secret。生产环境浏览器接入建议通过自有后端
调用 FacePhys，或为受控客户端签发短期、低额度的子密钥。

