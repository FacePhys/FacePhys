"""
FacePhys SDK — 封装 presign → OSS 直传 → 处理的完整流程

用法:
    from facephys_sdk import FacePhysClient

    client = FacePhysClient(key_id="your-key", secret_key="your-secret")
    result = client.process_video("/path/to/video.mp4")
    print(result)
"""

import base64
import hashlib
import hmac
import time
from pathlib import Path
from typing import Any, Dict, Optional

import requests


class FacePhysError(Exception):
    """FacePhys API 错误"""

    def __init__(self, message: str, status_code: int = 0):
        super().__init__(message)
        self.status_code = status_code


class FacePhysClient:
    """FacePhys API 客户端"""

    def __init__(
        self,
        key_id: str,
        secret_key: str,
        base_url: str = "https://www.facephys.com",
        timeout: int = 120,
    ):
        """
        Args:
            key_id: API Key ID
            secret_key: API Secret Key
            base_url: API 基础 URL
            timeout: 请求超时秒数
        """
        self.key_id = key_id
        self.secret_key = secret_key
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def _sign(self, timestamp: str) -> str:
        """生成 HMAC-SHA256 签名"""
        sig = hmac.new(
            self.secret_key.encode("utf-8"),
            timestamp.encode("utf-8"),
            hashlib.sha256,
        ).digest()
        return base64.b64encode(sig).decode()

    def _auth_headers(self) -> Dict[str, str]:
        """生成认证 headers"""
        timestamp = str(int(time.time()))
        signature = self._sign(timestamp)
        return {
            "x-key-id": self.key_id,
            "x-timestamp": timestamp,
            "x-signature": signature,
        }

    def _normalize_upload_url(self, url: str) -> str:
        """还原 uploadUrl path 中的 %2F，避免 OSS 签名路径不一致导致 403"""
        if not isinstance(url, str):
            return url
        path_part, sep, query = url.partition("?")
        normalized_path = path_part.replace("%2F", "/").replace("%2f", "/")
        return normalized_path + (sep + query if sep else "")

    def process_video(
        self,
        video_path: str,
        capability: str = "rppg",
    ) -> Dict[str, Any]:
        """
        视频处理完整流程：presign → OSS 直传 → 提交处理

        Args:
            video_path: 视频文件路径
            capability: 处理能力 (默认 "rppg")

        Returns:
            处理结果 dict

        Raises:
            FacePhysError: API 调用失败
            FileNotFoundError: 视频文件不存在
        """
        path = Path(video_path)
        if not path.is_file():
            raise FileNotFoundError(f"video file not found: {video_path}")

        # Step 1: 获取 presign URL
        presign_resp = requests.post(
            f"{self.base_url}/api/v2/video/upload/presign",
            headers=self._auth_headers(),
            timeout=self.timeout,
        )
        if presign_resp.status_code != 200:
            try:
                err = presign_resp.json().get("error", f"status {presign_resp.status_code}")
            except ValueError:
                err = presign_resp.text[:200] or f"status {presign_resp.status_code}"
            raise FacePhysError(f"presign failed: {err}", presign_resp.status_code)

        presign_data = presign_resp.json()
        upload_url = self._normalize_upload_url(presign_data["uploadUrl"])
        object_key = presign_data["objectKey"]

        # Step 2: 直传 OSS
        with open(video_path, "rb") as f:
            upload_resp = requests.put(
                upload_url,
                data=f,
                headers={"Content-Type": "video/mp4"},
                timeout=self.timeout,
            )
        if upload_resp.status_code >= 400:
            raise FacePhysError(
                f"OSS upload failed: {upload_resp.status_code}",
                upload_resp.status_code,
            )

        # Step 3: 提交处理（重新签名，上传可能耗时较长）
        process_resp = requests.post(
            f"{self.base_url}/api/v3/video/process",
            headers={**self._auth_headers(), "Content-Type": "application/json"},
            json={"objectKey": object_key, "capability": capability},
            timeout=self.timeout,
        )
        if process_resp.status_code != 200:
            try:
                err = process_resp.json().get("error", f"status {process_resp.status_code}")
            except ValueError:
                err = process_resp.text[:200] or f"status {process_resp.status_code}"
            raise FacePhysError(f"process failed: {err}", process_resp.status_code)

        return process_resp.json()
