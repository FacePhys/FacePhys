/**
 * FacePhys SDK — 封装 presign → OSS 直传 → 处理的完整流程
 *
 * 用法:
 *   import { FacePhysClient, processVideo } from './facephys-sdk.js';
 *   const result = await processVideo(file, keyId, secretKey);
 */

// ── HMAC-SHA256 签名 ──

async function generateSignature(timestamp, secretKey) {
  const encoder = new TextEncoder();
  const keyData = encoder.encode(secretKey);
  const messageData = encoder.encode(timestamp.toString());

  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );

  const signature = await crypto.subtle.sign('HMAC', cryptoKey, messageData);
  return btoa(String.fromCharCode(...new Uint8Array(signature)));
}

async function makeAuthHeaders(keyId, secretKey) {
  const timestamp = Math.floor(Date.now() / 1000);
  const signature = await generateSignature(timestamp, secretKey);
  return {
    'x-key-id': keyId,
    'x-timestamp': timestamp.toString(),
    'x-signature': signature,
  };
}

// ── 核心流程 ──

// 服务端偶发会把 uploadUrl 路径里的 `/` 编码成 `%2F`，签名却是按 `/` 计算的，
// 直接 PUT 会触发 OSS AccessDenied (403)。这里只对 query 之前的 path 段
// 把 %2F 还原回 /，query 串原样保留以免破坏签名里合法的编码字符。
function normalizeUploadUrl(url) {
  if (typeof url !== 'string') return url;
  const qIdx = url.indexOf('?');
  if (qIdx === -1) return url.replace(/%2F/gi, '/');
  return url.slice(0, qIdx).replace(/%2F/gi, '/') + url.slice(qIdx);
}

function resolveProcessPath(apiVersion) {
  const version = Number(apiVersion);
  if (version === 2) return '/api/v2/video/process';
  if (version === 3) return '/api/v3/video/process';
  throw new Error(`unsupported apiVersion: ${apiVersion}`);
}

/**
 * 视频处理（presign → OSS 直传 → 提交处理）
 *
 * @param {File|Blob} file - 视频文件
 * @param {string} keyId - API Key ID
 * @param {string} secretKey - API Secret Key
 * @param {object} [options]
 * @param {string} [options.baseUrl=''] - API 基础 URL（默认同源）
 * @param {string} [options.capability='rppg'] - 处理能力
 * @param {number} [options.apiVersion=3] - API 版本 (2 或 3)，决定调用哪个 process 端点
 * @param {function} [options.onProgress] - 上传进度回调 (0-100)
 * @returns {Promise<object>} 处理结果
 */
export async function processVideo(file, keyId, secretKey, options = {}) {
  const { baseUrl = '', capability = 'rppg', apiVersion = 3, onProgress } = options;
  const processPath = resolveProcessPath(apiVersion);

  // Step 1: 获取 presign URL（presign 本身没有版本之分，沿用 v2）
  const authHeaders = await makeAuthHeaders(keyId, secretKey);
  const presignResp = await fetch(`${baseUrl}/api/v2/video/upload/presign`, {
    method: 'POST',
    headers: {
      ...authHeaders,
      'Content-Type': 'application/json',
    },
  });

  if (!presignResp.ok) {
    const err = await presignResp.json().catch(() => ({}));
    throw new Error(err.error || `presign failed: ${presignResp.status}`);
  }

  const presignJson = await presignResp.json();
  const uploadUrl = normalizeUploadUrl(presignJson.uploadUrl);
  const { objectKey } = presignJson;

  // Step 2: 直传 OSS
  if (onProgress) {
    // 使用 XMLHttpRequest 以支持进度回调
    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('PUT', uploadUrl);
      xhr.setRequestHeader('Content-Type', 'video/mp4');
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      };
      xhr.onload = () => (xhr.status < 400 ? resolve() : reject(new Error(`OSS upload failed: ${xhr.status}`)));
      xhr.onerror = () => reject(new Error('OSS upload network error'));
      xhr.send(file);
    });
  } else {
    const uploadResp = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': 'video/mp4' },
      body: file,
    });
    if (!uploadResp.ok) {
      throw new Error(`OSS upload failed: ${uploadResp.status}`);
    }
  }

  // Step 3: 提交处理（重新签名，presign 和 upload 可能耗时较长）
  const processHeaders = await makeAuthHeaders(keyId, secretKey);
  const processResp = await fetch(`${baseUrl}${processPath}`, {
    method: 'POST',
    headers: {
      ...processHeaders,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ objectKey, capability }),
  });

  if (!processResp.ok) {
    const err = await processResp.json().catch(() => ({}));
    throw new Error(err.error || `process failed: ${processResp.status}`);
  }

  return processResp.json();
}

export class FacePhysClient {
  constructor({ keyId, secretKey, baseUrl = '', apiVersion = 3, capability = 'rppg' } = {}) {
    if (!keyId) throw new Error('keyId is required');
    if (!secretKey) throw new Error('secretKey is required');
    this.keyId = keyId;
    this.secretKey = secretKey;
    this.baseUrl = baseUrl;
    this.apiVersion = apiVersion;
    this.capability = capability;
  }

  processVideo(file, options = {}) {
    return processVideo(file, this.keyId, this.secretKey, {
      baseUrl: this.baseUrl,
      apiVersion: this.apiVersion,
      capability: this.capability,
      ...options,
    });
  }
}

export { generateSignature, makeAuthHeaders };
