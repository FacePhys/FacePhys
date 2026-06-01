import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * FacePhys SDK — presign → OSS 直传 → 处理
 *
 * <pre>
 * FacePhysClient client = new FacePhysClient("your-key-id", "your-secret", "https://www.facephys.com");
 * String result = client.processVideo("video.mp4");
 * System.out.println(result);
 * </pre>
 *
 * 零外部依赖，仅需 Java 11+。
 */
public class FacePhysClient {

    private final String keyId;
    private final String secretKey;
    private final String baseUrl;
    private final HttpClient http;

    public FacePhysClient(String keyId, String secretKey, String baseUrl) {
        this(keyId, secretKey, baseUrl, 120);
    }

    public FacePhysClient(String keyId, String secretKey, String baseUrl, int timeoutSeconds) {
        this.keyId = keyId;
        this.secretKey = secretKey;
        this.baseUrl = baseUrl.replaceAll("/+$", "");
        this.http = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(timeoutSeconds))
                .build();
    }

    // ── HMAC-SHA256 签名 ──

    private String sign(String timestamp) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secretKey.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            byte[] sig = mac.doFinal(timestamp.getBytes(StandardCharsets.UTF_8));
            return Base64.getEncoder().encodeToString(sig);
        } catch (Exception e) {
            throw new FacePhysException("Failed to generate signature: " + e.getMessage(), 0);
        }
    }

    private Map<String, String> authHeaders() {
        String timestamp = String.valueOf(System.currentTimeMillis() / 1000);
        String signature = sign(timestamp);
        Map<String, String> headers = new LinkedHashMap<>();
        headers.put("x-key-id", keyId);
        headers.put("x-timestamp", timestamp);
        headers.put("x-signature", signature);
        return headers;
    }

    // ── 核心流程 ──

    /**
     * 视频处理：presign → OSS 直传 → 提交处理
     *
     * @param videoPath  视频文件路径
     * @param capability 处理能力（默认 "rppg"）
     * @return 处理结果 JSON 字符串
     */
    public String processVideo(String videoPath, String capability) {
        Path path = Path.of(videoPath);
        if (!Files.isRegularFile(path)) {
            throw new FacePhysException("Video file not found: " + videoPath, 0);
        }

        try {
            // Step 1: 获取 presign URL
            Map<String, String> auth1 = authHeaders();
            HttpRequest.Builder presignBuilder = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + "/api/v2/video/upload/presign"))
                    .POST(HttpRequest.BodyPublishers.noBody())
                    .header("Content-Type", "application/json");
            auth1.forEach(presignBuilder::header);

            HttpResponse<String> presignResp = http.send(presignBuilder.build(),
                    HttpResponse.BodyHandlers.ofString());

            if (presignResp.statusCode() != 200) {
                throw new FacePhysException("presign failed: status " + presignResp.statusCode(),
                        presignResp.statusCode());
            }

            String uploadUrl = extractJsonString(presignResp.body(), "uploadUrl");
            String objectKey = extractJsonString(presignResp.body(), "objectKey");

            if (uploadUrl == null || objectKey == null) {
                throw new FacePhysException("presign response missing uploadUrl or objectKey", 0);
            }
            uploadUrl = normalizeUploadUrl(uploadUrl);

            // Step 2: 直传 OSS
            HttpRequest uploadReq = HttpRequest.newBuilder()
                    .uri(URI.create(uploadUrl))
                    .PUT(HttpRequest.BodyPublishers.ofFile(path))
                    .header("Content-Type", "video/mp4")
                    .build();

            HttpResponse<Void> uploadResp = http.send(uploadReq,
                    HttpResponse.BodyHandlers.discarding());

            if (uploadResp.statusCode() >= 400) {
                throw new FacePhysException("OSS upload failed: " + uploadResp.statusCode(),
                        uploadResp.statusCode());
            }

            // Step 3: 提交处理（重新签名）
            Map<String, String> auth2 = authHeaders();
            String processBody = "{\"objectKey\":\"" + escapeJson(objectKey)
                    + "\",\"capability\":\"" + escapeJson(capability) + "\"}";

            HttpRequest.Builder processBuilder = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + "/api/v3/video/process"))
                    .POST(HttpRequest.BodyPublishers.ofString(processBody))
                    .header("Content-Type", "application/json");
            auth2.forEach(processBuilder::header);

            HttpResponse<String> processResp = http.send(processBuilder.build(),
                    HttpResponse.BodyHandlers.ofString());

            if (processResp.statusCode() != 200) {
                throw new FacePhysException("process failed: status " + processResp.statusCode(),
                        processResp.statusCode());
            }

            return processResp.body();

        } catch (FacePhysException e) {
            throw e;
        } catch (IOException | InterruptedException e) {
            throw new FacePhysException("Network error: " + e.getMessage(), 0);
        }
    }

    /**
     * 视频处理（使用默认 capability "rppg"）
     */
    public String processVideo(String videoPath) {
        return processVideo(videoPath, "rppg");
    }

    // ── 工具方法 ──

    private static final Pattern JSON_STRING_PATTERN = Pattern.compile(
            "\"(%s)\"\\s*:\\s*\"((?:[^\"\\\\]|\\\\.)*)\"");

    private static String extractJsonString(String json, String key) {
        Matcher m = Pattern.compile(
                "\"" + Pattern.quote(key) + "\"\\s*:\\s*\"((?:[^\"\\\\]|\\\\.)*)\""
        ).matcher(json);
        return m.find() ? unescapeJsonString(m.group(1)) : null;
    }

    private static String escapeJson(String s) {
        return s.replace("\\", "\\\\").replace("\"", "\\\"");
    }

    private static String unescapeJsonString(String s) {
        StringBuilder out = new StringBuilder(s.length());
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            if (c != '\\' || i + 1 >= s.length()) {
                out.append(c);
                continue;
            }
            char next = s.charAt(++i);
            switch (next) {
                case '"': out.append('"'); break;
                case '\\': out.append('\\'); break;
                case '/': out.append('/'); break;
                case 'b': out.append('\b'); break;
                case 'f': out.append('\f'); break;
                case 'n': out.append('\n'); break;
                case 'r': out.append('\r'); break;
                case 't': out.append('\t'); break;
                case 'u':
                    if (i + 4 >= s.length()) {
                        out.append("\\u");
                        break;
                    }
                    String hex = s.substring(i + 1, i + 5);
                    try {
                        out.append((char) Integer.parseInt(hex, 16));
                        i += 4;
                    } catch (NumberFormatException e) {
                        out.append("\\u").append(hex);
                        i += 4;
                    }
                    break;
                default:
                    out.append(next);
            }
        }
        return out.toString();
    }

    private static String normalizeUploadUrl(String url) {
        int qIdx = url.indexOf('?');
        if (qIdx < 0) {
            return url.replaceAll("(?i)%2F", "/");
        }
        String path = url.substring(0, qIdx).replaceAll("(?i)%2F", "/");
        return path + url.substring(qIdx);
    }

    // ── 异常类 ──

    public static class FacePhysException extends RuntimeException {
        private final int statusCode;

        public FacePhysException(String message, int statusCode) {
            super(message);
            this.statusCode = statusCode;
        }

        public int getStatusCode() {
            return statusCode;
        }
    }
}
